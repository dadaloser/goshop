# Database Migrations

This directory stores reviewed database migrations for each service.

## Naming

Use monotonic timestamps and service names:

```text
YYYYMMDDHHMMSS_service_short_description.up.sql
YYYYMMDDHHMMSS_service_short_description.down.sql
```

Examples:

```text
202607050001_user_add_identity_columns.up.sql
202607050001_user_add_identity_columns.down.sql
202607050002_order_add_status_logs.up.sql
202607050002_order_add_status_logs.down.sql
```

## Rules

- Do not rely on Gorm auto-migration in production.
- Keep `mysql.auto-migrate` disabled in production. It exists only for local bootstrap or temporary development environments.
- Every schema change must have an `up` and `down` migration.
- Risky rollback steps must be marked with comments and reviewed manually.
- Add indexes and unique constraints explicitly.
- Run migrations in staging before production.
- Keep data backfills separate from schema migrations when they may take time.
- Keep startup-required schema in reviewed migrations. For example, `user-srv`
  startup validation currently requires `user.account_status`,
  `roles/user_roles/role_permissions/role_domains`, `user_audit_logs`, and
  `admin_audit_logs` to exist before production boots with `mysql.auto-migrate=false`.

## P0 Baseline

The first implementation milestone should add reviewed migrations for:

- goods core tables, category/brand/banner relations, and order core tables.
- order list filtering support and order status logs.
- admin users, roles, permissions, role bindings, and audit logs.
- inventory stock/reservation/log tables.
- user identity/session/account status fields.

## Schema Smoke Test

The repository now includes a real-MySQL schema smoke test for `user-srv`,
`goods-srv`, `order-srv`, `inventory-srv`, and `review-srv` startup validation:

```bash
make schema-integration-test
```

Set either:

- all of `GOSHOP_USER_SCHEMA_TEST_MYSQL_DSN`,
  `GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN`,
  `GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN`, and
  `GOSHOP_INVENTORY_SCHEMA_TEST_MYSQL_DSN`, and
  `GOSHOP_REVIEW_SCHEMA_TEST_MYSQL_DSN`
- or shared `GOSHOP_SCHEMA_TEST_MYSQL_USERNAME` / `GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD`
  with optional `GOSHOP_SCHEMA_TEST_MYSQL_HOST`,
  `GOSHOP_SCHEMA_TEST_MYSQL_PORT`,
  `GOSHOP_USER_SCHEMA_TEST_MYSQL_DATABASE`,
  `GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DATABASE`, and
  `GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DATABASE`, and
  `GOSHOP_INVENTORY_SCHEMA_TEST_MYSQL_DATABASE`, and
  `GOSHOP_REVIEW_SCHEMA_TEST_MYSQL_DATABASE`
- or existing service credentials:
  `USER_MYSQL_USERNAME` / `USER_MYSQL_PASSWORD`,
  `GOODS_MYSQL_USERNAME` / `GOODS_MYSQL_PASSWORD` and
  `ORDER_MYSQL_USERNAME` / `ORDER_MYSQL_PASSWORD` and
  `INVENTORY_MYSQL_USERNAME` / `INVENTORY_MYSQL_PASSWORD` and
  `REVIEW_MYSQL_USERNAME` / `REVIEW_MYSQL_PASSWORD`
  with optional `GOODS_MYSQL_HOST`, `GOODS_MYSQL_PORT`,
  `GOODS_MYSQL_DATABASE`, `ORDER_MYSQL_HOST`, `ORDER_MYSQL_PORT`,
  `ORDER_MYSQL_DATABASE`, `USER_MYSQL_HOST`, `USER_MYSQL_PORT`,
  `USER_MYSQL_DATABASE`, `INVENTORY_MYSQL_HOST`,
  `INVENTORY_MYSQL_PORT`, `INVENTORY_MYSQL_DATABASE`,
  `REVIEW_MYSQL_HOST`, `REVIEW_MYSQL_PORT`, and `REVIEW_MYSQL_DATABASE`

The test flow connects to each service database separately, drops the target
service tables, applies the reviewed service-specific migrations from scratch,
and verifies that startup schema validation passes with
`mysql.auto-migrate=false`.
