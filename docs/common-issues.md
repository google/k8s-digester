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
