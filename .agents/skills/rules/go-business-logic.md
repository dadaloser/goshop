# Go 工程业务开发规范

本文档用于约束 Go 服务、Worker、CLI 和业务库的工程组织方式。目标是让包职责清晰、依赖显式、错误可追踪、测试可复现，并避免把其他语言的重分层习惯机械搬到 Go 项目中。

## 1. Go 工程原则

- 按能力组织包，不按固定层级机械组织代码。不要为了“分层完整”强行创建空壳包。
- 包名表达提供的能力，避免 `utils`、`common`、`helper`、`base` 等泛化名称。
- `cmd/{app}/main.go` 只做启动相关工作：解析参数、加载配置、初始化依赖、启动程序和处理退出。
- 业务规则放在清晰命名的业务包中，例如 `order`、`payment`、`user`、`billing`，不要散落在 HTTP handler 或存储实现里。
- 接口由使用方定义，谁需要抽象谁定义接口；不要在实现包旁边提前设计一批大接口。
- 依赖方向保持简单：入口调用业务包，业务包依赖小接口，存储、消息、第三方客户端实现这些接口。
- 所有请求链路传递 `context.Context`，不要把 `context.Context` 存进结构体。
- 错误要么返回，要么在边界层记录并转换为响应；不要同一错误在多层重复日志。

## 2. 推荐目录结构

目录结构应服务于项目规模。小项目可以更扁平，业务变复杂后再拆包。

```text
.
├── cmd/
│   └── {app}/
│       └── main.go
├── internal/
│   ├── app/
│   ├── config/
│   ├── httpapi/
│   ├── order/
│   ├── payment/
│   ├── store/
│   ├── worker/
│   └── client/
├── pkg/
├── api/
├── configs/
├── migrations/
├── scripts/
├── docs/
├── testdata/
├── go.mod
├── go.sum
├── Makefile
└── .golangci.yml
```

| 目录 | 职责 |
| --- | --- |
| `cmd/{app}` | 可执行程序入口，一个程序一个目录 |
| `internal/app` | 应用组装、生命周期、健康检查、优雅退出 |
| `internal/config` | 配置结构、默认值、环境变量映射和校验 |
| `internal/httpapi` | HTTP 路由、请求解析、响应转换、中间件 |
| `internal/{business}` | 业务包，例如 `order`、`payment`，放业务类型、规则、用例 |
| `internal/store` | 数据库、缓存、消息队列等存储访问实现 |
| `internal/worker` | 定时任务、消息消费、批处理、补偿任务 |
| `internal/client` | 第三方 HTTP/gRPC 客户端封装 |
| `pkg` | 真正需要被外部项目导入的稳定公共库 |
| `api` | OpenAPI、Protobuf、GraphQL schema 或其他接口定义 |
| `configs` | 本地配置样例和部署配置模板，不存放生产密钥 |
| `migrations` | 数据库迁移脚本 |
| `scripts` | 本地开发、构建发布、代码生成和维护脚本 |
| `docs` | 架构、业务流程、接口、运维和排障文档 |

目录约束：

- 不要把所有业务都塞进几个按层命名的大包。
- 如果包内文件互相不共享非导出符号，且服务不同使用者，考虑拆成更小的包。
- 如果一个包只有一个类型且没有明确独立职责，优先合并回调用方包。
- 只有具备稳定 API 承诺的代码才放入 `pkg/`。

## 3. 包设计与命名

- Module path 必须匹配仓库地址，例如 `github.com/company/order-service`。
- 包名使用小写、短词、单数、语义明确的名称。
- Go 标识符使用 `MixedCaps` 或 `mixedCaps`，不要使用 snake_case 或 ALL_CAPS。
- 导出标识符必须有 godoc 风格注释，注释说明用途、约束和错误条件。
- 错误变量使用 `Err` 前缀，例如 `ErrOrderNotFound`。
- 错误字符串使用小写且不以标点结尾。
- 状态类型必须定义安全零值，例如 `StatusUnknown` 或 `StatusInvalid`。
- 包内只暴露调用方需要依赖的类型和函数，其余实现保持非导出。

