#!/usr/bin/env bash
set -euo pipefail

PROFILE="${COVERPROFILE:-/tmp/goshop-coverage.out}"

if [[ ! -f "${PROFILE}" ]]; then
  echo "coverage profile not found: ${PROFILE}" >&2
  exit 1
fi

declare -a GATES=(
  "money:70:goshop/pkg/money/"
  "authz:70:goshop/app/pkg/authz/"
  "order:70:goshop/app/goshop/api/internal/service/order/v1/"
  "payment:70:goshop/app/goshop/api/internal/payment/"
  "session:70:goshop/app/user/srv/internal/service/v1/session.go|goshop/app/user/srv/internal/controller/user/session.go|goshop/app/user/srv/internal/data/v1/session.go"
)

status=0
for gate in "${GATES[@]}"; do
  IFS=":" read -r name threshold patterns <<< "${gate}"
  summary="$(
    awk -v patterns="${patterns}" '
      BEGIN {
        count = split(patterns, matcher, "|")
      }
      NR == 1 { next }
      {
        split($1, location, ":")
        file = location[1]
        statements = $2 + 0
        covered_count = $3 + 0
        for (i = 1; i <= count; i++) {
          if (index(file, matcher[i]) == 1) {
            total += statements
            if (covered_count > 0) {
              covered += statements
            }
            matched = 1
            break
          }
        }
      }
      END {
        if (total == 0) {
          print "NOFILES"
          exit 0
        }
        printf "%.2f %d %d", (covered / total) * 100, covered, total
      }
    ' "${PROFILE}"
  )"

  if [[ "${summary}" == "NOFILES" ]]; then
    echo "[coverage] ${name}: no matching files in ${PROFILE}" >&2
    status=1
    continue
  fi

  read -r percent covered total <<< "${summary}"
  if ! awk -v actual="${percent}" -v minimum="${threshold}" 'BEGIN { exit !(actual + 0 >= minimum + 0) }'; then
    echo "[coverage] ${name}: ${percent}% (${covered}/${total}) is below ${threshold}%" >&2
    status=1
    continue
  fi
  echo "[coverage] ${name}: ${percent}% (${covered}/${total})"
done

exit "${status}"
