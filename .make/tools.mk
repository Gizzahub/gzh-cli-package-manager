# .make/tools.mk - Tool installation
# Included by main Makefile

# ISSUE-352: 패밀리 고정 묶음. gfm이 표를 처리하므로 tables 플러그인은 쓰지 않는다.
MDFORMAT_VERSION := 1.0.0
MDFORMAT_GFM_VERSION := 1.0.0
MDFORMAT_FRONTMATTER_VERSION := 2.1.2
MDFORMAT := $(CURDIR)/bin/tools/mdformat$(shell $(GO) env GOEXE)
MDFORMAT_TOOL_DIR := $(CURDIR)/bin/tools/python
MDFORMAT_PYTHON := $(MDFORMAT_TOOL_DIR)/mdformat/$(if $(shell $(GO) env GOEXE),Scripts/python.exe,bin/python)
MDFORMAT_CONSTRAINTS := $(CURDIR)/.make/mdformat-constraints.txt
# --version 출력을 awk에 그대로 파이프하지 않고 변수에 담는다: producer가 죽으면
# && 체인이 awk 이전에 멈추므로 파싱 실패가 producer 실패를 가리지 않는다.
MDFORMAT_VERSION_OK = { mdformat_version_output=$$("$(MDFORMAT)" --version 2>/dev/null) && printf '%s\n' "$$mdformat_version_output" | \
	awk -v core="mdformat $(MDFORMAT_VERSION)" -v a="mdformat-gfm $(MDFORMAT_GFM_VERSION)" -v b="mdformat_frontmatter $(MDFORMAT_FRONTMATTER_VERSION)" \
	'NR == 1 && sub(/\)$$/, "") { n = split($$0, p, / \(|, /); ok = n == 3 && p[1] == core && ((p[2] == a && p[3] == b) || (p[2] == b && p[3] == a)) } END { exit !ok }'; }
MDFORMAT_DEPS_OK = "$(MDFORMAT_PYTHON)" -I -c 'import sys, importlib.metadata as m; lines = [l for l in map(str.strip, open(sys.argv[1])) if l and l[0] != chr(35)]; got = lambda n: next((d.version for d in m.distributions(name=n)), None); pairs = [(l, l.split("==")) for l in lines]; bad = [l + (" (installed %s)" % got(p[0]) if len(p) == 2 else " (not name==version)") for l, p in pairs if len(p) != 2 or got(p[0]) != p[1]]; bad and print("mdformat dependency mismatch: " + "; ".join(bad), file=sys.stderr); sys.exit(not lines or bool(bad))' "$(MDFORMAT_CONSTRAINTS)"

.PHONY: install-tools install-lint install-fumpt install-mdformat

install-tools: install-lint install-fumpt install-mdformat ## Install all development tools
	@echo "✅ All tools installed"

# Reports the Go toolchain that built the installed linter, e.g. "go1.27.3",
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

install-mdformat: ## install the pinned Markdown bundle into bin/tools
	@mkdir -p "$(dir $(MDFORMAT))"
	@if ! { $(MDFORMAT_VERSION_OK) && $(MDFORMAT_DEPS_OK) 2>/dev/null; }; then \
		command -v uv >/dev/null 2>&1 || { echo "uv is required to install mdformat" >&2; exit 1; }; \
		UV_TOOL_DIR="$(MDFORMAT_TOOL_DIR)" UV_TOOL_BIN_DIR="$(dir $(MDFORMAT))" \
			uv tool install --quiet --force --constraints "$(MDFORMAT_CONSTRAINTS)" \
				--with "mdformat-gfm==$(MDFORMAT_GFM_VERSION)" \
				--with "mdformat-frontmatter==$(MDFORMAT_FRONTMATTER_VERSION)" \
				"mdformat==$(MDFORMAT_VERSION)" || exit 1; \
	fi
	@$(MDFORMAT_VERSION_OK) || { echo "mdformat version/plugin set mismatch: $(MDFORMAT)" >&2; exit 1; }
	@$(MDFORMAT_DEPS_OK) || { echo "mdformat dependencies do not match $(MDFORMAT_CONSTRAINTS)" >&2; exit 1; }

install-goreleaser: ## Install goreleaser
	@echo "Installing goreleaser..."
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		$(GO) install github.com/goreleaser/goreleaser@latest; \
	else \
		echo "goreleaser already installed"; \
	fi
