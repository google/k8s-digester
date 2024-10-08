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

package webhook

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/google/k8s-digester/pkg/handler"
	"github.com/google/k8s-digester/pkg/logging"
)

func Test_ResolvePublicGCRImage(t *testing.T) {
	ctx := ctrl.SetupSignalHandler()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("could not add core v1 Kubernetes resources to scheme")
	}
	mgr, err := ctrl.NewManager(testEnv.Config, ctrl.Options{
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Host:    testEnv.WebhookInstallOptions.LocalServingHost,
			Port:    testEnv.WebhookInstallOptions.LocalServingPort,
			CertDir: testEnv.WebhookInstallOptions.LocalServingCertDir,
		}),
	})
	if err != nil {
		t.Fatalf("could not create test controller manager: %+v", err)
	}
	t.Logf("webhook address: https://%v:%v%v",
		testEnv.WebhookInstallOptions.LocalServingHost,
		testEnv.WebhookInstallOptions.LocalServingPort,
		webhookPath)

	webhookServer := mgr.GetWebhookServer()
	webhookServer.Register(webhookPath, &webhook.Admission{
		Handler: &handler.Handler{
			Log: logging.CreateKlogLogger(),
		},
	})
	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Errorf("could not start test controller manager: %+v", err)
		}
	}()

	client := mgr.GetClient()

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-ns",
			Labels: map[string]string{
				"digest-resolution": "enabled",
			},
		},
	}
	if err := client.Create(ctx, namespace); err != nil {
		t.Fatalf("could not create test namespace: %+v", err)
	}

	podIn := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-ns",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "index.docker.io/docker/whalesay:latest",
				},
			},
		},
	}
	if err := client.Create(ctx, podIn); err != nil {
		t.Fatalf("could not create test pod: %+v", err)
	}

	podOut := &corev1.Pod{}
	if err := client.Get(ctx,
		types.NamespacedName{
			Name:      "test-pod",
			Namespace: "test-ns",
		}, podOut); err != nil {
		t.Fatalf("could not get test pod: %+v", err)
	}

	wantImage := "index.docker.io/docker/whalesay:latest@sha256:178598e51a26abbc958b8a2e48825c90bc22e641de3d31e18aaf55f3258ba93b"
	gotImage := podOut.Spec.Containers[0].Image
	if wantImage != gotImage {
		t.Errorf("wanted %s, got %s", wantImage, gotImage)
	}
}
