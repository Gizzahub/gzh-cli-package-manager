# ADR-002: Clean Architecture Adoption

**Date**: 2025-01-27
**Status**: Accepted
**Deciders**: Project Team
**Related**: ADR-001 (Standalone Extraction), ADR-003 (Hexagonal Ports)

---

## Context

We need to decide on the architectural pattern for gzh-cli-package-manager. The original gzh-cli PM code (~11,662 lines) uses a relatively flat structure with some separation of concerns, but lacks formal layering.

**Project Goals** (from PRD):
- 5+ year lifespan
- 90%+ test coverage
- Easy addition of new package managers
- Team collaboration (multiple developers)
- Long-term maintainability

**Architectural Options**:
1. Keep flat structure (minimal refactoring)
2. Adopt Clean Architecture with defined layers
3. Use microservices-style architecture
4. Event-driven architecture

---

## Decision

**Adopt Clean Architecture with 4 distinct layers: Domain, Application, Infrastructure, and Presentation.**

**Layer Structure**:
```
Presentation (cmd/gz-pm)
    ↓
Application (pkg/application)
    ↓
Domain (pkg/domain)
    ↑
Infrastructure (pkg/infrastructure)
```

**Dependency Rule**: Dependencies flow inward only. Outer layers depend on inner layers, never the reverse.

---

## Rationale

### Why Clean Architecture?

**1. Long-Term Maintainability**
- Clear boundaries make code easier to understand years later
- Changes in one layer don't ripple to others
- Onboarding new developers faster (predictable structure)

**2. Testability**
- Domain layer: pure functions, no mocks needed (95%+ coverage achievable)
- Application layer: mock all ports (90%+ coverage)
- Infrastructure layer: Docker-based integration tests (85%+ coverage)
- **Overall target: 90%+ maintained for years**

**3. Flexibility**
- Swap CLI for GUI without touching business logic
- Change from YAML to Database without affecting domain
- Add new package managers as plugins (adapter pattern)

**4. Team Collaboration**
- Clear ownership: domain experts own domain layer, infra experts own adapters
- Parallel development: layers can evolve independently
- Code reviews easier: reviewers focus on relevant layer

**5. Business Logic Protection**
- Domain layer has zero external dependencies
- Update strategies, version logic are pure functions
- Easy to verify correctness (no mocking frameworks needed)

### Why Not Other Patterns?

**Flat Structure (Alternative 1)**
- **Rejected**: Hard to maintain at 11,662 lines
- **Reason**: No clear boundaries → accidental coupling increases over time
- **Risk**: "Big ball of mud" anti-pattern after 2-3 years

**Microservices (Alternative 2)**
- **Rejected**: Overkill for CLI tool
- **Reason**: Adds network latency, deployment complexity
- **When to reconsider**: If we build hosted service (v2.0+)

**Event-Driven (Alternative 3)**
- **Rejected**: Unnecessary complexity
- **Reason**: Package updates are sequential operations, not async events
- **When to use**: If we add real-time notifications (future)

---

## Consequences

### Positive ✅

1. **Predictable Structure**
   - New developers know where code belongs
   - "Where does version comparison logic go?" → Domain layer
   - "Where does Homebrew API call go?" → Infrastructure layer

2. **Independent Testing**
   - Domain: `go test ./pkg/domain/... -v` (no setup needed)
   - Application: Mock all ports, test orchestration logic
   - Infrastructure: Docker containers with real package managers
   - Each layer's tests run in < 30 seconds

3. **Technology Independence**
   - Not locked into Cobra (could use  Bubble Tea TUI)
   - Not locked into YAML (could use SQLite, etcd)
   - Not locked into Go stdlib (could use external libs in infra)

4. **Plugin Architecture**
   - Adding Pacman: Create `infrastructure/adapter/manager/pacman`
   - Implement `ManagerAdapter` interface
   - Register in factory
   - **Estimated time: < 1 day per new manager**

5. **Clear Review Guidelines**
   - Domain PR: Check business logic correctness
   - Application PR: Check workflow orchestration
   - Infrastructure PR: Check external integration
   - Presentation PR: Check UX and input validation

### Negative ❌

