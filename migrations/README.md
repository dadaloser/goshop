# Database Migrations
此目录存储每个服务的已审核数据库迁移。

## Naming
使用单调时间戳和服务名称：

```text
YYYYMMDDHHMMSS_service_short_description.up.sql
YYYYMMDDHHMMSS_service_short_description.down.sql
```

例子:

```text
202607050001_user_add_identity_columns.up.sql
202607050001_user_add_identity_columns.down.sql
202607050002_order_add_status_logs.up.sql
202607050002_order_add_status_logs.down.sql
```

## Rules

- 不要在生产中依赖 Gorm 自动迁移。
- 在生产中保持“mysql.auto-migrate”禁用。它仅适用于本地引导程序或临时开发环境。 
- 每个架构更改都必须有“向上”和“向下”迁移。 - 有风险的回滚步骤必须标记注释并手动审核。 
- 显式添加索引和唯一约束。 
- 在生产之前在暂存中运行迁移。 
- 当数据回填可能需要时间时，将数据回填与架构迁移分开。 
- 在审核的迁移中保留启动所需的架构。例如，“user-srv”启动验证当前需要“user.account_status”、“rolesuser_rolesrole_permissionsrole_domains”、“user_audit_logs”和“admin_audit_logs”在使用“mysql.auto-migrate=false”启动生产之前存在


## P0 Baseline

首个实施里程碑应添加经评审的数据库迁移脚本，涵盖以下内容：
- 商品核心表、分类/品牌/轮播图关联表，以及订单核心表。
- 订单列表筛选功能支持及订单状态变更日志。
- 后台管理员用户、角色、权限、角色绑定关系及操作审计日志。
- 库存现货、预占及变动日志表。
- 用户身份标识、会话信息及账户状态字段。

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

测试流程会分别连接至各服务的数据库，删除目标服务的相关表，从零开始应用经评审的服务专属迁移脚本，并验证启动时的 Schema 校验能否通过。
`mysql.auto-migrate=false`.
