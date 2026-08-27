# Makefile for gz-pm (Package Manager Control)
# ==============================================================================
# Modular Makefile - includes from .make/ directory
# ==============================================================================

.DEFAULT_GOAL := help

# Include modular makefiles
include .make/vars.mk
include .make/build.mk
include .make/release.mk
include .make/deps.mk
include .make/test.mk
include .make/quality.mk
include .make/tools.mk
include .make/dev.mk
include .make/docker.mk

# ==============================================================================
# Help Target
# ==============================================================================

.PHONY: help
help: ## Display this help message
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  gz-pm - Package Manager Control"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@awk 'BEGIN {FS = ":.*##"; category=""} \
		/^##@ / { \
			category = substr($$0, 5); \
			printf "\n\033[1m%s\033[0m\n", category; \
		} \
		/^[a-zA-Z_0-9-]+:.*?##/ { \
			if (category == "") printf "\n\033[1mGeneral\033[0m\n"; \
			printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 \
		}' $(MAKEFILE_LIST) .make/*.mk
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "  Quick Start: make build && make test"
	@echo "  Quality Check: make quality"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""

# ==============================================================================
# Common Targets (organized by category)
# ==============================================================================

##@ Build
# Targets defined in .make/build.mk

##@ Dependencies
# Targets defined in .make/deps.mk

##@ Testing
# Targets defined in .make/test.mk

##@ Code Quality
# Targets defined in .make/quality.mk

##@ Development
# Targets defined in .make/dev.mk

##@ Docker
# Targets defined in .make/docker.mk

##@ Tools
# Targets defined in .make/tools.mk

# ==============================================================================
# Cleanup
# ==============================================================================

.PHONY: clean clean-all

clean: ## Remove build artifacts and coverage files
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_OUT) $(COVERAGE_HTML) bench.txt
	@find . -name "*.test" -type f -delete
	@echo "✅ Cleaned"

clean-all: clean docker-clean ## Deep clean (build + docker + cache)
	@echo "Performing deep clean..."
	@$(GO) clean -cache -testcache -modcache
	@echo "✅ Deep clean completed"

# ==============================================================================
# Project Info
# ==============================================================================

.PHONY: version info

version: ## Display version information
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $$(go version | awk '{print $$3}')"

info: ## Display project information
	@echo "Project Information:"
	@echo "  Binary Name:  $(BINARY_NAME)"
	@echo "  Build Dir:    $(BUILD_DIR)"
	@echo "  Main Package: $(MAIN_PKG)"
	@echo "  Go Version:   $$(go version | awk '{print $$3}')"
	@echo ""
	@echo "Build Settings:"
	@echo "  CGO Enabled:  $${CGO_ENABLED:-0}"
	@echo "  Version:      $(VERSION)"
	@echo "  Git Commit:   $(GIT_COMMIT)"
	@echo "  Build Date:   $(BUILD_DATE)"
