# ADR-003: Hexagonal Architecture (Ports & Adapters)

**Date**: 2025-01-27
**Status**: Accepted
**Deciders**: Project Team
**Related**: ADR-002 (Clean Architecture)

---

## Context

Clean Architecture (ADR-002) provides layering, but we need a pattern for how layers interact, especially:
- How application layer calls external systems (Homebrew, ASDF, file system)
- How to make external dependencies swappable
- How to test business logic without external dependencies

**Design Goal**: Application core should be independent of:
- CLI frameworks (Cobra)
- Package manager implementations (Homebrew, APT)
- File system and OS specifics
- Network I/O

---

## Decision

**Adopt Hexagonal Architecture (Ports & Adapters pattern) to complement Clean Architecture layers.**

**Key Concepts**:
- **Ports**: Interfaces defining how application interacts with external world
- **Adapters**: Implementations of ports for specific technologies
- **Primary/Driving Adapters**: Trigger application (CLI, future REST API)
- **Secondary/Driven Adapters**: Used by application (Database, External APIs)

**Structure**:
```
          Primary Adapters
        (CLI, REST API, GUI)
                 ↓
        ┌─────────────────┐
        │  Input Ports    │ (Use Case Interfaces)
        └────────┬────────┘
                 ↓
        ┌─────────────────┐
        │ Application Core│ (Use Cases + Domain)
        │  Business Logic │
        └────────┬────────┘
                 ↓
        ┌─────────────────┐
        │  Output Ports   │ (Repository, Executor Interfaces)
        └────────┬────────┘
                 ↓
          Secondary Adapters
    (Homebrew, YAML, Shell Executor)
```

---

## Rationale

### Why Hexagonal Architecture?

**1. Testability Without Mocks**
- Application core has no knowledge of external systems
- Tests use simple in-memory adapters
- Example: Test update logic without actual Homebrew installation

```go
// Test with mock adapter
mockAdapter := &MockBrewAdapter{
    packages: []Package{
        {Name: "node", Version: "20.11.0"},
    },
}
// Use in tests without Docker
```

**2. Swappable Implementations**
- Replace Cobra with Bubble Tea TUI → Change primary adapter only
- Replace YAML with SQLite → Change secondary adapter only
- Application core unchanged

**3. Delayed Technology Decisions**
- Build core first with in-memory adapters
- Choose real implementations later
- Example: Started with in-memory config, added YAML later

**4. Clear Boundaries**
- Port interfaces document application's needs
- Adapters isolated to infrastructure layer
- Easy to find code: "Where's the Homebrew logic?" → `infrastructure/adapter/manager/homebrew/`

**5. Plugin Architecture**
- New package managers → Implement `ManagerAdapter` port
- Register in factory → No core changes
- Community can contribute adapters

---

## Consequences

### Positive ✅

1. **Framework Independence**
   - Not locked into Cobra
   - Could build web UI using same core
   - Example migration path:
     ```go
     // Before: CLI-specific
     cobra.Command → call use case

     // After: Framework-agnostic
     HTTP handler → call use case
     TUI screen → call use case
     ```

2. **Test Isolation**
   - Unit tests: in-memory adapters (fast)
   - Integration tests: real adapters in Docker (comprehensive)
   - E2E tests: full system with real environment
   - **Each test level independent**

3. **Parallel Development**
   - Team A: Build core with mock adapters
   - Team B: Implement Homebrew adapter
   - Team C: Implement ASDF adapter
   - **No blocking dependencies**

4. **Explicit Dependencies**
   - Port interfaces document what application needs
   - Example: `CommandExecutor` port shows we need shell execution
   - New developers understand dependencies from interfaces

5. **Future-Proof**
   - Technology changes contained to adapters
   - Example: Homebrew v5 → Update adapter only
   - Core business logic unaffected

### Negative ❌

1. **Interface Proliferation**
   - Many small interfaces (one per external dependency)
   - Can feel like boilerplate
   - Example: `Logger`, `Executor`, `Notifier`, `Repository`

   **Mitigation**:
   - Interface Segregation Principle (small, focused interfaces)
   - Generated documentation shows all ports
   - Worth it for testability gains

2. **Indirection Layers**
   - Port → Adapter → External system (3 hops)
   - Slight performance overhead
   - Example: Use Case → Port → Adapter → Homebrew CLI

   **Mitigation**:
   - Performance impact negligible (I/O is bottleneck)
   - Compiler optimizes indirection
   - Clarity > minimal performance cost

