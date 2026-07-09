.PHONY: help proto proto-check proto-tools panic-check migration-check config-secret-check startup-validation-check inventory-integration-test

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
	@echo "  make migration-check  Fail on unsafe application-managed schema migration"
	@echo "  make config-secret-check  Fail if configs contain known secrets or unsafe defaults"
	@echo "  make startup-validation-check  Fail if startup validation can be bypassed by log.development"
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

inventory-integration-test:
	bash ./scripts/run-inventory-integration-tests.sh
