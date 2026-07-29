# 依赖韧性

本文档说明出站 gRPC、Redis 与 MySQL 调用的 Sentinel 保护策略和 Prometheus 指标。

## 保护模型

每个受保护操作包含：

- 基于 `context` 的超时；
- Sentinel 并发隔离；
- 基于错误比例的熔断；
- 熔断前的最小请求数、统计窗口与恢复窗口；
- 当隔离或熔断拒绝请求时的快速失败回退。

资源标签保持有界，避免高基数指标：

| 依赖 | 资源标签 | 示例 |
| --- | --- | --- |
| gRPC | Protobuf 完整方法名 | `goshop.user.v1.User/GetUserById` |
| Redis | 命令名 | `get`、`set`、`pipeline` |
| MySQL | GORM 操作类别 | `create`、`query`、`update`、`delete`、`row`、`raw` |

Redis `Nil`、MySQL 记录不存在和约束冲突、调用方取消，以及正常的 gRPC 业务状态码都不会计入熔断错误比例。

## 配置

MySQL 与 Redis 策略放在各自依赖配置段；API 和订单服务的出站 RPC 使用 `rpc-client-resilience`：

```yaml
rpc-client-resilience:
  enabled: true
  timeout: 2s
  max-concurrency: 100
  error-ratio: 0.5
  min-request-amount: 20
  stat-interval: 10s
  recovery-timeout: 30s
```

将 `enabled` 设为 `false` 会关闭 Sentinel 隔离和熔断，但操作超时仍然生效，避免出现无限等待的调用。

## 指标

| 指标 | 标签 | 含义 |
| --- | --- | --- |
| `dependency_resilience_requests_total` | `dependency`、`resource`、`outcome` | 请求结果：`success`、`error`、`timeout`、`canceled` 或 `blocked`。 |
| `dependency_resilience_duration_ms` | `dependency`、`resource`、`outcome` | 操作耗时直方图。 |
| `dependency_resilience_inflight` | `dependency`、`resource` | 当前隔离并发数。 |
| `dependency_resilience_fallback_total` | `dependency`、`resource`、`reason` | 按 `isolation`、`circuit_open` 等原因统计的快速失败次数。 |
| `dependency_resilience_circuit_transitions_total` | `dependency`、`resource`、`from`、`to` | 熔断状态转换次数。 |
| `dependency_resilience_circuit_state` | `dependency`、`resource` | 当前状态：`0` 关闭、`1` 半开、`2` 打开。 |
| `dependency_resilience_recovery_total` | `dependency`、`resource` | 半开恢复到关闭的成功次数。 |

常用 Grafana 查询：

```promql
sum by (dependency, resource, outcome) (
  rate(dependency_resilience_requests_total[5m])
)
```

```promql
histogram_quantile(
  0.95,
  sum by (le, dependency, resource) (
    rate(dependency_resilience_duration_ms_bucket[5m])
  )
)
```

```promql
max by (dependency, resource) (dependency_resilience_circuit_state)
```

告警规则位于 `monitoring/prometheus/dependency-resilience-alerts.yaml`。并发阈值应与实际 `max-concurrency` 配置匹配，先观察告警再升级为分页通知。
