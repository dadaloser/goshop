# GoShop 新手使用教程

本教程用于在本机启动 GoShop 的一组可用服务。建议先完成最小闭环：MySQL 初始化 → 用户服务 → 用户 API；再按需启动商品、库存、订单、评价和后台服务。

> 配置目录中的地址、开发证书与示例令牌并不等同于你的本机环境。请使用自己的私有配置文件，不要提交密码、令牌或生产证书。

## 1. 前置准备

| 组件 | 最低用途 | 何时需要 |
| --- | --- | --- |
| Go 1.26.5 | 编译和运行项目 | 始终需要；版本以 `go.mod` 为准。 |
| MySQL 8.0 | 五个服务数据库与迁移 | 用户、商品、库存、订单、评价服务。 |
| Consul | 服务注册与发现 | 启动任意应用服务时需要。 |
| Redis | API、后台和库存缓存/限流 | 启动 API、后台或库存服务时需要。 |
| Nacos | 用户服务的 Sentinel 配置 | 启动用户服务时需要。 |
| Elasticsearch | 商品搜索 | 启动商品服务并启用搜索时需要。 |
| DTM | 分布式事务 | 启动订单服务时需要。 |

先确认本机工具：

```bash
go version
docker version       # 仅当你用 Docker 启动基础设施时需要
mysql --version      # 或准备一个可访问的 MySQL 8.0 实例
```

项目没有提供一键式 compose 文件；可使用团队已有的基础设施，或自行通过 Docker/本机服务启动上述依赖。

## 2. 获取代码并完成基础校验

```bash
git clone <仓库地址> goshop
cd goshop
go mod download
go test ./pkg/...
make config-secret-check
```

`make config-secret-check` 用于阻止明显的明文密钥与不安全默认值进入配置。它不是密钥管理系统的替代品。

## 3. 准备本地配置

每个启动命令都通过 `--config` 指定 YAML 文件。建议在仓库外创建私有配置目录：

```bash
mkdir -p ../goshop-local-configs
cp configs/user/srv.yaml ../goshop-local-configs/user.yaml
cp configs/api/api.yaml ../goshop-local-configs/api.yaml
```

至少替换下列字段：

| 配置段 | 必须调整的内容 |
| --- | --- |
| `registry` | 本机或测试环境的 Consul 地址。 |
| `mysql` | 主机、端口、用户名、密码和对应数据库名。 |
| `redis` | 主机、端口、用户名（使用 ACL 时）与密码。 |
| `nacos` | 用户服务所用的 Nacos 地址、命名空间和凭据。 |
| `es` | 商品服务所用的 Elasticsearch 地址。 |
| `dtm` | 订单服务所用的 DTM gRPC/HTTP 地址。 |
| `rpc-security` | 本地可使用 `configs/tls/dev/` 中的开发证书；生产必须替换为正式证书与 CA。 |
| `jwt`、`admin-auth`、`email`、`payment` | 仅使用你自己的密钥；生产环境由密钥管理系统注入。 |

不要直接依赖 YAML 注释中提到的环境变量覆盖。当前应用启动时会读取 YAML；只有代码中显式支持的配置项（例如后台令牌）才会读取相应环境变量。将敏感配置放在仓库外或由部署系统生成。

## 4. 导入数据库数据

### 4.1 初始化空 MySQL

初始化脚本会创建以下数据库并执行已审核 migration：

- `goshop_user_srv`
- `goshop_goods_srv`
- `goshop_order_srv`
- `goshop_inventory_srv`
- `goshop_review_srv`

执行前使用具备建库和 DDL/DML 权限的专用迁移账户：

```bash
cd .. #切到主目录

MYSQL_HOST=192.168.1.139 MYSQL_PORT=5488 \
MYSQL_USERNAME=admin MYSQL_PASSWORD='admin@admin' \
bash deploy/mysql/initialize.sh
```

脚本默认拒绝对非空数据库执行初始化。只有确认 `schema_migrations` 状态正确后，才允许设置 `GOSHOP_INIT_ALLOW_EXISTING=true` 继续。

