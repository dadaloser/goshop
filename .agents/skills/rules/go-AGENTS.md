# Go 开发 AGENTS 规则

## 规则文档索引

| 文档 | 内容 |
| --- | --- |
| `docs/rules/project-architecture.md` | 项目架构：模块职责、目录结构、依赖方向、核心数据流、关键接口、关键文件索引 |
| `docs/rules/go-coding-standards.md` | Go 编码规范：命名、包设计、错误处理、context、并发、日志、配置、测试、Lint、安全规范 |
| `docs/rules/api-contracts.md` | API 契约：HTTP/gRPC/GraphQL 接口、请求响应结构、错误码、兼容性约束 |
| `docs/rules/operations.md` | 运行维护：启动参数、环境变量、数据库迁移、观测性、部署与排障命令 |

代码规范约束：

- 后续实现默认都在当前工作区内完成，不创建或切换 worktree；除非用户明确要求使用 worktree。
- 写代码前必须先查看与变更相关的规则文档，优先级为：用户最新指令 > 本 `AGENTS.md` > `docs/rules/*.md` > 现有代码风格。
- 生成计划前应先阅读 `docs/rules/project-architecture.md`，理解当前模块边界、依赖方向和核心业务流程。
- 业务逻辑、模块职责、数据流或关键接口发生变化后，必须同步更新 `docs/rules/project-architecture.md` 或 `docs/rules/api-contracts.md`。
- 用户指出编码规范调整后，必须同步更新 `docs/rules/go-coding-standards.md`。
- 新增或修改 Go 代码时，必须保持 `gofmt`/`goimports` 格式化，不手工调整与格式化工具冲突的缩进或 import 顺序。
- 新增或修改导出包、导出类型、导出函数、导出方法、导出常量或导出变量时，必须补齐符合 godoc 风格的注释；注释应解释用途、约束和错误条件，不要只复述名称。
- 新增或修改 HTTP Handler、gRPC Service、Repository、Store、Usecase、Worker、Job、Config、DTO、Model、Middleware 时，必须同步检查命名、包路径、错误处理、context 传播、日志、敏感信息、测试和兼容性。
- 生成代码后至少执行与变更范围匹配的验证命令；优先级为 `go test ./...`、相关包测试、`golangci-lint run ./...`、`govulncheck ./...`。如果因既有无关问题无法全量通过，必须说明本次相关验证结果。

---

# Go 开发强制规范

## Required Go skills

处理 Go 相关任务时，必须优先应用以下 Go 开发技能与约束：

- `samber/cc-skills-golang@golang-code-style`
- `samber/cc-skills-golang@golang-data-structures`
- `samber/cc-skills-golang@golang-design-patterns`
- `samber/cc-skills-golang@golang-documentation`
- `samber/cc-skills-golang@golang-error-handling`
- `samber/cc-skills-golang@golang-modernize`
- `samber/cc-skills-golang@golang-naming`
- `samber/cc-skills-golang@golang-safety`
- `samber/cc-skills-golang@golang-security`
- `samber/cc-skills-golang@golang-testing`
- `samber/cc-skills-golang@golang-troubleshooting`

如项目使用数据库、gRPC、GraphQL、Cobra、Viper、Wire、Fx、Dig、samber/do、samber/lo 等技术栈，应按实际代码额外应用对应技能。

## Go 编码约束

