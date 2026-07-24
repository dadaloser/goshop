# Operations

This document describes local operation, startup, observability, and production
guardrails for `goshop`.

## Prerequisites

- Go `1.26`
- Consul for service discovery
- MySQL for service data
- Redis for API SMS code storage and inventory coordination
- Elasticsearch for goods search
- OpenTelemetry collector when tracing is enabled
- DTM for distributed transaction coordination
- Nacos when dynamic Sentinel configuration is enabled

## Local Startup

Each binary reads a config file with `--config`.

```bash
go run ./cmd/api --config=./configs/api/api.yaml
go run ./cmd/admin --config=./configs/admin/admin.yaml
go run ./cmd/user --config=./configs/user/srv.yaml
go run ./cmd/goods --config=./configs/goods/srv.yaml
go run ./cmd/inventory --config=./configs/inventory/srv.yaml
go run ./cmd/order --config=./configs/order/srv.yaml
```

Start dependencies before services. Start backend gRPC services before API
gateways when testing full flows.

## Configuration Rules

- Files under `configs/` are local examples or deployment templates.
- Do not commit production passwords, JWT keys, SMS secrets, private DSNs, or
  cloud credentials.
- Use placeholders such as `${MYSQL_PASSWORD}`, `${JWT_KEY}`, and
  `${REDIS_PASSWORD}` in committed templates.
- Production startup validation must reject unsafe settings.
- Config logging must use safe redaction.

Important production settings:

| Area | Required Guardrail |
| --- | --- |
| HTTP timeouts | Positive read-header, read, write, and idle timeouts. |
| JWT | Non-empty key outside development. |
| CORS | Explicit allowed origins; no wildcard in production. |
| pprof | Disabled or protected with an explicit bearer token. |
| Secrets | Injected by environment, secret manager, or deployment platform. |
| Logs | JSON format preferred for production ingestion. |
| Dependency resilience | Positive timeout and concurrency, error ratio in `(0,1]`, positive minimum requests, statistic window, and recovery window. |

## Lifecycle

`gmicro.App.RunContext` coordinates service startup and shutdown:

1. Start configured HTTP and gRPC servers.
2. Wait for servers exposing `Ready()` to bind.
3. Register the service instance in Consul.
4. Wait for SIGTERM, SIGQUIT, SIGINT, or context cancellation.
5. Deregister from Consul.
6. Stop servers with a bounded timeout.
7. Flush OpenTelemetry trace exporters with a bounded timeout.

Background loops, including Redis connection checks, must listen to the same
startup context and exit on cancellation.

## Health And Readiness

- `/healthz` means the process is alive.
- `/readyz` should represent readiness to serve traffic and should include
  critical dependency checks where implemented.
- A service should not register in Consul before its network listener is ready.

## Metrics, Tracing, And Logs

- HTTP metrics are exposed at `/metrics` when enabled.
- pprof is exposed under `/debug/pprof` only when enabled.
- Tracing uses OpenTelemetry configuration from service config.
- Logs should include stable operation names and useful IDs such as request ID,
  user ID, order SN, service name, and duration.
- Outbound gRPC, Redis, and MySQL protection exports
  `dependency_resilience_*` metrics. See `docs/dependency-resilience.md` and
  `monitoring/prometheus/dependency-resilience-alerts.yaml`.
- Do not log JWTs, passwords, SMS secrets, full DSNs, private keys, or user
  private payloads.

## Payment Provider Protocol

- Refunds use `POST payment.refund-url` with `request_id`, `order_sn`,
  `trade_no`, `amount_fen`, and `reason`. `request_id` is the provider
  idempotency key. Successful responses return `provider_refund_id` and
  `status`; terminal success values are `refunded` and `succeeded`.
- Reconciliation uses `GET payment.reconcile-url?from=<RFC3339>&to=<RFC3339>`.
  The JSON response contains `transactions` with stable provider `event_id`,
  order/trade IDs, event type, amount in fen, and occurrence time.
- Outbound provider requests sign
  `timestamp + "\n" + HTTP method + "\n" + exact body` with HMAC-SHA256.
- Callbacks must send `X-Payment-Timestamp`, an unpredictable unique
  `X-Payment-Nonce`, and `X-Payment-Signature`. The callback signature covers
  `timestamp + "\n" + provider + "\n" + nonce + "\n" + exact body`.
  Redis atomically reserves the provider/nonce pair for twice the allowed clock
  skew. The database `(provider,event_id)` constraint supplies a second
  idempotency layer.
- Refund jobs use database leases, bounded exponential retry, and a dead-letter
  state after `payment.max-attempts`. Dead refunds transition both the refund and
  order to `FAILED`/`REFUND_FAILED` for operator handling.

## Testing And Verification

Common verification commands:

```bash
go test ./...
go test -race ./gmicro/... ./app/...
golangci-lint run ./...
govulncheck ./...
```

Notes:

- The local `golangci-lint` binary must be built with a Go version that supports
  the module target version.
- Integration tests that require Consul, MySQL, Redis, Elasticsearch, DTM, or
  external SMS providers should use a build tag and explicit setup instructions.

## Deployment Checklist

Before deploying:

1. Confirm all required secrets are injected outside the repository.
2. Confirm service config passes startup validation.
3. Run unit tests and relevant integration tests.
4. Verify protobuf and HTTP contract changes are documented.
5. Confirm health, readiness, metrics, and tracing endpoints are reachable in the
   target environment.
6. Confirm rollback plan for application image and database migrations.
7. Watch error rate, latency, dependency failures, and registration health during
   rollout.

## Troubleshooting

| Symptom | First Checks |
| --- | --- |
| Service does not start | Config file path, startup validation, port binding, missing secrets. |
| Service not discoverable | Consul address, service registration logs, listener readiness, health check status. |
| API request times out | Gateway logs, gRPC client discovery, backend service health, context deadlines. |
| Redis-dependent feature fails | Redis config, `storage.Connected()`, Redis logs, network path. |
| Trace data missing | Telemetry endpoint, exporter config, shutdown flushing, collector logs. |
| pprof unavailable | `profiling` flag, profiling token, route protection, network policy. |

## Documentation Maintenance

- New config fields must update this document and the relevant config template.
- New operational endpoints must document auth, exposure, and expected response.
- New external dependencies must be added to prerequisites, startup order, and
  troubleshooting notes.
