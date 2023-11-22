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

package resolve

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/google/k8s-digester/pkg/logging"
)

type M map[string]interface{}

type anonymousKeychain struct{}

var _ authn.Keychain = &anonymousKeychain{}

func (kc *anonymousKeychain) Resolve(_ authn.Resource) (authn.Authenticator, error) {
	return authn.Anonymous, nil
}

var (
	ctx    = context.Background()
	log    = logging.CreateDiscardLogger()
	filter = &ImageTagFilter{
		Log:          log,
		Keychain:     &anonymousKeychain{},
		SkipPrefixes: &[]string{},
	}
)

func init() {
	// Implementation of resolveTagFn that computes the SHA-256 sum of the
	// image name. We could make this simpler since it's just for testing, but
	// it means the digest values have the same 'shape' as real values.
	resolveTagFn = func(image string, _ authn.Keychain) (string, error) {
		if image == "" || image == "error" {
			return "", fmt.Errorf("intentional error resolving image [%s]", image)
		}
		sumBytes := sha256.Sum256([]byte(image))
		digest := strings.TrimRight(hex.EncodeToString(sumBytes[:]), "\r\n")
		return fmt.Sprintf("sha256:%s", digest), nil
	}
}

func TestSkip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

func Test_ImageTagFilter_filterImage_Container(t *testing.T) {
	imageTag := "registry.example.com/repository/image:tag"
	node, err := yaml.FromMap(M{
		"name":  "test-container-image",
		"image": imageTag,
	})
	if err != nil {
		t.Fatalf("could not create container node: %v", err)
	}

	if err := filter.filterImage(node); err != nil {
		t.Fatalf("unexpected Filter error: %v", err)
	}

	imageNode, err := node.Pipe(yaml.Lookup("image"))
	if err != nil {
		t.Fatalf("could not get image field from node: %s: %v", node.MustString(), err)
	}
	gotImage := yaml.GetValue(imageNode)
	// wantDigest=$(echo -n "registry.example.com/repository/image:tag" | shasum -a 256 | cut -d' ' -f1)
	wantDigest := "sha256:90dc9a6dfb6f86fe35508c9e94d255922bce69c3d5c520b29d65685fab4ee18d"
	wantImage := fmt.Sprintf("%s@%s", imageTag, wantDigest)
	if wantImage != gotImage {
		t.Errorf("wanted [%s], got [%s]", wantImage, gotImage)
	}
}

func Test_ImageTags_Pod(t *testing.T) {
	node, err := createPodNode([]string{"image0", "image1"}, []string{"image2", "image3"})
	if err != nil {
		t.Fatalf("could not create pod node: %v", err)
	}

	if err := ImageTags(ctx, log, nil, node, []string{}); err != nil {
		t.Fatalf("problem resolving image tags: %v", err)
	}
	t.Log(node.MustString())

	assertContainer(t, node, "image0@sha256:07d7d43fe9dd151e40f0a8d54c5211a8601b04e4a8fa7ad57ea5e73e4ffa7e4a", "spec", "containers", "[name=container0]")
	assertContainer(t, node, "image1@sha256:cc292b92ce7f10f2e4f727ecdf4b12528127c51b6ddf6058e213674603190d06", "spec", "containers", "[name=container1]")
	assertContainer(t, node, "image2@sha256:5bb21ac469b5e7df4e17899d4aae0adfb430f0f0b336a2242ef1a22d25bd2e53", "spec", "initContainers", "[name=initcontainer0]")
	assertContainer(t, node, "image3@sha256:b0542da3f90bad69318e16ec7fcb6b13b089971886999e08bec91cea34891f0f", "spec", "initContainers", "[name=initcontainer1]")
}