- 包名必须小写、短、语义明确，避免 `utils`、`common`、`helper` 等泛化包名；包名不要与导出标识符重复表达同一概念。
- Go 标识符使用 MixedCaps 或 mixedCaps，不使用 snake_case 或 ALL_CAPS；常量也使用 MixedCaps。
- `cmd/{app}/main.go` 只负责解析配置、初始化依赖、启动程序和处理退出；业务逻辑必须放在 `internal/` 或明确可复用的 `pkg/` 中。
- 默认使用 `internal/` 放置项目私有代码；只有确实要对外复用、并能承担兼容性责任时才使用 `pkg/`。
- 函数应短小聚焦；参数超过 4 个时优先使用参数结构体或 options；`context.Context` 必须作为第一个参数并命名为 `ctx`。
- 不要把 `context.Context` 存进结构体；请求链路中必须传递同一个 `ctx`，不能在中间随意创建 `context.Background()`。
- `WithCancel`、`WithTimeout`、`WithDeadline` 返回的 `cancel` 必须在所有控制路径上调用，除非所有权被明确转移。
- 错误必须检查，不允许用 `_` 静默丢弃错误；无法处理时使用 `%w` 包装上下文继续返回。
- 错误字符串使用小写且不以标点结尾；错误要么记录日志，要么返回，禁止同一层既 log 又 return。
- 预期错误不能使用 `panic`；`panic` 仅用于不可恢复的程序员错误或启动期硬失败。
- 日志默认使用结构化日志，优先 `log/slog`；日志中不得输出 token、密码、密钥、身份证、手机号、邮箱等敏感信息。
- Slice 和 map 默认显式初始化，避免 nil map 写入 panic，也避免 API JSON 中空切片意外编码为 `null`。
- 结构体字面量必须使用字段名，避免字段顺序变化引入隐蔽 bug。
- 比较同一变量的多个分支时优先使用 `switch`；错误和边界条件优先早返回，减少嵌套。
- 禁止不必要的全局可变状态；并发访问共享状态必须使用 channel、mutex、atomic 或其他明确同步机制。
- 涉及 goroutine 的代码必须设计退出路径，必须响应 context 取消，测试中应考虑 goroutine 泄漏。
- 涉及用户输入、SQL、Shell、文件路径、模板、网络请求、鉴权、加密、压缩包解包时，必须进行安全审查。

## 测试约束

- 新增业务逻辑必须补充测试；修复 bug 必须先补充能复现问题的测试，或说明无法测试的原因。
- 单元测试文件与被测代码同包同目录，命名为 `*_test.go`；测试夹具放在 `testdata/`。
- 表格驱动测试必须包含 `name` 字段，并通过 `t.Run(tt.name, ...)` 运行具名子测试。
- 测试只验证可观察行为，不绑定无关实现细节。
- 独立且无共享状态的测试可以使用 `t.Parallel()`；并发、时间、网络、文件系统测试必须保证确定性。
- 集成测试必须使用 build tag 隔离，例如 `//go:build integration`，并通过 `go test -tags=integration ./...` 单独运行。
- 并发代码应使用 `go test -race ./...` 验证；存在后台 goroutine 的包应考虑 `go.uber.org/goleak`。
- Benchmark 必须报告有意义的输入规模和分配情况；优化前后应使用基准测试或 profile 证明收益。

## Lint 与格式化约束

- 所有 Go 文件提交前必须通过 `gofmt` 或 `gofumpt` 格式化，并保持 import 有序。
- 项目应维护 `.golangci.yml`；Lint 规则以配置文件为准，不在代码中随意绕过。
- `//nolint` 必须指定具体 linter，并附带理由；禁止无理由的裸 `//nolint`。
- 安全相关 linter，例如 `gosec`、`bodyclose`、`sqlclosecheck`，不得轻易 suppress。

---

# 项目索引模板

## 项目概览

这是一个 Go 项目。请在接入具体仓库后补充以下信息：

- 项目名称：
- Go module：
- Go 版本：
- 项目类型：CLI / HTTP API / gRPC 服务 / Worker / Library / Monorepo
- 主要能力：
- 主要外部依赖：数据库、缓存、消息队列、第三方 API、对象存储等

## 推荐目录结构

```text
.
├── cmd/
│   └── {app}/
│       └── main.go
├── internal/
│   ├── config/
│   ├── app/
│   ├── handler/
│   ├── service/
│   ├── repository/
│   └── worker/
├── pkg/
│   └── {public-package}/
├── api/
├── migrations/
├── docs/
│   └── rules/
├── testdata/
├── go.mod
├── go.sum
├── Makefile
└── .golangci.yml
```