3. **Learning Curve**
   - Developers must understand ports vs adapters
   - "Where do I add code?" questions initially
   - Example confusion: "Is this a port or adapter?"

   **Mitigation**:
   - Clear naming: `*Port` vs `*Adapter`
   - Documentation with examples
   - Code review catches misplacements

### Neutral 🔄

1. **Port Interface Design**
   - Must get interfaces right upfront
   - Changes affect multiple adapters
   - Example: Adding parameter to `Execute()` breaks all adapters

   **Approach**: Start with simple interfaces, evolve carefully

2. **Adapter Implementations**
   - Each adapter is independent
   - Code duplication across adapters acceptable
   - Example: Both Homebrew and APT parse version strings

   **Trade-off**: Duplication > wrong abstraction

---

## Implementation

### Port Definitions

**Input Ports** (Application → Presentation):
```go
// pkg/application/port/input/update_port.go
package input

// Primary/Driving port (how presentation calls application)
type UpdateUseCase interface {
    Execute(ctx context.Context, req *dto.UpdateAllRequest) (*dto.UpdateAllResponse, error)
}
```

**Output Ports** (Application → Infrastructure):
```go
// pkg/application/port/output/executor_port.go
package output

// Secondary/Driven port (application needs this from infrastructure)
type CommandExecutor interface {
    Execute(ctx context.Context, cmd string, args ...string) (*ExecutionResult, error)
}

// pkg/application/port/output/repository_port.go
type ManagerRepository interface {
    FindInstalled(ctx context.Context) ([]*domain.Manager, error)
    Save(ctx context.Context, m *domain.Manager) error
}
```

### Adapter Implementations

**Primary Adapter** (CLI → Application):
```go
// cmd/pm/command/update.go
package command

func NewUpdateCommand(updateUC port.UpdateUseCase) *cobra.Command {
    return &cobra.Command{
        Use: "update",
        RunE: func(cmd *cobra.Command, args []string) error {
            // Translate CLI input to DTO
            req := buildRequest(cmd.Flags())

            // Call use case through port
            resp, err := updateUC.Execute(cmd.Context(), req)

            // Format output
            return formatResponse(resp)
        },
    }
}
```

**Secondary Adapter** (Application → Homebrew):
```go
// pkg/infrastructure/adapter/manager/homebrew/adapter.go
package homebrew

// Implements ManagerAdapter port
type HomebrewAdapter struct {
    executor port.CommandExecutor  // Uses another port
    logger   port.Logger
}

func (a *HomebrewAdapter) Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
    // Execute brew update && brew upgrade
    result, err := a.executor.Execute(ctx, "brew", "update")
    // Parse output, return domain entities
}
```

### Dependency Injection (Wire Adapters to Ports)

```go
// cmd/pm/main.go
func main() {
    // Create adapters (infrastructure)
    logger := logger.NewStructuredLogger("gz-pm")
    executor := executor.NewShellExecutor(logger)  // CommandExecutor adapter
    homebrewAdapter := homebrew.NewAdapter(executor, logger)  // ManagerAdapter
    managerRepo := repository.NewManagerRepository(executor, logger)  // Repository adapter

    // Create use cases (application) - inject ports
    updateUC := update.NewUpdateAllManagersUseCase(
        managerRepo,  // ManagerRepository port
        executor,     // CommandExecutor port
        logger,       // Logger port
        update.NewStableStrategy(),
    )

    // Create commands (presentation) - inject use cases
    rootCmd := command.NewRootCommand()
    rootCmd.AddCommand(command.NewUpdateCommand(updateUC))  // UpdateUseCase port

    rootCmd.Execute()
}
```

**Flow**:
1. CLI Command (primary adapter) calls UpdateUseCase (input port)
2. UpdateUseCase calls ManagerRepository (output port)
3. ManagerRepository calls ManagerAdapter implementations
4. ManagerAdapter calls CommandExecutor (output port)
5. Shell executor (secondary adapter) runs actual commands

---

## Port Design Principles

### Interface Segregation

**BAD** - One big interface:
```go
type PackageManager interface {
    Detect() bool
    Update() error
    Install(pkg string) error
    Remove(pkg string) error
    Search(query string) []Package
    Export() Config
    Import(cfg Config) error
    // ... 20 more methods
}
```

**GOOD** - Small, focused interfaces:
```go
type ManagerDetector interface {
    Detect(ctx context.Context) (bool, error)
}

type ManagerUpdater interface {
    Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
}

type ManagerExporter interface {
    Export(ctx context.Context) (*Config, error)
}
```

**Benefit**: Adapters implement only what they need.

### Port Naming Conventions

