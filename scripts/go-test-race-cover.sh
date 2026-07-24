#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COVERPROFILE="${COVERPROFILE:-/tmp/goshop-coverage.out}"
GOCACHE_DIR="${GOCACHE:-/tmp/goshop-gocache}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-3}"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/goshop-cover.XXXXXX")"
trap 'rm -rf "${TMP_DIR}"' EXIT

cd "${ROOT_DIR}"

packages="$(
  go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./... |
    sed '/^$/d'
)"

if [[ -z "${packages}" ]]; then
  echo "no packages with tests found" >&2
  exit 1
fi

printf 'mode: atomic\n' > "${COVERPROFILE}"

run_package_test() {
  local pkg="$1"
  local profile="$2"
  local attempt=1

  while (( attempt <= MAX_ATTEMPTS )); do
    if env GOCACHE="${GOCACHE_DIR}" \
      go test -count=1 -race -shuffle=on -covermode=atomic -coverprofile="${profile}" "${pkg}"; then
      return 0
    fi

    if (( attempt == MAX_ATTEMPTS )); then
      return 1
    fi

    echo "retrying ${pkg} (${attempt}/${MAX_ATTEMPTS}) after test failure" >&2
    rm -f "${profile}"
    attempt=$((attempt + 1))
  done
}

index=0
while IFS= read -r pkg; do
  [[ -z "${pkg}" ]] && continue
  profile="${TMP_DIR}/$(printf '%04d' "${index}").out"
  run_package_test "${pkg}" "${profile}"
  if [[ -f "${profile}" ]]; then
    tail -n +2 "${profile}" >> "${COVERPROFILE}"
  fi
  index=$((index + 1))
done <<< "${packages}"

echo "coverage profile written to ${COVERPROFILE}"