func Test_ImageTags_Pod_Skip_Prefixes(t *testing.T) {
	node, err := createPodNode([]string{"image0", "skip1.local/image1"}, []string{"image2", "skip2.local/image3"})
	if err != nil {
		t.Fatalf("could not create pod node: %v", err)
	}

	if err := ImageTags(ctx, log, nil, node, []string{"skip1.local", "skip2.local"}); err != nil {
		t.Fatalf("problem resolving image tags: %v", err)
	}
	t.Log(node.MustString())

	assertContainer(t, node, "image0@sha256:07d7d43fe9dd151e40f0a8d54c5211a8601b04e4a8fa7ad57ea5e73e4ffa7e4a", "spec", "containers", "[name=container0]")
	assertContainer(t, node, "skip1.local/image1", "spec", "containers", "[name=container1]")
	assertContainer(t, node, "image2@sha256:5bb21ac469b5e7df4e17899d4aae0adfb430f0f0b336a2242ef1a22d25bd2e53", "spec", "initContainers", "[name=initcontainer0]")
	assertContainer(t, node, "skip2.local/image3", "spec", "initContainers", "[name=initcontainer1]")
}

func Test_ImageTags_CronJob(t *testing.T) {
	node, err := createCronJobNode([]string{"image0", "image1"}, []string{"image2", "image3"})
	if err != nil {
		t.Fatalf("could not create pod node: %v", err)
	}

	if err := ImageTags(ctx, log, nil, node, []string{}); err != nil {
		t.Fatalf("problem resolving image tags: %v", err)
	}
	t.Log(node.MustString())

	assertContainer(t, node, "image0@sha256:07d7d43fe9dd151e40f0a8d54c5211a8601b04e4a8fa7ad57ea5e73e4ffa7e4a", "spec", "jobTemplate", "spec", "template", "spec", "containers", "[name=container0]")
	assertContainer(t, node, "image1@sha256:cc292b92ce7f10f2e4f727ecdf4b12528127c51b6ddf6058e213674603190d06", "spec", "jobTemplate", "spec", "template", "spec", "containers", "[name=container1]")
	assertContainer(t, node, "image2@sha256:5bb21ac469b5e7df4e17899d4aae0adfb430f0f0b336a2242ef1a22d25bd2e53", "spec", "jobTemplate", "spec", "template", "spec", "initContainers", "[name=initcontainer0]")
	assertContainer(t, node, "image3@sha256:b0542da3f90bad69318e16ec7fcb6b13b089971886999e08bec91cea34891f0f", "spec", "jobTemplate", "spec", "template", "spec", "initContainers", "[name=initcontainer1]")
}

func Test_ImageTags_Deployment(t *testing.T) {
	node, err := createDeploymentNode([]string{"image0", "image1"}, []string{"image2", "image3"})
	if err != nil {
		t.Fatalf("could not create deployment node: %v", err)
	}

	if err := ImageTags(ctx, log, nil, node, []string{}); err != nil {
		t.Fatalf("problem resolving image tags: %v", err)
	}
	t.Log(node.MustString())

	assertContainer(t, node, "image0@sha256:07d7d43fe9dd151e40f0a8d54c5211a8601b04e4a8fa7ad57ea5e73e4ffa7e4a", "spec", "template", "spec", "containers", "[name=container0]")
	assertContainer(t, node, "image1@sha256:cc292b92ce7f10f2e4f727ecdf4b12528127c51b6ddf6058e213674603190d06", "spec", "template", "spec", "containers", "[name=container1]")
	assertContainer(t, node, "image2@sha256:5bb21ac469b5e7df4e17899d4aae0adfb430f0f0b336a2242ef1a22d25bd2e53", "spec", "template", "spec", "initContainers", "[name=initcontainer0]")
	assertContainer(t, node, "image3@sha256:b0542da3f90bad69318e16ec7fcb6b13b089971886999e08bec91cea34891f0f", "spec", "template", "spec", "initContainers", "[name=initcontainer1]")
}

func assertContainer(t *testing.T, n *yaml.RNode, imageWithDigest string, path ...string) {
	container, err := n.Pipe(yaml.Lookup(path...), yaml.Get("image"))
	if err != nil {
		t.Fatalf("could not find container-0: %v", err)
	}
	got := yaml.GetValue(container)
	want := imageWithDigest
	if want != got {
		t.Errorf("wanted [%s], got [%s]", want, got)
	}
}
