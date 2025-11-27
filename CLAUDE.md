# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

---

## Project Overview

**gz-pm** (Package Manager Control) is a CLI tool that orchestrates multiple package managers (Homebrew, asdf, npm, pip, etc.) through a unified interface. Think of it as a "package manager for package managers."

**Binary name**: `gz-pm`
**Source directory**: `cmd/pm/`
**Current status**: Documentation phase (Week 2) - implementation pending

---

## Essential Commands

### Build & Development

```bash
# Build binary (outputs to bin/gz-pm)
make build

# Build for all platforms (macOS, Linux, Windows)
make build-all

# Install to /usr/local/bin
sudo make install

# Run in development mode
make dev ARGS="--version"

# Watch for changes and rebuild (requires entr)
make watch
```

### Testing

```bash
# Run unit tests only (fast, no external dependencies)
make test-unit

# Run integration tests (requires Docker)
make test-integration

# Run all tests with coverage report (generates coverage.html)
make test-coverage

# Test specific package
go test -v ./pkg/domain/manager/...

# Test with race detection
go test -race ./...
```

### Code Quality

```bash
# Format code (required before commits)
make fmt

# Run go vet
make vet

# Run linter (golangci-lint required)
make lint

# Auto-fix linting issues
make lint-fix

# Run all validation checks (fmt + vet + lint + test-unit)
make validate

# Full CI validation (includes coverage)
make ci
```

### Project Information

```bash
# Display version, git commit, build date
make version

# Display build configuration
make info

# Clean build artifacts
make clean
```

---

## Architecture

This project follows **Clean Architecture** with **Hexagonal (Ports & Adapters)** pattern. Understanding this is critical for contributing effectively.

### Layer Structure

```
┌─────────────────────────────────────┐
│  Presentation (cmd/pm)              │  CLI, input validation, formatting
│  - Cobra commands                   │
│  - Output formatters                │
└──────────────┬──────────────────────┘
               │ uses ↓
┌──────────────▼──────────────────────┐
│  Application (pkg/application)      │  Use cases, business workflows
│  - Use case orchestration           │
│  - DTOs                             │
│  - Port interfaces (input/output)   │
└──────────────┬──────────────────────┘
               │ uses ↓
┌──────────────▼──────────────────────┐
│  Domain (pkg/domain)                │  Core business logic (NO external deps)
│  - Entities (Manager, Package)      │
│  - Domain services                  │
│  - Repository interfaces            │
└──────────────▲──────────────────────┘
               │ implements ↑
┌──────────────┴──────────────────────┐
│  Infrastructure (pkg/infrastructure)│  External system integrations
│  - Package manager adapters         │
│  - Repository implementations       │
│  - Command execution                │
└─────────────────────────────────────┘
```

### Critical Architecture Rules

**Dependency Direction**: Dependencies ONLY flow inward (outer → inner).

1. **Domain layer** (`pkg/domain/`):
   - Pure Go stdlib only - NO external libraries
   - NO imports from application/infrastructure/presentation
   - Contains: entities, value objects, domain services, repository interfaces
   - Test: Pure unit tests, no mocks needed (95%+ coverage target)

2. **Application layer** (`pkg/application/`):
   - Imports: domain layer ONLY
   - Defines port interfaces (input/output)
   - Contains: use cases, DTOs, orchestration logic
   - Test: Mock all ports with gomock (90%+ coverage target)

3. **Infrastructure layer** (`pkg/infrastructure/`):
   - Implements domain/application interfaces
   - External libraries allowed here
   - Contains: adapters (Homebrew, asdf, npm), repositories, executors
   - Test: Docker-based integration tests (85%+ coverage target)

4. **Presentation layer** (`cmd/pm/`):
   - Imports: application + infrastructure (for DI only)
   - Contains: CLI commands, input validation, output formatting
   - Thin layer - delegates to application use cases

### Validating Architecture Compliance

```bash
# Check domain layer has no forbidden imports
go list -test -deps ./pkg/domain/... | grep -E "(application|infrastructure|cmd)" && echo "VIOLATION" || echo "OK"

# Check application layer doesn't import infrastructure
go list -test -deps ./pkg/application/... | grep -E "(infrastructure|cmd)" && echo "VIOLATION" || echo "OK"

# Check file sizes (should be < 500 lines / ~10KB)
find pkg cmd -name "*.go" -exec wc -l {} \; | awk '$1 > 500 {print $2 ": " $1 " lines (TOO LARGE)"}'
```

---