1. **Initial Complexity**
   - More files than flat structure (~50 files vs ~20)
   - Learning curve for Clean Architecture concepts
   - Junior developers may struggle initially

   **Mitigation**:
   - Comprehensive ARCHITECTURE.md documentation
   - Code examples for each layer
   - Onboarding guide with video walkthrough

2. **Boilerplate Code**
   - Port interfaces need definition
   - DTOs for cross-layer communication
   - Adapter implementations for each manager

   **Mitigation**:
   - Code generators for adapters (`make generate-adapter name=pacman`)
   - Templates in `docs/development/patterns.md`
   - Acceptable trade-off for long-term maintainability

3. **Indirection Overhead**
   - More function calls between layers
   - Slight performance impact (measurable but not noticeable)

   **Mitigation**:
   - Performance tests in CI (must be within 10% of baseline)
   - Go compiler optimizes away most indirection
   - Real bottleneck is network/disk I/O, not CPU

4. **Potential Over-Engineering**
   - Risk of creating unnecessary abstractions
   - "Analysis paralysis" when deciding layer boundaries

   **Mitigation**:
   - Follow established patterns from DDD/Clean Architecture literature
   - YAGNI principle: don't create abstractions until needed
   - Code reviews catch over-abstraction

### Neutral 🔄

1. **Learning Investment**
   - Team must learn Clean Architecture principles
   - Initial development slower (weeks 3-4)
   - Pays off in weeks 5+ with faster feature development

2. **File Organization**
   - More directories to navigate
   - IDE helps (jump to definition)
   - Tree view shows structure clearly

---

## Implementation Guidelines

### Layer Definitions

**Domain Layer** (`pkg/domain/`):
- **Contains**: Entities, Value Objects, Domain Services, Repository Interfaces
- **Dependencies**: Go stdlib only (no external libraries)
- **Example**: `Manager`, `Package`, `UpdateStrategy`
- **Rule**: No `import` statements to other layers

**Application Layer** (`pkg/application/`):
- **Contains**: Use Cases, DTOs, Input/Output Ports
- **Dependencies**: Domain layer, port interfaces
- **Example**: `UpdateAllManagersUseCase`
- **Rule**: No infrastructure imports (use ports)

**Infrastructure Layer** (`pkg/infrastructure/`):
- **Contains**: Adapters, Repository Implementations, External Integrations
- **Dependencies**: Domain (interfaces), Application (ports), External libraries
- **Example**: `HomebrewAdapter`, `YAMLConfigRepository`
- **Rule**: Implements domain and application interfaces

**Presentation Layer** (`cmd/gz-pm/`):
- **Contains**: CLI Commands, Input Validation, Output Formatting
- **Dependencies**: Application (use cases), Infrastructure (for DI)
- **Example**: `UpdateCommand`, `TextFormatter`
- **Rule**: Thin layer, delegates to application

### Dependency Injection

Manual DI in `main.go` (no DI frameworks):

```go
func main() {
    // 1. Infrastructure
    logger := logger.NewStructuredLogger("gz-pm")
    executor := executor.NewShellExecutor(logger)

    // 2. Repositories
    managerRepo := repository.NewManagerRepository(executor, logger)

    // 3. Use Cases
    updateUC := update.NewUpdateAllManagersUseCase(
        managerRepo,
        executor,
        logger,
        update.NewStableStrategy(),
    )

    // 4. CLI
    rootCmd := command.NewRootCommand()
    rootCmd.AddCommand(command.NewUpdateCommand(updateUC))
    rootCmd.Execute()
}
```

**Why Manual DI?**
- Explicit, understandable
- No magic/reflection
- Compile-time safety
- Easy to debug

### File Naming Conventions

- Entities: `entity.go`, `manager.go`, `package.go`
- Repository interfaces: `repository.go`
- Repository implementations: `{type}_repository.go` (e.g., `yaml_repository.go`)
- Adapters: `adapter.go`, `{manager}_adapter.go`
- Use cases: `{action}_{entity}.go` (e.g., `update_all_managers.go`)
- Tests: `{file}_test.go` alongside source

### Testing Strategy Per Layer

**Domain Layer**:
```go
// No setup, pure functions
func TestStableStrategy_SelectVersion(t *testing.T) {
    strategy := NewStableStrategy()
    result, err := strategy.SelectVersion("1.0.0", []string{"1.1.0", "2.0.0-beta"})
    assert.Equal(t, "1.1.0", result)
}
```

