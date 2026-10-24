# Makefile for Datadog CLI

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -ldflags="-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)"

# Binary name
BINARY := dd

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod

# Directories
BUILD_DIR := bin
DIST_DIR := dist
CMD_DIR := ./cmd

# Platforms for cross-compilation
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64

.PHONY: all
all: clean lint test build

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build          - Build binary for current platform"
	@echo "  make build-all      - Build binaries for all platforms"
	@echo "  make test           - Run tests"
	@echo "  make test-coverage  - Run tests with coverage"
	@echo "  make test-race      - Run tests with race detector"
	@echo "  make lint           - Run linters"
	@echo "  make fmt            - Format code"
	@echo "  make vet            - Run go vet"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install        - Install binary to GOPATH/bin"
	@echo "  make docker         - Build Docker image"
	@echo "  make deps           - Download dependencies"
	@echo "  make verify         - Verify dependencies"

.PHONY: build
build:
	@echo "Building $(BINARY) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY)"

.PHONY: build-all
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(DIST_DIR)
	@$(foreach platform,$(PLATFORMS), \
		$(eval GOOS := $(word 1,$(subst /, ,$(platform)))) \
		$(eval GOARCH := $(word 2,$(subst /, ,$(platform)))) \
		$(eval OUTPUT := $(DIST_DIR)/$(BINARY)-$(GOOS)-$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,)) \
		echo "Building $(GOOS)/$(GOARCH)..." && \
		GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(OUTPUT) $(CMD_DIR) && \
	) echo "Cross-compilation complete"

.PHONY: install
install:
	@echo "Installing $(BINARY)..."
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY) $(CMD_DIR)
	@echo "Installed to $(GOPATH)/bin/$(BINARY)"

.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

.PHONY: test-race
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -race ./...

.PHONY: bench
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run --timeout=5m

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .
	@echo "Code formatted"

.PHONY: fmt-check
fmt-check:
	@echo "Checking code formatting..."
	@test -z "$$($(GOFMT) -s -l .)" || (echo "Code needs formatting. Run 'make fmt'" && $(GOFMT) -s -l . && exit 1)

.PHONY: vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

.PHONY: verify
verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@rm -f $(BINARY)
	@echo "Clean complete"

.PHONY: docker
docker:
	@echo "Building Docker image..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t datadog-cli:$(VERSION) \
		-t datadog-cli:latest \
		.
	@echo "Docker image built: datadog-cli:$(VERSION)"

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run --rm datadog-cli:latest version

.PHONY: ci
ci: fmt-check vet lint test-race test-coverage
	@echo "CI checks complete"

.PHONY: release-prep
release-prep: clean ci build-all
	@echo "Release preparation complete"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
