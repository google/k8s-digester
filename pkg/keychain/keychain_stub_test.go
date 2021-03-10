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

package keychain

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func createFakeClient() (kubernetes.Interface, error) {
	return createClientFn(nil)
}

func createDockerConfigSecret(ctx context.Context, client kubernetes.Interface, namespace, secretName, registry, username, password string) error {
	secretData := map[string]interface{}{
		"auths": map[string]interface{}{
			registry: map[string]string{
				"auth": base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", username, password))),
			},
		},
	}
	secretBytes, err := json.Marshal(secretData)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: secretBytes,
		},
	}
	_, err = client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func createServiceAccount(ctx context.Context, client kubernetes.Interface, namespace, serviceAccountName string, imagePullSecrets ...string) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceAccountName,
		},
	}
	for _, secret := range imagePullSecrets {
		serviceAccount.ImagePullSecrets = append(serviceAccount.ImagePullSecrets,
			corev1.LocalObjectReference{Name: secret})
	}
	_, err := client.CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

type M map[string]interface{}

func createPodNode(namespace, serviceAccountName, imageTag string, imagePullSecrets ...string) (*yaml.RNode, error) {
	node, err := yaml.FromMap(M{
		"apiVersion": "v1",
		"kind":       "Pod",
		"metadata": M{
			"name": "test-pod",
		},
		"spec": M{
			"containers": []M{
				{
					"name":  "test-image",
					"image": "example.com/repository/image",
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if err := node.PipeE(
		yaml.LookupCreate(yaml.ScalarNode, "metadata", "namespace"),
		yaml.FieldSetter{StringValue: namespace},
	); err != nil {
		return nil, err
	}
	if err := node.PipeE(
		yaml.LookupCreate(yaml.ScalarNode, "spec", "serviceAccountName"),
		yaml.FieldSetter{StringValue: serviceAccountName},
	); err != nil {
		return nil, err
	}
	if err := node.PipeE(
		yaml.Lookup("spec", "containers", "[name=test-image]"),
		yaml.FieldSetter{
			Name:        "image",
			StringValue: imageTag,
		},
	); err != nil {
		return nil, err
	}
	for _, secret := range imagePullSecrets {
		if err := node.PipeE(
			yaml.LookupCreate(yaml.SequenceNode, "spec", "imagePullSecrets"),
			yaml.Append(yaml.NewMapRNode(&map[string]string{"name": secret}).YNode()),
		); err != nil {
			return nil, err
		}
	}
	return node, nil
}

func createDeploymentNode(namespace, serviceAccountName, imageTag string, imagePullSecrets ...string) (*yaml.RNode, error) {
	node, err := yaml.FromMap(M{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": M{
			"name": "test-deployment",
		},
		"spec": M{
			"replicas": 1,
			"selector": M{
				"matchLabels": M{
					"app": "test",
				},
			},
			"template": M{
				"metadata": M{
					"labels": M{
						"app": "test",
					},
				},
				"spec": M{
					"containers": []M{
						{
							"name":  "test-image",
							"image": "example.com/repository/image",
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if err := node.PipeE(
		yaml.LookupCreate(yaml.ScalarNode, "metadata", "namespace"),
		yaml.FieldSetter{StringValue: namespace},
	); err != nil {
		return nil, err
	}
	if err := node.PipeE(
		yaml.LookupCreate(yaml.ScalarNode, "spec", "template", "spec", "serviceAccountName"),
		yaml.FieldSetter{StringValue: serviceAccountName},
	); err != nil {
		return nil, err
	}
	if err := node.PipeE(
		yaml.Lookup("spec", "template", "spec", "containers", "[name=test-image]"),
		yaml.FieldSetter{
			Name:        "image",
			StringValue: imageTag,
		},
	); err != nil {
		return nil, err
	}
	for _, secret := range imagePullSecrets {
		if err := node.PipeE(
			yaml.LookupCreate(yaml.SequenceNode, "spec", "template", "spec", "imagePullSecrets"),
			yaml.Append(yaml.NewMapRNode(&map[string]string{"name": secret}).YNode()),
		); err != nil {
			return nil, err
		}
	}
	return node, nil
}
