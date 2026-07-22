#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
README="${ROOT_DIR}/migrations/README.md"
USER_DB_FILE="app/user/srv/internal/data/v1/db/mysql.go"
MIGRATIONS_DIR="${ROOT_DIR}/migrations"

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

assert_up_migration_contains() {
  local pattern="$1"
  local description="$2"

  if ! rg -q --glob '*.up.sql' "${pattern}" "${MIGRATIONS_DIR}"; then
    echo "missing reviewed migration coverage for ${description}" >&2
    exit 1
  fi
}

# user-srv startup validation requires these reviewed schema changes to exist
# before production can safely keep mysql.auto-migrate disabled.
assert_up_migration_contains 'account_status' 'user.account_status'
assert_up_migration_contains 'CREATE TABLE IF NOT EXISTS `roles`' 'roles table'
assert_up_migration_contains 'CREATE TABLE IF NOT EXISTS `user_roles`' 'user_roles table'
assert_up_migration_contains 'CREATE TABLE IF NOT EXISTS `role_permissions`' 'role_permissions table'
assert_up_migration_contains 'CREATE TABLE IF NOT EXISTS `role_domains`' 'role_domains table'
assert_up_migration_contains 'CREATE TABLE `user_audit_logs`' 'user_audit_logs table'
assert_up_migration_contains 'CREATE TABLE `admin_audit_logs`' 'admin_audit_logs table'

echo "migration check passed"
