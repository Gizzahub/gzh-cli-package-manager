# .make/dev.mk - Development workflow targets
# Included by main Makefile

.PHONY: dev dev-fast verify pr-check ci-local
.PHONY: watch pre-commit release-dry
.PHONY: comments todo deps-graph changelog

# ==============================================================================
# Development Workflows
# ==============================================================================

dev: fmt lint test ## Standard development workflow (format, lint, test)
	@echo "✅ Development workflow completed"

dev-fast: fmt test-unit ## Quick development cycle (format + unit tests only)
	@echo "✅ Fast development cycle completed"

verify: fmt lint test test-coverage ## Complete verification before PR
	@echo "✅ Verification completed"

pr-check: fmt lint test ## Pre-PR submission check
	@echo "✅ Pre-PR check passed - ready for submission"

ci-local: clean quality build ## Run full CI pipeline locally
	@echo "✅ Local CI pipeline completed"

# ==============================================================================
# Development Helpers
# ==============================================================================

watch: ## Watch for changes and run tests (requires entr)
	@echo "Watching for changes..."
	@command -v entr >/dev/null 2>&1 || { \
		echo "❌ entr not installed. brew install entr (macOS) / apt install entr (Linux)"; \
		exit 1; \
	}
	@find . -name "*.go" | entr -c make test-unit

pre-commit: quality ## Run pre-commit checks
	@echo "✅ Pre-commit checks passed"

# Reports failure when goreleaser is absent, for the same reason as `security`.
#
# Note that this target cannot succeed as the repository stands: there is no
# .goreleaser.yml, so goreleaser has nothing to read even when installed. The
# release path that is actually wired up and exercised is
# `make release-snapshot` plus .github/workflows/build.yml. Add a goreleaser
# config before relying on this, or drop the target -- what should not continue
# is a second, half-present release path that reports success by doing nothing.
release-dry: ## Dry run release (needs goreleaser + a .goreleaser.yml)
	@echo "Running release dry run..."
	@command -v goreleaser >/dev/null 2>&1 || { \
		echo "❌ goreleaser not installed. Run: make install-goreleaser"; \
		exit 1; \
	}
	goreleaser release --snapshot --clean

# ==============================================================================
# Code Analysis
# ==============================================================================

comments: ## Show all TODO/FIXME/NOTE comments
	@echo "=== TODO comments ==="
	@grep -rn "TODO" --include="*.go" . 2>/dev/null | grep -v vendor | grep -v .git || echo "No TODOs found"
	@echo ""
	@echo "=== FIXME comments ==="
	@grep -rn "FIXME" --include="*.go" . 2>/dev/null | grep -v vendor | grep -v .git || echo "No FIXMEs found"
	@echo ""
	@echo "=== NOTE comments ==="
	@grep -rn "NOTE" --include="*.go" . 2>/dev/null | grep -v vendor | grep -v .git || echo "No NOTEs found"

todo: comments ## Alias for comments

deps-graph: ## Show module dependency graph
	@echo "Module dependency graph:"
	@go mod graph

# ==============================================================================
# Documentation
# ==============================================================================

# Reports failure when the tool is absent. Warning and exiting 0 made "no
# changelog was generated" indistinguishable from "a changelog was generated",
# so a caller reading only the exit status could not tell the two apart.
#
# As shipped this target cannot succeed here, and that is deliberate rather
# than overlooked: it needs both git-chglog on PATH and a .chglog/ config,
# neither of which this repository carries. git-chglog fails loudly by itself
# on the missing config. Add both before invoking it; nothing else calls it.
changelog: ## Generate CHANGELOG.md on demand (needs git-chglog + .chglog/ config)
	@command -v git-chglog >/dev/null 2>&1 || { \
		echo "❌ git-chglog not installed. See: https://github.com/git-chglog/git-chglog"; \
		exit 1; \
	}
	git-chglog -o CHANGELOG.md
	@echo "✅ Changelog generated"

docs-serve: ## Serve documentation locally (requires mdbook or similar)
	@if command -v mdbook >/dev/null 2>&1; then \
		cd docs && mdbook serve; \
	else \
		echo "⚠️  mdbook not installed. Using python http.server..."; \
		cd docs && python3 -m http.server 8000; \
	fi
