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

// Package webhook provides the command to run the Kubernetes mutating admission webhook.
package webhook

import (
	"context"
	"flag"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/google/k8s-digester/pkg/handler"
	"github.com/google/k8s-digester/pkg/logging"
	"github.com/google/k8s-digester/pkg/util"
)

const (
	caName             = "digester-ca"
	caOrganization     = "digester"
	defaultCertDir     = "/certs"
	defaultMetricsAddr = ":8888"
	defaultHealthAddr  = ":9090"
	defaultPort        = 8443
	secretName         = "digester-webhook-server-cert"            // matches the Secret name
	serviceName        = "digester-webhook-service"                // matches the Service name
	webhookName        = "digester-mutating-webhook-configuration" // matches the MutatingWebhookConfiguration name
	webhookPath        = "/v1/mutate"                              // matches the MutatingWebhookConfiguration clientConfig path
)

// Cmd is the webhook controller manager sub-command
var Cmd = &cobra.Command{
	Use:   "webhook",
	Short: "Start a Kubernetes mutating admission webhook controller manager",
	RunE: func(cmd *cobra.Command, _ []string) error {
		return run(cmd.Context())
	},
}

var (
	dnsName  = fmt.Sprintf("%s.%s.svc", serviceName, util.GetNamespace())
	webhooks = []rotator.WebhookInfo{
		{
			Name: webhookName,
			Type: rotator.Mutating,
		},
	}
)

var (
	certDir             string
	disableCertRotation bool
	dryRun              bool
	healthAddr          string
	metricsAddr         string
	offline             bool
	port                int
	ignoreErrors        bool
	skipPrefixes        string
)

func init() {
	Cmd.Flags().StringVar(&certDir, "cert-dir", defaultCertDir, "directory where TLS certificates and keys are stored")
	Cmd.Flags().BoolVar(&disableCertRotation, "disable-cert-rotation", false, "disable automatic generation and rotation of webhook TLS certificates/keys")
	Cmd.Flags().BoolVar(&dryRun, "dry-run", false, "if true, do not mutate any resources")
	Cmd.Flags().StringVar(&healthAddr, "health-addr", defaultHealthAddr, "health endpoint address")
	Cmd.Flags().AddGoFlag(flag.Lookup("kubeconfig"))
	Cmd.Flags().StringVar(&metricsAddr, "metrics-addr", defaultMetricsAddr, "metrics endpoint address")
	Cmd.Flags().BoolVar(&offline, "offline", false, "do not connect to API server to retrieve imagePullSecrets")
	Cmd.Flags().IntVar(&port, "port", defaultPort, "webhook server port")
	Cmd.Flags().BoolVar(&ignoreErrors, "ignore-errors", false, "do not fail on webhook admission errors, just log them")
	Cmd.Flags().StringVar(&skipPrefixes, "skip-prefixes", "", "(optional) image prefixes that should not be resolved to digests, colon separated")
}

func run(ctx context.Context) error {
	syncLogger, err := logging.CreateZapLogger("manager")
	if err != nil {
		return fmt.Errorf("could not create zap logger %w", err)
	}
	defer syncLogger.Sync()
	log := syncLogger.Log

	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get kubeconfig: %w", err)
	}
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("could not add core/v1 Kubernetes resources to scheme: %w", err)
	}
	mgr, err := manager.New(cfg, manager.Options{
		Scheme:         scheme,
		Logger:         log,
		LeaderElection: false,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: healthAddr,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    port,
			CertDir: certDir,
		}),
	})
	if err != nil {
		return fmt.Errorf("unable to set up manager: %w", err)
	}
	if err := mgr.AddReadyzCheck("default", healthz.Ping); err != nil {
		return fmt.Errorf("unable to create readyz check: %w", err)
	}
	if err := mgr.AddHealthzCheck("default", healthz.Ping); err != nil {
		return fmt.Errorf("unable to create healthz check: %w", err)
	}
	certSetupFinished := make(chan struct{})
	if !disableCertRotation {
		log.Info("setting up cert rotation")
		if err := rotator.AddRotator(mgr, &rotator.CertRotator{
			SecretKey: types.NamespacedName{
				Namespace: util.GetNamespace(),
				Name:      secretName,
			},
			CertDir:        certDir,
			CAName:         caName,
			CAOrganization: caOrganization,
			DNSName:        dnsName,
			IsReady:        certSetupFinished,
			Webhooks:       webhooks,
		}); err != nil {
			return fmt.Errorf("unable to set up cert rotation: %w", err)
		}
	} else {
		log.Info("skipping certificate provisioning setup")
		close(certSetupFinished)
	}

	go setupControllers(mgr, log, dryRun, ignoreErrors, certSetupFinished, util.StringArray(skipPrefixes))

	log.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return fmt.Errorf("problem running manager: %w", err)
	}
	return nil
}

func setupControllers(mgr manager.Manager, log logr.Logger, dryRun bool, ignoreErrors bool, certSetupFinished chan struct{}, skipPrefixes []string) {
	log.Info("waiting for cert rotation setup")
	<-certSetupFinished
	log.Info("done waiting for cert rotation setup")
	var k8sClientConfig *rest.Config
	if !offline {
		k8sClientConfig = mgr.GetConfig()
	}
	whh := &handler.Handler{
		Log:          log.WithName("webhook"),
		DryRun:       dryRun,
		IgnoreErrors: ignoreErrors,
		Config:       k8sClientConfig,
		SkipPrefixes: skipPrefixes,
	}
	mwh := &admission.Webhook{Handler: whh}
	log.Info("starting webhook server", "path", webhookPath)
	mgr.GetWebhookServer().Register(webhookPath, mwh)
}
