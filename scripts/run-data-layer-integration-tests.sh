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

check_coverage_gate() {
  local name="$1" prefix="$2" minimum="$3"
  local result
  result="$(awk -v prefix="${prefix}" '
    NR > 1 {
      split($1, location, ":")
      if (index(location[1], prefix) == 1) {
        total += $2
        if ($3 > 0) covered += $2
      }
    }
    END { if (total == 0) print "NOFILES"; else printf "%.2f %d %d", covered * 100 / total, covered, total }
  ' "${PROFILE}")"
  if [[ "${result}" == NOFILES ]]; then
    echo "[data-coverage] ${name}: no matching statements" >&2
    return 1
  fi
  read -r percent covered total <<< "${result}"
  if ! awk -v actual="${percent}" -v minimum="${minimum}" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; then
    echo "[data-coverage] ${name}: ${percent}% (${covered}/${total}) is below ${minimum}%" >&2
    return 1
  fi
  echo "[data-coverage] ${name}: ${percent}% (${covered}/${total})"
}

# 数据层覆盖率仅由本集成测试产生的 profile 使用，因此就近校验，避免维护独立脚本。
check_coverage_gate goods-db goshop/app/goods/srv/internal/data/v1/db/ 15
check_coverage_gate inventory-db goshop/app/inventory/srv/internal/data/v1/db/ 25
