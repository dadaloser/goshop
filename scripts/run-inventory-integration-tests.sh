#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PKG="./app/inventory/srv/internal/service/v1"
PATTERN="${GO_TEST_PATTERN:-TestInventory(.*RealDB)}"

has_direct_dsn=0
if [[ -n "${GOSHOP_INVENTORY_TEST_MYSQL_DSN:-}" ]]; then
  has_direct_dsn=1
fi

has_env_pair=0
if [[ -n "${INVENTORY_MYSQL_USERNAME:-}" && -n "${INVENTORY_MYSQL_PASSWORD:-}" ]]; then
  has_env_pair=1
fi

if [[ ${has_direct_dsn} -eq 0 && ${has_env_pair} -eq 0 ]]; then
  echo "inventory integration tests require GOSHOP_INVENTORY_TEST_MYSQL_DSN or INVENTORY_MYSQL_USERNAME/INVENTORY_MYSQL_PASSWORD" >&2
  exit 1
fi

cd "${ROOT_DIR}"
env GOCACHE="${GOCACHE:-/tmp/goshop-gocache}" \
  go test "${PKG}" -run "${PATTERN}" -v
