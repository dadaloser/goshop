#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOCACHE_DIR="${GOCACHE:-/tmp/goshop-gocache}"
PROFILE="${GOSHOP_DATA_LAYER_COVERAGE_PROFILE:-/tmp/goshop-data-layer-coverage.out}"

for value in GOODS_MYSQL_USERNAME GOODS_MYSQL_PASSWORD INVENTORY_MYSQL_USERNAME INVENTORY_MYSQL_PASSWORD; do
  if [[ -z "${!value:-}" ]]; then
    echo "data-layer integration tests require ${value}" >&2
    exit 1
  fi
done

cd "${ROOT_DIR}"
tmp_dir="$(mktemp -d "${TMPDIR:-/tmp}/goshop-data-cover.XXXXXX")"
trap 'rm -rf "${tmp_dir}"' EXIT
printf 'mode: atomic\n' > "${PROFILE}"

packages=(
  ./app/goods/srv/internal/data/v1/db
  ./app/inventory/srv/internal/data/v1/db
)
for index in "${!packages[@]}"; do
  profile="${tmp_dir}/${index}.out"
  env GOCACHE="${GOCACHE_DIR}" go test -count=1 -covermode=atomic -coverprofile="${profile}" "${packages[${index}]}" -run 'Test(GoodsStore|Outbox|InventoryStore).*RealDB' -v
  tail -n +2 "${profile}" >> "${PROFILE}"
done

GOSHOP_DATA_LAYER_COVERAGE_PROFILE="${PROFILE}" bash ./scripts/data-layer-coverage-check.sh
