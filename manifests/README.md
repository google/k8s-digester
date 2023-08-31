# Digester webhook package

Package for the [digester](https://github.com/google/k8s-digester)
Kubernetes mutating admission webhook.

The digester mutating admission webhook resolves tags to digests for container
and init container images in Kubernetes CronJob, Pod and Pod template specs.

## Preparing for deployment

The digester webhook requires Kubernetes v1.16 or later.

If you use Google Kubernetes Engine (GKE), grant yourself the
`cluster-admin` Kubernetes
[cluster role](https://kubernetes.io/docs/reference/access-authn-authz/rbac/):

```sh
kubectl create clusterrolebinding cluster-admin-binding \
    --clusterrole cluster-admin \
    --user "$(gcloud config get core/account)"
```

To configure how the webhook authenticates to your container image registries,
see the documentation on
[Authenticating to container image registries](https://github.com/google/k8s-digester/blob/main/docs/authentication.md#authenticating-to-container-image-registries).

If you use a private GKE cluster, see additional steps for
[creating a firewall rule](../README.md#private-clusters).

## Deploying the webhook using kustomize

1.  Install [kustomize](https://github.com/kubernetes-sigs/kustomize).

2.  Apply this package:

    ```sh
    VERSION=v0.1.12
    kustomize build "https://github.com/google/k8s-digester.git/manifests?ref=$VERSION" | kubectl apply -f -
    ```

3.  Add the `digest-resolution: enabled` label to namespaces where you want
    the webhook to resolve tags to digests:

    ```sh
    kubectl label namespace [NAMESPACE] digest-resolution=enabled
    ```

## Deploying the webhook using kpt

1.  Install [kpt](https://kpt.dev/installation/) v1.0.0-beta.1 or later.

3.  Fetch this package:

    ```sh
    VERSION=v0.1.12
    kpt pkg get "https://github.com/google/k8s-digester.git/manifests@${VERSION}" manifests
    ```

4.  Setup inventory tracking for the package:

    ```sh
    kubectl create namespace digester-system
    kpt live init manifests
    ```

5.  Apply the package:

    ```sh
    kpt live apply manifests --reconcile-timeout=3m --output=table
    ```

6.  Add the `digest-resolution: enabled` label to namespaces where you want
    the webhook to resolve tags to digests:

    ```sh
    kubectl label namespace [NAMESPACE] digest-resolution=enabled
    ```
