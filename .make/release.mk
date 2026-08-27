# .make/release.mk - Release inventory targets
# Included by main Makefile

.PHONY: release-runtime-deps release-runtime-deps-check

##@ Release
release-runtime-deps: ## Print the release runtime dependency manifest
	@./scripts/release/runtime-deps.sh

release-runtime-deps-check: ## Verify the committed runtime dependency manifest
	@./scripts/release/runtime-deps.sh --check
