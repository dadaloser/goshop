# Prometheus Rules

Apply all current Goshop alert rules with:

```bash
kubectl apply -k monitoring/prometheus
```

This kustomization deploys:

- `dependency-resilience-alerts.yaml`
- `order-ops-alerts.yaml`

Before applying in production, confirm the target namespace and Prometheus
Operator label selectors match your cluster conventions.
