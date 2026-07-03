.PHONY: help proto proto-check proto-tools

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

proto:
	go generate ./api

proto-check:
	./scripts/proto-check.sh

proto-tools:
	./scripts/proto-install-tools.sh
