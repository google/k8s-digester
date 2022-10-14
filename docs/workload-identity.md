# Workload Identity on Google Kubernetes Engine

If you use
[Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/docs),
you can authenticate to
[Container Registry](https://cloud.google.com/container-registry/docs) and
[Artifact Registry](https://cloud.google.com/artifact-registry/docs) using
[Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity).

The following steps assume that the
[Google service account](https://cloud.google.com/iam/docs/service-accounts)
is in the same
[project](https://cloud.google.com/resource-manager/docs/creating-managing-projects)
as the Container Registry and Artifact Registry image repositories.

1.  Enable the GKE and Artifact Registry APIs:

    ```sh
    gcloud services enable \
        container.googleapis.com \
        artifactregistry.googleapis.com
    ```

    Note that enabling the GKE API also enables the Container Registry API.

2.  Create a GKE cluster with Workload Identity, and assign the
    [`cloud-platform` access scope](https://cloud.google.com/compute/docs/access/service-accounts#service_account_permissions)
    to the nodes:

    ```sh
    PROJECT_ID=$(gcloud config get core/project)
    ZONE=us-central1-f

    gcloud container clusters create digester-webhook-test \
        --enable-ip-alias \
        --release-channel regular \
        --scopes cloud-platform \
        --workload-pool $PROJECT_ID.svc.id.goog \
        --zone $ZONE
    ```

3.  Create a Google service account:

    ```sh
    GSA_NAME=digester-webhook
    GSA=$GSA_NAME@$PROJECT_ID.iam.gserviceaccount.com

    gcloud iam service-accounts create $GSA_NAME \
        --display-name "Digester webhook service account"
    ```

    The digester webhook
    [Kubernetes service account](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)
    impersonates this Google service account to authenticate to Container
    Registry and Artifact Registry.

4.  Grant the
    [Container Registry Service Agent role](https://cloud.google.com/iam/docs/understanding-roles#service-agents-roles)
    to the Google service account at the project level:

    ```sh
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member "serviceAccount:$GSA" \
        --role roles/containerregistry.ServiceAgent
    ```

5.  Grant the
    [Artifact Registry Reader](https://cloud.google.com/iam/docs/understanding-roles#artifact-registry-roles)
    to the Google service account at the project level:

    ```sh
    gcloud projects add-iam-policy-binding $PROJECT_ID \
        --member "serviceAccount:$GSA" \
        --role roles/artifactregistry.reader
    ```

6.  Grant the
    [Workload Identity User role](https://cloud.google.com/iam/docs/understanding-roles#service-accounts-roles)
    to the `digester-admin` Kubernetes service account in the `digester-system`
    namespace on the Google service account:

    ```sh
    gcloud iam service-accounts add-iam-policy-binding "$GSA" \
        --member "serviceAccount:$PROJECT_ID.svc.id.goog[digester-system/digester-admin]" \
        --role roles/iam.workloadIdentityUser
    ```

7.  Add the Workload Identity annotation to the digester webhook Kubernetes
    service account:

    ```sh
    kubectl annotate serviceaccount digester-admin --namespace digester-system \
        "iam.gke.io/gcp-service-account=$GSA"
    ```

    This annotation informs GKE that the Kubernetes service account
    `digester-admin` in the namespace `digester-system` can impersonate the
    Google service account `$GSA`.

Workload Identity works with both
[online and offline authentication](authentication.md).

If you use Workload Identity to authenticate to Container Registry or Artifact
Registry, and if you do not rely on `imagePullSecrets` to authenticate to
other container image registries, you can enable offline authentication on the
digester webhook without providing a Docker config file, see
[`authentication.md`](authentication.md).
