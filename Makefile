.PHONY: help proto proto-check proto-tools panic-check migration-check config-secret-check startup-validation-check inventory-integration-test schema-integration-test format-check vet-check lint-check rpcserver-flake-check release-check

GOLANGCI_LINT_VERSION ?= v2.12.2

# Fixed protobuf workflow.
#
# Local development:
#   1. Edit api/**/*.proto.
#   2. Run `make proto`.
#   3. Commit the .proto file and regenerated *.pb.go files together.
#
# First-time setup:
#   make proto-tools
#
# CI drift check:
#   make proto-check
#
# Notes:
#   - Do not edit api/**/*.pb.go, *_grpc.pb.go, *_gin.pb.go or *_http.pb.go by hand.
#   - protoc-gen-go-gin must be installed on PATH, or installed by setting
#     PROTOC_GEN_GO_GIN_INSTALL=module@version before `make proto-tools`.
help:
	@echo "Available targets:"
	@echo "  make proto        Generate api/**/*.pb.go from api/**/*.proto"
	@echo "  make proto-check  Regenerate proto files and fail if git diff changes api/"
	@echo "  make proto-tools  Install pinned protoc Go plugins"
	@echo "  make panic-check  Fail if business code contains implement-me panics"
	@echo "  make migration-check  Fail on unsafe AutoMigrate usage or missing reviewed user/RBAC schema coverage"
	@echo "  make config-secret-check  Fail if configs contain known secrets or unsafe defaults"
	@echo "  make startup-validation-check  Fail if startup validation can be bypassed by log.development"
	@echo "  make schema-integration-test  Run real-MySQL goods/order startup schema integration tests"
	@echo "  make format-check  Fail if gofmt would change tracked Go files"
	@echo "  make vet-check  Run go vet on app/gmicro/pkg business code"
	@echo "  make lint-check  Run pinned golangci-lint version"
	@echo "  make rpcserver-flake-check  Run rpcserver tests with -count=50"
	@echo "  make release-check  Run the trusted release baseline gate"
	@echo "  make inventory-integration-test  Run inventory real-DB integration tests"

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
	env GOCACHE=/tmp/goshop-gocache go vet ./app/... ./gmicro/... ./pkg/...

lint-check:
	GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) bash ./scripts/lint.sh

rpcserver-flake-check:
	env GOCACHE=/tmp/goshop-gocache go test -count=50 ./gmicro/server/rpcserver

release-check:
	GOLANGCI_LINT_VERSION=$(GOLANGCI_LINT_VERSION) bash ./scripts/release-check.sh

inventory-integration-test:
	bash ./scripts/run-inventory-integration-tests.sh

schema-integration-test:
	bash ./scripts/run-schema-integration-tests.sh
