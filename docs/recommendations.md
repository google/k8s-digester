# Recommendations

Digester can run either as a mutating admission webhook in a Kubernetes
cluster, or as a client-side config function.

We recommend that you use both the config function and the webhook in your
environment.

The reason for this recommendation is that the config function and webhook
complement each other. There are some situations that only the config function
or the webhook can handle. By using both the config function and the webhook,
you get better coverage of different situations.

The following sections describe drawbacks of the config function and webhook
components, and how you can overcome these drawbacks by using the other
component.

## Injected containers

A drawback of the digester config function is that it will not resolve digests
for container and initContainer images injected by other mutating admission
webhooks, such as the
[Istio sidecar injector](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/#automatic-sidecar-injection).

The digester webhook resolves digests for container and initContainer images
injected by other mutating webhooks because its
[`reinvocationPolicy` is set to `IfNeeded`](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#reinvocation-policy).
This policy means that the API server executes the digester webhook again if
another webhook mutates the resource after the digester webhook first executed.

## Race condition

The digester webhook resolves the image digest at the time of deployment. This
means that you might encounter a race condition where the image associated with
the tag has changed in the time between when you decided to deploy the image,
and when you actually deployed the image to your cluster. This race condition
might result in deploying an unexpected image.

Additionally, if you want to deploy the same image to multiple clusters, the
deployments to each cluster might not happen at the same time, and the image
tag might change in between the deployments. The result might be that you
deploy different images to your clusters.

You can run the digester config function at the time you decide to deploy and
store the resulting manifest with digest in source control. You can then use
the manifest with digest to deploy, and this avoids the race condition.

## Rolling back

With the digester webhook, a situation similar to the race condition described
above might occur if you want to roll back an image to a previous version. As
an example, let's say that you are currently running an image with the tag `v1`
in your cluster, and the webhook resolved this tag to a digest value. You then
deploy a new version `v2` of your image, but there is a problem and you decide
that you want to roll back to version `v1`.

If you did not record the digest of the `v1` image from the first deployment,
the digester webhook will resolve the `v1` tag to a digest again when you roll
back. The tag `v1` might have changed to point to a new image in the time
between when you first deployed `v1` and when you rolled back. The result might
be that your rollback results in deploying a different image to the first time
you deployed `v1`.

You can run the digester config function the first time you deploy the `v1`
image and store the resulting manifest with digest in source control. Then,
when you want to roll back from `v2` to `v1`, you can apply the manifest you
stored. This manifest contains the image digest, so after you roll back, you
will be running the same image as before you deployed `v2`.

## Resource coverage

Another drawback of the digester webhook is that it only mutates the resource
types listed in the rules of the `MutatingWebhookConfiguration`. This includes
pods, podtemplates, replicationcontrollers, daemonsets, deployments,
replicasets, statefulsets, cronjobs, jobs, and Knative Eventing
containersources.

The digester config function does not inspect the resource type or Kind,
and it resolves digests for any resource that contains the fields
`spec.containers`, `spec.initContainers`, `spec.template.spec.containers`, and
`spec.template.spec.initContainers`.
