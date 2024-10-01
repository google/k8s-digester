// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package resolve looks up image references in resources and resolves tags to
// digests using the `crane` package from `go-containerregistry`.
package resolve

import (
	"context"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"strings"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/google/k8s-digester/pkg/keychain"
	"github.com/google/k8s-digester/pkg/version"
)

var resolveTagFn = resolveTag // override for unit testing

// ImageTags looks up the digest and adds it to the image field
// for containers and initContainers in pods and pod template specs.
//
// It looks for image fields with tags in these sequence nodes:
// - `spec.containers`
// - `spec.initContainers`
// - `spec.template.spec.containers`
// - `spec.template.spec.initContainers`
// - `spec.jobTemplate.spec.template.spec.containers`
// - `spec.jobTemplate.spec.template.spec.initContainers`
// The `config` input parameter can be null. In this case, the function
// will not attempt to retrieve imagePullSecrets from the cluster.
func ImageTags(ctx context.Context, log logr.Logger, config *rest.Config, n *yaml.RNode, skipPrefixes []string, platform string) error {
	kc, err := keychain.Create(ctx, log, config, n)
	if err != nil {
		return fmt.Errorf("could not create keychain: %w", err)
	}
	imageTagFilter := &ImageTagFilter{
		Log:          log,
		Keychain:     kc,
		SkipPrefixes: &skipPrefixes,
		Platform:     platform,
	}
	// if input is a CronJob, we need to look up the image tags in the
	// `spec.jobTemplate.spec.template.spec` path as well
	if n.GetKind() == "CronJob" {
		return n.PipeE(
			yaml.Lookup("spec", "jobTemplate", "spec", "template", "spec"),
			yaml.Tee(yaml.Lookup("containers"), imageTagFilter),
			yaml.Tee(yaml.Lookup("initContainers"), imageTagFilter),
		)
	}
	// otherwise, we look up the image tags in the `spec.template.spec` path
	return n.PipeE(
		yaml.Lookup("spec"),
		yaml.Tee(yaml.Lookup("containers"), imageTagFilter),
		yaml.Tee(yaml.Lookup("initContainers"), imageTagFilter),
		yaml.Lookup("template", "spec"),
		yaml.Tee(yaml.Lookup("containers"), imageTagFilter),
		yaml.Tee(yaml.Lookup("initContainers"), imageTagFilter),
	)
}

// ImageTagFilter resolves image tags to digests
type ImageTagFilter struct {
	Log          logr.Logger
	Keychain     authn.Keychain
	SkipPrefixes *[]string
	Platform     string
}

var _ yaml.Filter = &ImageTagFilter{}

// Filter to resolve image tags to digests for a list of containers
func (f *ImageTagFilter) Filter(n *yaml.RNode) (*yaml.RNode, error) {
	if err := n.VisitElements(f.filterImage); err != nil {
		return nil, err
	}
	return n, nil
}

func (f *ImageTagFilter) filterImage(n *yaml.RNode) error {
	imageNode, err := n.Pipe(yaml.Lookup("image"))
	if err != nil {
		s, _ := n.String()
		return fmt.Errorf("could not lookup image in node %v: %w", s, err)
	}
	image := yaml.GetValue(imageNode)
	for _, prefix := range *f.SkipPrefixes {
		if strings.HasPrefix(image, prefix) {
			// Image should be excluded from digest resolution
			return nil
		}
	}
	if strings.Contains(image, "@") {
		return nil // already has digest, skip
	}
	digest, err := resolveTagFn(image, f.Platform, f.Keychain)
	if err != nil {
		return fmt.Errorf("could not get digest for %s: %w", image, err)
	}
	f.Log.V(1).Info("resolved tag to digest", "image", image, "digest", digest)
	imageWithDigest := fmt.Sprintf("%s@%s", image, digest)
	if _, err := n.Pipe(yaml.Lookup("image"), yaml.Set(yaml.NewStringRNode(imageWithDigest))); err != nil {
		return fmt.Errorf("could not set image: %w", err)
	}
	return nil
}

func resolveTag(image string, platform string, keychain authn.Keychain) (string, error) {
	opts := []crane.Option{
		crane.WithAuthFromKeychain(keychain),
		crane.WithUserAgent(fmt.Sprintf("cloud-solutions/%s-%s", "k8s-digester", version.Version)),
	}
	if platform != "" {
		pf, err := v1.ParsePlatform(platform)
		if err != nil {
			return "", err
		}
		opts = append(opts, crane.WithPlatform(pf))
	}
	return crane.Digest(image, opts...)
}
