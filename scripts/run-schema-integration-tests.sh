#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
has_direct_dsn=0
if [[ -n "${GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_USER_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_INVENTORY_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_REVIEW_SCHEMA_TEST_MYSQL_DSN:-}" ]]; then
  has_direct_dsn=1
fi

has_env_pair=0
if [[ -n "${GOSHOP_SCHEMA_TEST_MYSQL_USERNAME:-}" && -n "${GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD:-}" ]]; then
  has_env_pair=1
fi

has_service_pairs=0
if [[ -n "${GOODS_MYSQL_USERNAME:-}" && -n "${GOODS_MYSQL_PASSWORD:-}" && -n "${ORDER_MYSQL_USERNAME:-}" && -n "${ORDER_MYSQL_PASSWORD:-}" && -n "${USER_MYSQL_USERNAME:-}" && -n "${USER_MYSQL_PASSWORD:-}" && -n "${INVENTORY_MYSQL_USERNAME:-}" && -n "${INVENTORY_MYSQL_PASSWORD:-}" && -n "${REVIEW_MYSQL_USERNAME:-}" && -n "${REVIEW_MYSQL_PASSWORD:-}" ]]; then
  has_service_pairs=1
fi

if [[ ${has_direct_dsn} -eq 0 && ${has_env_pair} -eq 0 && ${has_service_pairs} -eq 0 ]]; then
  echo "schema integration tests require all five *_SCHEMA_TEST_MYSQL_DSN values, shared GOSHOP_SCHEMA_TEST_MYSQL_USERNAME/GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD, or GOODS_/ORDER_/USER_/INVENTORY_/REVIEW_MYSQL_* credentials" >&2
  exit 1
fi

cd "${ROOT_DIR}"
env GOCACHE="${GOCACHE:-/tmp/goshop-gocache}" \
  go test ./app/user/srv/internal/data/v1/db ./app/goods/srv/internal/data/v1/db ./app/order/srv/internal/data/v1/db ./app/inventory/srv/internal/data/v1/db ./app/review/srv/internal/data/db -run 'Test(User|Goods|Order|Inventory|Review)StartupValidationRealDB' -v
