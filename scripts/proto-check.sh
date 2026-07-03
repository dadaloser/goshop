#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# CI guard for generated protobuf drift.
# Usage:
#   make proto-check
#
# This regenerates api/**/*.pb.go and fails if the result differs from git.
# Add it to Jenkins/CI before build/test so manual edits to generated files are
# caught early.
"${ROOT_DIR}/scripts/proto-gen.sh"

if ! git -C "${ROOT_DIR}" diff --exit-code -- api ':!api/generate.go'; then
  echo "generated proto files are out of date; run make proto and commit the result" >&2
  exit 1
fi

echo "generated proto files are up to date"
