# Kubernetes Monitoring Manifests

Apply the default in-cluster monitoring edge with:

```bash
kubectl apply -k monitoring/kubernetes
```

This bundle installs:

- `ServiceMonitor` resources for `goshop-api` and `goshop-admin`
- a namespace-local `NetworkPolicy` that keeps the shared HTTP observability
  ports reachable only from internal cluster ranges and the `monitoring`/`goshop`
  namespaces

## Label Assumptions

The manifests assume the workload Services or Pods carry:

- `app.kubernetes.io/part-of: goshop`
- `app.kubernetes.io/name: goshop-api` or `goshop-admin`

`ServiceMonitor` assumes the selected Services ultimately expose pod target ports
`8049` and `8050`.

## ServiceMonitor vs PodMonitor

`kustomization.yaml` enables `ServiceMonitor` by default.

Use `podmonitor-api-admin.yaml` only when your cluster standardizes on
`PodMonitor` instead of `ServiceMonitor`. Do not enable both for the same
workload unless you intentionally want duplicate scrapes.

## Important Limitation

`goshop-api` and `goshop-admin` currently serve business traffic and
observability routes on the same HTTP ports. This NetworkPolicy narrows direct
pod and ClusterIP access to internal sources, but it cannot filter individual
paths. If a public Ingress still routes `/metrics`, `/readyz`, `/healthz`, or
`/debug/pprof`, external callers may still traverse that Ingress path.

Keep the application-level route guard enabled and ensure public Ingress rules
do not expose these observability paths. A future dedicated management port
would let the NetworkPolicy become fully path-agnostic and stricter.
