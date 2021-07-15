# Digester webhook kpt package

kpt package for the [digester](https://github.com/google/k8s-digester)
Kubernetes mutating admission webhook.

The digester mutating admission webhook resolves tags to digests for container
and init container images in Kubernetes Pod and Pod template specs.

## Deploying the webhook

The digester webhook requires Kubernetes v1.16 or later.

1.  If you use GKE, grant yourself the `cluster-admin` Kubernetes
    [cluster role](https://kubernetes.io/docs/reference/access-authn-authz/rbac/):

    ```sh
    kubectl create clusterrolebinding cluster-admin-binding \
        --clusterrole cluster-admin \
        --user "$(gcloud config get-value core/account)"
    ```

2.  Install [kpt](https://kpt.dev/installation/) v1.0.0-beta.1 or later.

3.  Fetch this package:

    ```sh
    VERSION=v0.1.5
    kpt pkg get https://github.com/google/k8s-digester.git/manifests@$VERSION manifests
    ```

4.  Setup inventory tracking for the package:

    ```sh
    kpt live init manifests
    ```

5.  Apply the package:

    ```sh
    kpt live apply manifests --reconcile-timeout=3m --output=table
    ```

6.  Add the `digest-resolution: enabled` label to namespaces where you want the
    webhook to resolve tags to digests:

    ```sh
    kubectl label namespace [NAMESPACE] digest-resolution=enabled
    ```

To configure how the webhook authenticates to your container image registries,
see the documentation on
[Authenticating to container image registries](https://github.com/google/k8s-digester/blob/main/docs/authentication.md#authenticating-to-container-image-registries).

## Deploying using kubectl

We recommend deploying the webhook using kpt as described above. If you are
unable to use kpt, you can deploy the digester using kubectl:

```sh
git clone https://github.com/google/k8s-digester.git digester
cd digester
VERSION=v0.1.5
git checkout $VERSION
kubectl apply -f manifests/namespace.yaml
kubectl apply -f manifests/
```

## Private clusters

If you install the webhook in a
[private Google Kubernetes Engine (GKE) cluster](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters),
you must add a firewall rule. In a private cluster, the nodes only have
[internal IP addresses](https://cloud.google.com/vpc/docs/ip-addresses).
The firewall rule allows the API server to access the webhook running on port
8443 on the cluster nodes.

1.  Create an environment variable called `CLUSTER`. The value is the name of
    your cluster that you see when running `gcloud container clusters list`:

    ```sh
    CLUSTER=[your private GKE cluster name]
    ```

2.  Look up the IP address range for the cluster API server and store it in an
    environment variable:

    ```sh
    API_SERVER_CIDR=$(gcloud container clusters describe $CLUSTER \
        --format 'value(privateClusterConfig.masterIpv4CidrBlock)')
    ```

3.  Look up the
    [network tags](https://cloud.google.com/vpc/docs/add-remove-network-tags)
    for your cluster nodes and store them comma-separated in an environment
    variable:

    ```sh
    TARGET_TAGS=$(gcloud compute firewall-rules list \
        --filter "name~^gke-$CLUSTER" \
        --format 'value(targetTags)' | uniq | paste -d, -s -)
    ```

4.  Create a firewall rule that allow traffic from the API server to cluster
    nodes on TCP port 8443:

    ```sh
    gcloud compute firewall-rules create allow-api-server-to-digester-webhook \
        --action ALLOW \
        --direction INGRESS \
        --source-ranges "$API_SERVER_CIDR" \
        --rules tcp:8443 \
        --target-tags "$TARGET_TAGS"
    ```

You can read more about private cluster firewall rules in the
[GKE private cluster documentation](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules).
