#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/goshop-proto-check.XXXXXX")"
trap 'rm -rf "${TMP_DIR}"' EXIT

# CI 中用于防止 protobuf 生成结果漂移。
# 用法：
#   make proto-check
#
# 本脚本会在临时目录中重新生成 api/**/*.pb.go；若结果与 Git 中的文件不同
# 则失败。应在 Jenkins/CI 的构建和测试前执行，以便及早发现手工修改的生成文件，
# 且不会改动工作区。
PROTO_OUT_ROOT="${TMP_DIR}" "${ROOT_DIR}/scripts/proto-gen.sh"

current_files="$(
  cd "${ROOT_DIR}" &&
    find api -type f \( -name '*.pb.go' -o -name '*_grpc.pb.go' -o -name '*_gin.pb.go' -o -name '*_http.pb.go' \) |
    sort
)"
generated_files="$(
  cd "${TMP_DIR}" &&
    find api -type f \( -name '*.pb.go' -o -name '*_grpc.pb.go' -o -name '*_gin.pb.go' -o -name '*_http.pb.go' \) |
    sort
)"

if [[ "${current_files}" != "${generated_files}" ]]; then
  echo "generated proto file set is out of date" >&2
  diff -u <(printf '%s\n' "${current_files}") <(printf '%s\n' "${generated_files}") || true
  exit 1
fi

while IFS= read -r file; do
  [[ -z "${file}" ]] && continue
  if ! cmp -s "${ROOT_DIR}/${file}" "${TMP_DIR}/${file}"; then
    echo "generated proto file differs: ${file}" >&2
    git -C "${ROOT_DIR}" diff --no-index -- "${ROOT_DIR}/${file}" "${TMP_DIR}/${file}" || true
    exit 1
  fi
done <<< "${current_files}"

echo "generated proto files are up to date"