## 4. main 与应用启动

推荐使用 `run() error` 模式，让启动逻辑可测试：

```go
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Load config, initialize dependencies, start the app.
	return nil
}
```

启动流程：

```text
cmd/{app}/main.go
        ↓
config.Load 读取默认值、配置文件、环境变量和启动参数
        ↓
config.Validate 校验必填项、范围和组合约束
        ↓
app.New 装配 logger、db、cache、clients、stores、业务包、handler、worker
        ↓
app.Run 启动 HTTP/gRPC/Worker
        ↓
监听 context cancellation / OS signal
        ↓
优雅关闭 server、worker、db、trace exporter
```

启动规则：

- `os.Exit`、`log.Fatal` 只在 `main` 或最外层启动代码中使用。
- 其他包返回 `error`，不要直接退出进程。
- 启动期依赖初始化失败可以返回错误终止进程。
- 后台 goroutine 必须有退出路径，并响应 `ctx.Done()`。
- 优雅退出必须设置超时，避免进程无限等待。
- `app.Run(ctx)` 返回前释放 server、worker、数据库连接、trace exporter 等资源。

## 5. 业务包组织

业务包以能力命名，例如 `order`。包内可以按需要组织文件，而不是套用固定层级：

```text
internal/order/
├── order.go          # 订单类型、状态和核心规则
├── create.go         # 创建订单用例
├── pay.go            # 支付订单用例
├── store.go          # 本包需要的最小存储接口
├── errors.go         # 本包错误
└── *_test.go
```

业务包规则：

- 包内类型表达业务概念，例如 `Order`、`Payment`、`Invoice`。
- 状态流转集中在业务包中，禁止在 handler 或 store 中直接散落状态赋值。
- 业务包按需要定义小接口，例如 `type Store interface { ... }`，由 `internal/store` 实现。
- 包内函数和方法使用业务语言命名，例如 `Create`、`Pay`、`Cancel`、`Expire`。
- 复杂操作使用参数结构体，例如 `CreateParams`，避免函数参数过长。
- 对外返回稳定错误，内部错误使用 `%w` 包装保留根因。

状态类型建议：

| 阶段 | 示例 |
| --- | --- |
| 初始化 | `StatusUnknown`、`StatusDraft`、`StatusPending` |
| 执行中 | `StatusProcessing`、`StatusWaitingReview`、`StatusRetrying` |
| 成功结束 | `StatusSucceeded`、`StatusCompleted`、`StatusPublished` |
| 失败结束 | `StatusFailed`、`StatusCancelled`、`StatusExpired` |
| 人工介入 | `StatusBlocked`、`StatusManualReview` |

示例调用链：

```text
httpapi.CreateOrder
        ↓
order.Service.Create(ctx, params)
        ↓
加载用户、商品和库存
        ↓
order.New 校验输入并构造订单
        ↓
transaction 保存订单、扣减库存、写入 outbox
        ↓
after commit 发布业务事件或由 outbox worker 投递
        ↓
返回响应结构体
```

## 6. HTTP、gRPC 与接口定义

入口层规则：

- HTTP handler、gRPC server、CLI command 只处理协议细节、参数解析、基础校验、鉴权信息提取、响应转换。
- 入口层不拼 SQL，不直接操作事务，不直接组合多个外部资源完成业务。
- 所有入口设置合理超时，并将 `ctx` 传递到业务包。
- 请求和响应结构体按协议命名，例如 `CreateOrderRequest`、`CreateOrderResponse`。
- 不把数据库结构体直接作为 HTTP 或 gRPC 响应。
- 日志记录请求摘要和业务关键 ID，不记录密码、token、私钥、完整 DSN 或隐私正文。

接口定义规则：