目录约束：

- `cmd/`：程序入口，只做启动编排。
- `internal/`：项目私有业务代码，外部项目不可导入。
- `pkg/`：对外可复用库代码，新增前必须确认兼容性责任。
- `api/`：OpenAPI、protobuf、GraphQL schema 或其他接口契约。
- `migrations/`：数据库迁移脚本。
- `docs/rules/`：AI 和团队共同遵循的项目规则文档。

## 核心模块

接入具体项目后，请按实际模块补充：

- `config`：配置加载、环境变量解析、默认值与校验
- `app`：应用组装、生命周期、依赖注入、健康检查、优雅退出
- `handler`：HTTP/gRPC/CLI 入参解析、鉴权、响应转换
- `service`：业务用例与领域规则
- `repository`：数据库、缓存、外部存储访问
- `worker`：异步任务、定时任务、消息消费
- `client`：第三方服务客户端
- `model` 或 `entity`：领域对象与持久化对象

## 常用命令

```bash
# 查看 Go 版本
go version

# 下载依赖
go mod download

# 整理依赖
go mod tidy

# 格式化
gofmt -w .

# 运行全部测试
go test ./...

# 运行指定包测试
go test ./internal/service/...

# 运行指定测试
go test -run TestName ./...

# 运行竞态检测
go test -race ./...

# 运行集成测试
go test -tags=integration ./...

# 生成覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out

# 运行基准测试
go test -bench=. -benchmem ./...

# 运行 Lint
golangci-lint run ./...

# 自动修复部分 Lint 问题
golangci-lint run --fix ./...

# 运行漏洞扫描
govulncheck ./...

# 本地启动
go run ./cmd/{app}
```

如项目使用 Makefile，应优先使用项目封装命令：

```bash
make tidy
make fmt
make lint
make test
make test-race
make build
make run
```

## 关键配置

推荐通过环境变量或配置文件管理运行配置。生产敏感值必须来自环境变量、密钥管理系统或部署平台，不得写入代码、测试快照、日志或文档示例。

常见配置项：

```yaml
APP_ENV: dev
APP_NAME: your-service
HTTP_ADDR: :8080
LOG_LEVEL: info
DATABASE_DSN: ${DATABASE_DSN}
REDIS_ADDR: ${REDIS_ADDR}
OTEL_EXPORTER_OTLP_ENDPOINT: ${OTEL_EXPORTER_OTLP_ENDPOINT}
```

敏感配置约束：

- 不要把 token、password、secret、private key、AK/SK、数据库 DSN 明文写入提交说明、日志、文档或测试数据。
- 示例配置必须使用占位符，例如 `${DATABASE_DSN}`、`<redacted>`、`example-token`。
- 打印配置时必须脱敏；错误信息返回给用户前必须转换为安全、稳定的业务错误。

## API 与兼容性

- HTTP API 必须明确状态码、错误响应格式和字段兼容性。
- gRPC API 必须维护 protobuf 的字段编号兼容性，删除字段前必须先 reserve。
- 对外导出的 Go API 必须谨慎变更；删除或改签名属于破坏性变更。
- 数据库迁移必须可重复执行、可回滚或具备明确补偿方案。

## 关键文件索引

接入具体项目后，请在这里补充真实路径：

- `go.mod`：模块路径、Go 版本、依赖声明
- `cmd/{app}/main.go`：程序入口
- `internal/config/`：配置加载
- `internal/app/`：应用启动与依赖组装
- `internal/handler/`：入口层
- `internal/service/`：业务逻辑
- `internal/repository/`：数据访问
- `.golangci.yml`：Lint 配置
- `Makefile`：常用开发命令
- `docs/rules/project-architecture.md`：架构与业务说明
- `docs/rules/go-coding-standards.md`：编码规范


  详见 docs/rules/go-business-logic.md 中的“关键文件索引”章节。