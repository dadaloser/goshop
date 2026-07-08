#!/usr/bin/env bash

# Central place for the protobuf generator versions used by this repository.
# Usage:
#   source scripts/proto-versions.sh
#
# Keep these versions stable. Changing any value below can rewrite generated
# api/**/*.pb.go headers and should be reviewed in the same PR as the generated
# output.

export PROTOC_GEN_GO_VERSION="${PROTOC_GEN_GO_VERSION:-v1.36.11}"
export PROTOC_GEN_GO_GRPC_VERSION="${PROTOC_GEN_GO_GRPC_VERSION:-v1.6.0}"
export PROTOC_GEN_GO_HTTP_VERSION="${PROTOC_GEN_GO_HTTP_VERSION:-v2.0.0-20260404020628-f149714c1d54}"

# protoc-gen-go-gin is required for api/*/v1/*_gin.pb.go.
# This repository does not currently know its public Go module path, so callers
# must either install the exact binary on PATH or set this to module@version:
#
#   PROTOC_GEN_GO_GIN_INSTALL=example.com/tools/protoc-gen-go-gin@v1.2.3 make proto-tools
#
export PROTOC_GEN_GO_GIN_INSTALL="${PROTOC_GEN_GO_GIN_INSTALL:-}"
