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
	@"$(GOLANGCI_LINT_BIN)" run ./...

fmt: ## Format code with gofmt and gofumpt
	@echo "Formatting code..."
	$(GOFMT) ./...
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
	else \
		echo "gofumpt not installed, using go fmt only"; \
	fi

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

fmt-diff: ## Format only changed Go files
	@echo "Formatting changed files..."
	@git diff --name-only --diff-filter=ACMR HEAD | grep '\.go$$' | xargs -r gofmt -w
	@if command -v gofumpt >/dev/null 2>&1; then \
		git diff --name-only --diff-filter=ACMR HEAD | grep '\.go$$' | xargs -r gofumpt -w; \
	fi
	@echo "✅ Changed files formatted"

lint-diff: install-lint ## Lint only changed Go files
	@echo "Linting changed files..."
	@"$(GOLANGCI_LINT_BIN)" run --new-from-rev=HEAD~1 ./...

fmt-check: ## Check if code is formatted (for CI)
	@echo "Checking code format..."
	@test -z "$$(gofmt -l .)" || { echo "Code is not formatted. Run: make fmt"; exit 1; }
	@echo "✅ Code is properly formatted"

lint-check: install-lint ## Run lint without fixing (for CI)
	@echo "Checking lint..."
	@"$(GOLANGCI_LINT_BIN)" run ./...

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
