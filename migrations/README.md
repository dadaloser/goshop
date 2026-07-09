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

## P0 Baseline

The first implementation milestone should add reviewed migrations for:

- order list filtering support and order status logs.
- admin users, roles, permissions, role bindings, and audit logs.
- inventory stock/reservation/log tables.
- user identity/session/account status fields.
