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
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/google/k8s-digester/pkg/logging"
)

var (
	ctx = context.Background()
	// log = logging.CreateDiscardLogger()
	log = logging.CreateStdLogger("keychain_test")

	authConfigComparer = cmp.Comparer(func(l, r *authn.AuthConfig) bool {
		return l.Username == r.Username && l.Password == r.Password
	})
)

func init() {
	createClientFn = func(_ *rest.Config) (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(), nil
	}
}

func Test_Create_Keychain_for_nil_config(t *testing.T) {
	kc, err := Create(ctx, log, nil, nil)
	if err != nil {
		t.Fatalf("error creating keychain: %v", err)
	}
	if kc == nil {
		t.Errorf("expected keychain, got nil")
	}
}

func Test_createK8schain_ImagePullSecretOnPod(t *testing.T) {
	namespace := "test-ns"
	serviceAccountName := "test-sa"
	secretName := "test-secret"
	registry := "registry.example.com"
	username := "username"
	password := "password"
	imageTag := "registry.example.com/repository/image:tag"
	client, err := createFakeClient()
	if err != nil {
		t.Fatalf("error creating fake Kubernetes client: %v", err)
	}
	if err := createDockerConfigSecret(ctx, client, namespace, "wrong-secret", "wrong.example.com", "foo", "bar"); err != nil {
		t.Fatalf("error creating secret: %v", err)
	}
	if err := createDockerConfigSecret(ctx, client, namespace, secretName, registry, username, password); err != nil {
		t.Fatalf("error creating secret: %v", err)
	}
	if err := createServiceAccount(ctx, client, namespace, serviceAccountName); err != nil {
		t.Fatalf("error creating service account: %v", err)
	}
	node, err := createPodNode(namespace, serviceAccountName, imageTag, "wrong-secret", secretName)
	if err != nil {
		t.Fatalf("error creating pod node: %v", err)
	}
	t.Logf("%s", node.MustString())

	kc, err := createK8schain(ctx, log, client, node)
	if err != nil {
		t.Fatalf("error creating k8schain: %v", err)
	}

	authConfig, err := getAuthConfig(kc, imageTag)
	if err != nil {
		t.Fatalf("error getting authconfig: %v", err)
	}
	want := &authn.AuthConfig{
		Username: username,
		Password: password,
	}
	if diff := cmp.Diff(want, authConfig, authConfigComparer); diff != "" {
		t.Errorf("createK8schain() authConfig mismatch (-want +got):\n%s", diff)
	}
}

func Test_createK8schain_ImagePullSecretOnPodTemplateSpec(t *testing.T) {
	namespace := "test-ns"
	serviceAccountName := "test-sa"
	secretName := "test-secret"
	registry := "registry.example.com"
	username := "username"
	password := "password"
	imageTag := "registry.example.com/repository/image:tag"
	client, err := createFakeClient()
	if err != nil {
		t.Fatalf("error creating fake Kubernetes client: %v", err)
	}
	if err := createDockerConfigSecret(ctx, client, namespace, secretName, registry, username, password); err != nil {
		t.Fatalf("error creating secret: %v", err)
	}
	if err := createServiceAccount(ctx, client, namespace, serviceAccountName); err != nil {
		t.Fatalf("error creating service account: %v", err)
	}
	node, err := createDeploymentNode(namespace, serviceAccountName, imageTag, secretName)
	if err != nil {
		t.Fatalf("error creating deployment node: %v", err)
	}
	t.Logf("%s", node.MustString())

	kc, err := createK8schain(ctx, log, client, node)
	if err != nil {
		t.Fatalf("error creating k8schain: %v", err)
	}

	authConfig, err := getAuthConfig(kc, imageTag)
	if err != nil {
		t.Fatalf("error getting authconfig: %v", err)
	}
	want := &authn.AuthConfig{
		Username: username,
		Password: password,
	}
	if diff := cmp.Diff(want, authConfig, authConfigComparer); diff != "" {
		t.Errorf("createK8schain() authConfig mismatch (-want +got):\n%s", diff)
	}
}

func Test_createK8schain_ImagePullSecretOnDefaultServiceAccount(t *testing.T) {
	namespace := "default"
	serviceAccountName := "default"
	secretName := "test-secret"
	registry := "registry.example.com"
	username := "username"
	password := "password"
	imageTag := "registry.example.com/repository/image:tag"
	client, err := createFakeClient()
	if err != nil {
		t.Fatalf("error creating fake Kubernetes client: %v", err)
	}
	if err := createDockerConfigSecret(ctx, client, namespace, secretName, registry, username, password); err != nil {
		t.Fatalf("error creating secret: %v", err)
	}
	if err := createServiceAccount(ctx, client, namespace, serviceAccountName, secretName); err != nil {
		t.Fatalf("error creating service account: %v", err)
	}

	kc, err := createK8schain(ctx, log, client, nil)
	if err != nil {
		t.Fatalf("error creating k8schain: %v", err)
	}

	authConfig, err := getAuthConfig(kc, imageTag)
	if err != nil {
		t.Fatalf("error getting authconfig: %v", err)
	}
	want := &authn.AuthConfig{
		Username: username,
		Password: password,
	}
	if diff := cmp.Diff(want, authConfig, authConfigComparer); diff != "" {
		t.Errorf("createK8schain() authConfig mismatch (-want +got):\n%s", diff)
	}
}

func getAuthConfig(kc authn.Keychain, imageTag string) (*authn.AuthConfig, error) {
	tag, err := name.NewTag(imageTag)
	if err != nil {
		return nil, err
	}
	authenticator, err := kc.Resolve(tag)
	if err != nil {
		return nil, err
	}
	return authenticator.Authorization()
}
