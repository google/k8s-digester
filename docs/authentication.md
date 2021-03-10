# Authenticating to container image registries

To resolve digests for private images, digester requires credentials to
authenticate to your container image registry.

## Authentication modes

Digester supports two modes of authentication: offline and online.

### Offline authentication

When using offline authentication, digester uses credentials available on the
node or machine where it runs. This includes the following credentials:

1.  Google service account credentials available via
    [Application Default Credentials](https://cloud.google.com/docs/entication/production#auth-cloud-implicit-go)
    for authenticating to
    [Container Registry](https://cloud.google.com/container-registry/docs) and
    [Artifact Registry](https://cloud.google.com/artifact-registry/docs).

    For implementation details, see the
    [github.com/google/go-containerregistry/pkg/v1/google](https://pkg.go.github.com/google/go-containerregistry/pkg/v1/google)
    and
    [golang.org/x/oauth2/google](https://pkg.go.dev/golang.org/x/oauth2/le)
    Go packages.

2.  Credentials and credential helpers specified in the
    [Docker config file](https://github.com/google/go-containerregistry/tree//pkg/authn#the-config-file),
    for authenticating to any container image registry. The file name is
    `config.json`, and the default file location is the directory
    `$HOME/.docker`. You can override the default location of the config file
    using the `DOCKER_CONFIG` environment variable.

    For implementation details, see the
    [github.com/google/go-containerregistry/pkg/authn](https://pkg.go.dev/ub.com/google/go-containerregistry/pkg/authn)
    and
    [github.com/docker/cli/cli/config](https://pkg.go.dev/github.com/docker/cli/config)
    Go packages.

### Online authentication

When using online authentication, digester authenticates using the following
credentials:

1.  The `imagePullSecrets` listed in the
    [pod specification](https://kubernetes.io/docs/concepts/containers/es/#specifying-imagepullsecrets-on-a-pod)
    and the
    [service account](https://kubernetes.io/docs/tasks/igure-pod-container/configure-service-account/-imagepullsecrets-to-a-service-account)
    used by the pod. Digester retrieves these secrets from the Kubernetes
    cluster API server.

    For implementation details, see the
    [github.com/google/go-containerregistry/pkg/authn/k8schain](https://pkg.ev/github.com/google/go-containerregistry/pkg/authn/k8schain)
    Go package.

2.  Cloud provider-specific implementations of the Kubernetes
    [`DockerConfigProvider` interface](https://pkg.go.dev/github.com/eester/k8s-pkg-credentialprovider#DockerConfigProvider).
    For instance, the implementation for Google Cloud retrieves credentials
    from the node or Workload Identity
    [metadata server](https://cloud.google.com/compute/docs/ing-retrieving-metadata).

    For implementation details, see the provider-specific subdirectories of the
    [github.com/vdemeester/k8s-pkg-credentialprovider](https://pkg.go.dev/ub.com/vdemeester/k8s-pkg-credentialprovider)
    Go package.

3.  Credentials specified in the
    [Docker config file](https://github.com/google/go-containerregistry/tree//pkg/authn#the-config-file),
    for authenticating to any container image registry. The file name is
    `config.json`, and the location can be the container working directory
    (`$PWD/config.json`), or a directory called `.docker` under the user
    home directory (`$HOME/.docker/config.json`) or the file system root
    directory (`/.docker/config.json`).

    For implementation details, see the
    [github.com/vdemeester/k8s-pkg-credentialprovider](https://pkg.go.dev/ub.com/vdemeester/k8s-pkg-credentialprovider)
    Go package.

The client-side config function defaults to offline authentication, whereas the
webhook defaults to online authentication. You can override the default
authentication mode using the `--offline` command-line flag or the `OFFLINE`
environment variable.

## Config function offline authentication

The config function uses offline authentication by default. By running digester
as an
[exec function](https://googlecontainertools.github.io/kpt/guides/producer/functions/exec/),
the config function has access to container image registry credentials in the
current environment, such as the current user's
[Docker config file](https://github.com/google/go-containerregistry/blob/main/pkg/authn/README.md#the-config-file)
and
[credential helpers](https://docs.docker.com/engine/reference/commandline/login/#credential-helper-protocol).

To run digester as an exec function, use the `--enable-exec` and `--execpath`
flags:

```bash
kpt fn source [manifest file or directory] \
  | kpt fn run --enable-exec --exec-path ./digester
```

If your Docker config file contains your container image registry credentials
and you do not need a credential helper, you can run digester in a container.
Mount your Docker config file in the container using the `--mount` flag:

```bash
kpt fn source [manifest file or directory] \
  | kpt fn run --network \
    --env DOCKER_CONFIG=/.docker \
    --mount type=bind,src="$HOME/.docker/config.json",dst=/.docker/config.json \
    --image gcr.io/cloud-solutions-images/digester
```

The `--network` flag provides external network access to digester running in
the container. Digester requires this to connect to the container image
registry.

## Config function online authentication

To use online authentication with the digester config function, set the
`OFFLINE=false` environment variable. To use the kpt
[exec runtime](https://googlecontainertools.github.io/kpt/guides/producer/functions/exec/),
you can use this command:

```bash
kpt fn source [manifest file or directory] \
  | OFFLINE=false kpt fn run --enable-exec --exec-path ./digester
```

If you use the kpt
[container runtime](https://googlecontainertools.github.io/kpt/guides/producer/functions/container/),
mount your kubeconfig file in the container:

```bash
kpt fn source [manifest file or directory] \
  | kpt fn run --network \
    --env OFFLINE=false --env KUBECONFIG=/.kube/config \
    --mount type=bind,src="$HOME/.kube/config",dst=/.kube/config \
    --image gcr.io/cloud-solutions-images/digester
```

When using online authentication, digester connects to the Kubernetes cluster
defined by your current
[kubeconfig context](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/).

The user defined by the current context must have permissions to read the
`imagePullSecrets` and service accounts listed in the Pod specifications.

You can provide an alternative kubeconfig file by setting the value of the
`--kubeconfig` command-line flag or the `KUBECONFIG` environment variable to
the full path of an alternative kubeconfig file.

## Webhook online authentication

The webhook uses online authentication by default, and it uses the
`digester-admin` Kubernetes service account to authenticate to the API server.

The `digester-manager-role` ClusterRole provides read access to all
Secrets and ServiceAccounts in the cluster, and the
`digester-manager-rolebinding` ClusterRoleBinding binds this role to the
`digester-admin` Kubernetes service account in the `digester-system` namespace.

## Webhook offline authentication

If you don't want to give the digester webhook read access to Secrets and
ServiceAccounts in the cluster, you can enable offline authentication
(`--offline=true`). With offline authentication, you can provide credentials to
the webhook using a
[Docker config file](https://github.com/google/go-containerregistry/blob/main/pkg/authn/README.md#the-config-file):

1.  Set the `offline` flag value to `true` in the webhook Deployment manifest:

    ```bash
    kpt cfg set manifests/ offline true
    ```

2.  Create a Docker config file containing map entries with usernames and
    passwords for your registries:

    ```bash
    REGISTRY_HOST=[your container image registry authority, e.g., registry.gitlab.com]
    REGISTRY_USERNAME=[your container image registry user name]
    REGISTRY_PASSWORD=[your container image registry password or token]

    cat << EOF > docker-config.json
    {
      "auths": {
        "$REGISTRY_HOST": {
          "username": "$REGISTRY_USERNAME",
          "password": "$REGISTRY_PASSWORD"
        }
      }
    }
    EOF
    ```

3.  Create a Secret in the `digester-system` namespace containing the config
    file:

    ```bash
    kubectl create secret generic docker-config --namespace digester-system \
      --from-file config.json=$(pwd)/docker-config.json
    ```

4.  Create a patch file for the `webhook-controller-manager` Deployment. The
    patch adds the Docker config file Secret as a volume, and mounts the volume
    on the Pods:

    ```bash
    cat << EOF > manifests/docker-config-patch.json
    [
      {
        "op": "add",
        "path": "/spec/template/spec/containers/0/volumeMounts/-",
        "value":{
          "mountPath": ".docker",
          "name": "docker",
          "readOnly": true
        }
      },
      {
        "op": "add",
        "path": "/spec/template/spec/volumes/-",
        "value": {
          "name": "docker",
          "secret": {
            "defaultMode": 420,
            "secretName": "docker-config"
          }
        }
      }
    ]
    EOF
    ```

4.  Create a kustomize manifest with the patch:

    ```bash
    cat << EOF > manifests/Kustomization
    apiVersion: kustomize.config.k8s.io/v1beta1
    kind: Kustomization
    resources:
    - cluster-role-binding.yaml
    - cluster-role.yaml
    - deployment.yaml
    - inventory-template.yaml
    - mutating-webhook-configuration.yaml
    - namespace.yaml
    - role-binding.yaml
    - role.yaml
    - secret.yaml
    - service-account.yaml
    - service.yaml
    patches:
    - path: docker-config-patch.json
      target:
        group: apps
        version: v1
        kind: Deployment
        name: digester-controller-manager
    EOF
    ```

4.  Deploy the webhook with the patch:

    ```bash
    kustomize build manifests/ | kpt live apply
    ```

If you use offline authentication, you can remove the rule in the
`digester-manager-role` ClusterRole that grants access to `secrets` and
`serviceaccounts`, see
[`manifests/cluster-role.yaml`](../manifests/cluster-role.yaml).
