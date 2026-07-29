#!/usr/bin/env bash

# 本仓库 protobuf 生成器版本的统一定义位置。
# 用法：
#   source scripts/proto-versions.sh
#
# 请保持这些版本稳定。修改以下任一值都可能改写生成的 api/**/*.pb.go 文件头，
# 必须在同一个 PR 中审查相应生成结果。

export PROTOC_GEN_GO_VERSION="${PROTOC_GEN_GO_VERSION:-v1.36.11}"
export PROTOC_GEN_GO_GRPC_VERSION="${PROTOC_GEN_GO_GRPC_VERSION:-v1.6.0}"
export PROTOC_GEN_GO_HTTP_VERSION="${PROTOC_GEN_GO_HTTP_VERSION:-v2.0.0-20260404020628-f149714c1d54}"

# api/*/v1/*_gin.pb.go 需要 protoc-gen-go-gin。本仓库目前未记录该工具公开的
# Go 模块路径，因此调用方必须将准确的二进制放入 PATH，或设置 module@version：
#
#   PROTOC_GEN_GO_GIN_INSTALL=example.com/tools/protoc-gen-go-gin@v1.2.3 make proto-tools
#
export PROTOC_GEN_GO_GIN_INSTALL="${PROTOC_GEN_GO_GIN_INSTALL:-}"
