# Common Tasks - gzh-cli-package-manager

## Adding a New Package Manager Adapter

This is a common task. Follow this pattern:

### 1. Create adapter package
`pkg/infrastructure/adapter/manager/newmanager/`

### 2. Implement ManagerAdapter interface

```go
type ManagerAdapter interface {
    Detect(ctx context.Context) (bool, error)
    GetVersion(ctx context.Context) (string, error)
    Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
    ListInstalled(ctx context.Context) ([]Package, error)
    // ... other methods
}
```

### 3. Register in factory

```go
// cmd/pm/main.go
adapterFactory.Register("newmanager", adapter)
```

### 4. Write tests
- Unit tests: `adapter_test.go` (mock executor)
- Integration tests: `adapter_integration_test.go` (Docker container)

### 5. Target
< 1 day per new manager

## Implementing a New Use Case

### Step-by-step workflow

1. **Define domain entities** (`pkg/domain/`) - pure business logic
2. **Create use case** (`pkg/application/`) - orchestration
3. **Define ports** (`pkg/application/port/`) - interfaces
4. **Implement adapters** (`pkg/infrastructure/`) - external integrations
5. **Add CLI command** (`cmd/pm/command/`) - user interface
6. **Wire dependencies** (`cmd/pm/main.go`) - manual DI

## Before Making Changes

### 1. Read architecture docs
- `ARCHITECTURE.md`
- Relevant ADRs in `docs/architecture/adr/`

### 2. Understand layer boundaries
- Where does this change belong?

### 3. Check existing patterns
- How are similar features implemented?

### 4. Verify test coverage
- Will your changes maintain 90%+ coverage?

## During Implementation

1. **Follow architecture rules**: Respect dependency directions
2. **Keep files small**: Target < 300 lines per file
3. **Write tests first** (TDD for domain/application layers)
4. **No CGO dependencies**: Pure Go only

## Before Committing

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

## Important Files

- `ARCHITECTURE.md` - Full architecture documentation
- `CONTRIBUTING.md` - Development guidelines
- `PRD.md` - Product vision and roadmap
- `REQUIREMENTS.md` - Functional/non-functional requirements
- `docs/architecture/adr/` - Architecture Decision Records
- `docs/specifications/` - Use case specifications
- `Makefile` - Build automation (authoritative command reference)
