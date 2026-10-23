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

.PHONY: release-coverage-check release-quality release-homebrew
release-coverage-check: ## Check current test report against PRODUCT's 85% floor
	@./scripts/release/coverage-check.sh $(COVERAGE_OUT)

release-quality: quality ## Run quality and enforce release coverage
	@$(MAKE) release-coverage-check

RELEASE_VERSION ?= 0.1.0
RELEASE_CHECKSUMS ?= $(BUILD_DIR)/snapshot/checksums.txt
release-homebrew: ## Generate cask from release checksums; does not publish
	@mkdir -p $(BUILD_DIR)/homebrew
	@./scripts/release/homebrew-cask.sh "$(RELEASE_VERSION)" "$(RELEASE_CHECKSUMS)" > $(BUILD_DIR)/homebrew/gz-pm.rb
