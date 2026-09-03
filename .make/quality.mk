# .make/quality.mk - Code quality targets
# Included by main Makefile

.PHONY: lint fmt vet quality check
.PHONY: fmt-diff lint-diff fmt-check lint-check
.PHONY: fmt-md security

# ==============================================================================
# Standard Quality Checks
# ==============================================================================

lint: install-lint ## Run golangci-lint
	@echo "Running golangci-lint..."
	@"$(GOLANGCI_LINT_BIN)" run $(GOLANGCI_LINT_RUN_FLAGS) ./...

# Formats with the pinned golangci-lint rather than a gofumpt from PATH. The
# previous form fell back to plain go fmt with a notice and exited 0, so a
# machine without gofumpt reported success having applied neither gofumpt nor
# gci -- and gci was never applied anywhere, because .golangci.yml declared its
# settings without enabling it. Both now come from the same pinned binary the
# Lint job uses, so local and hosted formatting cannot drift apart.
fmt: install-lint ## Format code (gofumpt + gci, via pinned golangci-lint)
	@echo "Formatting code..."
	@"$(GOLANGCI_LINT_BIN)" fmt ./...
	@echo "✅ Formatted"

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

quality: fmt lint test ## Run all quality checks (fmt + lint + test)
	@echo "✅ All quality checks passed"

check: vet lint ## Quick check (vet + lint, no tests)
	@echo "✅ Quick checks passed"

# ==============================================================================
# Diff-Aware Checks (faster for incremental development)
# ==============================================================================

fmt-diff: install-lint ## Format only changed Go files
	@echo "Formatting changed files..."
	@git diff --name-only --diff-filter=ACMR HEAD | grep '\.go$$' | \
		xargs -r "$(GOLANGCI_LINT_BIN)" fmt
	@echo "✅ Changed files formatted"

# Base is the integration branch tip (not HEAD~1): multi-commit task
# branches and the first commit on a branch both need that baseline.
LINT_DIFF_BASE ?= origin/master

lint-diff: install-lint ## Lint changes since origin/master (LINT_DIFF_BASE=...)
	@echo "Linting changes since $(LINT_DIFF_BASE)..."
	@"$(GOLANGCI_LINT_BIN)" run $(GOLANGCI_LINT_RUN_FLAGS) --new-from-rev="$(LINT_DIFF_BASE)" ./...

fmt-check: ## Check if code is formatted (for CI)
	@echo "Checking code format..."
	@test -z "$$(gofmt -l .)" || { echo "Code is not formatted. Run: make fmt"; exit 1; }
	@echo "✅ Code is properly formatted"

lint-check: install-lint ## Run lint without fixing (for CI)
	@echo "Checking lint..."
	@"$(GOLANGCI_LINT_BIN)" run $(GOLANGCI_LINT_RUN_FLAGS) ./...

# ==============================================================================
# Additional Quality Tools
# ==============================================================================

fmt-md: ## Format markdown files (requires mdformat)
	@echo "Formatting markdown..."
	@if command -v mdformat >/dev/null 2>&1; then \
		find . -name "*.md" -not -path "./vendor/*" -not -path "./.git/*" | xargs mdformat; \
		echo "✅ Markdown formatted"; \
	else \
		echo "⚠️  mdformat not installed. Install: pip install mdformat"; \
	fi

security: ## Run security scan (gosec)
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -exclude-generated ./...; \
	else \
		echo "⚠️  gosec not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi
