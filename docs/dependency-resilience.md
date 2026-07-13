# Dependency Resilience

This document describes Sentinel protection and Prometheus signals for outbound
gRPC, Redis, and MySQL operations.

## Protection Model

Every protected operation has:

- a context timeout;
- Sentinel concurrency isolation;
- an error-ratio circuit breaker;
- a minimum request count before the circuit can open;
- a statistic window and an open-circuit recovery window;
- a fail-fast fallback when isolation or circuit breaking rejects an operation.

Resource names are deliberately bounded:

| Dependency | Resource label | Example |
| --- | --- | --- |
| gRPC | Protobuf full method | `goshop.user.v1.User/GetUserById` |
| Redis | Command name | `get`, `set`, `pipeline` |
| MySQL | GORM operation class | `create`, `query`, `update`, `delete`, `row`, `raw` |

Redis `Nil`, MySQL record-not-found and constraint conflicts, caller cancellation,
and normal gRPC business status codes do not contribute to the circuit error ratio.

## Configuration

MySQL and Redis policies live under their dependency section. API and order
outbound RPC policies use `rpc-client-resilience`.

```yaml
rpc-client-resilience:
  enabled: true
  timeout: 2s
  max-concurrency: 100
  error-ratio: 0.5
  min-request-amount: 20
  stat-interval: 10s
  recovery-timeout: 30s
```

Disabling `enabled` turns off Sentinel isolation and circuit breaking. The
operation timeout remains active so a disabled breaker cannot create unbounded
calls.

## Metrics

| Metric | Labels | Meaning |
| --- | --- | --- |
| `dependency_resilience_requests_total` | `dependency`, `resource`, `outcome` | Operations split into `success`, `error`, `timeout`, `canceled`, or `blocked`. |
| `dependency_resilience_duration_ms` | `dependency`, `resource`, `outcome` | Operation latency histogram. |
| `dependency_resilience_inflight` | `dependency`, `resource` | Current isolated concurrency. |
| `dependency_resilience_fallback_total` | `dependency`, `resource`, `reason` | Fail-fast fallback count by `isolation`, `circuit_open`, or another Sentinel block reason. |
| `dependency_resilience_circuit_transitions_total` | `dependency`, `resource`, `from`, `to` | Circuit state changes. |
| `dependency_resilience_circuit_state` | `dependency`, `resource` | Current state: `0` closed, `1` half-open, `2` open. |
| `dependency_resilience_recovery_total` | `dependency`, `resource` | Successful half-open to closed recoveries. |

Useful Grafana panels:

```promql
sum by (dependency, resource, outcome) (
  rate(dependency_resilience_requests_total[5m])
)
```

```promql
histogram_quantile(
  0.95,
  sum by (le, dependency, resource) (
    rate(dependency_resilience_duration_ms_bucket[5m])
  )
)
```

```promql
max by (dependency, resource) (dependency_resilience_circuit_state)
```

```promql
sum by (dependency, resource) (
  rate(dependency_resilience_fallback_total[5m])
)
```

Recommended rules are in
`monitoring/prometheus/dependency-resilience-alerts.yaml`. Tune concurrency
thresholds to the configured `max-concurrency` and observe warning rules before
turning them into paging alerts.
