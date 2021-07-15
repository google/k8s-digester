# Digester

Digester resolves tags to
[digests](https://cloud.google.com/solutions/using-container-images) for
container and init container images in Kubernetes
[Pod](https://kubernetes.io/docs/concepts/workloads/pods/) and
[Pod template](https://kubernetes.io/docs/concepts/workloads/pods/#pod-templates)
specs.

It replaces container image references that use tags:

```yaml
spec:
  containers:
  - image: gcr.io/google-containers/echoserver:1.10
```

With references that use the image digest:

```yaml
spec:
  containers:
  - image: gcr.io/google-containers/echoserver:1.10@sha256:cb5c1bddd1b5665e1867a7fa1b5fa843a47ee433bbb75d4293888b71def53229
```

Digester can run either as a
[mutating admission webhook](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
in a Kubernetes cluster, or as a client-side
[Kubernetes Resource Model (KRM) function](https://kpt.dev/book/02-concepts/03-functions)
with the [kpt](https://kpt.dev/) or
[kustomize](https://kubectl.docs.kubernetes.io/guides/introduction/kustomize/)
command-line tools.

If a tag points to an
[image index](https://github.com/opencontainers/image-spec/blob/master/image-index.md#oci-image-index-specification)
or
[manifest list](https://docs.docker.com/registry/spec/manifest-v2-2/#manifest-list),
digester resolves the tag to the digest of the image index or manifest list.

The webhook is opt-in at the namespace level by label, see
[Deploying the webhook](#deploying-the-webhook).

If you use
[Binary Authorization](https://cloud.google.com/binary-authorization/docs),
digester can help to ensure that only verified container images can be deployed
to your clusters. A Binary Authorization
[attestation](https://cloud.google.com/binary-authorization/docs/key-concepts#attestations)
is valid for a particular container image digest. You must deploy container
images by digest so that Binary Authorization can verify the attestations for
the container image. You can use digester to deploy container images by digest.

## Running the KRM function

1.  Download the digester binary for your platform from the
    [Releases page](../../releases).

    Alternatively, you can download the latest version using these commands:

    ```sh
    VERSION=v0.1.7
    curl -Lo digester "https://github.com/google/k8s-digester/releases/download/${VERSION}/digester_$(uname -s)_$(uname -m)"
    chmod +x digester
    ```

2.  [Install kpt](https://kpt.dev/installation/) v1.0.0-beta.1 or later, and/or
    [install kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/).

3.  Run the digester KRM function:

    -   Using kpt:

        ```sh
        kpt fn eval [manifest directory] --exec ./digester
        ```

    -  Using kustomize:

        ```sh
        kustomize fn run [manifest directory] --enable-exec --exec-path ./digester
        ```

    By running as an executable, the KRM function has access to container
    image registry credentials in the current environment, such as the current
    user's
    [Docker config file](https://github.com/google/go-containerregistry/blob/main/pkg/authn/README.md#the-config-file)
    and
    [credential helpers](https://docs.docker.com/engine/reference/commandline/login/#credential-helper-protocol).
    For more information, see the digester documentation on
    [Authenticating to container image registries](docs/authentication.md).

## Deploying the webhook

Install the digester webhook in your Kubernetes cluster by following the steps
in the [kpt package documentation](manifests/README.md).

## Documentation

-   [Tutorial](https://cloud.google.com/architecture/using-container-image-digests-in-kubernetes-manifests#using_digester)

-   [Motivation](docs/motivation.md)

-   [Recommendations](docs/recommendations.md)

-   [Authenticating to container image registries](docs/authentication.md)

-   [Configuring GKE Workload Identity for authenticating to Container Registry and Artifact Registry](docs/workload-identity.md)

-   [Resolving common issues](docs/common-issues.md)

-   [Troubleshooting](docs/troubleshooting.md)

-   [Building digester](docs/build.md)

-   [Developing digester](docs/development.md)

-   [Releasing digester](docs/release.md)

## Disclaimer

This is not an officially supported Google product.