**Application Layer**:
```go
// Mock all ports
func TestUpdateAllUseCase(t *testing.T) {
    mockRepo := mock_port.NewMockManagerRepository(ctrl)
    mockRepo.EXPECT().FindInstalled(gomock.Any()).Return(managers, nil)

    uc := NewUpdateAllManagersUseCase(mockRepo, mockExec, mockLogger, strategy)
    resp, err := uc.Execute(ctx, req)
    // assertions
}
```

**Infrastructure Layer**:
```go
// Docker-based integration tests
func TestHomebrewAdapter_Update(t *testing.T) {
    container := setupHomebrewContainer(t)
    defer container.Terminate()

    adapter := homebrew.NewAdapter(executor, logger)
    result, err := adapter.Update(ctx, opts)
    // assertions on real Homebrew output
}
```

---

## Validation

### Architectural Conformance

**Dependency Check**:
```bash
# Domain layer must not import other layers
go list -test -deps ./pkg/domain/... | grep -E "(application|infrastructure|cmd)" && echo "VIOLATION" || echo "OK"

# Application layer must not import infrastructure
go list -test -deps ./pkg/application/... | grep -E "(infrastructure|cmd)" && echo "VIOLATION" || echo "OK"
```

**File Size Check** (LLM-Friendly):
```bash
# No file should exceed 500 lines
find pkg cmd -name "*.go" -exec wc -l {} \; | awk '$1 > 500 {print $2 ": " $1 " lines (TOO LARGE)"}'
```

**Cyclomatic Complexity**:
```bash
gocyclo -over 15 pkg cmd
# Should return empty (all functions < 15 complexity)
```

### Success Criteria

- [ ] All layer dependency rules followed (validation scripts pass)
- [ ] 90%+ test coverage achieved
- [ ] Each layer testable independently
- [ ] Adding new package manager takes < 1 day
- [ ] New developers productive within 1 week

---

## Migration from Flat Structure

**Original gzh-cli Structure**:
```
cmd/pm/
  update/
    update.go (600 lines, all logic mixed)
internal/pm/
  bootstrap/
    manager.go (800 lines, manager + orchestration + execution)
```

**New Clean Architecture**:
```
pkg/domain/
  manager/
    entity.go (100 lines, pure entities)
  update/
    strategy.go (150 lines, pure business logic)

pkg/application/
  update/
    update_all.go (200 lines, orchestration only)

pkg/infrastructure/
  adapter/manager/homebrew/
    adapter.go (300 lines, Homebrew-specific)

cmd/gz-pm/
  command/
    update.go (100 lines, CLI only)
```

**Benefits**:
- 600-line God file → 5 files of 100-300 lines each
- Clear responsibilities
- Each piece independently testable

---

## Alternatives Considered

### Alternative 1: Keep Flat Structure
**Why Rejected**:
- Technical debt accumulates quickly in flat structures
- Testing becomes harder as coupling increases
- Violates project's 5+ year lifespan goal

### Alternative 2: Hexagonal Architecture Only (No Explicit Layers)
**Why Rejected**:
- Hexagonal is good for ports/adapters but doesn't enforce layering
- Clean Architecture provides both layering AND hexagonal
- Adopted Hexagonal as complementary pattern (see ADR-003)

### Alternative 3: DDD Tactical Patterns (Aggregates, Entities, Value Objects)
**Why Partially Adopted**:
- Using Entities and Value Objects from DDD
- Not using Aggregates (overkill for this domain)
- Not using Domain Events (YAGNI for v1.0)

---

## References

- **Clean Architecture** by Robert C. Martin (Uncle Bob)
- **Domain-Driven Design** by Eric Evans
- **Implementing Domain-Driven Design** by Vaughn Vernon
- **ARCHITECTURE.md**: `/ARCHITECTURE.md` - Full architecture documentation
- **Go Clean Architecture Example**: https://github.com/bxcodec/go-clean-arch

---

## Revision History

| Date | Changes | Author |
|------|---------|--------|
| 2025-01-27 | Initial version | Claude Code |

---

## Status Notes

**Current Status**: Accepted and being implemented

**Next Review**: After Week 8 (validation phase) - evaluate if architecture meets 90%+ test coverage goal and < 1 day per new manager target.