初始化会导入正式 RBAC 角色和权限目录，但**不会**创建默认管理员、用户、密码或应急账号。测试业务数据应通过业务 API、受控导入脚本或经过评审的 SQL 单独导入。

若 MySQL 只在 Docker 容器中运行：

```bash
cd .. #切到主目录

MYSQL_CLIENT="$PWD/deploy/mysql/docker-mysql-client.sh" \
MYSQL_DOCKER_CONTAINER=mysql-server \
MYSQL_USERNAME=admin MYSQL_PASSWORD='admin@admin' \
bash deploy/mysql/initialize.sh
```

详细说明见 [数据库初始化](database-initialization.md)。

### 4.2 导入现有业务数据

导入已有数据前，先完成 schema 初始化，并确认源数据版本与 `schema_migrations` 一致。推荐流程：

1. 在隔离的 MySQL 实例恢复备份。
2. 检查五个数据库的 `schema_migrations` 与本仓库 migration 一致。
3. 用最小权限账号导入数据；不要覆盖正式 RBAC 种子或生产凭据。
4. 以只读查询和服务健康检查验证，再切换应用连接。

## 5. 按依赖顺序启动服务

先确保 Consul、MySQL、Redis、Nacos（用户服务）已经可用。每个命令在独立终端执行。

```bash
# 1. 用户服务
go run ./cmd/user --config ../goshop-local-configs/user.yaml

# 2. 用户 API
go run ./cmd/api --config ../goshop-local-configs/api.yaml
```

按需继续启动：

```bash
go run ./cmd/goods --config ../goshop-local-configs/goods.yaml
go run ./cmd/inventory --config ../goshop-local-configs/inventory.yaml
go run ./cmd/order --config ../goshop-local-configs/order.yaml
go run ./cmd/review --config ../goshop-local-configs/review.yaml
go run ./cmd/admin --config ../goshop-local-configs/admin.yaml
```

建议启动顺序：用户 → 商品/库存 → 订单/评价 → API/后台。商品服务需要 ES；订单服务需要 DTM；API、后台和库存服务需要 Redis。

## 6. 启动后验证

1. 检查终端日志，确认应用已经完成服务注册。
2. 在 Consul 中确认服务实例为健康状态。
3. 用配置中的 HTTP 端口访问 `/livez`；管理端口及 `/metrics`、`/readyz`、`/healthz` 受 `built-in-route-cidrs` 限制。
4. 运行基础测试：

   ```bash
   go test ./pkg/...
   make vet-check
   ```

5. 在进行写操作前，使用受控测试账户验证登录、鉴权和一个最小业务流程。

## 7. 后续要求

- **配置与密钥**：生产配置必须使用真实 TLS、显式 CORS 来源、私有运维端点和密钥管理系统；禁止提交密钥。
- **数据库**：生产环境保持 `mysql.auto-migrate: false`；仅通过审核过的 migration 变更 schema。
- **接口变更**：修改 `.proto` 后执行 `make proto` 与 `make proto-check`，并一并提交生成文件。
- **发布**：发布前执行 `make release-check`，并按团队发布前检查清单完成验证。
- **故障处理**：遵循团队的部署、回滚与故障处理流程；不要直接在生产库执行未审核 SQL。

## 常见问题

| 现象 | 优先检查 |
| --- | --- |
| 服务无法注册 | `registry.address`、Consul 可达性、服务监听地址和证书配置。 |
| 服务启动即退出 | YAML 路径、MySQL/Redis/Nacos/ES/DTM 地址和启动校验错误。 |
| 无法访问健康检查或指标 | 请求来源是否在 `built-in-route-cidrs` 中，端口是否正确。 |
| 初始化脚本拒绝执行 | 数据库不是空库；先检查 `schema_migrations`，不要盲目设置允许已有库。 |
| Proto 校验失败 | 执行 `make proto`，检查并提交 `api/` 下生成文件。 |
