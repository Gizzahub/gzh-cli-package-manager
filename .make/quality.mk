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

# Checks with the same formatters `fmt` applies. It tested `gofmt -l` alone,
# which knows nothing about gofumpt's extra rules or gci's import grouping, so
# the target named "for CI" could pass on code the Lint job rejects -- the exact
# local/hosted drift enabling the formatters was meant to close.
#
# Measured against the pinned v2.13.1: `golangci-lint fmt --diff` exits 1 when
# it emits a diff and 0 when it emits nothing, so the exit status and the
# emptiness of the output agree. This tests the output because it has to
# capture it either way -- printing the diff next to "run make fmt" is what
# makes the failure actionable rather than just red.
#
# An earlier version of this comment asserted the command exits 0 even with a
# diff, and that testing its status would rebuild a gate that cannot fail. That
# was never measured and it is wrong. It is recorded here because the claim was
# specific enough to be believed and would have argued against the simpler
# check.
fmt-check: install-lint ## Check if code is formatted (for CI)
	@echo "Checking code format..."
	@out="$$("$(GOLANGCI_LINT_BIN)" fmt --diff ./... 2>&1)"; \
	if [ -n "$$out" ]; then \
		printf '%s\n' "$$out"; \
		echo "❌ Code is not formatted. Run: make fmt"; \
		exit 1; \
	fi
	@echo "✅ Code is properly formatted"

lint-check: install-lint ## Run lint without fixing (for CI)
	@echo "Checking lint..."
	@"$(GOLANGCI_LINT_BIN)" run $(GOLANGCI_LINT_RUN_FLAGS) ./...

# ==============================================================================
# Additional Quality Tools
# ==============================================================================

fmt-md: ## Format markdown files (requires mdformat)
	@echo "Formatting markdown..."
	@command -v mdformat >/dev/null 2>&1 || { \
		echo "❌ mdformat not installed. Install: pip install mdformat"; \
		exit 1; \
	}
	@find . -name "*.md" -not -path "./vendor/*" -not -path "./.git/*" | xargs mdformat
	@echo "✅ Markdown formatted"

# Reports failure when gosec is absent. Warning and exiting 0 made "no scan ran"
# indistinguishable from "the scan found nothing", which is the more dangerous of
# the two to mistake for the other.
#
# This target is not redundant with the gosec enabled in .golangci.yml. That one
# runs under exclusions.presets, whose common-false-positives preset drops
# "Potential file inclusion via variable" (G304) outright: deleting the #nosec in
# pkg/application/usecase/bootstrap/bootstrap.go still leaves `golangci-lint run`
# at 0 issues, while gosec reports the finding. Standalone gosec is where those
# rules are actually checked, so an exit code it cannot fail hides real findings
# rather than duplicated ones.
security: ## Run security scan (gosec)
	@echo "Running security scan..."
	@command -v gosec >/dev/null 2>&1 || { \
		echo "❌ gosec not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	}
	gosec -exclude-generated ./...
