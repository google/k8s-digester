# Build

This document explains how to build your own binaries or container images for
digester.

Before you proceed, clone the Git repository and install the following tools:

- [Go distribution](https://golang.org/doc/install) v1.17 or later
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) v3.7.0 or later 
- [Skaffold](https://skaffold.dev/docs/install/#standalone-binary) v1.37.2 or later

## Building binaries and container images

- Build the binary:

  ```sh
  go build -o digester .
  ```

- Build a container image and load it into your local Docker daemon:

  ```sh
  skaffold build --cache-artifacts=false --push=false
  ```

- Build a container image and push it to Container Registry:

  ```sh
  skaffold build --push --default-repo gcr.io/$(gcloud config get core/project)
  ```

The base image is `gcr.io/distroless/static:nonroot`. If you want to use a
different base image, change the value of the `defaultBaseImage` field in the
file [`.ko.yaml`](ko.yaml). For instance, if you want to use a base image that
contains credential helpers for a number of container registries, you can use a
base image from the `gcr.io/kaniko-project/executor` repository.

## Building and deploying the webhook

1.  (optional) If you use a Google Kubernetes Engine (GKE) cluster with
    [Workload Identity](workload-identity.md), and either Container Registry or
    Artifact Registry, annotate the digester Kubernetes service account:

    ```sh
    kustomize cfg annotate manifests \
        --kind ServiceAccount \
        --name digester-admin \
        --namespace digester-system \
        --kv "iam.gke.io/gcp-service-account=$GSA"
    ```

    This annotation informs GKE that the Kubernetes service account
    `digester-admin` in the namespace `digester-system` can impersonate the
    Google service account `$GSA`.

2.  Build and push the webhook container image, and deploy to your Kubernetes cluster:

    ```sh
    skaffold run --push --default-repo gcr.io/$(gcloud config get core/project)
    ```
