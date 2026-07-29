# Proto 生成说明

本仓库将生成的 Protobuf 文件提交到版本库。禁止手工编辑下列文件：

- `api/**/*.pb.go`
- `api/**/*_grpc.pb.go`
- `api/**/*_gin.pb.go`
- `api/**/*_http.pb.go`

## 日常流程

```bash
# 修改 api/**/*.proto 后重新生成
make proto

# 在提交前确认生成结果没有漂移
make proto-check
```

生成器版本由项目统一管理，以保证开发机和 CI 使用相同流程。首次配置工具链时执行：

```bash
make proto-tools
```

固定插件版本见 `scripts/proto-versions.sh`。生成 `api/*/v1/*_gin.pb.go` 还需要 `protoc-gen-go-gin`；仓库当前未记录该工具的模块路径。请将对应二进制放入 `PATH`，或指定本项目实际使用的 `module@version`：

```bash
PROTOC_GEN_GO_GIN_INSTALL=example.com/tools/protoc-gen-go-gin@v1.2.3 make proto-tools
make proto
```

## 提交要求

1. 只编辑 `.proto` 源文件。
2. 执行 `make proto`。
3. 审查 `api/` 下的生成差异。
4. 执行 `make proto-check`。
5. 将 `.proto` 与对应生成文件放入同一个提交。

`make proto-check` 失败表示生成文件与源定义不一致；不要通过手工修改生成文件来修复。
