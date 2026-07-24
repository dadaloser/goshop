#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
README="${ROOT_DIR}/migrations/README.md"
USER_DB_FILE="app/user/srv/internal/data/v1/db/mysql.go"
REVIEW_DB_FILE="app/review/srv/internal/data/db/mysql.go"
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

unexpected="$(printf '%s\n' "${matches}" | awk -v allowed_user="${ROOT_DIR}/${USER_DB_FILE}" -v allowed_review="${ROOT_DIR}/${REVIEW_DB_FILE}" -F: '$1 != allowed_user && $1 != allowed_review { print }')"
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
if ! rg -q 'if mysqlOpts\.AutoMigrate' "${ROOT_DIR}/${REVIEW_DB_FILE}"; then
  echo "review service AutoMigrate must remain guarded by mysql.auto-migrate" >&2
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

# startup validation requires these reviewed schema changes to exist before
# production can safely keep mysql.auto-migrate disabled.
assert_up_migration_contains 'CREATE TABLE `category`' 'goods category table'
assert_up_migration_contains 'CREATE TABLE `brands`' 'goods brands table'
assert_up_migration_contains 'CREATE TABLE `banner`' 'goods banner table'
assert_up_migration_contains 'CREATE TABLE `goodscategorybrand`' 'goods category-brand table'
assert_up_migration_contains 'CREATE TABLE `goods`' 'goods table'
assert_up_migration_contains 'CREATE TABLE `orderinfo`' 'orderinfo table'
assert_up_migration_contains 'CREATE TABLE `ordergoods`' 'ordergoods table'
assert_up_migration_contains 'CREATE TABLE `shoppingcart`' 'shoppingcart table'
assert_up_migration_contains 'CREATE TABLE `user`' 'user table'
assert_up_migration_contains 'account_status' 'user.account_status'
assert_up_migration_contains 'idx_username' 'user.username unique index'
assert_up_migration_contains 'idx_email' 'user.email unique index'
assert_up_migration_contains 'CREATE TABLE `user_sessions`' 'user_sessions table'
assert_up_migration_contains 'CREATE TABLE `verification_codes`' 'verification_codes table'
assert_up_migration_contains 'CREATE TABLE IF NOT EXISTS `roles`' 'roles table'
assert_up_migration_contains 'idx_role_name' 'roles.name unique index'
assert_up_migration_contains 'CREATE TABLE IF NOT EXISTS `user_roles`' 'user_roles table'
assert_up_migration_contains 'idx_user_role_user' 'user_roles.user_id index'
assert_up_migration_contains 'idx_user_role_role' 'user_roles.role_id index'
assert_up_migration_contains 'CREATE TABLE IF NOT EXISTS `role_permissions`' 'role_permissions table'
assert_up_migration_contains 'idx_role_permission_role' 'role_permissions.role_id index'
assert_up_migration_contains 'CREATE TABLE IF NOT EXISTS `role_domains`' 'role_domains table'
assert_up_migration_contains 'idx_role_domain_role' 'role_domains.role_id index'
assert_up_migration_contains 'CREATE TABLE `user_audit_logs`' 'user_audit_logs table'
assert_up_migration_contains 'idx_user_audit_logs_user' 'user_audit_logs.user_id index'
assert_up_migration_contains 'idx_user_audit_logs_actor' 'user_audit_logs.actor_user_id index'
assert_up_migration_contains 'CREATE TABLE `admin_audit_logs`' 'admin_audit_logs table'
assert_up_migration_contains 'idx_admin_audit_logs_target' 'admin_audit_logs.target_user_id index'
assert_up_migration_contains 'idx_admin_audit_logs_actor' 'admin_audit_logs.actor_user_id index'
assert_up_migration_contains 'CREATE TABLE `inventory`' 'inventory table'
assert_up_migration_contains 'CREATE TABLE `stockselldetail`' 'stockselldetail table'
assert_up_migration_contains 'CREATE TABLE `inventory_adjustment_logs`' 'inventory_adjustment_logs table'
assert_up_migration_contains 'CREATE TABLE `reviews`' 'reviews table'
assert_up_migration_contains 'CREATE TABLE `review_appends`' 'review_appends table'
assert_up_migration_contains 'CREATE TABLE `review_replies`' 'review_replies table'
assert_up_migration_contains 'CREATE TABLE `review_audit_logs`' 'review_audit_logs table'
assert_up_migration_contains 'CREATE TABLE `review_outbox_events`' 'review_outbox_events table'
assert_up_migration_contains 'CREATE TABLE `review_product_ratings`' 'review_product_ratings table'
assert_up_migration_contains 'market_price_fen' 'goods.market_price_fen column'
assert_up_migration_contains 'shop_price_fen' 'goods.shop_price_fen column'
assert_up_migration_contains 'order_mount_fen' 'orderinfo.order_mount_fen column'
assert_up_migration_contains 'goods_price_fen' 'ordergoods.goods_price_fen column'
assert_up_migration_contains 'DROP COLUMN `shop_price`' 'goods.shop_price legacy drop'
assert_up_migration_contains 'DROP COLUMN `market_price`' 'goods.market_price legacy drop'
assert_up_migration_contains 'DROP COLUMN `order_mount`' 'orderinfo.order_mount legacy drop'
assert_up_migration_contains 'DROP COLUMN `goods_price`' 'ordergoods.goods_price legacy drop'

echo "migration check passed"
