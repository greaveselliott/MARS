GO ?= go
BINARY := mars-harness
BUILD_DIR := build
GOBIN := $(shell $(GO) env GOBIN)
GOPATH := $(shell $(GO) env GOPATH)
INSTALL_BIN := $(if $(GOBIN),$(GOBIN),$(GOPATH)/bin)

FUZZTIME ?= 10s
GOPATH_BIN := $(shell $(GO) env GOPATH)/bin

.PHONY: build install update-tool test vet lint check coverage-check vuln fuzz-smoke dogfood clean

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
	$(GO) test ./... -race -count=1 -parallel=4 -coverprofile=coverage.out -covermode=atomic -cover | tee coverage-report.txt
	$(GO) tool cover -func=coverage.out | tail -n 5
	scripts/check-coverage.sh --input coverage-report.txt
	$(GO) run ./cmd/mars-harness validation check-closure --report docs/validation/reports/2026-06-13-foundation-wsd-closure-replay.md
	$(MAKE) vuln
	$(MAKE) fuzz-smoke
	$(MAKE) lint

coverage-check:
	scripts/check-coverage.sh

vuln:
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	elif [ -x "$(GOPATH_BIN)/govulncheck" ]; then \
		"$(GOPATH_BIN)/govulncheck" ./...; \
	else \
		echo "govulncheck not found; skipping vulnerability scan."; \
		echo "Fix: go install golang.org/x/vuln/cmd/govulncheck@latest"; \
	fi

fuzz-smoke:
	$(GO) test ./internal/agent -run '^$$' -fuzz FuzzToolCallsFromAssistantMessage -fuzztime $(FUZZTIME)
	$(GO) test ./internal/tools -run '^$$' -fuzz FuzzDecodeStringSliceArg -fuzztime $(FUZZTIME)
	$(GO) test ./internal/tools -run '^$$' -fuzz FuzzParsePythonStyleStringList -fuzztime $(FUZZTIME)
	$(GO) test ./internal/tools -run '^$$' -fuzz FuzzNormalizeShellExecArgv -fuzztime $(FUZZTIME)

dogfood: build
	$(BUILD_DIR)/$(BINARY) version
	$(BUILD_DIR)/$(BINARY) run foundation-maintainer --repo . --dry-run --no-init --trace
	$(BUILD_DIR)/$(BINARY) run qa --repo . --dry-run --no-init --trace
	$(BUILD_DIR)/$(BINARY) run engineer --repo . --dry-run --no-init --trace

clean:
	rm -rf $(BUILD_DIR)
