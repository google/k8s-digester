# Build

This document explains how to build your own binaries or container images for
digester.

Before you proceed, clone the Git repository and install the following tools:

-   [Go distribution](https://golang.org/doc/install)
-   [ko](https://github.com/google/ko#installation)
-   [kpt](https://googlecontainertools.github.io/kpt/installation/)

## Building binaries and container images

-   Build the binary:

    ```bash
    go build .
    ```

-  Build a container image and load it into your local Docker daemon:

    ```bash
    export GOROOT=$(go env GOROOT)
    ko publish --base-import-paths --local .
    ```

-   Build a container image and publish it to Container Registry:

    ```bash
    export GOROOT=$(go env GOROOT)
    export KO_DOCKER_REPO=gcr.io/$(gcloud config get-value core/project)
    ko publish --base-import-paths .
    ```

The base image is `gcr.io/distroless/static:nonroot`. If you want to use a
different base image, change the value of the `defaultBaseImage` field in the
file [`.ko.yaml`](ko.yaml). For instance, if you want to use a base image that
contains credential helpers for a number of container registries, you can use a
base image from the `gcr.io/kaniko-project/executor` repository.

## Building and deploying the webhook

1.  Set environment variables for `ko`:

    ```bash
    export GOROOT=$(go env GOROOT)
    export KO_DOCKER_REPO=gcr.io/$(gcloud config get-value core/project)
    ```

2.  Build and publish the webhook container image, and set the image name (with
    digest) in the webhook Deployment manifest:

    ```bash
    IMAGE=$(ko publish --base-import-paths .)
    kpt cfg set manifests/ image $IMAGE
    ```

3.  (optional) If you use a Google Kubernetes Engine (GKE) cluster with
    [Workload Identity](workload-identity.md), and either Container Registry or
    Artifact Registry, annotate the digester Kubernetes service account:

    ```bash
    kpt cfg annotate manifests/ \
      --kind ServiceAccount \
      --name digester-admin \
      --namespace digester-system \
      --kv "iam.gke.io/gcp-service-account=$GSA"
    ```

    This annotation informs GKE that the Kubernetes service account
    `digester-admin` in the namespace `digester-system` can impersonate the
    Google service account `$GSA`.

4.  Deploy the webhook:

    ```bash
    kpt live apply manifests/ --reconcile-timeout=3m --output=table
    ```
