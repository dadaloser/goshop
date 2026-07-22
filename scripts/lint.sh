#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-v2.12.2}"
TOOLS_DIR="${TOOLS_DIR:-/tmp/goshop-tools}"
GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-/tmp/goshop-golangci-cache}"

cd "${ROOT_DIR}"

case "$(uname -s)" in
  Darwin) platform="darwin" ;;
  Linux) platform="linux" ;;
  *)
    echo "unsupported platform: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

version_no_v="${GOLANGCI_LINT_VERSION#v}"
cache_dir="${TOOLS_DIR}/golangci-lint/${version_no_v}/${platform}-${arch}"
binary="${cache_dir}/golangci-lint"

if [[ ! -x "${binary}" ]]; then
  mkdir -p "${cache_dir}"
  archive="golangci-lint-${version_no_v}-${platform}-${arch}.tar.gz"
  url="https://github.com/golangci/golangci-lint/releases/download/${GOLANGCI_LINT_VERSION}/${archive}"
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "${tmp_dir}"' EXIT
  curl -fL "${url}" -o "${tmp_dir}/${archive}"
  tar -xzf "${tmp_dir}/${archive}" -C "${tmp_dir}"
  mv "${tmp_dir}/golangci-lint-${version_no_v}-${platform}-${arch}/golangci-lint" "${binary}"
  chmod +x "${binary}"
fi

"${binary}" run --timeout=5m
