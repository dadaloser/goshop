#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# contextutil 负责唯一允许的进程根上下文。其他生产代码必须传播调用方的
# context，或从该包获取有明确生命周期的 context。
matches="$(rg -n 'context\.Background\(\)' --glob '*.go' --glob '!**/*_test.go' --glob '!gmicro/contextutil/context.go' . || true)"
if [[ -n "${matches}" ]]; then
	printf '%s\n' "${matches}" >&2
	echo 'production code must not call context.Background directly; use caller context or gmicro/contextutil' >&2
	exit 1
fi
