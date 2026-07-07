#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
README="${ROOT_DIR}/migrations/README.md"
USER_DB_FILE="app/user/srv/internal/data/v1/db/mysql.go"

if [[ ! -s "${README}" ]]; then
  echo "migrations/README.md is required and must document the migration policy" >&2
  exit 1
fi

matches="$(rg -n 'AutoMigrate\(' "${ROOT_DIR}/app" || true)"
if [[ -z "${matches}" ]]; then
  echo "migration check passed"
  exit 0
fi

unexpected="$(printf '%s\n' "${matches}" | awk -v allowed="${ROOT_DIR}/${USER_DB_FILE}" -F: '$1 != allowed { print }')"
if [[ -n "${unexpected}" ]]; then
  echo "unexpected GORM AutoMigrate usage found:" >&2
  printf '%s\n' "${unexpected}" >&2
  echo "use reviewed SQL migrations instead of application startup schema changes" >&2
  exit 1
fi

if ! rg -q 'if mysqlOpts\.AutoMigrate' "${ROOT_DIR}/${USER_DB_FILE}"; then
  echo "user service AutoMigrate must remain guarded by mysql.auto-migrate" >&2
  exit 1
fi

echo "migration check passed"
