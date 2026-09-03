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
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -c make test-unit; \
	else \
		echo "⚠️  entr not installed. Install with: brew install entr (macOS) or apt install entr (Linux)"; \
	fi

pre-commit: quality ## Run pre-commit checks
	@echo "✅ Pre-commit checks passed"

release-dry: ## Dry run release (goreleaser)
	@echo "Running release dry run..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean; \
	else \
		echo "⚠️  goreleaser not installed. Run: make install-goreleaser"; \
	fi

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
