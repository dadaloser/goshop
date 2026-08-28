.PHONY: help proto proto-check proto-tools panic-check migration-check config-secret-check startup-validation-check inventory-integration-test data-layer-integration-test schema-integration-test test-race-cover coverage-threshold-check format-check vet-check lint-check rpcserver-flake-check gmicro-resilience-check release-check ops-check architecture-check context-check error-contract-audit error-contract-check

GOLANGCI_LINT_VERSION ?= v2.12.2

# 本地开发：
# 1. 编辑 api/**/*.proto 文件。
# 2. 运行 make proto。
# 3. 将 .proto 文件与重新生成的 *.pb.go 文件一并提交。
#首次环境配置：
# make proto-tools
# CI 一致性检查：
# make proto-check
# 注意事项：
# - 请勿手动编辑 api/**/.pb.go、_grpc.pb.go、*_gin.pb.go 或 *_http.pb.go 文件。
# - protoc-gen-go-gin 必须已安装在 PATH 中，或者在执行 make proto-tools 前
# 通过设置 PROTOC_GEN_GO_GIN_INSTALL=module@version 来自动安装。

help:
	@echo "Available targets:"
	@echo "  make proto        Generate api/**/*.pb.go from api/**/*.proto"
	@echo "  make proto-check  Regenerate proto files and fail if git diff changes api/"
	@echo "  make proto-tools  Install pinned protoc Go plugins"
	@echo "  make panic-check  Fail if business code contains implement-me panics"
	@echo "  make migration-check  Fail on unsafe AutoMigrate usage or missing reviewed startup schema coverage"
	@echo "  make config-secret-check  Fail if configs contain known secrets or unsafe defaults"
	@echo "  make startup-validation-check  Fail if startup validation can be bypassed by log.development"
	@echo "  make schema-integration-test  Run real-MySQL user/goods/order/inventory/review startup schema integration tests"
	@echo "  make test-race-cover  Run race-enabled tests with merged coverage output"
	@echo "  make coverage-threshold-check  Enforce coverage thresholds for core money/authz/order/payment/session code"
	@echo "  make format-check  Fail if gofmt would change tracked Go files"
	@echo "  make vet-check  Run go vet with an approved murmur3 exception policy"
	@echo "  make lint-check  Run pinned golangci-lint version"
	@echo "  make rpcserver-flake-check  Run rpcserver tests with -count=50"
	@echo "  make gmicro-resilience-check  Run local gmicro registration, readiness, shutdown, and resolver resilience checks"
	@echo "  make release-check  Run the trusted release baseline gate"
	@echo "  make inventory-integration-test  Run inventory real-DB integration tests"
	@echo "  make data-layer-integration-test  Run goods/inventory real-DB data tests and coverage gates"
	@echo "  make ops-check  Validate monitoring, runbooks, Jenkins gates, canary and chaos assets"
	@echo "  make architecture-check  Enforce formal RBAC and thin API/Admin dependency boundaries"
	@echo "  make context-check  Reject direct context.Background calls outside documented context boundaries"
	@echo "  make error-contract-audit  Report legacy database error wrapping and string matching"
	@echo "  make error-contract-check  Enforce database error wrapping and typed error matching"

proto:
	go generate ./api

proto-check:
	./scripts/proto-check.sh

proto-tools:
	./scripts/proto-install-tools.sh

panic-check:
	! rg 'panic\("implement me"\)' app api gmicro

migration-check:
	bash ./scripts/migration-check.sh

config-secret-check:
	bash ./scripts/config-secret-check.sh

startup-validation-check:
	bash ./scripts/startup-validation-check.sh

format-check:
	test -z "$$(gofmt -l .)"

vet-check:
	bash ./scripts/go-vet-check.sh

lint-check:
	GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) bash ./scripts/lint.sh

rpcserver-flake-check:
	env GOCACHE=/tmp/goshop-gocache go test -count=50 ./gmicro/server/rpcserver

gmicro-resilience-check:
	bash ./scripts/gmicro-resilience-check.sh

release-check:
	GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) bash ./scripts/release-check.sh

inventory-integration-test:
	bash ./scripts/run-inventory-integration-tests.sh

data-layer-integration-test:
	bash ./scripts/run-data-layer-integration-tests.sh

schema-integration-test:
	bash ./scripts/run-schema-integration-tests.sh

test-race-cover:
	bash ./scripts/go-test-race-cover.sh

coverage-threshold-check:
	bash ./scripts/coverage-threshold-check.sh

ops-check:
	bash ./scripts/ops-readiness-check.sh

architecture-check:
	bash ./scripts/architecture-boundary-check.sh

context-check:
	bash ./scripts/context-boundary-check.sh

error-contract-audit:
	bash ./scripts/error-contract-audit.sh

error-contract-check:
	bash ./scripts/error-contract-audit.sh --strict
