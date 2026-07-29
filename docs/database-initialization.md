# MySQL 初始化

`deploy/mysql/initialize.sh` 用于初始化**空的** MySQL 实例：创建五个 GoShop 服务数据库、执行已审核 migration、把版本写入 `schema_migrations`，最后只导入正式 RBAC 角色和权限目录。

```bash
MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 \
MYSQL_USERNAME=goshop_migrator MYSQL_PASSWORD='from-secret-manager' \
bash deploy/mysql/initialize.sh
```

若 MySQL 只运行在本地 Docker 容器中，可使用附带的客户端适配器；容器名称不是 `mysql-server` 时请设置 `MYSQL_DOCKER_CONTAINER`：

```bash
MYSQL_CLIENT="$PWD/deploy/mysql/docker-mysql-client.sh" \
MYSQL_USERNAME=goshop_migrator MYSQL_PASSWORD='from-secret-manager' \
bash deploy/mysql/initialize.sh
```

初始化器会拒绝非空数据库。只有在核实 `schema_migrations` 后，才可设置 `GOSHOP_INIT_ALLOW_EXISTING=true` 恢复已验证的安装；已记录的版本会被跳过。迁移账户需要建库及 DDL/DML 权限。

脚本不会创建默认操作员、密码、客户、初始管理员或应急凭据。RBAC 种子与 `app/pkg/authz/roles.go` 保持一致，只包含 `support`、`ops`、`finance`、`catalog`、`review`、`admin` 和 `super_admin`。

`make schema-integration-test` 用于独立的、可丢弃 MySQL 实例启动校验；该测试要求 `mysql.auto-migrate=false`。
