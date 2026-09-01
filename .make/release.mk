# .make/release.mk - Release inventory and archive targets
# Included by main Makefile

.PHONY: release-runtime-deps release-runtime-deps-check
.PHONY: release-targets-check release-payload-check release-snapshot

##@ Release
release-runtime-deps: ## Print the release runtime dependency manifest
	@./scripts/release/runtime-deps.sh

release-runtime-deps-check: ## Verify the committed runtime dependency manifest
	@./scripts/release/runtime-deps.sh --check

release-targets-check: ## Verify workflow, Make, and inventory target sets match
	@./scripts/release/targets-check.sh

release-payload-check: ## Verify license payload hashes and inventory coverage
	@./scripts/release/payload-check.sh

release-snapshot: ## Build snapshot archives, license payload, and checksums
	@mkdir -p $(BUILD_DIR)/snapshot
	@./scripts/release/snapshot-archives.sh --output-dir $(BUILD_DIR)/snapshot
