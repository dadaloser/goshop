#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

matches="$(rg -n 'Log\.Development' "${ROOT_DIR}/app" -g 'config.go' || true)"
if [[ -n "${matches}" ]]; then
  echo "startup validation must not be bypassed by log.development:" >&2
  printf '%s\n' "${matches}" >&2
  exit 1
fi

echo "startup validation check passed"
