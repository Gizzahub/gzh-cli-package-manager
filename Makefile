.PHONY: help build build-all install clean test test-unit test-integration test-integration-fast test-e2e test-coverage lint lint-fix fmt vet check-go-version deps

# Variables
BINARY_NAME=gz-pm
BUILD_DIR=./build
BIN_DIR=./bin
CMD_DIR=./cmd/pm
PKG_DIRS=$(shell go list ./... | grep -v /vendor/)

# Build variables
VERSION?=0.1.0
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X github.com/gizzahub/gzh-cli-package-manager/internal/version.Version=$(VERSION) -X github.com/gizzahub/gzh-cli-package-manager/internal/version.GitCommit=$(GIT_COMMIT) -X github.com/gizzahub/gzh-cli-package-manager/internal/version.BuildDate=$(BUILD_DATE)"

# Go build settings (ADR-006: No CGO)
export CGO_ENABLED=0
export GO111MODULE=on

# Colors for output
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m
COLOR_BLUE=\033[34m

##@ General

help: ## Display this help message
	@echo "$(COLOR_BOLD)gz-pm Makefile$(COLOR_RESET)"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make $(COLOR_BLUE)<target>$(COLOR_RESET)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(COLOR_BLUE)%-20s$(COLOR_RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(COLOR_BOLD)%s$(COLOR_RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

check-go-version: ## Check Go version meets requirement (1.24+)
	@echo "$(COLOR_YELLOW)Checking Go version...$(COLOR_RESET)"
	@GO_VERSION=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	REQUIRED="1.24"; \
	if [ "$$(printf '%s\n' "$$REQUIRED" "$$GO_VERSION" | sort -V | head -n1)" != "$$REQUIRED" ]; then \
		echo "$(COLOR_RED)Error: Go $$REQUIRED or higher required, found $$GO_VERSION$(COLOR_RESET)"; \
		exit 1; \
	fi; \
	echo "$(COLOR_GREEN)✓ Go $$GO_VERSION$(COLOR_RESET)"

deps: check-go-version ## Download dependencies
	@echo "$(COLOR_YELLOW)Downloading dependencies...$(COLOR_RESET)"
	@go mod download
	@go mod tidy
	@echo "$(COLOR_GREEN)✓ Dependencies downloaded$(COLOR_RESET)"

##@ Build

build: check-go-version ## Build gz-pm binary
	@echo "$(COLOR_YELLOW)Building $(BINARY_NAME)...$(COLOR_RESET)"
	@mkdir -p $(BIN_DIR)
	@go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "$(COLOR_GREEN)✓ Built: $(BIN_DIR)/$(BINARY_NAME)$(COLOR_RESET)"

build-all: check-go-version ## Build binaries for all platforms
	@echo "$(COLOR_YELLOW)Building for all platforms...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)

	# macOS (Intel)
	@echo "  → macOS amd64"
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)

	# macOS (Apple Silicon)
	@echo "  → macOS arm64"
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

	# Linux (amd64)
	@echo "  → Linux amd64"
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)

	# Linux (arm64)
	@echo "  → Linux arm64"
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

	# Windows (amd64)
	@echo "  → Windows amd64"
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

	@echo "$(COLOR_GREEN)✓ Built all platform binaries in $(BUILD_DIR)/$(COLOR_RESET)"
	@ls -lh $(BUILD_DIR)/

install: build ## Install gz-pm to GOPATH/bin (no sudo required)
	@if [ -z "$(GOPATH)" ]; then \
		echo "$(COLOR_YELLOW)⚠ GOPATH not set, using default: $(HOME)/go$(COLOR_RESET)"; \
		GOPATH_BIN="$(HOME)/go/bin"; \
	else \
		GOPATH_BIN="$(GOPATH)/bin"; \
	fi; \
	echo "$(COLOR_YELLOW)Installing $(BINARY_NAME) to $$GOPATH_BIN...$(COLOR_RESET)"; \
	mkdir -p $$GOPATH_BIN; \
	cp $(BIN_DIR)/$(BINARY_NAME) $$GOPATH_BIN/$(BINARY_NAME); \
	echo "$(COLOR_GREEN)✓ Installed: $$GOPATH_BIN/$(BINARY_NAME)$(COLOR_RESET)"; \
	echo "$(COLOR_YELLOW)Note: Ensure $$GOPATH_BIN is in your PATH$(COLOR_RESET)"

clean: ## Remove build artifacts
	@echo "$(COLOR_YELLOW)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(BIN_DIR) $(BUILD_DIR)
	@rm -f coverage.txt coverage.html *.coverprofile
	@find . -name "*.test" -delete
	@find . -name "*.out" -delete
	@echo "$(COLOR_GREEN)✓ Cleaned$(COLOR_RESET)"

##@ Testing

test: test-unit ## Run all tests (alias for test-unit)

test-unit: ## Run unit tests
	@echo "$(COLOR_YELLOW)Running unit tests...$(COLOR_RESET)"
	@go test -v -short $(PKG_DIRS)
	@echo "$(COLOR_GREEN)✓ Unit tests passed$(COLOR_RESET)"

test-integration: build ## Run integration tests (requires built binary)
	@echo "$(COLOR_YELLOW)Running integration tests...$(COLOR_RESET)"
	@go test -v -tags=integration ./test/integration/...
	@echo "$(COLOR_GREEN)✓ Integration tests passed$(COLOR_RESET)"

test-integration-fast: build ## Run fast integration tests (CLI help/version only)
	@echo "$(COLOR_YELLOW)Running fast integration tests...$(COLOR_RESET)"
	@go test -v -tags=integration ./test/integration/... -run "TestCLI_(Version|Help|InvalidCommand|InvalidFlag|.*_Exists)"
	@echo "$(COLOR_GREEN)✓ Fast integration tests passed$(COLOR_RESET)"

test-e2e: build ## Run end-to-end tests
	@echo "$(COLOR_YELLOW)Running E2E tests...$(COLOR_RESET)"
	@./test/e2e/run-tests.sh
	@echo "$(COLOR_GREEN)✓ E2E tests passed$(COLOR_RESET)"

test-coverage: ## Run tests with coverage report
	@echo "$(COLOR_YELLOW)Running tests with coverage...$(COLOR_RESET)"
	@go test -v -coverprofile=coverage.txt -covermode=atomic $(PKG_DIRS)
	@go tool cover -html=coverage.txt -o coverage.html
	@go tool cover -func=coverage.txt | grep total | awk '{print "Coverage: " $$3}'
	@echo "$(COLOR_GREEN)✓ Coverage report: coverage.html$(COLOR_RESET)"

##@ Quality

fmt: ## Format Go code
	@echo "$(COLOR_YELLOW)Formatting code...$(COLOR_RESET)"
	@go fmt $(PKG_DIRS)
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

vet: ## Run go vet
	@echo "$(COLOR_YELLOW)Running go vet...$(COLOR_RESET)"
	@go vet $(PKG_DIRS)
	@echo "$(COLOR_GREEN)✓ go vet passed$(COLOR_RESET)"

lint: ## Run golangci-lint
	@echo "$(COLOR_YELLOW)Running linters...$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout 5m; \
		echo "$(COLOR_GREEN)✓ Linting passed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint not installed, skipping...$(COLOR_RESET)"; \
		echo "  Install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

lint-fix: ## Run golangci-lint with auto-fix
	@echo "$(COLOR_YELLOW)Running linters with auto-fix...$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --fix --timeout 5m; \
		echo "$(COLOR_GREEN)✓ Linting fixes applied$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint not installed$(COLOR_RESET)"; \
	fi

##@ Development

dev: ## Build and run gz-pm in development mode
	@echo "$(COLOR_YELLOW)Running gz-pm in development mode...$(COLOR_RESET)"
	@go run $(CMD_DIR) $(ARGS)

watch: ## Watch for changes and rebuild (requires entr)
	@echo "$(COLOR_YELLOW)Watching for changes...$(COLOR_RESET)"
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -r make build; \
	else \
		echo "$(COLOR_YELLOW)⚠ entr not installed$(COLOR_RESET)"; \
		echo "  Install: brew install entr"; \
	fi

##@ Validation

validate: check-go-version fmt vet lint test-unit ## Run all validation checks
	@echo ""
	@echo "$(COLOR_GREEN)✓ All validation checks passed$(COLOR_RESET)"

ci: validate test-coverage ## Run CI validation pipeline
	@echo ""
	@echo "$(COLOR_GREEN)✓ CI validation complete$(COLOR_RESET)"

##@ Utilities

version: ## Display version information
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $$(go version | awk '{print $$3}')"

info: ## Display build information
	@echo "Binary Name:  $(BINARY_NAME)"
	@echo "Build Dir:    $(BUILD_DIR)"
	@echo "Bin Dir:      $(BIN_DIR)"
	@echo "CMD Dir:      $(CMD_DIR)"
	@echo "CGO Enabled:  $(CGO_ENABLED)"
	@echo "Go Modules:   $(GO111MODULE)"

# Default target
.DEFAULT_GOAL := help
