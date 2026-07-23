#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
has_direct_dsn=0
if [[ -n "${GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN:-}" ]]; then
  has_direct_dsn=1
fi

has_env_pair=0
if [[ -n "${GOSHOP_SCHEMA_TEST_MYSQL_USERNAME:-}" && -n "${GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD:-}" ]]; then
  has_env_pair=1
fi

has_service_pairs=0
if [[ -n "${GOODS_MYSQL_USERNAME:-}" && -n "${GOODS_MYSQL_PASSWORD:-}" && -n "${ORDER_MYSQL_USERNAME:-}" && -n "${ORDER_MYSQL_PASSWORD:-}" ]]; then
  has_service_pairs=1
fi

if [[ ${has_direct_dsn} -eq 0 && ${has_env_pair} -eq 0 && ${has_service_pairs} -eq 0 ]]; then
  echo "schema integration tests require both GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN and GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN, shared GOSHOP_SCHEMA_TEST_MYSQL_USERNAME/GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD, or GOODS_MYSQL_*/ORDER_MYSQL_* credentials" >&2
  exit 1
fi

cd "${ROOT_DIR}"
env GOCACHE="${GOCACHE:-/tmp/goshop-gocache}" \
  go test ./app/goods/srv/internal/data/v1/db ./app/order/srv/internal/data/v1/db -run 'Test(Goods|Order)StartupValidationRealDB' -v
