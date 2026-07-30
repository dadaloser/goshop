# NATS JetStream（本地开发）

账号注销事件使用 JetStream，消息主题限定为 `goshop.account.deletion.>`。

启动：

```bash
docker compose -f deploy/nats/compose.yaml up -d
```

服务端口为 `4222`，监控端口为 `8222`。生产环境必须通过受管 NATS 集群、TLS 和账户权限策略部署；不要直接复用此开发配置。
