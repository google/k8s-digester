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
	"flag"
	"os"
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/google/k8s-digester/pkg/logging"
	"github.com/google/k8s-digester/pkg/util"
)

var testEnv *envtest.Environment

// TestMain starts the API server
func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		stdlog := logging.CreateStdLogger("webhook_suite_test")
		stdlog.Info("skipping integration test suite in short mode")
		return
	}
	apiServerFlags := append(envtest.DefaultKubeAPIServerFlags,
		"--enable-admission-plugins=ValidatingAdmissionWebhook,MutatingAdmissionWebhook",
	)
	admissionRegistrationNamespacedScope := admissionregistrationv1.NamespacedScope
	admissionRegistrationFailurePolicyFail := admissionregistrationv1.Fail
	admissionregistrationIfNeededReinvocationPolicy := admissionregistrationv1.IfNeededReinvocationPolicy
	admissionregistrationSideEffectClassNone := admissionregistrationv1.SideEffectClassNone
	clientConfigWebhookPath := webhookPath

	testEnv = &envtest.Environment{
		KubeAPIServerFlags: apiServerFlags,
		WebhookInstallOptions: envtest.WebhookInstallOptions{
			MutatingWebhooks: []client.Object{
				&admissionregistrationv1.MutatingWebhookConfiguration{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "admissionregistration.k8s.io/v1",
						Kind:       "MutatingWebhookConfiguration",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: webhookName,
					},
					Webhooks: []admissionregistrationv1.MutatingWebhook{
						{
							Name:                    dnsName,
							AdmissionReviewVersions: []string{"v1", "v1beta1"},
							ClientConfig: admissionregistrationv1.WebhookClientConfig{
								Service: &admissionregistrationv1.ServiceReference{
									Name:      serviceName,
									Namespace: util.GetNamespace(),
									Path:      &clientConfigWebhookPath,
								},
							},
							FailurePolicy: &admissionRegistrationFailurePolicyFail,
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"digest-resolution": "enabled",
								},
							},
							ReinvocationPolicy: &admissionregistrationIfNeededReinvocationPolicy,
							Rules: []admissionregistrationv1.RuleWithOperations{
								{
									Operations: []admissionregistrationv1.OperationType{
										admissionregistrationv1.Create,
										admissionregistrationv1.Update,
									},
									Rule: admissionregistrationv1.Rule{
										APIGroups:   []string{""},
										APIVersions: []string{"v1"},
										Resources:   []string{"pods"},
										Scope:       &admissionRegistrationNamespacedScope,
									},
								},
							},
							SideEffects: &admissionregistrationSideEffectClassNone,
						},
					},
				},
			},
		},
	}

	log := logging.CreateKlogLogger()
	if _, err := testEnv.Start(); err != nil {
		log.Error(err, "problem starting API server")
		os.Exit(1)
	}

	exitCode := m.Run()
	defer os.Exit(exitCode)

	if err := testEnv.Stop(); err != nil {
		log.Error(err, "problem stopping API server")
		os.Exit(1)
	}
}