- Ports: `{Capability}Port` or just `{Capability}` (e.g., `ExecutorPort`, `Repository`)
- Adapters: `{Technology}Adapter` (e.g., `HomebrewAdapter`, `YAMLRepository`)
- Input ports: Verb-based (`UpdateUseCase`, `BootstrapUseCase`)
- Output ports: Noun-based (`ManagerRepository`, `CommandExecutor`)

---

## Testing with Ports & Adapters

### Unit Tests (Application Layer)

```go
func TestUpdateAllUseCase(t *testing.T) {
    // Use mock adapters (simple implementations)
    mockRepo := &InMemoryManagerRepository{
        managers: []*domain.Manager{
            {ID: "brew", Installed: true},
        },
    }

    mockExecutor := &MockCommandExecutor{
        responses: map[string]*ExecutionResult{
            "brew update": {ExitCode: 0, Stdout: "Updated"},
        },
    }

    // Test use case with mocks
    uc := NewUpdateAllManagersUseCase(mockRepo, mockExecutor, nullLogger, strategy)
    resp, err := uc.Execute(ctx, &dto.UpdateAllRequest{All: true})

    // Assertions
    assert.NoError(t, err)
    assert.Equal(t, 1, len(resp.Results))
}
```

**Benefits**:
- No external dependencies
- Fast (< 1ms per test)
- Deterministic

### Integration Tests (Infrastructure Layer)

```go
func TestHomebrewAdapter_Update(t *testing.T) {
    // Use real adapter with real external system (Docker)
    container := setupHomebrewContainer(t)
    defer container.Terminate()

    executor := executor.NewShellExecutor(logger)
    adapter := homebrew.NewAdapter(executor, logger)

    // Test against real Homebrew
    result, err := adapter.Update(ctx, UpdateOptions{DryRun: true})

    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

**Benefits**:
- Validates actual integration
- Catches API changes
- Confidence in production behavior

---

## Examples of Ports

### Current Ports (v1.0)

**Input Ports** (Use Case Interfaces):
- `UpdateUseCase`
- `BootstrapUseCase`
- `SyncUseCase`
- `ExportUseCase`
- `StatusUseCase`

**Output Ports** (Dependency Interfaces):
- `CommandExecutor` - Execute shell commands
- `ManagerRepository` - Manager CRUD operations
- `ConfigurationRepository` - Config load/save
- `Logger` - Structured logging
- `Notifier` - Send notifications (future)

### Future Ports (v1.1+)

- `EventPublisher` - Publish domain events
- `CacheStore` - Caching layer
- `MetricsCollector` - Usage metrics
- `SecretsManager` - Secure token storage

---

## Validation

### Architecture Conformance

**Check port usage**:
```bash
# Application layer should only import port interfaces
grep -r "import.*infrastructure" pkg/application/ && echo "VIOLATION" || echo "OK"

# Count ports
find pkg/application/port -name "*.go" | wc -l
# Should be ~10 port files for v1.0
```

### Success Criteria

- [ ] All external dependencies accessed through ports
- [ ] Application layer has zero infrastructure imports
- [ ] Each adapter implements at least one port
- [ ] Unit tests use mock adapters only
- [ ] Integration tests use real adapters
- [ ] Adding new package manager: < 1 day (adapter implementation)

---

## Alternatives Considered

### Alternative 1: Direct Dependencies (No Ports)

```go
// BAD: Application directly uses infrastructure
func (uc *UpdateUseCase) Execute() {
    homebrewAdapter := homebrew.NewAdapter()  // Direct coupling
    homebrewAdapter.Update()
}
```

**Rejected**: Hard to test, tight coupling, breaks Clean Architecture

### Alternative 2: Dependency Injection Framework (Wire, Fx)

**Rejected for v1.0**: Manual DI is simpler, more explicit, easier to debug

**Reconsider**: If DI wiring becomes complex (>50 dependencies)

### Alternative 3: Service Locator Pattern

**Rejected**: Anti-pattern, hides dependencies, hard to test

---

## References

- **Hexagonal Architecture** by Alistair Cockburn
- **Ports and Adapters Pattern**: https://alistair.cockburn.us/hexagonal-architecture/
- **Growing Object-Oriented Software, Guided by Tests** by Steve Freeman & Nat Pryce
- **ADR-002**: Clean Architecture provides the layering
- **ARCHITECTURE.md**: `/ARCHITECTURE.md` - Implementation details

---

## Revision History

| Date | Changes | Author |
|------|---------|--------|
| 2025-01-27 | Initial version | Claude Code |

---

## Status Notes

**Current Status**: Accepted, being implemented alongside ADR-002

**Next Review**: Week 5 (after application layer implementation) - evaluate if port design is clean and adapters are straightforward to implement.
