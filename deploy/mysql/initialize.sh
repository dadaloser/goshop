#!/usr/bin/env bash
# Initializes an empty MySQL instance. Reviewed migrations are the sole table DDL source.
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
: "${MYSQL_HOST:=127.0.0.1}"
: "${MYSQL_PORT:=3306}"
: "${MYSQL_USERNAME:?set MYSQL_USERNAME to an account with CREATE DATABASE privileges}"
: "${MYSQL_PASSWORD:?set MYSQL_PASSWORD without putting it in the command line}"
: "${MYSQL_CLIENT:=mysql}"
: "${GOSHOP_INIT_ALLOW_EXISTING:=false}"

mysql_base=("${MYSQL_CLIENT}" --protocol=TCP --host="${MYSQL_HOST}" --port="${MYSQL_PORT}" --user="${MYSQL_USERNAME}" --default-character-set=utf8mb4)
mysql_exec() { MYSQL_PWD="${MYSQL_PASSWORD}" "${mysql_base[@]}" "$@"; }

ensure_database() {
  local database="$1" table_count
  mysql_exec -e "CREATE DATABASE IF NOT EXISTS \`${database}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci"
  table_count="$(mysql_exec -N -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='${database}' AND table_name <> 'schema_migrations'")"
  if [[ "${GOSHOP_INIT_ALLOW_EXISTING}" != true && "${table_count}" != 0 ]]; then
    echo "refusing to initialize non-empty database ${database}; set GOSHOP_INIT_ALLOW_EXISTING=true only after verifying schema_migrations" >&2
    exit 1
  fi
  mysql_exec "${database}" -e 'CREATE TABLE IF NOT EXISTS schema_migrations (version varchar(128) NOT NULL, applied_at datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3), PRIMARY KEY (version)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4'
}
already_applied() { [[ "$(mysql_exec "$1" -N -e "SELECT COUNT(*) FROM schema_migrations WHERE version='$2'")" != 0 ]]; }
record_migration() { mysql_exec "$1" -e "INSERT INTO schema_migrations (version) VALUES ('$2')"; }
apply_file() {
  local database="$1" migration="$2" version
  version="$(basename "${migration}" .up.sql)"
  already_applied "${database}" "${version}" && return
  echo "apply ${database}: ${version}"
  MYSQL_PWD="${MYSQL_PASSWORD}" "${mysql_base[@]}" "${database}" < "${ROOT_DIR}/${migration}"
  record_migration "${database}" "${version}"
}
# One historical migration owns statements for three service databases. Extract
# the service-owned SQL at runtime; do not duplicate table definitions.
apply_fragment() {
  local database="$1" version="$2" expression="$3"
  already_applied "${database}" "${version}" && return
  echo "apply ${database}: ${version}"
  awk -v expression="${expression}" 'BEGIN { RS=";" } $0 ~ expression { print $0 ";" }' "${ROOT_DIR}/migrations/202607230002_admin_resource_scopes_and_refunds.up.sql" | MYSQL_PWD="${MYSQL_PASSWORD}" "${mysql_base[@]}" "${database}"
  record_migration "${database}" "${version}"
}

for database in goshop_user_srv goshop_goods_srv goshop_order_srv goshop_inventory_srv goshop_review_srv; do ensure_database "${database}"; done

apply_file goshop_user_srv migrations/202607040001_user_create_core_tables.up.sql
apply_file goshop_user_srv migrations/202607050001_user_add_identity_columns.up.sql
apply_file goshop_user_srv migrations/202607200001_user_add_audit_logs.up.sql
apply_file goshop_user_srv migrations/202607210001_user_add_admin_audit_logs.up.sql
apply_file goshop_user_srv migrations/202607220001_user_add_account_status.up.sql
apply_file goshop_user_srv migrations/202607220002_user_add_rbac_core_tables.up.sql
apply_file goshop_user_srv migrations/202607230001_user_add_identity_and_sessions.up.sql
apply_fragment goshop_user_srv 202607230002_user_resource_scopes 'CREATE TABLE `user_resource_scopes`|ALTER TABLE `admin_audit_logs`'
apply_file goshop_user_srv migrations/202607230006_user_add_review_rbac.up.sql
apply_file goshop_user_srv migrations/202607240001_user_role_fk_constraints.up.sql
apply_file goshop_user_srv migrations/202607250001_auth_resource_scope_staff_sessions_break_glass.up.sql

apply_file goshop_goods_srv migrations/202607070001_goods_create_core_tables.up.sql
apply_file goshop_goods_srv migrations/202607080001_goods_add_outbox_events.up.sql
apply_file goshop_goods_srv migrations/202607220003_goods_add_money_fen_columns.up.sql
apply_file goshop_goods_srv migrations/202607220005_goods_drop_float_money_columns.up.sql
apply_file goshop_goods_srv migrations/202607230004_goods_outbox_claim_and_sku.up.sql
apply_file goshop_goods_srv migrations/202607260001_auth_resource_ownership_store_ids.up.sql

apply_file goshop_order_srv migrations/202607070002_order_create_core_tables.up.sql
apply_file goshop_order_srv migrations/202607100001_order_add_status_logs.up.sql
apply_file goshop_order_srv migrations/202607220004_order_add_money_fen_columns.up.sql
apply_file goshop_order_srv migrations/202607220006_order_drop_float_money_columns.up.sql
apply_fragment goshop_order_srv 202607230002_order_refund_requests 'CREATE TABLE `order_refund_requests`'
apply_file goshop_order_srv migrations/202607230003_order_add_payment_events.up.sql
apply_file goshop_order_srv migrations/202607240001_payment_refund_outbox_reconciliation.up.sql
apply_file goshop_order_srv migrations/202607260002_order_add_store_id.up.sql

apply_file goshop_inventory_srv migrations/202607040002_inventory_create_core_tables.up.sql
apply_file goshop_inventory_srv migrations/202607090001_inventory_add_stock_lifecycle_columns.up.sql
apply_fragment goshop_inventory_srv 202607230002_inventory_adjustment_logs 'CREATE TABLE `inventory_adjustment_logs`'
apply_file goshop_review_srv migrations/202607230005_review_domain.up.sql

if ! already_applied goshop_user_srv seed_formal_rbac_v1; then
  echo 'seed goshop_user_srv: formal RBAC roles and permissions'
  MYSQL_PWD="${MYSQL_PASSWORD}" "${mysql_base[@]}" goshop_user_srv < "${ROOT_DIR}/deploy/mysql/seed-formal-rbac.sql"
  record_migration goshop_user_srv seed_formal_rbac_v1
fi
echo 'GoShop MySQL initialization complete.'
