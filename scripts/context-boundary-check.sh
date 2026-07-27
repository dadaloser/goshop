#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# contextutil owns the only documented process roots. All other production
# code must propagate a caller context or obtain a bounded context from it.
matches="$(rg -n 'context\.Background\(\)' --glob '*.go' --glob '!**/*_test.go' --glob '!pkg/common/contextutil/context.go' . || true)"
if [[ -n "${matches}" ]]; then
	printf '%s\n' "${matches}" >&2
	echo 'production code must not call context.Background directly; use caller context or pkg/common/contextutil' >&2
	exit 1
fi
