# NATS JetStream 与账号注销事件

## 流程

账号注销使用“事务 Outbox + JetStream”：用户服务先将账号置为 `deletion_pending` 并撤销会话，再将申请写入 `user_account_deletion_outbox`。

```text
注销申请 → 用户 Outbox → goshop.account.deletion.requested
         → 订单审核 → confirmed / rejected → 用户服务最终处理
```

## 本地启动与配置

```bash
cd /Users/tanglanqin/GolandProjects/goshop
docker compose -f deploy/nats/compose.yaml up -d
docker compose -f deploy/nats/compose.yaml ps
```

客户端地址为 `nats://127.0.0.1:4222`，监控端点为 `http://127.0.0.1:8222`。

```yaml
account-deletion-events:
  url: "nats://127.0.0.1:4222"
  poll-interval: 2s
  batch-size: 50
  max-retries: 20
```

尚未部署 NATS 时，将 `url` 设为空字符串；注销申请仍会写入 Outbox，但不投递也不会产生连接失败日志。

## Outbox 行为

用户服务以 `FOR UPDATE SKIP LOCKED` 领取 `PENDING` 事件，以事件 ID 作为 `Nats-Msg-Id` 发布 `goshop.account.deletion.requested`。发布成功后标记为 `PUBLISHED`；失败会退避重试。

“发布成功、更新 Outbox 失败”会造成重复投递，这是至少一次投递的预期行为。订单消费者必须按事件 ID 使用 Inbox 去重，并且仅在本地事务提交后 ACK。

## 上线要求与边界

- NATS 必须启用 JetStream，并创建包含 `goshop.account.deletion.>` 的 Stream。
- 生产使用 TLS、账户认证和最小主题权限，不可复用本地 compose 配置。
- 对 Outbox 积压、重试、死信以及消费者延迟设置监控告警。
- 当前已完成用户侧请求 Outbox 与发布工作器；订单 durable consumer、结果 Outbox 投递，以及用户服务消费 `confirmed` / `rejected` 仍需完成，未完成前注销审核并未闭环。

## 故障处理

`nats: no servers available for connection` 表示配置地址没有可用 NATS。事件仍保留在用户 Outbox；恢复 NATS 后由工作器重试。不要手动将事件标记为 `PUBLISHED`。
