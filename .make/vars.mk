# .make/vars.mk - Common variables
# Included by main Makefile

# Project settings
BINARY_NAME := gz-pm
BUILD_DIR := build
MAIN_PKG := ./cmd/pm

# Version information
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildDate=$(BUILD_DATE)"

# Go commands
GO := go
GOBUILD := $(GO) build
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
