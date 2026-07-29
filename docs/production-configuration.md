# 生产配置说明

`configs/` 中的 YAML 是示例和安全默认值，不得保存生产密钥。生产环境应由部署系统生成私有配置，并通过密钥管理系统提供敏感值。

## 配置来源

应用使用 Viper 读取由 `--config` 指定的文件。Viper 的环境变量前缀由服务 basename 决定，点号和连字符会转换为下划线；但只有代码显式读取的环境变量才可作为可靠覆盖机制。部署前请验证实际生效的配置，不要仅依赖 YAML 注释。

后台初始化令牌支持显式回退环境变量：

```text
GOSHOP_ADMIN_TOKEN
GOSHOP_ADMIN_CONFIRMATION_TOKEN
GOSHOP_ADMIN_PREVIOUS_TOKEN
```

## 必须由密钥系统管理的值

- MySQL、Redis 与 Nacos 凭据；
- JWT 签名密钥；
- 管理端令牌、应急令牌轮换元数据及高风险操作确认令牌；
- 短信、邮件与支付提供方密钥；
- 生产 TLS 私钥与 CA。

应急身份不拥有 RBAC 角色或权限；员工授权始终以 RBAC 数据库为准。应急会话必须使用独立令牌、短 TTL，并保留审计记录。

## 上线前最小检查

```bash
make config-secret-check
make migration-check
make startup-validation-check
make release-check
```

还应确认：

1. `server.profiling` 在生产环境为 `false`，或被限制在受保护的内部端点。
2. CORS 使用显式来源，不使用通配符。
3. `/metrics`、`/livez`、`/readyz`、`/healthz` 只允许来自 `built-in-route-cidrs`、私有入口或受信任代理的访问。
4. `mysql.auto-migrate` 为 `false`；schema 仅通过审核过的 migration 变更。
5. 跨主机 RPC 使用 TLS 或 mTLS；若临时使用明文，必须限制在私有网络并记录风险边界。

## 库存真实数据库集成测试

库存集成测试支持完整 DSN，或服务范围的 MySQL 环境变量：

```text
GOSHOP_INVENTORY_TEST_MYSQL_DSN
INVENTORY_MYSQL_USERNAME
INVENTORY_MYSQL_PASSWORD
INVENTORY_MYSQL_HOST
INVENTORY_MYSQL_PORT
INVENTORY_MYSQL_DATABASE
```

执行：

```bash
make inventory-integration-test
```

该测试覆盖并发扣减不超卖、重复扣减幂等、扣减确认、扣减释放、确认幂等，以及延迟扣减与释放的时序。
