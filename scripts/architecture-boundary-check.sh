#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

if rg -n '"(gorm\.io|database/sql|github\.com/go-sql-driver|github\.com/olivere/elastic|github\.com/segmentio/kafka-go|github\.com/rabbitmq)' app/goshop/api app/goshop/admin --glob '*.go'; then
  echo "API/Admin aggregation layers must not import persistence, search, or broker implementations" >&2
  exit 1
fi

if rg -n '^\s*(role|permissions):' configs/admin/admin.yaml; then
  echo "bootstrap role/permission configuration is forbidden; use database-backed RBAC" >&2
  exit 1
fi

if rg -n 'GOSHOP_ADMIN_(ROLE|PERMISSIONS)|admin-auth\.(role|permissions)' app configs --glob '!**/*_test.go'; then
  echo "bootstrap role/permission environment or config authorization is forbidden" >&2
  exit 1
fi

while IFS= read -r file; do
  lines="$(wc -l < "$file" | tr -d ' ')"
  if (( lines > 750 )); then
    echo "$file has $lines lines; split aggregation responsibilities before adding more" >&2
    exit 1
  fi
done < <(rg --files app/goshop/api app/goshop/admin -g '*.go' -g '!**/*_test.go' | sort)

echo "architecture boundary check passed"
