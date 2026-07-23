# Performance and Capacity Validation

Run `BASE_URL=https://canary.example ACCESS_TOKEN=... GOODS_ID=... TPS=200 DURATION=10m k6 run performance/k6/core-business.js` against an isolated production-like environment.

Release gate: sustained 200 TPS, P95 below 500 ms, P99 below 1 s, error rate below 1%, no oversell, no dead-letter increase, CPU below 70%, memory below 75%, DB pool below 80% and zero critical alerts for 10 minutes. Starting capacity is 200 TPS per API replica; validate linearly at 200/400/800 TPS and record the first saturated resource. Required replicas are `ceil(peak TPS / measured safe TPS per replica × 1.5)`.

Store k6 JSON output, dashboard snapshots, image digest, dataset size and configuration with the release record. A result from a laptop or shared development database is not a release qualification result.