- REST API 使用 OpenAPI 维护对外字段、状态码和错误响应。
- gRPC API 使用 Protobuf，删除字段前必须 `reserved` 字段编号和字段名。
- 对外字段只增不改语义；删除或改语义属于破坏性变更。
- 对外错误响应保持稳定，内部错误只在日志和 trace 中保留详细上下文。
- `/healthz` 只验证进程存活。
- `/readyz` 验证数据库、缓存、消息队列等关键依赖是否就绪。
- `/metrics`、`/debug/pprof`、`/admin/*` 必须限制网络、鉴权、限流和审计。

## 7. 存储、缓存与消息

存储规则：

- 存储包负责 SQL、事务、缓存、消息队列等外部资源细节。
- 存储实现满足业务包定义的小接口，不反向要求业务包依赖具体实现。
- 查询必须明确字段列表，避免在核心路径使用 `SELECT *`。
- 分页查询必须限制最大 `pageSize`，大批量扫描使用游标或稳定排序键。
- 写接口要明确唯一约束、乐观锁、幂等键或业务状态校验。
- 数据库迁移必须与代码兼容，优先采用 expand-migrate-contract。

缓存规则：

- 缓存 key 必须包含业务前缀、版本和关键维度，例如 `order:v1:detail:{orderID}`。
- 缓存值必须设置 TTL；热点 key 需要考虑击穿、穿透和雪崩。
- 删除或更新缓存必须说明一致性策略，例如 cache aside、write through、事件驱动刷新。
- 本地缓存只用于配置、字典、低频变更元数据，并提供刷新或过期机制。

消息规则：

- 消息体只放必要业务字段，大对象通过数据库或对象存储引用。
- 生产端记录业务 key、消息 ID、目标 topic 和投递结果。
- 消费端必须幂等，使用业务唯一键或消息 ID 去重。
- 瞬时错误有限重试，永久错误进入死信、失败表或人工处理。
- 消费端记录耗时、成功数、失败数、重试数和最后错误。

## 8. 配置管理

配置加载优先级：

```text
默认配置
        ↓
配置文件
        ↓
环境变量
        ↓
启动参数
        ↓
Validate 校验必填项、范围和组合约束
```

配置规则：

- 配置结构体集中定义，并提供默认值和校验逻辑。
- 生产敏感配置来自环境变量、KMS、Vault 或部署平台，不写入仓库。
- 示例配置使用占位符，例如 `${DATABASE_DSN}`、`<redacted>`、`example-token`。
- 日志和错误中禁止输出密码、token、私钥、完整 DSN、AK/SK 和用户隐私数据。
- 超时、重试、连接池、并发度、批大小和熔断参数必须可配置。

常见配置项：

| 类型 | 示例 |
| --- | --- |
| 应用 | app name、env、http addr、shutdown timeout |
| 数据库 | DSN、连接池、最大空闲连接、慢查询阈值 |
| 缓存 | Redis 地址、DB、TTL、连接池 |
| 外部服务 | endpoint、timeout、retry、circuit breaker |
| 安全 | JWT key、OAuth client、TLS、密钥路径 |
| 观测 | log level、trace exporter、metrics namespace |
| 任务 | cron、batch size、parallelism、retry interval |

## 9. Worker、定时任务与补偿

任务类型：

| 类型 | 职责 |
| --- | --- |
| Cron Job | 周期性同步、统计、清理、过期处理 |
| Queue Worker | 消息消费、异步处理、削峰填谷 |
| Batch Job | 大批量导入导出、历史数据回填、离线计算 |
| Repair Job | 数据修复、幂等补偿、人工触发重算 |
| Scheduler | 任务分片、租约控制、并发度控制 |

执行流程：

```text
job trigger
        ↓
获取分布式锁、任务租约或分片
        ↓
按 batch size 分页扫描待处理数据
        ↓
worker pool 并发处理
        ↓
记录成功、失败、跳过和耗时
        ↓
失败数据进入 retry / dead letter / repair table
```

任务规则：

