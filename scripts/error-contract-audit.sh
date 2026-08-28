#!/usr/bin/env bash
set -euo pipefail

exec go run ./scripts/error-contract-check.go "$@"
