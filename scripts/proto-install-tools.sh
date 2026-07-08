#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Install the pinned protoc plugins used by `make proto`.
# Usage:
#   make proto-tools
#   PROTOC_GEN_GO_GIN_INSTALL=example.com/tools/protoc-gen-go-gin@v1.2.3 make proto-tools
#
# protoc itself is intentionally not installed here; install it with brew/apt or
# your build image, then verify with `protoc --version`.
source "${ROOT_DIR}/scripts/proto-versions.sh"

go install "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"
go install "google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}"
go install "github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@${PROTOC_GEN_GO_HTTP_VERSION}"

if [[ -n "${PROTOC_GEN_GO_GIN_INSTALL}" ]]; then
  go install "${PROTOC_GEN_GO_GIN_INSTALL}"
else
  (cd "${ROOT_DIR}" && go install ./tools/protoc-gen-go-gin)
fi

if ! command -v protoc-gen-go-gin >/dev/null 2>&1; then
  cat >&2 <<'EOF'
protoc-gen-go-gin is still required for api/*/v1/*_gin.pb.go.

The current repository contains generated *_gin.pb.go files, but does not record
the Go module path of the generator that produced them.

Set PROTOC_GEN_GO_GIN_INSTALL to the exact module@version used by the project,
or install the same protoc-gen-go-gin binary and make sure it is on PATH.

After that, run:
  make proto
EOF
  exit 127
fi