- 长任务必须支持分页、断点续跑、限流、取消和重入。
- 并发任务必须限制 worker 数量，不能无界创建 goroutine。
- 任务必须记录执行 ID、触发来源、扫描范围、影响行数、失败原因和耗时。
- 人工修复入口必须记录操作者、入参、影响范围和执行结果。
- 补偿任务只能修复明确状态的数据，不得绕过统一的状态流转校验。

## 10. 错误处理与重试

错误分类：

| 类型 | 处理方式 |
| --- | --- |
| 参数错误 | 返回 4xx 或业务参数错误码，不重试 |
| 权限错误 | 返回 401/403，记录安全审计日志 |
| 业务冲突 | 返回明确业务错误，例如状态不允许、库存不足 |
| 外部依赖瞬时错误 | 按退避策略有限重试 |
| 外部依赖永久错误 | 直接失败，保留错误上下文 |
| 数据一致性错误 | 触发告警，必要时进入补偿流程 |
| Panic | 边界层 recover，记录堆栈，返回通用错误 |

处理规则：

- 使用 `fmt.Errorf("create order: %w", err)` 包装错误，保留根因。
- 使用 `errors.Is`、`errors.As` 判断哨兵错误和错误类型。
- 低层不要直接向用户返回错误文案，边界层统一转换为稳定错误码。
- 重试必须设置最大次数、退避间隔、整体超时和可观测日志。
- 只有确认操作幂等或可补偿时才允许自动重试。
- 普通业务失败返回 `error`，`panic` 只用于不可恢复的程序员错误。

## 11. 可观测性

日志、指标和 trace 必须围绕业务操作组织：

- 结构化日志默认包含 `trace_id`、`request_id`、`user_id`、`operation`、`duration_ms`、`error`。
- 指标至少覆盖请求量、错误率、耗时、队列堆积、任务成功率、外部依赖耗时和数据库慢查询。
- HTTP/gRPC、数据库、缓存、消息生产消费链路需要透传 trace context。
- 健康检查区分存活检查和就绪检查。
- pprof 只在内部网络或调试环境启用，并受权限控制。

请求观测流程：

```text
middleware 注入 request id / trace id
        ↓
handler 记录入口日志和参数摘要
        ↓
业务包 / store / client 创建 span
        ↓
metrics 记录耗时、状态码和错误类型
        ↓
middleware 输出请求完成日志
```

禁止在日志中输出密码、token、完整身份证号、银行卡号、私钥、用户隐私正文和大体积 payload。

## 12. 测试策略

测试类型：

| 类型 | 目标 |
| --- | --- |
| 单元测试 | 验证纯函数、业务规则、错误路径和状态流转 |
| 表驱动测试 | 用统一结构覆盖多组输入输出 |
| 集成测试 | 验证数据库、缓存、消息队列和外部依赖适配 |
| 契约测试 | 验证 OpenAPI/Protobuf 与真实实现一致 |
| 回归测试 | 固化历史 bug，防止再次出现 |
| 基准测试 | 验证性能敏感逻辑的吞吐、分配和延迟 |
| E2E 测试 | 验证关键业务链路从入口到存储的完整行为 |

测试规则：

- `_test.go` 与被测代码放在同一 package 目录。
- 测试数据放入 `testdata/`。
- 表驱动测试必须包含 `name` 字段，并使用 `t.Run(tt.name, ...)`。
- 测试 helper 使用 `t.Helper()`。
- 时间、随机数、外部依赖通过接口或注入方式可控。
- 集成测试使用 build tag 隔离，例如 `//go:build integration`。
- 并发代码应使用 `go test -race ./...` 验证；后台 goroutine 需要考虑泄漏检测。

核心业务模块至少覆盖：

- 成功路径。
- 参数错误。
- 状态冲突。
- 外部依赖失败。
- 事务回滚。
- 幂等重试。
- 权限失败。

## 13. CI/CD 与发布

推荐质量门禁：

```text
gofmt / gofmt check
        ↓
go vet ./...
        ↓
golangci-lint run ./...
        ↓
go test ./...
        ↓
go test -race ./...（并发敏感模块）
        ↓
go test -bench=. -benchmem ./...（性能敏感模块）
        ↓
govulncheck ./... / 镜像扫描 / SBOM
        ↓
build binary / docker image
```

