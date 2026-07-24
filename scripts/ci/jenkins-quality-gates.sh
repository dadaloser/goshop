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

if [[ (-n "${GOSHOP_USER_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_GOODS_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_ORDER_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_INVENTORY_SCHEMA_TEST_MYSQL_DSN:-}" && -n "${GOSHOP_REVIEW_SCHEMA_TEST_MYSQL_DSN:-}") || (-n "${GOSHOP_SCHEMA_TEST_MYSQL_USERNAME:-}" && -n "${GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD:-}") || (-n "${USER_MYSQL_USERNAME:-}" && -n "${USER_MYSQL_PASSWORD:-}" && -n "${GOODS_MYSQL_USERNAME:-}" && -n "${GOODS_MYSQL_PASSWORD:-}" && -n "${ORDER_MYSQL_USERNAME:-}" && -n "${ORDER_MYSQL_PASSWORD:-}" && -n "${INVENTORY_MYSQL_USERNAME:-}" && -n "${INVENTORY_MYSQL_PASSWORD:-}" && -n "${REVIEW_MYSQL_USERNAME:-}" && -n "${REVIEW_MYSQL_PASSWORD:-}") ]]; then
  make schema-integration-test
else
  echo "schema integration skipped on Jenkins: provide all five *_SCHEMA_TEST_MYSQL_DSN values, shared GOSHOP_SCHEMA_TEST_MYSQL_USERNAME/GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD, or USER_/GOODS_/ORDER_/INVENTORY_/REVIEW_MYSQL_* credentials"
fi

govulncheck ./...
gitleaks detect --source . --no-git --redact --exit-code 1
