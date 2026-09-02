# .make/tools.mk - Tool installation
# Included by main Makefile

.PHONY: install-tools install-lint install-fumpt

install-tools: install-lint install-fumpt ## Install all development tools
	@echo "✅ All tools installed"

# Reports the Go toolchain that built the installed linter, e.g. "go1.26.7",
# or nothing when it is absent or unreadable.
#
# Read from the binary's embedded build info rather than from golangci-lint's
# own `version` banner. The banner is that project's display text and it is free
# to reword it in any release; the build info is Go's own format. A reworded
# banner would make this probe return nothing, and the post-install assertion
# below would then turn an upgrade into a hard build failure. This also reads
# the binary instead of executing it.
GOLANGCI_LINT_BUILT_WITH = $(GO) version -m "$(GOLANGCI_LINT_BIN)" 2>/dev/null | awk 'NR==1{print $$NF}'

install-lint: ## Install golangci-lint
	@want_go="$$($(GO) env GOVERSION)"; \
	if [ -x "$(GOLANGCI_LINT_BIN)" ] && \
		"$(GOLANGCI_LINT_BIN)" version --short 2>/dev/null | grep -qxF "$(GOLANGCI_LINT_BARE)" && \
		[ "$$($(GOLANGCI_LINT_BUILT_WITH))" = "$$want_go" ]; then \
		echo "golangci-lint $(GOLANGCI_LINT_VERSION) already installed: $(GOLANGCI_LINT_BIN)"; \
	else \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION) to $(GOLANGCI_LINT_BIN)..."; \
		mkdir -p "$(GOLANGCI_LINT_DIR)"; \
		GOBIN="$(GOLANGCI_LINT_DIR)" $(GOINSTALL) github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		"$(GOLANGCI_LINT_BIN)" version --short 2>/dev/null | grep -qxF "$(GOLANGCI_LINT_BARE)" || { \
			echo "golangci-lint installation did not produce $(GOLANGCI_LINT_VERSION): $(GOLANGCI_LINT_BIN)" >&2; \
			exit 1; \
		}; \
		got_go="$$($(GOLANGCI_LINT_BUILT_WITH))"; \
		[ "$$got_go" = "$$want_go" ] || { \
			echo "golangci-lint was built with $$got_go but the active toolchain is $$want_go: $(GOLANGCI_LINT_BIN)" >&2; \
			exit 1; \
		}; \
	fi

install-fumpt: ## Install gofumpt
	@echo "Installing gofumpt..."
	@if ! command -v gofumpt >/dev/null 2>&1; then \
		$(GO) install mvdan.cc/gofumpt@latest; \
	else \
		echo "gofumpt already installed"; \
	fi

install-goreleaser: ## Install goreleaser
	@echo "Installing goreleaser..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		$(GO) install github.com/goreleaser/goreleaser@latest; \
	else \
		echo "goreleaser already installed"; \
	fi
