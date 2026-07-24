#!/usr/bin/env bash
set -euo pipefail

command -v govulncheck >/dev/null || { echo "govulncheck is required on Jenkins agents" >&2; exit 1; }
command -v gitleaks >/dev/null || { echo "gitleaks is required on Jenkins agents" >&2; exit 1; }

make format-check
make vet-check
make lint-check
make panic-check
make migration-check
make config-secret-check
make startup-validation-check
make proto-check
make ops-check
make architecture-check
make schema-integration-test

govulncheck ./...
gitleaks detect --source . --no-git --redact --exit-code 1
