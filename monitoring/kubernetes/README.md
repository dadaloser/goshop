# Kubernetes Monitoring Manifests

Apply the default in-cluster monitoring edge with:

```bash
kubectl apply -k monitoring/kubernetes
```

This bundle installs:

- `ServiceMonitor` resources for API, admin, goods, inventory, order, and user
- a namespace-local `NetworkPolicy` that keeps the shared HTTP observability
  ports reachable only from internal cluster ranges and the `monitoring`/`goshop`
  namespaces

## Label Assumptions

The manifests assume the workload Services or Pods carry:

- `app.kubernetes.io/part-of: goshop`
- `app.kubernetes.io/name` matching one of the six service names in the manifests

API/admin expose dedicated management target ports `8149` and `8150`. Goods uses
`8051`; inventory, order, and user Services must expose their metrics endpoint
through a port named `metrics`.

## ServiceMonitor vs PodMonitor

`kustomization.yaml` enables `ServiceMonitor` by default.

Use `podmonitor-api-admin.yaml` only when your cluster standardizes on
`PodMonitor` instead of `ServiceMonitor`. Do not enable both for the same
workload unless you intentionally want duplicate scrapes.

## Important Limitation

`goshop-api` and `goshop-admin` now expose observability endpoints on dedicated
management ports. Keep public Ingress rules pointed only at the business HTTP
ports and do not route traffic to the management ports.
