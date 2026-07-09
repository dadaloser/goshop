# Go Coding Standards

These standards apply to all Go code in `goshop`.

## Formatting And Imports

- Run `gofmt` on changed Go files.
- Keep import groups ordered by standard library, blank line, then project and
  third-party imports as produced by Go tooling.
- Do not hand-format code against `gofmt`.

## Naming

- Use `MixedCaps` or `mixedCaps`; do not use snake_case or ALL_CAPS for Go
  identifiers.
- Package names must be lowercase, short, and capability-oriented.
- Avoid new packages named `util`, `common`, `helper`, or `base`. Existing
  packages with those names should not grow without review.
- Keep receiver names short and consistent across methods.

## Package Boundaries

- Keep business logic in service packages, not in handlers or storage code.
- Define small interfaces in the package that consumes them.
- Return concrete implementations from producer packages unless an interface is
  required by callers.
- Add code under `pkg/` only when other modules or multiple internal services
  should rely on that API.

## Context

- `ctx context.Context` is the first parameter for operations that can block,
  perform I/O, call another service, or run under a request lifecycle.
- Do not store `context.Context` in structs.
- Do not use `context.Background()` inside request, service, data, or boundary
  paths.
- Use `context.TODO()` only as a temporary compatibility placeholder when a
  legacy function cannot yet accept a context.
- Always call cancel functions returned by `WithCancel`, `WithTimeout`, or
  `WithDeadline`, unless ownership is explicitly returned to the caller.
- Background loops must exit on `ctx.Done()`.

## Error Handling

- Never discard returned errors with `_` unless the operation is intentionally
  best-effort and the reason is documented in a nearby comment.
- Wrap errors with context using `%w`.
- Error strings should be lowercase and should not end with punctuation.
- Handle each error once: either return it or log it at the boundary, not both.
- Do not panic for expected failures. Panics are limited to programmer errors,
  impossible invariants, or startup-time hard failures.
- Convert internal errors to stable business codes at HTTP/gRPC boundaries.
- Do not expose secrets, DSNs, tokens, passwords, or private payloads in error
  messages.

## Logging

- Use the project logger instead of `fmt.Println` for operational logs.
- Log messages should be stable strings; put request IDs, user IDs, order IDs,
  counts, and durations in structured fields where available.
- Do not log passwords, JWTs, SMS secrets, full DSNs, private keys, or user
  privacy data.
- Avoid repeated logging of the same error in controller, service, and data
  layers.

## Configuration

- Each service config type must implement validation.
- Production startup must reject missing required secrets, unsafe CORS wildcard
  settings, unbounded HTTP timeouts, and unauthenticated pprof exposure.
- Printing config must use safe redaction.
- Repository config files are examples or local templates. Production values
  must come from environment variables, deployment systems, or secret managers.

## Data Access

- Gorm calls must use `WithContext(ctx)` when a caller context is available.
- Transactions must rollback on every error path before returning.
- Do not silently ignore `Commit`, `Rollback`, `DB`, `Close`, or migration
  errors unless the operation is documented as best-effort.
- Paginated queries must enforce bounded page sizes.
- Order clauses derived from input must be allowlisted before use.

## Concurrency

- Every goroutine must have a clear owner and exit path.
- Do not start unbounded goroutines from request handlers.
- Shared mutable state must use a mutex, atomic value, channel ownership, or
  another explicit synchronization mechanism.
- Concurrency-sensitive packages should have race tests where practical.

## Testing

- New business behavior requires tests for success and important failure paths.
- Bug fixes should include a regression test or document why one is not
  feasible.
- Table-driven tests must include a `name` field and run with `t.Run`.
- Tests that use real Consul, Redis, MySQL, Elasticsearch, DTM, or SMS providers
  must be behind an integration build tag.
- Prefer `httptest` and in-process gRPC servers over mocks for transport
  behavior.

## Generated And Third-Party Code

- Do not edit `*.pb.go`, `*_grpc.pb.go`, or `*_gin.pb.go` by hand.
- Update the source `.proto` and regenerate.
- Avoid editing `third_party/` except for explicit dependency upgrades or
  tracked patches.

## Required Validation

For code changes, run the narrowest useful check and broaden when shared code is
affected:

```bash
go test ./...
go test -race ./gmicro/... ./app/...   # for concurrency-sensitive changes
golangci-lint run ./...                # when the local lint tool supports the module Go version
govulncheck ./...                      # before releases or dependency changes
```

If a check cannot run because of toolchain or existing unrelated failures, record
the command and the reason.
