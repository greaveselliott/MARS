GO ?= go
BINARY := mars-harness
BUILD_DIR := build
GOBIN := $(shell $(GO) env GOBIN)
GOPATH := $(shell $(GO) env GOPATH)
INSTALL_BIN := $(if $(GOBIN),$(GOBIN),$(GOPATH)/bin)

.PHONY: build install test vet clean

build:
	CGO_ENABLED=0 $(GO) build -o $(BUILD_DIR)/$(BINARY) ./cmd/mars-harness

install:
	CGO_ENABLED=0 $(GO) install ./cmd/mars-harness
	@echo "Installed $(BINARY) to $(INSTALL_BIN)/$(BINARY)"
	@echo "Run: $(BINARY) version"

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR)
