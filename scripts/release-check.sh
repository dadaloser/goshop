#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOCACHE_DIR="${GOCACHE:-/tmp/goshop-gocache}"
GOTOOLCHAIN_VERSION="${GOTOOLCHAIN_VERSION:-go1.26.0}"

cd "${ROOT_DIR}"

echo "[release-check] format"
test -z "$(gofmt -l .)"

echo "[release-check] vet"
env GOCACHE="${GOCACHE_DIR}" go vet ./app/... ./gmicro/... ./pkg/...

echo "[release-check] panic scan"
make panic-check

echo "[release-check] migration policy"
make migration-check

echo "[release-check] config secrets"
make config-secret-check

echo "[release-check] startup validation"
make startup-validation-check

if [[ (-n "${GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN:-}") || (-n "${GOSHOP_SCHEMA_TEST_MYSQL_USERNAME:-}" && -n "${GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD:-}") || (-n "${GOODS_MYSQL_USERNAME:-}" && -n "${GOODS_MYSQL_PASSWORD:-}" && -n "${ORDER_MYSQL_USERNAME:-}" && -n "${ORDER_MYSQL_PASSWORD:-}") ]]; then
  echo "[release-check] schema integration"
  make schema-integration-test
else
  echo "[release-check] schema integration skipped (set both GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN and GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN, shared GOSHOP_SCHEMA_TEST_MYSQL_USERNAME/GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD, or GOODS_MYSQL_*/ORDER_MYSQL_* credentials)"
fi

echo "[release-check] protobuf drift"
make proto-check

echo "[release-check] rpcserver stability"
env GOCACHE="${GOCACHE_DIR}" go test -count=50 ./gmicro/server/rpcserver

echo "[release-check] unit/integration tests"
env GOCACHE="${GOCACHE_DIR}" go test -race -shuffle=on ./...

echo "[release-check] lint"
bash ./scripts/lint.sh

echo "[release-check] vulnerability scan"
env GOCACHE="${GOCACHE_DIR}" GOTOOLCHAIN="${GOTOOLCHAIN_VERSION}" go run golang.org/x/vuln/cmd/govulncheck@latest ./...
