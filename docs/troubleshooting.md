# Troubleshooting

Be sure to check out solutions to [common issues](common-issues.md).

## Config function troubleshooting

If the config function fails to look up the image digest, you can increase the
logging verbosity by using the `DEBUG` environment variable:

```bash
export DEBUG=true
kpt fn ...
```

## Webhook troubleshooting

The webhook fails open because the
[`failurePolicy`](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#failure-policy)
on the `MutatingWebhookConfiguration` is set to `Ignore`. This means that if
there is an error calling the webhook, the API server allows the request to
continue.

If the webhook fails to look up the image digest, you can enable development
mode logging and increase the logging verbosity.

1.  Set the `DEBUG` environment variable to `true` in the webhook Deployment
    manifest and redeploy the webhook:

    ```bash
    kpt cfg set manifests/ debug true
    kpt live apply manifests/
    ```

2.  Tail the webhook logs:

    ```bash
    kubectl logs --follow deployment/digester-controller-manager \
      --namespace digester-system --all-containers=true
    ```
