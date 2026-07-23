#!/usr/bin/env bash
set -euo pipefail
image="${1:?image reference required}"
artifact="${2:?artifact prefix required}"
command -v syft >/dev/null || { echo "syft is required" >&2; exit 1; }
command -v trivy >/dev/null || { echo "trivy is required" >&2; exit 1; }
syft "$image" -o "spdx-json=${artifact}.spdx.json"
trivy image --exit-code 1 --severity HIGH,CRITICAL --ignore-unfixed "$image"
