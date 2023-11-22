# Resolving common issues

This section provides solutions to common issues encountered when using
digester.

## Self-signed and untrusted certificates

If your container image registry uses a self-signed certificate, or a
certificate issued by a certificate authority (CA) that is not trusted by the
CA bundle used by digester
([`ca-certificates`](https://packages.debian.org/stable/ca-certificates)), you
can configure digester with your own CA bundle.

To do so, set the
[`SSL_CERT_FILE` or `SSL_CERT_DIR` environment variables](https://golang.org/pkg/crypto/x509/#SystemCertPool)
on the `manager` container in the webhook
[deployment resource](../manifests/deployment.yaml).
The steps below use the `SSL_CERT_DIR` environment variable.

1.  Create a Kubernetes generic Secret containing you CA bundle certificates,
    called `my-ca-bundle`, in the `digester-system` namespace:

    ```sh
    kubectl create secret generic my-ca-bundle --namespace digester-system \
        --from-file=cert1=/path/to/cert1 --from-file=cert2=/path/to/cert2
    ```

2.  Create a JSON patch file called `ca-bundle-patch.json` that adds the
    `SSL_CERT_DIR` environment variable, a volume, and a volume mount to the
    webhook deployment:

    ```json
    [
      {
        "op": "add",
        "path": "/spec/template/spec/containers/0/env/-",
        "value":{
          "name": "SSL_CERT_DIR",
          "value": "/my-ca-certs"
        }
      },
      {
        "op": "add",
        "path": "/spec/template/spec/containers/0/volumeMounts/-",
        "value":{
          "mountPath": "/my-ca-certs",
          "name": "my-ca-bundle-volume",
          "readOnly": true
        }
      },
      {
        "op": "add",
        "path": "/spec/template/spec/volumes/-",
        "value": {
          "name": "my-ca-bundle-volume",
          "secret": {
            "defaultMode": 420,
            "secretName": "my-ca-bundle"
          }
        }
      }
    ]
    ```

3.  Apply the patch:

    ```sh
    kubectl patch deployment/digester-controller-manager -n digester-system \
        --type json --patch-file ca-bundle-patch.json
    ```

Ref: https://knative.dev/docs/serving/tag-resolution/#custom-certificates

## Corporate proxies

If digester needs to traverse a corporate HTTP proxy to reach the container
registry, you can configure digester to use the proxy.

To do so, set the
[`HTTP_PROXY` or `HTTPS_PROXY` environment variables](https://golang.org/pkg/net/http/#ProxyFromEnvironment)
on the `manager` container in the webhook
[deployment resource](../manifests/deployment.yaml).
The steps below use the `HTTPS_PROXY` environment variable.

1.  Create a JSON patch file called `http-proxy-patch.json` that adds the
    `HTTPS_PROXY` environment variable to the webhook deployment:

    ```json
    [
      {
        "op": "add",
        "path": "/spec/template/spec/containers/0/env/-",
        "value":{
          "name": "HTTPS_PROXY",
          "value": "http://myproxy.example.com:3128"
        }
      }
    ]
    ```

2.  Apply the patch:

    ```sh
    kubectl patch deployment/digester-controller-manager -n digester-system \
        --type json --patch-file http-proxy-patch.json
    ```

Note that this will not work for proxies that require NTLM authentication.

Ref: https://knative.dev/docs/serving/tag-resolution/#corporate-proxy

## Interaction with systems expecting tags, particularly cloud managed services

If digester updates an image tag that is being actively managed by a cloud controller then
it may cause the cloud controller to behave unexpectedly.

One example of this is the Anthos Service Mesh Managed Dataplane Controller which looks
for specific tagged versions of the istio-proxy sidecar injected by the mutating webhook.

Replacement of the tagged names with digest values can, under these circumstances, create
an edge case for the cloud managed services handling unepected values in unforseen ways such
as updating pods and terminating them once they have already been updated (since the image
does not match the value set by the controller with only the tag).

In these circumstances and if you are using digester to provide a tag feature when using
Binary Authorization it is worth noting that there is a capability to whitelist certain
image registries and repo locations within Binary Authorization. ASM images are by default
whitelisted by the policy.

To avoid digester replacing the tagged version expected by mdp-controller in these instances
one can utilise the --skip-prefixes option to the webhook which takes a set of prefixes
separated by a colon (if multiple prefixes are needed).

The parameter can be added to the webhook args in the deployment, the following is an
example
```
        args:
        - webhook
        - --cert-dir=/certs # kpt-set: --cert-dir=${cert-dir}
        - --disable-cert-rotation=false # kpt-set: --disable-cert-rotation=${disable-cert-rotation}
        - --dry-run=false # kpt-set: --dry-run=${dry-run}
        - --health-addr=:9090 # kpt-set: --health-addr=:${health-port}
        - --metrics-addr=:8888 # kpt-set: --metrics-addr=:${metrics-port}
        - --offline=false # kpt-set: --offline=${offline}
        - --port=8443 # kpt-set: --port=${port}
        - --skip-prefixes=gcr.io/gke-release/asm/mdp:gcr.io/gke-release/asm/proxyv2
```