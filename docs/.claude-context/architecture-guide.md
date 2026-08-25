# Architecture Guide - gzh-cli-package-manager

## Layer Structure

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

## Critical Architecture Rules

**Dependency Direction**: Dependencies ONLY flow inward (outer → inner).

### 1. Domain Layer (`pkg/domain/`)
- Pure Go stdlib only - NO external libraries
- NO imports from application/infrastructure/presentation
- Contains: entities, value objects, domain services, repository interfaces
- Test: Pure unit tests, no mocks needed (95%+ coverage target)

### 2. Application Layer (`pkg/application/`)
- Imports: domain layer ONLY
- Defines port interfaces (input/output)
- Contains: use cases, DTOs, orchestration logic
- Test: Mock all ports with gomock (90%+ coverage target)

### 3. Infrastructure Layer (`pkg/infrastructure/`)
- Implements domain/application interfaces
- External libraries allowed here
- Contains: adapters (Homebrew, asdf, npm), repositories, executors
- Test: Docker-based integration tests (85%+ coverage target)

### 4. Presentation Layer (`cmd/pm/`)
- Imports: application + infrastructure (for DI only)
- Contains: CLI commands, input validation, output formatting
- Thin layer - delegates to application use cases

## Validating Architecture Compliance

```bash
# Check domain layer has no forbidden imports
go list -test -deps ./pkg/domain/... | grep -E "(application|infrastructure|cmd)" && echo "VIOLATION" || echo "OK"

# Check application layer doesn't import infrastructure
go list -test -deps ./pkg/application/... | grep -E "(infrastructure|cmd)" && echo "VIOLATION" || echo "OK"

# Check file sizes (should be < 500 lines / ~10KB)
find pkg cmd -name "*.go" -exec wc -l {} \; | awk '$1 > 500 {print $2 ": " $1 " lines (TOO LARGE)"}'
```

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
- **Consumer minimum**: Go 1.24.0+ (ADR-004)
- **Recommended development toolchain**: Go 1.26.7 (`toolchain go1.26.7`)
- **Compatibility evidence**: CI builds, tests, and vets with Go 1.24.11 using
  `GOTOOLCHAIN=local` and `-mod=readonly`

## Manual Dependency Injection Pattern

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

## Common Pitfalls to Avoid

1. ❌ Importing infrastructure in domain layer
2. ❌ Bypassing use cases (CLI → infrastructure directly)
3. ❌ Large God files (> 500 lines)
4. ❌ External libraries in domain layer
5. ❌ CGO dependencies
6. ❌ Mocking in domain layer tests (should be pure functions)

## Project-Specific Patterns

- **Use case naming**: `{Action}{Entity}UseCase` (e.g., `UpdateAllManagersUseCase`)
- **Repository pattern**: Interface in domain, implementation in infrastructure
- **Adapter pattern**: All package managers implement `ManagerAdapter` interface
- **Strategy pattern**: Update strategies in domain layer (stable, latest, minor, fixed)
- **Port interfaces**: Defined in application layer, implemented in infrastructure
