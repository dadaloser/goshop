#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

bash -n scripts/ci/jenkins-quality-gates.sh scripts/ci/jenkins-image-gates.sh scripts/ci/jenkins-sign-image.sh scripts/verify-db-compatibility.sh
ruby -e 'require "yaml"; ARGV.each { |f| Psych.parse_stream(File.read(f)) }' \
  monitoring/kubernetes/*.yaml monitoring/prometheus/*.yaml deployments/kubernetes/*.yaml deployments/kubernetes/canary/*.yaml chaos/kubernetes/*.yaml .github/workflows/*.yml

for service in goshop-api goshop-admin goshop-goods-srv goshop-inventory-srv goshop-order-srv goshop-review-srv goshop-user-srv; do
  rg -q "$service" monitoring/kubernetes || { echo "missing ServiceMonitor coverage for $service" >&2; exit 1; }
done

for file in build/docker/Jenkinsfile build/docker/*/Jenkinsfile; do
  rg -q 'jenkins-quality-gates.sh' "$file"
  rg -q 'spdx|jenkins-image-gates.sh' "$file"
  rg -q 'cosign|jenkins-sign-image.sh' "$file"
done

alerts="$(rg '^\s*- alert:' monitoring/prometheus | wc -l | tr -d ' ')"
owners="$(rg 'owner:' monitoring/prometheus | wc -l | tr -d ' ')"
runbooks="$(rg '^\s+runbook_url:' monitoring/prometheus | wc -l | tr -d ' ')"
test "$alerts" -eq "$owners"
test "$alerts" -eq "$runbooks"

# 运行手册、SLO 与历史演练已按主题收敛为当前有效的中文文档。
test -f docs/operations-and-release.md
test -f docs/production-baseline.md
test -f performance/k6/core-business.js
test -f chaos/records/drill-record-template.md
test -f chaos/records/2026-07-24-review-canary-rollback-drill.md

echo "operations readiness check passed"
