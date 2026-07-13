# Project Architecture

This document records the current architecture of `goshop` and the boundaries
that changes must preserve.

## Overview

- Module: `goshop`
- Go version: `1.26`
- Project type: multi-service Go application with gRPC services, HTTP gateways,
  shared framework code, generated protobuf contracts, and deployment assets.
- Main executables: `cmd/api`, `cmd/admin`, `cmd/user`, `cmd/goods`,
  `cmd/inventory`, and `cmd/order`.
- External systems: Consul, MySQL, Redis, Elasticsearch, OpenTelemetry collector,
  DTM, Nacos, and Aliyun SMS.

## Top-Level Layout

| Path | Responsibility |
| --- | --- |
| `cmd/` | Process entrypoints. Keep startup orchestration here only. |
| `api/` | Protobuf contracts and generated HTTP/gRPC bindings. |
| `app/` | Product services and gateways. Service-private code lives under each service's `internal/` tree. |
| `app/pkg/` | Shared application-level helpers used by multiple services, such as clients, options, codes, Gorm helpers, and translators. |
| `gmicro/` | Local microservice framework code: app lifecycle, registry, trace, metrics, RPC server, REST server, middleware, and load balancing. |
| `pkg/` | Reusable lower-level packages. Treat exported APIs here as shared contracts. |
| `configs/` | Local configuration examples and deployment templates. Do not commit production secrets. |
| `docs/` | Architecture, rules, operations, and hardening notes. |
| `migrations/` | Database migration scripts. |
| `scripts/` | Local development and maintenance scripts. |
| `build/` | Docker and CI packaging assets. |
| `third_party/` | Vendored protocol definitions and forked code. Avoid editing unless upgrading or patching intentionally. |

## Service Boundaries

| Service | Path | Role |
| --- | --- | --- |
| User service | `app/user/srv` | User creation, lookup, mobile lookup, password verification, and user update over gRPC. |
| Goods service | `app/goods/srv` | Goods, brand, category, banner, category-brand relation, and Elasticsearch-backed search. |
| Inventory service | `app/inventory/srv` | Inventory query, sell, rollback, and Redis-backed coordination. |
| Order service | `app/order/srv` | Order and shopping cart flows. Calls goods and inventory and coordinates DTM Saga operations. |
| User API gateway | `app/goshop/api` | Public HTTP API that aggregates backend gRPC services. |
| Admin gateway | `app/goshop/admin` | Admin HTTP service. |

## Layering Rules

1. `cmd/{service}` only starts the configured application.
2. HTTP handlers and gRPC controllers parse protocol input, run shallow
   validation, extract auth context, and convert responses.
3. Service packages hold business rules, state transitions, and orchestration.
4. Data packages own SQL, Gorm, Redis, Elasticsearch, and transaction mechanics.
5. Boundary/client packages own cross-service calls and convert external DTOs
   into local interfaces.
6. `pkg/` and `gmicro/` must not import service-private `app/*/internal` code.
7. New reusable code goes to `pkg/` only when it has a stable API responsibility.
8. Outbound gRPC, Redis, and MySQL operations use `gmicro/resilience` for
   timeout, Sentinel isolation/circuit breaking, fail-fast fallback, and metrics.

## Request Lifecycle

HTTP gateway flow:

```text
cmd/api
  -> app/goshop/api.NewApp
  -> config load and startup validation
  -> restserver.Server
  -> middleware: recovery, cors, context, auth, metrics, pprof when enabled
  -> controller
  -> service
  -> data or gRPC boundary
  -> backend service
```

gRPC service flow:

```text
cmd/{service}
  -> app/{service}/srv.NewApp
  -> config load and startup validation
  -> gmicro.App.RunContext
  -> rpcserver.Server
  -> controller
  -> service
  -> data, boundary, or external dependency
```

## Lifecycle And Context

- `pkg/app.RunFunc` receives the top-level command context.
- `gmicro.App.RunContext` owns server startup, registry registration, signal
  handling, deregistration, server stop, and trace shutdown.
- Startup code must propagate the same context into Redis loops, gRPC dials,
  discovery probes, and other long-running background work.
- Cleanup should use bounded contexts derived with `context.WithoutCancel(ctx)`
  and `context.WithTimeout`, so cancellation starts shutdown without preventing
  deregistration or exporter flush.
- Do not create `context.Background()` inside request or service paths. Use the
  caller's `ctx`; use `context.TODO()` only as a compatibility placeholder.

## Error Boundaries

- Low-level packages wrap and return errors with context.
- Controllers, interceptors, and middleware are the response/logging boundary.
- Avoid logging and returning the same error in intermediate service layers.
- Convert internal errors to stable business codes before returning responses.

## Key Files

| Path | Purpose |
| --- | --- |
| `pkg/app/app.go` | CLI command assembly, config loading, and top-level run callback. |
| `gmicro/app/app.go` | Service lifecycle, signal handling, registration, shutdown, and trace cleanup. |
| `gmicro/server/restserver/server.go` | Gin HTTP server, health, metrics, pprof, and production startup checks. |
| `gmicro/server/rpcserver/server.go` | gRPC server startup and shutdown. |
| `gmicro/resilience/` | Shared dependency policies, Sentinel guards, circuit state listeners, and Prometheus metrics. |
| `app/pkg/gorm/resilience.go` | GORM callback plugin for MySQL dependency protection. |
| `pkg/storage/redis_resilience.go` | go-redis hook for Redis dependency protection. |
| `app/goshop/api/router.go` | Public HTTP API routing. |
| `app/order/srv/internal/service/v1/order.go` | Order creation and DTM Saga orchestration. |
| `app/*/srv/config/config.go` | Per-service configuration structure, validation, and safe printing. |
| `api/*/v1/*.proto` | Public gRPC contracts for each service. |

## Change Rules

- Changes to service boundaries, request flow, lifecycle, or shared framework
  behavior must update this document.
- Changes to exported HTTP/gRPC fields, status mapping, or protobuf fields must
  update `docs/rules/api-contracts.md`.
- Changes to configuration, deployment, health checks, metrics, or startup
  commands must update `docs/rules/operations.md`.
- Changes to coding rules, lint policy, testing policy, or error/context rules
  must update `docs/rules/go-coding-standards.md`.
