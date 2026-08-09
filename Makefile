GO ?= go
BINARY := mars
BUILD_DIR := build
GOBIN := $(shell $(GO) env GOBIN)
GOPATH := $(shell $(GO) env GOPATH)
INSTALL_BIN := $(if $(GOBIN),$(GOBIN),$(GOPATH)/bin)

FUZZTIME ?= 10s
GOVULNCHECK_VERSION := v1.6.0
GOVULNCHECK ?= $(INSTALL_BIN)/govulncheck
export GOVULNCHECK

.PHONY: build install update-tool test vet lint check coverage-check vuln fuzz-smoke dependency-notices dogfood clean

build:
	CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/mars

install:
	CGO_ENABLED=0 $(GO) install ./cmd/mars
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
	CGO_ENABLED=0 $(GO) build ./cmd/mars
	$(GO) test ./... -race -count=1 -parallel=4 -coverprofile=coverage.out -covermode=atomic -cover | tee coverage-report.txt
	$(GO) tool cover -func=coverage.out | tail -n 5
	scripts/check-coverage.sh --input coverage-report.txt
	$(MAKE) vuln
	$(MAKE) fuzz-smoke
	$(MAKE) lint

coverage-check:
	scripts/check-coverage.sh

vuln:
	@if ! scanner=$$(command -v "$${GOVULNCHECK}" 2>/dev/null); then \
		echo "govulncheck not found at '$${GOVULNCHECK}'; vulnerability scanning is required." >&2; \
		echo "Fix: go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)" >&2; \
		exit 1; \
	fi; \
	"$$scanner" ./...

fuzz-smoke:
	$(GO) test ./internal/agent -run '^$$' -fuzz FuzzToolCallsFromAssistantMessage -fuzztime $(FUZZTIME)
	$(GO) test ./internal/tools -run '^$$' -fuzz FuzzDecodeStringSliceArg -fuzztime $(FUZZTIME)
	$(GO) test ./internal/tools -run '^$$' -fuzz FuzzParsePythonStyleStringList -fuzztime $(FUZZTIME)
	$(GO) test ./internal/tools -run '^$$' -fuzz FuzzNormalizeShellExecArgv -fuzztime $(FUZZTIME)

dependency-notices:
	@set -eu; \
		goroot="$$($(GO) env GOROOT)"; \
		toolgo="$$goroot/bin/go"; \
		test "$$($$toolgo env GOVERSION)" = "go1.26.5"; \
		tmp="$$(mktemp -d "$${TMPDIR:-/tmp}/mars-notices.XXXXXX")"; \
		trap 'rm -rf "$$tmp"' EXIT; \
		cd tools/third-party-notices; \
		GOTOOLCHAIN=local "$$toolgo" mod verify; \
		GOCACHE="$$tmp/build-cache" GOROOT="$$goroot" GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off "$$toolgo" build -trimpath -o "$$tmp/generate" ./cmd/generate; \
		GOROOT="$$goroot" GOCACHE="$$tmp/graph-cache" GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off "$$tmp/generate" --repo "$(CURDIR)"

dogfood: build
	$(BUILD_DIR)/$(BINARY) version
	$(BUILD_DIR)/$(BINARY) run foundation-maintainer --repo . --dry-run --no-init --trace
	$(BUILD_DIR)/$(BINARY) run qa --repo . --dry-run --no-init --trace
	$(BUILD_DIR)/$(BINARY) run engineer --repo . --dry-run --no-init --trace

clean:
	rm -rf $(BUILD_DIR)
