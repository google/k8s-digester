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

// Package keychain creates credentials for authenticating to container image registries.
package keychain

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/authn/k8schain"
	"github.com/google/go-containerregistry/pkg/v1/google"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var createClientFn = createClient // override for testing

// Create a multi keychain based in input arguments
func Create(ctx context.Context, log logr.Logger, config *rest.Config, n *yaml.RNode) (authn.Keychain, error) {
	if config == nil {
		log.V(1).Info("creating offline keychain")
		return authn.NewMultiKeychain(google.Keychain, authn.DefaultKeychain), nil
	}
	client, err := createClientFn(config)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes Clientset: %w", err)
	}
	kkc, err := createK8schain(ctx, log, client, n)
	if err != nil {
		return nil, fmt.Errorf("could not create k8schain: %w", err)
	}
	log.V(1).Info("creating k8s keychain")
	return authn.NewMultiKeychain(kkc, authn.DefaultKeychain), nil
}

func createClient(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}

func createK8schain(ctx context.Context, log logr.Logger, client kubernetes.Interface, n *yaml.RNode) (authn.Keychain, error) {
	var namespace string
	namespaceNode, err := n.Pipe(yaml.Lookup("metadata", "namespace"))
	if err == nil {
		namespace = yaml.GetValue(namespaceNode)
	}
	var serviceAccountName string
	podServiceAccountNameNode, err := n.Pipe(yaml.Lookup("spec", "serviceAccountName"))
	if err == nil {
		serviceAccountName = yaml.GetValue(podServiceAccountNameNode)
	}
	if serviceAccountName == "" {
		podTemplateServiceAccountNameNode, err := n.Pipe(yaml.Lookup("spec", "template", "spec", "serviceAccountName"))
		if err == nil {
			serviceAccountName = yaml.GetValue(podTemplateServiceAccountNameNode)
		}
	}
	var imagePullSecrets []string
	podImagePullSecretsNode, err := n.Pipe(yaml.Lookup("spec", "imagePullSecrets"))
	if err == nil {
		imagePullSecrets, _ = podImagePullSecretsNode.ElementValues("name")
	}
	if len(imagePullSecrets) == 0 {
		podTemplateImagePullSecretsNode, err := n.Pipe(yaml.Lookup("spec", "template", "spec", "imagePullSecrets"))
		if err == nil {
			imagePullSecrets, _ = podTemplateImagePullSecretsNode.ElementValues("name")
		}
	}
	log.V(1).Info("creating k8schain",
		"namespace", namespace,
		"serviceAccountName", serviceAccountName,
		"imagePullSecrets", imagePullSecrets)
	return k8schain.New(ctx, client, k8schain.Options{
		Namespace:          namespace,          // k8schain defaults to "default" if empty
		ServiceAccountName: serviceAccountName, // k8schain defaults to "default" if empty
		ImagePullSecrets:   imagePullSecrets,
	})
}
