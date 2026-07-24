#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
THIRD_PARTY_DIR="${ROOT_DIR}/third_party"
PROTO_OUT_ROOT="${PROTO_OUT_ROOT:-${ROOT_DIR}}"

# Regenerate all checked-in protobuf artifacts under api/.
# Usage:
#   make proto
#   ./scripts/proto-gen.sh
#
# Do not edit generated *.pb.go files by hand. Change the matching .proto file,
# run this command, and commit both the .proto and generated files together.

require_bins() {
  local missing=()
  for bin in "$@"; do
    if ! command -v "${bin}" >/dev/null 2>&1; then
      missing+=("${bin}")
    fi
  done

  if (( ${#missing[@]} > 0 )); then
    printf 'missing required proto tool(s): %s\n' "${missing[*]}" >&2
    cat >&2 <<'EOF'
Install the pinned tools with:
  make proto-tools

If protoc-gen-go-gin is still missing, install the exact project generator:
  PROTOC_GEN_GO_GIN_INSTALL=module@version make proto-tools
EOF
    exit 127
  fi
}

generate_business_proto() {
  local dir="$1"
  local proto="$2"
  local out_dir="${PROTO_OUT_ROOT}/${dir}"

  echo "==> generate ${dir}/${proto}"
  (
    cd "${ROOT_DIR}/${dir}"
    mkdir -p "${out_dir}"
    protoc \
      --proto_path=. \
      --proto_path="${THIRD_PARTY_DIR}" \
      --go_out=paths=source_relative:"${out_dir}" \
      --go-grpc_out=paths=source_relative:"${out_dir}" \
      --go-gin_out=paths=source_relative:"${out_dir}" \
      "${proto}"
  )
}

generate_metadata_proto() {
  echo "==> generate api/metadata/metadata.proto"
  (
    cd "${ROOT_DIR}/api/metadata"
    mkdir -p "${PROTO_OUT_ROOT}/api/metadata"
    protoc \
      --proto_path=. \
      --proto_path="${THIRD_PARTY_DIR}" \
      --go_out=paths=source_relative:"${PROTO_OUT_ROOT}/api/metadata" \
      --go-grpc_out=paths=source_relative:"${PROTO_OUT_ROOT}/api/metadata" \
      --go-http_out=paths=source_relative:"${PROTO_OUT_ROOT}/api/metadata" \
      metadata.proto
  )
}

echo "==> protoc version: $(protoc --version)"
require_bins protoc protoc-gen-go protoc-gen-go-grpc protoc-gen-go-gin protoc-gen-go-http

generate_business_proto api/user/v1 user.proto
generate_business_proto api/goods/v1 goods.proto
generate_business_proto api/inventory/v1 inventory.proto
generate_business_proto api/order/v1 order.proto
generate_business_proto api/review/v1 review.proto
generate_metadata_proto

echo "proto generation complete"
