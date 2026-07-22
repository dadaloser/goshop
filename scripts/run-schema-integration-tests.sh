#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if [[ -z "${GOSHOP_SCHEMA_TEST_MYSQL_DSN:-}" ]]; then
  echo "schema integration tests require GOSHOP_SCHEMA_TEST_MYSQL_DSN" >&2
  exit 1
fi

cd "${ROOT_DIR}"
env GOCACHE="${GOCACHE:-/tmp/goshop-gocache}" \
  go test ./app/goods/srv/internal/data/v1/db ./app/order/srv/internal/data/v1/db -run 'Test(Goods|Order)StartupValidationRealDB' -v