发布规则：

- Binary 或镜像必须包含版本、commit、构建时间。
- Docker 镜像使用非 root 用户和最小运行时基础镜像。
- 数据库迁移与应用发布必须评估兼容顺序和回滚方案。
- 配置变更和代码发布分离，关键开关支持灰度。
- 发布失败时保留构建日志、部署日志、健康检查结果和告警上下文。

回滚规则：

- 应用发布支持上一版本镜像快速回滚。
- 数据库变更优先保持向前兼容，必要时用补偿脚本修复数据。
- 对外接口破坏性变更必须有迁移窗口和版本策略。

## 14. 安全要求

- 身份认证可使用 JWT、OAuth2、mTLS 或内部签名，具体方案以项目安全设计为准。
- 权限控制必须基于角色、资源、租户或数据范围校验。
- 所有外部输入在入口层完成基础校验，业务包完成业务校验。
- SQL、Shell、文件路径、模板、网络请求、压缩包解包等入口必须做注入和越权风险检查。
- 密钥不入库、不入仓库、不打日志，使用 KMS、Vault、环境变量或部署平台注入。
- 管理接口、数据导出接口、补偿接口和调试接口默认视为高风险入口，需要额外鉴权、限流和审计。
- 定期扫描 Go module、镜像、CI workflow 和基础镜像漏洞。

敏感操作必须记录审计日志：

- 登录、授权、权限变更。
- 管理后台操作。
- 数据导出。
- 人工补偿和数据修复。
- 密钥和配置变更。

## 15. 文档维护

必备文档：

| 文档 | 职责 |
| --- | --- |
| `README.md` | 项目说明、快速开始、运行方式、主要功能 |
| `CONTRIBUTING.md` | 本地开发、测试、提交、PR 流程 |
| `CHANGELOG.md` | 面向使用者的版本变更记录 |
| `docs/architecture.md` | 包边界、核心流程、依赖关系和设计取舍 |
| `docs/business-logic.md` | 业务模块、状态流转、任务和接口说明 |
| `docs/runbook.md` | 部署、监控、告警、排障和回滚手册 |
| `docs/api.md` | API 字段、错误码、鉴权方式和示例 |
| `llms.txt` | 给 AI Agent 使用的项目结构、关键入口和约束说明 |

维护规则：

- 新增包必须同步补充包职责、核心流程和关键配置。
- 新增对外接口必须同步 OpenAPI/Proto、错误码和示例。
- 新增定时任务必须说明 cron、幂等规则、失败重试和人工补偿方式。
- 修改状态流转必须同步业务逻辑文档和回归测试。
- 修改配置项必须同步配置样例、部署说明和脱敏规则。
- 发布版本必须更新 CHANGELOG，说明对使用者可见的变化。

## 16. 开发主流程

```text
需求分析
        ↓
业务建模与接口设计
        ↓
按能力拆分业务包、入口包、存储实现和 worker
        ↓
编写测试用例和测试数据
        ↓
实现业务逻辑
        ↓
本地 go test / lint / race
        ↓
代码评审
        ↓
CI 构建、测试、扫描
        ↓
部署测试环境联调
        ↓
灰度发布
        ↓
监控指标和告警观察
        ↓
全量发布或回滚
```

## 17. 开发完成定义

一项 Go 业务开发完成时，应满足以下条件：

- 业务用例实现完成，关键状态流转清晰。
- 包边界符合本文档要求，没有入口层或存储层散落核心业务规则。
- 参数校验、权限校验、错误处理、日志、指标和 trace 齐全。
- 数据库变更具备迁移脚本、兼容顺序和回滚或补偿预案。
- 单元测试、集成测试和关键回归测试覆盖主要成功与失败路径。
- OpenAPI/Proto、README、业务逻辑文档和运维手册已按影响范围更新。
- CI 通过，构建制品可部署，版本信息可追踪。
