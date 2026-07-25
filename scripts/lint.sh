#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-v2.12.2}"
GOTOOLCHAIN_VERSION="${GOTOOLCHAIN_VERSION:-go1.26.5}"
GOCACHE_DIR="${GOCACHE:-/tmp/goshop-gocache}"
GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-/tmp/goshop-golangci-cache}"
GOLANGCI_LINT_MODE="${GOLANGCI_LINT_MODE:-module}"

cd "${ROOT_DIR}"

run_lint() {
  env \
    GOCACHE="${GOCACHE_DIR}" \
    GOTOOLCHAIN="${GOTOOLCHAIN_VERSION}" \
    GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE}" \
    go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}" run --timeout=5m
}

case "${GOLANGCI_LINT_MODE}" in
  module)
    run_lint
    ;;
  *)
    echo "unsupported GOLANGCI_LINT_MODE: ${GOLANGCI_LINT_MODE}" >&2
    echo "supported values: module" >&2
    exit 1
    ;;
esac
