BUILD_DIR := $(CURDIR)/build
APP_BIN_DIR := $(BUILD_DIR)/app-bins

GATEWAY_APP := ./cmd/gateway/

LINUX_BUILD_FLAGS := -trimpath -ldflags='-w -s -linkmode external -extldflags "-fno-PIC -static"'

.PHONY: gen-docs
gen-docs:
	swag init -g $(GATEWAY_APP)main.go

.PHONY: pre-commit
pre-commit:
	@pre-commit run --all-files

.PHONY: build-gateway-linux-arm64
build-gateway-linux-arm64:
	@echo "building gateway for Linux ARM64 with musl (static)..."
	@mkdir -p $(APP_BIN_DIR)/linux_arm64
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-musl-gcc CXX=aarch64-linux-musl-g++ go build $(LINUX_BUILD_FLAGS) -v -o $(APP_BIN_DIR)/linux_arm64/gateway $(GATEWAY_APP)

.PHONY: build-gateway-linux-amd64
build-gateway-linux-amd64:
	@echo "building gateway for Linux AMD64 with musl (static)..."
	@mkdir -p $(APP_BIN_DIR)/linux_amd64
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ go build $(LINUX_BUILD_FLAGS) -v -o $(APP_BIN_DIR)/linux_amd64/gateway $(GATEWAY_APP)

.PHONY: clean
clean:
	@echo "Removing build directory..."
	@rm -rf $(BUILD_DIR)
	@echo "Cleanup complete."
