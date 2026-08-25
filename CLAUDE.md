# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Quick Start (30s scan)

**Binary**: `gz-pm` (Package Manager Control)
**Status**: Documentation phase (Week 2) - implementation pending
**Architecture**: Clean Architecture + Hexagonal (Ports & Adapters)
**Go Version**: 1.26+

Think of it as a "package manager for package managers" - unified interface for:
- **System**: Homebrew, apt, pacman, winget (Windows)
- **Version**: asdf
- **Language**: npm, pip, cargo

---

## Top 10 Commands

| Command | Purpose | Usage |
|---------|---------|-------|
| `make build` | Build binary → `build/gz-pm` | Before testing |
| `make test-unit` | Unit tests (fast, no Docker) | Quick validation |
| `make quality` | fmt + lint + test | Pre-commit |
| `make test-coverage` | Full coverage report | Coverage check |
| `make fmt` | Format code (required) | Before commit |
| `make lint` | Run golangci-lint | Fix issues |
| `make install` | Install to $GOPATH/bin | Local install |
| `make dev ARGS="..."` | Run in dev mode | Quick test |
| `make clean` | Clean artifacts | Fresh start |
| `make version` | Show version info | Debug builds |

---

## Absolute Rules (DO/DON'T)

### DO
- ✅ Use `gzh-cli-core` for common utilities (logger, testutil, errors, config)
- ✅ Follow Clean Architecture layers (Domain → Application → Infrastructure → Presentation)
- ✅ Run `make quality` before every commit
- ✅ Maintain 90%+ test coverage
- ✅ Keep files < 300 lines (~10KB)

### DON'T
- ❌ Import external libraries in domain layer (stdlib only)
- ❌ Add CGO dependencies (pure Go only)
- ❌ Bypass use cases (CLI → infrastructure directly)
- ❌ Create large God files (> 500 lines)
- ❌ Mock in domain layer tests (pure functions only)

---

## Directory Structure

```
.
├── cmd/pm/              # CLI entry (Presentation)
├── pkg/
│   ├── domain/          # Core logic (NO external deps)
│   ├── application/     # Use cases + ports
│   └── infrastructure/  # Adapters + repos
├── internal/            # Private utilities
├── docs/
│   ├── .claude-context/ # Context docs (see below)
│   └── 10-architecture/ # Full docs (layers, ADRs)
└── Makefile             # Authoritative commands
```

---

## Context Documentation

| Guide | Purpose |
|-------|---------|
| [Architecture Guide](docs/.claude-context/architecture-guide.md) | Layer rules, ADRs, DI pattern |
| [Testing Guide](docs/.claude-context/testing-guide.md) | Coverage targets, test organization |
| [Build Guide](docs/.claude-context/build-guide.md) | Build commands, CGO policy |
| [Common Tasks](docs/.claude-context/common-tasks.md) | Workflows, code style |

---

## Common Mistakes (Top 3)

1. **Importing infrastructure in domain layer**
   - ⚠️ Violates Clean Architecture
   - ✅ Check: `go list -test -deps ./pkg/domain/... | grep infrastructure`

2. **Skipping `make quality` before commit**
   - ⚠️ CI will fail
   - ✅ Run: `make quality` (includes fmt + lint + test)

3. **Adding CGO dependencies**
   - ⚠️ Breaks cross-compilation
   - ✅ Check: `./scripts/check-cgo.sh`

---

## Shared Library (gzh-cli-core)

**IMPORTANT**: Use `gzh-cli-core` for common utilities. DO NOT create local duplicates.

```go
import (
    "github.com/gizzahub/gzh-cli-core/logger"
    "github.com/gizzahub/gzh-cli-core/errors"
    "github.com/gizzahub/gzh-cli-core/testutil"
    "github.com/gizzahub/gzh-cli-core/config"
    "github.com/gizzahub/gzh-cli-core/cli"
)
```

---

## Git Commit Format

**Required format** (enforced in reviews):

```
{type}({scope}): {imperative verb} {what}

{detailed description}

Model: claude-{model}
Co-Authored-By: Claude <noreply@anthropic.com>
```

**Types**: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`
**Scopes** (mandatory): `domain`, `application`, `infrastructure`, `cli`, `build`, `docs`, `test`

---

## Project Status

- **Current Phase**: Week 2 - Documentation (Pre-implementation)
- **Next Milestone**: v1.0 MVP (Week 9)
- **Test Coverage**: Target 90%+ (setup phase)
- **Platforms**: macOS, Linux, Windows (native winget + WSL2)

---

## Important Files

- `ARCHITECTURE.md` - Architecture overview + index
- `docs/10-architecture/` - Full architecture documentation (split by topic)
- `CONTRIBUTING.md` - Development guidelines
- `PRD.md` - Product vision and roadmap
- `REQUIREMENTS.md` - Functional/non-functional requirements
- `docs/10-architecture/adr/` - Architecture Decision Records
- `Makefile` - Build automation

---

**Last Updated**: 2025-12-27
**Previous**: 528 lines → **Current**: 138 lines (74% reduction)
