#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONFIG_DIR="${ROOT_DIR}/configs"

if [[ ! -d "${CONFIG_DIR}" ]]; then
  echo "configs directory not found" >&2
  exit 1
fi

patterns=(
  'admin@admin'
  'password:[[:space:]]*["'\'']?nacos["'\'']?'
  'username:[[:space:]]*["'\'']?admin["'\'']?'
  'key:[[:space:]]*["'\'']?nf6C74WZ0OReB0K1QpKhcee9lmBohGSq["'\'']?'
)

failed=0
for pattern in "${patterns[@]}"; do
  matches="$(rg -n "${pattern}" "${CONFIG_DIR}" || true)"
  if [[ -n "${matches}" ]]; then
    echo "forbidden config secret/default found for pattern: ${pattern}" >&2
    printf '%s\n' "${matches}" >&2
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  echo "remove secrets from configs/ and inject them through environment variables or a secret manager" >&2
  exit 1
fi

echo "config secret check passed"
