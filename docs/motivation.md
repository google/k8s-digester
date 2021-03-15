# Motivation

We created digester to make it easier for Kubernetes users to deploy container
images by digest, and to assist users of
[Binary Authorization](https://cloud.google.com/binary-authorization/docs).

## What are container image digests?

A
[container image digest](https://github.com/opencontainers/image-spec/blob/master/descriptor.md#digests)
uniquely and immutably identifies a container image.

The digest value is the result of applying a
[collision-resistant hash function](https://wikipedia.org/wiki/Collision_resistance),
typically [SHA-256](https://wikipedia.org/wiki/SHA-2),
to the image index, manifest list, or image manifest.

If you are not familiar with image digests, read the document
[Using container image digests](https://cloud.google.com/solutions/using-container-images).

## Why deploy with digests instead of tags?

When you deploy images by digest, you avoid the downsides of deploying by
[image tags](https://github.com/opencontainers/distribution-spec/blob/master/spec.md).

Tags are commonly used to refer to different revisions of a container image,
for example, `v1.0.1`, to refer to a version that you call 1.0.1. Tags make
image revisions easy to look up by human-readable strings. However, tags are
mutable references, which means the image referenced by a tag can change.

If you publish a new image using the same tag as an existing image, the tag
stops pointing to the existing image and starts pointing to your new image.

Because tags are mutable, they have the following disadvantages when you use
them to deploy an image:

-   In Kubernetes, deploying by tag can result in unexpected results. For
    example, assume that you have an existing Deployment resource that
    references a container image by tag `v1.0.1`. To fix a bug or make a small
    change, your build process creates a new image with the same tag `v1.0.1`.
    New Pods that are created from your Deployment resource can end up using
    either the old or the new image, even if you don't change your Deployment
    resource specification. This problem also applies to other Kubernetes
    resources such as StatefulSets, DaemonSets, ReplicaSets, and Jobs.

-   If you use tools to scan or analyze images, results from these tools are
    only valid for the image that was scanned. To ensure that you deploy the
    image that was scanned, you cannot rely on the tag because the image
    referred to by the tag might have changed.

-   If you use
    [Binary Authorization](https://cloud.google.com/binary-authorization/docs)
    with
    [Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/docs),
    tag-based deployment is disallowed because it's impossible to determine
    the exact image that is used when a Pod is created.

-   You must decide on which
    [imagePullPolicy](https://kubernetes.io/docs/concepts/configuration/overview/#container-images)
    to use for the containers in your Pods.

When you deploy your images, you can use an image digest to avoid these
disadvantages of using tags. You can still add tags to your images if you like,
but you don't have to do so.

## Software supply chain security benefits

Because digests are immutable and unique, using them to deploy images means
that you can cryptographically verify that the image that's running in your
production environment is the exact same image that you produced in your
build process by comparing the digest value.

In addition, if you want to ensure you only deploy approved images to your
Google Kubernetes Engine (GKE) clusters, you can use
[Binary Authorization](https://cloud.google.com/binary-authorization/docs).

## Other solutions

There are many ways to add image digests to Kubernetes manifests. Some of them
are documented in the tutorial
[Using container image digests in Kubernetes manifests](https://cloud.google.com/solutions/using-container-image-digests-in-kubernetes-manifests).

[Cloud Run](https://cloud.google.com/run/docs/deploying#service),
[Cloud Run for Anthos](https://cloud.google.com/kuberun/docs/deploying#service),
and
[Knative Serving](https://knative.dev/docs/serving/tag-resolution/)
resolve image tags to digests on deployment. The digest is stored in a service
revision, and all instances of that service revision use the digest.

## References

-   [Using container image digests](https://cloud.google.com/solutions/using-container-images)
-   [Using container image digests in Kubernetes manifests](https://cloud.google.com/solutions/using-container-image-digests-in-kubernetes-manifests)
-   [k/k#1697: Image name/tag resolution preprocessing pass](https://github.com/kubernetes/kubernetes/issues/1697)
-   [Why we resolve tags in Knative](https://docs.google.com/presentation/d/e/2PACX-1vTgyp2lGDsLr_bohx3Ym_2mrTcMoFfzzd6jocUXdmWQFdXydltnraDMoLxvEe6WY9pNPpUUvM-geJ-g/pub?resourcekey=0-FH5lN4C2sbURc_ds8XRHeA)
