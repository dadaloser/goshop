#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

schema_test_database="${GOSHOP_SCHEMA_TEST_MYSQL_DATABASE:-goshop_schema_ci}"
schema_test_container=""

cleanup() {
  if [[ -n "${schema_test_container}" ]]; then
    docker rm -f "${schema_test_container}" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

set_default_schema_databases() {
  export GOSHOP_USER_SCHEMA_TEST_MYSQL_DATABASE="${GOSHOP_USER_SCHEMA_TEST_MYSQL_DATABASE:-${schema_test_database}}"
  export GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DATABASE="${GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DATABASE:-${schema_test_database}}"
  export GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DATABASE="${GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DATABASE:-${schema_test_database}}"
  export GOSHOP_INVENTORY_SCHEMA_TEST_MYSQL_DATABASE="${GOSHOP_INVENTORY_SCHEMA_TEST_MYSQL_DATABASE:-${schema_test_database}}"
  export GOSHOP_REVIEW_SCHEMA_TEST_MYSQL_DATABASE="${GOSHOP_REVIEW_SCHEMA_TEST_MYSQL_DATABASE:-${schema_test_database}}"
}

maybe_start_schema_mysql() {
  if [[ -n "${GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_USER_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_INVENTORY_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_REVIEW_SCHEMA_TEST_MYSQL_DSN:-}" ]]; then
    return 0
  fi

  if [[ -n "${GOSHOP_SCHEMA_TEST_MYSQL_USERNAME:-}" && -n "${GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD:-}" ]]; then
    set_default_schema_databases
    return 0
  fi

  if [[ -n "${GOODS_MYSQL_USERNAME:-}" && -n "${GOODS_MYSQL_PASSWORD:-}" && -n "${ORDER_MYSQL_USERNAME:-}" && -n "${ORDER_MYSQL_PASSWORD:-}" && -n "${USER_MYSQL_USERNAME:-}" && -n "${USER_MYSQL_PASSWORD:-}" && -n "${INVENTORY_MYSQL_USERNAME:-}" && -n "${INVENTORY_MYSQL_PASSWORD:-}" && -n "${REVIEW_MYSQL_USERNAME:-}" && -n "${REVIEW_MYSQL_PASSWORD:-}" ]]; then
    return 0
  fi

  if ! command -v docker >/dev/null 2>&1; then
    return 1
  fi

  schema_test_container="goshop-schema-ci-$RANDOM-$RANDOM"
  host_port="$(
    docker run -d --rm \
      --name "${schema_test_container}" \
      -e MYSQL_ROOT_PASSWORD=integration \
      -e MYSQL_DATABASE="${schema_test_database}" \
      -p 127.0.0.1::3306 \
      mysql:8.4
  )"
  host_port="$(docker port "${schema_test_container}" 3306/tcp | awk -F: 'END{print $NF}')"
  if [[ -z "${host_port}" ]]; then
    echo "failed to determine mapped MySQL port for schema integration container" >&2
    exit 1
  fi

  for _ in $(seq 1 60); do
    if docker exec "${schema_test_container}" mysqladmin ping -h 127.0.0.1 -pintegration --silent >/dev/null 2>&1; then
      export GOSHOP_SCHEMA_TEST_MYSQL_USERNAME=root
      export GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD=integration
      export GOSHOP_SCHEMA_TEST_MYSQL_HOST=127.0.0.1
      export GOSHOP_SCHEMA_TEST_MYSQL_PORT="${host_port}"
      set_default_schema_databases
      return 0
    fi
    sleep 1
  done

  echo "timed out waiting for temporary schema integration MySQL" >&2
  exit 1
}

maybe_start_schema_mysql || {
  echo "schema integration tests require all five *_SCHEMA_TEST_MYSQL_DSN values, shared GOSHOP_SCHEMA_TEST_MYSQL_USERNAME/GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD, GOODS_/ORDER_/USER_/INVENTORY_/REVIEW_MYSQL_* credentials, or a local docker runtime to start temporary MySQL" >&2
  exit 1
}

has_direct_dsn=0
if [[ -n "${GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_USER_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_INVENTORY_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_REVIEW_SCHEMA_TEST_MYSQL_DSN:-}" ]]; then
  has_direct_dsn=1
fi

has_env_pair=0
if [[ -n "${GOSHOP_SCHEMA_TEST_MYSQL_USERNAME:-}" && -n "${GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD:-}" ]]; then
  set_default_schema_databases
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
