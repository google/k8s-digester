# Development

During development, you can run the config function and the webhook locally.
You can also use Skaffold to set up a watch loop that automatically deploys
the webhook to a Kubernetes cluster on source code changes.

## Running the config function during development

-   Apply the function to a Pod manifest:

    ```bash
    DEBUG=true go run . < build/examples/pod.yaml
    ```

## Running the webhook locally during development

1.  Create a self-signed certificate:

    ```bash
    mkdir -p build/cert

    openssl req -x509 -newkey rsa:4096 -nodes -sha256 -days 3650 \
      -keyout build/cert/tls.key -out build/cert/tls.crt -extensions san \
      -config \
      <(echo "[req]";
        echo distinguished_name=req;
        echo "[san]";
        echo subjectAltName=DNS:localhost,IP:127.0.0.1
        ) \
      -subj '/CN=localhost'
    ```

2.  Run the webhook locally:

    ```bash
    DEBUG=true go run . webhook --cert-dir=build/cert --disable-cert-rotation=true --offline=true
    ```

    Setting the `DEBUG=true` environment variable enabled development mode
    logging.

    The `--cert-dir` and `--disable-cert-rotation=true` flags means that the
    webhook uses the certificate you created in the previous step, instead of
    retrieving a certificate from the API server.

    The `--offline=true` flag means that the webhook will not retrieve
    `imagePullSecrets` from a Kubernetes API server.

3.  In another terminal window, send an admission review request for a
    Deployment that uses a public image:

    ```bash
    curl -sk -X POST -H "Content-Type: application/json" \
      --data @build/test/request.json \
      https://localhost:8443/v1/mutate \
      | jq -r '.response.patch' | base64 --decode | jq
    ```

    The output is the list of JSON patches that the API server admission
    process applies to the request object.

4.  Publish a private image by using `crane` to copy a public image:

    ```bash
    export PROJECT_ID=$(gcloud config get-value core/project)

    curl -sL "https://github.com/google/go-containerregistry/releases/download/v0.4.0/go-containerregistry_$(uname -s)_$(uname -m).tar.gz" \
      | tar -zxf - crane gcrane

    ./crane cp gcr.io/google-samples/hello-app:1.0 gcr.io/$PROJECT_ID/hello-app:1.0
    ```

5.  Send an admission review request for a Deployment that uses the private
    image:

    ```bash
    curl -sk -X POST -H "Content-Type: application/json" \
      --data @<(envsubst < build/test/request-authn.json) \
      https://localhost:8443/v1/mutate \
      | jq -r '.response.patch' | base64 --decode | jq
    ```

## Redeploying the webhook to a Kubernetes cluster on source code changes

1.  Create a development Kubernetes cluster, for instance using
    [Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/docs),
    [Minikube](https://minikube.sigs.k8s.io/), or
    [kind](https://kind.sigs.k8s.io/).

2.  Install these tools:

    -   [crane](https://github.com/google/go-containerregistry/tree/main/cmd/crane#installation)
    -   [ko](https://github.com/google/ko#installation)
    -   [kpt](https://googlecontainertools.github.io/kpt/installation/)
    -   [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/)
    -   [Skaffold](https://skaffold.dev/docs/install/)

3.  Set the Skaffold default container image registry:

    ```bash
    export SKAFFOLD_DEFAULT_REPO=gcr.io/$(gcloud config get-value core/project)
    ```

4.  (optional) Enable debug mode for more verbose logging:

    ```bash
    kpt cfg set manifests/ debug true
    ```

5.  (optional) Set `replicas` to 1:

    ```bash
    kpt cfg set manifests/ replicas 1
    ```

6.  Deploy the webhook and start the watch loop:

    ```bash
    skaffold dev
    ```