## Key Design Decisions (ADRs)

### ADR-002: Clean Architecture
- **Why**: 5+ year lifespan, 90%+ test coverage, easy plugin architecture
- **Impact**: Clear layer boundaries, dependency inversion, high testability
- **Trade-off**: More files/boilerplate vs long-term maintainability

### ADR-006: No CGO Dependencies
- **Why**: Cross-compilation simplicity, static binaries, faster builds
- **Impact**: `CGO_ENABLED=0` enforced in Makefile
- **Consequence**: Must use pure Go alternatives (e.g., `modernc.org/sqlite` not `go-sqlite3`)
- **Validation**: Run `./scripts/check-cgo.sh` to ensure no CGO deps

### Go Version Requirement
- **Minimum**: Go 1.24.0+
- **Check**: `make check-go-version`

---

## Adding a New Package Manager Adapter

This is a common task. Follow this pattern:

1. **Create adapter package**: `pkg/infrastructure/adapter/manager/newmanager/`

2. **Implement ManagerAdapter interface**:
   ```go
   type ManagerAdapter interface {
       Detect(ctx context.Context) (bool, error)
       GetVersion(ctx context.Context) (string, error)
       Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
       ListInstalled(ctx context.Context) ([]Package, error)
       // ... other methods
   }
   ```

3. **Register in factory** (`main.go`):
   ```go
   adapterFactory.Register("newmanager", adapter)
   ```

4. **Write tests**:
   - Unit tests: `adapter_test.go` (mock executor)
   - Integration tests: `adapter_integration_test.go` (Docker container)

5. **Target**: < 1 day per new manager

---

## Testing Strategy

### Test Organization

```
pkg/domain/manager/
├── entity.go          # Core logic
└── entity_test.go     # Pure unit tests (no mocks)

pkg/application/update/
├── update_all.go      # Use case
└── update_all_test.go # Mocked ports (gomock)

pkg/infrastructure/adapter/manager/homebrew/
├── adapter.go                  # Implementation
├── adapter_test.go             # Unit tests (mocked executor)
└── adapter_integration_test.go # Real Homebrew in Docker
```

### Coverage Targets

| Layer | Target | Minimum |
|-------|--------|---------|
| Domain | 95% | 90% |
| Application | 90% | 85% |
| Infrastructure | 85% | 75% |
| **Overall** | **90%** | **85%** |

### Running Tests

```bash
# Fast unit tests only
go test -short ./...

# With race detection
go test -race ./...

# Integration tests (requires Docker)
go test -tags=integration ./...

# Single package with verbose output
go test -v ./pkg/domain/manager/...
```

---

## File Size Policy

**Critical for LLM-friendly codebase**:

| Size | Status | Action |
|------|--------|--------|
| < 10KB (~300 lines) | ✅ Ideal | Maintain |
| 10-50KB | ⚠️ Warning | Consider splitting |
| > 50KB | ❌ Problem | Must split |

**Reason**: Keep files digestible for AI code analysis and human review.

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

**Example**:
```
feat(infrastructure): add Homebrew adapter

Implement ManagerAdapter interface for Homebrew package manager.
Supports detection, version checking, and dry-run updates.

- Add adapter implementation
- Add unit and integration tests
- Register in adapter factory

Model: claude-sonnet-4-5-20250929
Co-Authored-By: Claude <noreply@anthropic.com>
```

---

## Common Workflows

### Implementing a New Use Case

1. **Define domain entities** (`pkg/domain/`) - pure business logic
2. **Create use case** (`pkg/application/`) - orchestration
3. **Define ports** (`pkg/application/port/`) - interfaces
4. **Implement adapters** (`pkg/infrastructure/`) - external integrations
5. **Add CLI command** (`cmd/pm/command/`) - user interface
6. **Wire dependencies** (`cmd/pm/main.go`) - manual DI

### Manual Dependency Injection Pattern

```go
// cmd/pm/main.go
func main() {
    // 1. Infrastructure setup
    logger := logger.NewStructuredLogger("gz-pm")
    executor := executor.NewShellExecutor(logger)

    // 2. Repository setup
    managerRepo := repository.NewManagerRepository(executor, logger)

    // 3. Use case initialization
    updateUC := update.NewUpdateAllManagersUseCase(
        managerRepo,
        executor,
        logger,
        update.NewStableStrategy(),
    )

    // 4. CLI command setup
    rootCmd := command.NewRootCommand()
    rootCmd.AddCommand(command.NewUpdateCommand(updateUC))
    rootCmd.Execute()
}
```

