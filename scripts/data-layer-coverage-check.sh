#!/usr/bin/env bash
set -euo pipefail

PROFILE="${GOSHOP_DATA_LAYER_COVERAGE_PROFILE:-/tmp/goshop-data-layer-coverage.out}"
if [[ ! -f "${PROFILE}" ]]; then
  echo "data-layer coverage profile not found: ${PROFILE}" >&2
  exit 1
fi

check_gate() {
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

check_gate goods-db goshop/app/goods/srv/internal/data/v1/db/ 15
check_gate inventory-db goshop/app/inventory/srv/internal/data/v1/db/ 25
