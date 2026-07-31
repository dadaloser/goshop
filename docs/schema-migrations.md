# MySQL `schema_migrations` 操作指南

## 用途

每个服务数据库都有一张 `schema_migrations` 表，用于记录**已成功执行**的迁移版本：

```sql
CREATE TABLE schema_migrations (
  version varchar(128) NOT NULL,
  applied_at datetime(3) NOT NULL,
  PRIMARY KEY (version)
);
```

`deploy/mysql/initialize.sh` 会先检查迁移文件名（去掉 `.up.sql` 后缀）是否已存在于该表中：存在则跳过，不存在才执行 SQL，并在成功后写入版本。因此同一迁移可安全地重复执行初始化脚本；失败的迁移不会被记录为已完成。

不要把用户库的版本写入订单库，或反过来执行迁移文件；迁移必须按服务所属数据库执行。

## 执行已有数据库的缺失迁移

先备份目标库，并确认迁移 SQL 已审核。对于已有数据库，使用项目脚本执行：

```bash
cd ..

MYSQL_HOST=192.168.1.139 \
MYSQL_PORT=5488 \
MYSQL_USERNAME=admin \
MYSQL_PASSWORD='admin@admin' \
GOSHOP_INIT_ALLOW_EXISTING=true \
bash deploy/mysql/initialize.sh
```

`GOSHOP_INIT_ALLOW_EXISTING=true` 只适用于已经核验过的已有安装。脚本会执行该项目中所有尚未记录的受审迁移，而非只执行最近新增的一个文件。

空实例不要设置该变量，直接按照[数据库初始化](database-initialization.md)执行即可。

## 执行前核验

分别连接对应数据库，检查已执行版本：

```sql
USE goshop_user_srv;
SELECT version, applied_at
FROM schema_migrations
ORDER BY applied_at, version;

USE goshop_order_srv;
SELECT version, applied_at
FROM schema_migrations
ORDER BY applied_at, version;
```

仅核验账号注销相关迁移时：

```sql
SELECT version, applied_at
FROM schema_migrations
WHERE version IN (
  '202607310001_user_device_blacklist',
  '202607310002_user_session_client_metadata',
  '202607300002_order_add_account_deletion_events'
);
```

注意：前两条记录应出现在 `goshop_user_srv`，账号注销事件迁移应出现在 `goshop_order_srv`。

## 执行后核验

```sql
USE goshop_user_srv;
SHOW TABLES LIKE 'user_account_deletion_outbox';
DESCRIBE user_account_deletion_outbox;

USE goshop_order_srv;
SHOW TABLES LIKE 'order_account_deletion_inbox';
SHOW TABLES LIKE 'order_account_deletion_outbox';
```

然后重启相关服务。服务以 `mysql.auto-migrate=false` 启动时会进行 schema 校验；若表或关键列缺失，服务会在启动时失败，从而避免以不完整 schema 运行。

## 失败与回滚

- 迁移失败时，不要手动向 `schema_migrations` 插入版本记录。
- 先修正失败原因，再重新运行初始化脚本；未记录的迁移会再次尝试。
- `*.down.sql` 是人工审核后的回滚脚本，初始化器不会自动执行。
- 回滚前先确认没有依赖新表/新列的应用版本正在运行，并备份受影响数据。
- 不要删除 `schema_migrations` 中的历史记录来“强制重跑”迁移；这可能导致重复建表、重复数据变更或破坏线上数据。
