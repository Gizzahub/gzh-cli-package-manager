# .make/vars.mk - Common variables
# Included by main Makefile

# Project settings
BINARY_NAME := gz-pm
BUILD_DIR := build
MAIN_PKG := ./cmd/gz-pm

# Version information
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION_PKG := github.com/gizzahub/gzh-cli-package-manager/internal/version
LDFLAGS := -ldflags "-X $(VERSION_PKG).Version=$(VERSION) -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT) -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)"

# Go commands
GO := go
# -trimpath keeps the building machine's absolute paths out of the binary, so a
# build is reproducible from source alone and a published checksum can be
# verified by someone who does not share our directory layout. It is set here
# rather than per-target so every build in this repository agrees; the release
# workflow and scripts/release/snapshot-archives.sh pass it explicitly for the
# same reason.
GOBUILD := $(GO) build -trimpath
GOTEST := $(GO) test
GOINSTALL := $(GO) install
GOMOD := $(GO) mod
GOFMT := $(GO) fmt
GOVET := $(GO) vet

# Test settings
COVERAGE_OUT := coverage.out
COVERAGE_HTML := coverage.html
TEST_TIMEOUT := 5m
RACE_FLAG := -race

# Linter settings
# Keep the linter under the ignored repository-local /bin directory so every
# Make target runs the same pinned binary regardless of PATH contents.
GOLANGCI_LINT_VERSION := v2.13.1
GOLANGCI_LINT_BARE := $(patsubst v%,%,$(GOLANGCI_LINT_VERSION))
GOLANGCI_LINT_DIR ?= $(CURDIR)/bin/tools
GOLANGCI_LINT_BIN := $(GOLANGCI_LINT_DIR)/golangci-lint$(shell $(GO) env GOEXE)

# golangci-lint takes a machine-global lock so concurrent runs cannot exhaust
# memory. Without this flag a run that loses the race exits immediately with
# "parallel golangci-lint is running", so linting any other repository on this
# machine turns this repository's gate red. The flag makes the run wait for the
# lock and execute serially instead, which reports on the code rather than on
# the machine's concurrency state.
GOLANGCI_LINT_RUN_FLAGS ?= --allow-serial-runners
