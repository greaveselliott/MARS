GO ?= go
BINARY := mars-harness
BUILD_DIR := build
GOBIN := $(shell $(GO) env GOBIN)
GOPATH := $(shell $(GO) env GOPATH)
INSTALL_BIN := $(if $(GOBIN),$(GOBIN),$(GOPATH)/bin)

.PHONY: build install update-tool test vet lint check dogfood clean

build:
	CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/mars-harness

install:
	CGO_ENABLED=0 $(GO) install ./cmd/mars-harness
	$(INSTALL_BIN)/$(BINARY) path setup --install-dir $(INSTALL_BIN)
	@echo "Installed $(BINARY) to $(INSTALL_BIN)/$(BINARY)"
	@echo "Run now: $(INSTALL_BIN)/$(BINARY) version"
	@echo "After opening a new terminal or reloading your shell: $(BINARY) version"

update-tool:
	GO=$(GO) BINARY_NAME=$(BINARY) scripts/update-tool.sh

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found; running go vet ./..."; \
		$(GO) vet ./...; \
	fi

check:
	CGO_ENABLED=0 $(GO) build ./cmd/mars-harness
	$(GO) test ./... -race -count=1 -parallel=4 -coverprofile=coverage.out -covermode=atomic
	$(GO) tool cover -func=coverage.out | tail -n 5
	$(MAKE) lint

dogfood: build
	$(BUILD_DIR)/$(BINARY) version
	$(BUILD_DIR)/$(BINARY) run foundation-maintainer --repo . --dry-run --no-init --trace
	$(BUILD_DIR)/$(BINARY) run qa --repo . --dry-run --no-init --trace
	$(BUILD_DIR)/$(BINARY) run engineer --repo . --dry-run --no-init --trace

clean:
	rm -rf $(BUILD_DIR)
