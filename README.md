# GoShop

GoShop 是一个以 Go 实现的电商微服务示例。服务通过 gRPC 通信，并以 HTTP API 与后台管理端作为外部入口。

## 服务组成

| 服务 | 目录 | 主要职责 |
| --- | --- | --- |
| 用户服务 | `app/user/srv` | 用户、认证、会话和 RBAC 数据。 |
| 商品服务 | `app/goods/srv` | 商品、品牌、分类与搜索同步。 |
| 库存服务 | `app/inventory/srv` | 库存查询、扣减和调整。 |
| 订单服务 | `app/order/srv` | 订单、购物车、支付事件及生命周期任务。 |
| 评价服务 | `app/review/srv` | 评价领域及评分同步任务。 |
| 用户 API | `app/goshop/api` | 面向用户的 HTTP API 网关。 |
| 后台管理 | `app/goshop/admin` | 后台 HTTP API、RBAC 和审计。 |

## 从这里开始

完整的本地使用说明见 [新手使用教程](docs/getting-started.md)，其中包含环境准备、配置、MySQL 数据导入、服务启动、验证与日常维护。

常用命令：

```bash
make proto-check
make format-check
make vet-check
go test ./...
```

更多文档请参阅 [文档索引](docs/README.md)。生产环境请先阅读 [生产配置](docs/production-configuration.md) 与发布检查要求。