**Why manual DI?**: Explicit, no magic, compile-time safety, easy debugging

---

## Code Style & Conventions

### Naming

- Interfaces: `ManagerRepository`, `CommandExecutor` (no "I" prefix)
- Structs: `Manager`, `UpdateResult`
- Functions: `FindInstalled`, `Update` (verb/verb phrase)
- No Hungarian notation, avoid abbreviations unless well-known (`ID`, `URL`, `HTTP`)

### Error Handling

```go
// ✅ GOOD - wrap with context
managers, err := uc.managerRepo.FindInstalled(ctx)
if err != nil {
    return nil, fmt.Errorf("failed to find installed managers: %w", err)
}

// ❌ BAD - lose context
if err != nil {
    return nil, err
}
```

### Context Usage

- First parameter is always `context.Context`
- Check `ctx.Done()` in long operations
- Don't store context in structs

---

## When Modifying Code

### Before Making Changes

1. **Read architecture docs**: `ARCHITECTURE.md`, relevant ADRs in `docs/architecture/adr/`
2. **Understand layer boundaries**: Where does this change belong?
3. **Check existing patterns**: How are similar features implemented?
4. **Verify test coverage**: Will your changes maintain 90%+ coverage?

### During Implementation

1. **Follow architecture rules**: Respect dependency directions
2. **Keep files small**: Target < 300 lines per file
3. **Write tests first** (TDD for domain/application layers)
4. **No CGO dependencies**: Pure Go only

### Before Committing

```bash
# Run validation suite
make validate

# Check architecture compliance
go list -test -deps ./pkg/domain/... | grep -E "(application|infrastructure|cmd)"

# Ensure tests pass
make test

# Format code
make fmt

# Run linter
make lint
```

---

## Important Files

- `ARCHITECTURE.md` - Full architecture documentation
- `CONTRIBUTING.md` - Development guidelines
- `PRD.md` - Product vision and roadmap
- `REQUIREMENTS.md` - Functional/non-functional requirements
- `docs/architecture/adr/` - Architecture Decision Records
- `docs/specifications/` - Use case specifications
- `Makefile` - Build automation (authoritative command reference)

---

## Project Status

- **Current Phase**: Week 2 - Documentation (Pre-implementation)
- **Next Milestone**: v1.0 MVP (Week 9)
- **Test Coverage**: Target 90%+ (setup phase)
- **Binary Output**: `bin/gz-pm`
- **Platforms**: macOS, Linux (Ubuntu/Arch), Windows (WSL2)

---

## Notes for Claude Code

### When Working in This Codebase

1. **Architecture first**: Always consider which layer changes belong in
2. **Pure domain**: Domain layer must have ZERO external dependencies
3. **Test everything**: Every change needs tests (90%+ coverage is strict)
4. **File size**: If a file grows beyond 500 lines, split it
5. **No CGO**: Never add dependencies that require CGO
6. **Manual DI**: Don't use DI frameworks - wire dependencies explicitly in main.go

### Common Pitfalls to Avoid

1. ❌ Importing infrastructure in domain layer
2. ❌ Bypassing use cases (CLI → infrastructure directly)
3. ❌ Large God files (> 500 lines)
4. ❌ External libraries in domain layer
5. ❌ CGO dependencies
6. ❌ Mocking in domain layer tests (should be pure functions)

### Project-Specific Patterns

- **Use case naming**: `{Action}{Entity}UseCase` (e.g., `UpdateAllManagersUseCase`)
- **Repository pattern**: Interface in domain, implementation in infrastructure
- **Adapter pattern**: All package managers implement `ManagerAdapter` interface
- **Strategy pattern**: Update strategies in domain layer (stable, latest, minor, fixed)
- **Port interfaces**: Defined in application layer, implemented in infrastructure

---

## Quick Reference

```bash
# Most common commands
make build           # Build binary
make test           # Run unit tests
make validate       # Full validation (fmt + vet + lint + test)
make clean          # Clean artifacts

# Binary location after build
./bin/gz-pm --version

# Directory structure
cmd/pm/              # CLI entry point (Presentation)
pkg/domain/          # Business logic (NO external deps)
pkg/application/     # Use cases (orchestration)
pkg/infrastructure/  # Adapters, repos (external integrations)
internal/            # Private utilities
docs/                # Documentation
```

---

**Last Updated**: 2025-01-27
**Document Version**: 1.0
**For Questions**: See `CONTRIBUTING.md` or architecture documentation
