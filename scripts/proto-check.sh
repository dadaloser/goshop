#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/goshop-proto-check.XXXXXX")"
trap 'rm -rf "${TMP_DIR}"' EXIT

# CI guard for generated protobuf drift.
# Usage:
#   make proto-check
#
# This regenerates api/**/*.pb.go into a temp directory and fails if the result
# differs from git. Add it to Jenkins/CI before build/test so manual edits to
# generated files are caught early without mutating the worktree.
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
