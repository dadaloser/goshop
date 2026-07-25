#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOCACHE_DIR="${GOCACHE:-/tmp/goshop-gocache}"
APPROVED_FILE="${ROOT_DIR}/third_party/forked/murmur3/go-vet-approved.txt"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/goshop-vet.XXXXXX")"
trap 'rm -rf "${TMP_DIR}"' EXIT

cd "${ROOT_DIR}"

packages=()
while IFS= read -r pkg; do
  [[ -z "${pkg}" ]] && continue
  packages+=("${pkg}")
done < <(env GOCACHE="${GOCACHE_DIR}" go list ./... | rg -v '^goshop/third_party/forked/murmur3$')

if ((${#packages[@]} > 0)); then
  env GOCACHE="${GOCACHE_DIR}" go vet "${packages[@]}"
fi

approved_output="$(sed '/^#/d;/^$/d' "${APPROVED_FILE}")"
actual_file="${TMP_DIR}/murmur3.out"
if ! env GOCACHE="${GOCACHE_DIR}" go vet ./third_party/forked/murmur3 >"${actual_file}" 2>&1; then
  :
fi
actual_output="$(sed '/^$/d' "${actual_file}")"

if [[ "${actual_output}" != "${approved_output}" ]]; then
  echo "third_party/forked/murmur3 go vet output differs from approved exception list" >&2
  diff -u "${APPROVED_FILE}" "${actual_file}" || true
  exit 1
fi

if [[ -n "${actual_output}" ]]; then
  printf '%s\n' "${actual_output}"
fi
