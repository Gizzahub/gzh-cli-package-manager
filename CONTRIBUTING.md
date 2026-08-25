# Contributing to gzh-cli-package-manager (gz-pm)

Welcome! We're excited that you're interested in contributing to gz-pm. This guide will help you get started with development, understand our architecture, and contribute effectively.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Architecture Overview](#architecture-overview)
- [Coding Standards](#coding-standards)
- [Common Patterns and Best Practices](#common-patterns-and-best-practices)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Documentation](#documentation)
- [Release Process](#release-process)

---

## Code of Conduct

This project follows standard open-source community guidelines:

- **Be respectful** - Treat all contributors with respect
- **Be constructive** - Provide helpful feedback and criticism
- **Be collaborative** - Work together to improve the project
- **Be patient** - Remember that everyone was a beginner once

---

## Getting Started

### Prerequisites

- **Go 1.24.0+** (required - see ADR-004); Go 1.26.7 is recommended for development
- **Git** for version control
- **Make** for build automation
- **Docker** (optional, for integration tests)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/gizzahub/gzh-cli-package-manager.git
cd gzh-cli-package-manager

# Install dependencies
go mod download

# Build the project
make build

# Run tests
make test

# Install locally
make install
```

---

## Development Setup

### 1. Install the recommended Go 1.26.7 development toolchain

**macOS (Homebrew)**:
```bash
brew install go@1.26
```

**Linux**:
```bash
wget https://go.dev/dl/go1.26.7.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.7.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

**Verify Installation**:
```bash
go version  # Should show go1.26.7
```

Go 1.24.0+ remains the consumer minimum. Go 1.24.11 is exercised only by the
CI compatibility job; it is not the recommended local development installation.

### 2. Set Up Your Environment

```bash
# Set GOPATH (if not already set)
export GOPATH=$HOME/go
export PATH=$PATH:$GOPATH/bin

# Enable Go modules (default in Go 1.16+)
export GO111MODULE=on

# Disable CGO (required - see ADR-006)
export CGO_ENABLED=0
```

### 3. Install Development Tools

```bash
# Linting and formatting
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.1

# Testing tools
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install gotest.tools/gotestsum@latest

# Code coverage
go install github.com/axw/gocov/gocov@latest
go install github.com/AlekSi/gocov-xml@latest
```

### 4. Clone and Build

```bash
git clone https://github.com/gizzahub/gzh-cli-package-manager.git
cd gzh-cli-package-manager

# Confirm the selected Go toolchain
go version

# Download dependencies
go mod download

# Build project
make build

# Verify build
./build/gz-pm version
```

### 5. Run Tests

```bash
# Unit tests only
make test-unit

# Integration tests (requires Docker)
make test-integration

# All tests with coverage
make test-coverage

# Specific package tests
go test -v ./pkg/domain/manager/...
```

---

## Architecture Overview

gz-pm follows **Clean Architecture** with **Hexagonal (Ports & Adapters)** pattern.

### Layer Structure

```
┌─────────────────────────────────────────────┐
│  Presentation Layer (cmd/gz-pm)             │
│  - CLI commands (Cobra)                     │
│  - Input validation                         │
│  - Output formatting                        │
└──────────────┬──────────────────────────────┘
               │ depends on ↓
┌──────────────▼──────────────────────────────┐
│  Application Layer (pkg/application)        │
│  - Use cases (business workflows)           │
│  - DTOs (data transfer objects)             │
│  - Input/Output ports (interfaces)          │
└──────────────┬──────────────────────────────┘
               │ depends on ↓
┌──────────────▼──────────────────────────────┐
│  Domain Layer (pkg/domain)                  │
│  - Entities (Manager, Package, Config)      │
│  - Value Objects                            │
│  - Domain Services                          │
│  - Repository Interfaces                    │
└──────────────▲──────────────────────────────┘
               │ implements ↑
┌──────────────┴──────────────────────────────┐
│  Infrastructure Layer (pkg/infrastructure)  │
│  - Adapters (Homebrew, ASDF, npm, pip, etc)│
│  - Repositories (YAML, SQLite)              │
│  - External integrations                    │
└─────────────────────────────────────────────┘
```

### Key Architectural Rules

1. **Dependency Rule**: Dependencies only point inward
   - Presentation → Application → Domain
   - Infrastructure → Domain (implements interfaces)
   - **Never**: Domain → Application or Infrastructure

2. **Pure Domain Layer**: No external dependencies
   - Only Go stdlib imports allowed
   - Pure functions, testable without mocks

3. **Interface-Based Design**: All external dependencies via ports
   - Domain defines repository interfaces
   - Application defines executor, logger ports
   - Infrastructure implements ports

4. **No CGO**: Pure Go only (see ADR-006)
   - Cross-compilation support
   - Static binaries
   - Simple builds

### File Naming Conventions

- **Entities**: `manager.go`, `package.go`, `config.go`
- **Repositories**: `repository.go` (interface), `yaml_repository.go` (implementation)
- **Adapters**: `adapter.go`, `homebrew_adapter.go`
- **Use Cases**: `update_all_managers.go`, `bootstrap_system.go`
- **Tests**: `*_test.go` (alongside source files)

### Package Organization

```
pkg/
├── domain/               # Core business logic
│   ├── manager/         # Manager entity and repository interface
│   ├── package/         # Package entity
│   ├── config/          # Configuration entity
│   └── update/          # Update strategies (domain services)
│
├── application/         # Use cases and workflows
│   ├── port/           # Input and output port interfaces
│   │   ├── input/      # Use case interfaces
│   │   └── output/     # Dependency interfaces
│   ├── update/         # Update use cases
│   ├── bootstrap/      # Bootstrap use cases
│   └── dto/            # Data transfer objects
│
└── infrastructure/      # External integrations
    ├── adapter/        # Package manager adapters
    │   └── manager/
    │       ├── homebrew/
    │       ├── asdf/
    │       └── npm/
    ├── repository/     # Repository implementations
    │   ├── config/     # Config repositories
    │   └── state/      # State repositories
    └── executor/       # Command execution
```

---

## Coding Standards

### Go Code Style

We follow standard Go conventions with some additional requirements:

#### 1. Formatting

```bash
# Format code with gofmt
go fmt ./...

# OR use goimports (preferred - manages imports)
goimports -w .
```

**Requirements**:
- Use `gofmt` for formatting (enforced in CI)
- Use `goimports` for import management
- Line length: prefer < 100 characters, max 120
- Use tabs for indentation (Go standard)

#### 2. Naming Conventions

```go
// ✅ GOOD - Clear, Go-style naming
type ManagerRepository interface {
    FindInstalled(ctx context.Context) ([]*Manager, error)
    Save(ctx context.Context, m *Manager) error
}

type HomebrewAdapter struct {
    executor port.CommandExecutor
    logger   port.Logger
}

func (a *HomebrewAdapter) Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
    // Implementation
}

// ❌ BAD - Non-Go conventions
type IManagerRepository interface { ... }  // No "I" prefix
type manager_repository struct { ... }     // No snake_case
func UpdateMgr(...) { ... }                 // Unclear abbreviation
```

**Rules**:
- Interfaces: Noun or noun phrase (`ManagerRepository`, `Executor`)
- Structs: Noun (`Manager`, `UpdateResult`)
- Functions/Methods: Verb or verb phrase (`FindInstalled`, `Update`)
- No Hungarian notation, no "I" prefix for interfaces
- Use full words (avoid abbreviations unless well-known: `ID`, `URL`, `HTTP`)

#### 3. Error Handling

```go
// ✅ GOOD - Wrap errors with context
func (uc *UpdateAllUseCase) Execute(ctx context.Context, req *dto.UpdateAllRequest) (*dto.UpdateAllResponse, error) {
    managers, err := uc.managerRepo.FindInstalled(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to find installed managers: %w", err)
    }

    for _, mgr := range managers {
        if err := uc.updateManager(ctx, mgr); err != nil {
            return nil, fmt.Errorf("failed to update %s: %w", mgr.Name, err)
        }
    }

    return &dto.UpdateAllResponse{}, nil
}

// ❌ BAD - Losing error context
func (uc *UpdateAllUseCase) Execute(ctx context.Context, req *dto.UpdateAllRequest) (*dto.UpdateAllResponse, error) {
    managers, err := uc.managerRepo.FindInstalled(ctx)
    if err != nil {
        return nil, err  // Lost context
    }
    // ...
}
```

**Rules**:
- Always wrap errors with `fmt.Errorf("context: %w", err)`
- Use `%w` for error wrapping (Go 1.13+)
- Provide actionable context in error messages
- Don't log and return error (caller decides)

#### 4. Context Usage

```go
// ✅ GOOD - Pass context as first parameter
func (r *YAMLRepository) Load(ctx context.Context) (*Config, error) {
    // Use context for cancellation, deadlines, values
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
        // Continue
    }

    data, err := os.ReadFile(r.configPath)
    // ...
}

// ❌ BAD - No context support
func (r *YAMLRepository) Load() (*Config, error) {
    // No way to cancel long operations
}
```

**Rules**:
- First parameter is always `context.Context`
- Check `ctx.Done()` in long-running operations
- Pass context down the call chain
- Don't store context in structs

#### 5. Interface Segregation

```go
// ✅ GOOD - Small, focused interfaces
type ManagerDetector interface {
    Detect(ctx context.Context) (bool, error)
}

type ManagerUpdater interface {
    Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
}

type ManagerExporter interface {
    Export(ctx context.Context) (*Config, error)
}

// ❌ BAD - Large, monolithic interface
type PackageManager interface {
    Detect(ctx context.Context) (bool, error)
    Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
    Export(ctx context.Context) (*Config, error)
    Install(ctx context.Context, pkg string) error
    Remove(ctx context.Context, pkg string) error
    Search(ctx context.Context, query string) ([]Package, error)
    // ... 20 more methods
}
```

**Rules**:
- Prefer many small interfaces over large ones
- Compose interfaces when needed
- Clients implement only what they need

#### 6. File Size Limits

**From `~/.claude/ctx/FILE_SIZE_LIMITS.md`**:

| File Size | Status | Action Required |
|-----------|--------|-----------------|
| < 10KB | ✅ Ideal | Maintain |
| 10-50KB | ⚠️ Warning | Consider splitting |
| 50-100KB | ⚠️ Problem | Split recommended |
| > 100KB | ❌ Critical | Must split |

**Guidelines**:
- Target: Files under 500 lines (~10KB)
- Max: 1000 lines (rarely justified)
- Split large files by responsibility

### Linting

We use `golangci-lint` with strict settings:

```bash
# Run linter
make lint

# Auto-fix issues where possible
golangci-lint run --fix
```

**Enabled Linters**:
- `gofmt`, `goimports` - Formatting
- `govet` - Suspicious constructs
- `errcheck` - Unchecked errors
- `staticcheck` - Static analysis
- `gosec` - Security issues
- `ineffassign` - Ineffectual assignments
- `misspell` - Spelling errors

**Linter Configuration**: `.golangci.yml`

---

## Common Patterns and Best Practices

This section documents common patterns used in the codebase. Following these patterns ensures consistency and maintainability.

### Shared Utilities

When implementing adapters or use cases, leverage existing shared utilities instead of duplicating code:

#### cmdutil Package (Command Execution Helpers)

Location: `pkg/infrastructure/adapter/manager/cmdutil/`

The `cmdutil` package provides standardized helpers for command execution patterns used across package manager adapters.

**When to use**:
- Executing commands and validating results
- Extracting version information from command output
- Checking command availability (e.g., `which` results)
- Parsing JSON output from commands

**Available functions**:

```go
// Validate command execution result
func CheckResult(result *output.ExecutionResult, err error, operation string) error

// Extract trimmed stdout
func ExtractStdout(result *output.ExecutionResult) string

// Parse version field from output (e.g., "npm 9.0.0" → "9.0.0")
func ExtractVersionField(stdout string, fieldIndex int, operation string) (string, error)

// Check if command is available (interprets which/where results)
func IsCommandAvailable(result *output.ExecutionResult, err error) bool

// Unmarshal JSON output
func UnmarshalJSON(result *output.ExecutionResult, v interface{}, operation string) error
```

**Example usage** (from `pkg/infrastructure/adapter/manager/homebrew/homebrew.go`):

```go
// Before
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
    result, err := a.executor.Execute(ctx, "which", "brew")
    if err != nil {
        return false, nil
    }
    return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) != "", nil
}

// After
func (a *Adapter) Detect(ctx context.Context) (bool, error) {
    result, err := a.executor.Execute(ctx, "which", "brew")
    return cmdutil.IsCommandAvailable(result, err), nil
}
```

**Benefits**:
- Reduces code duplication (~20-30 lines per adapter)
- Standardizes error messages
- Simplifies adapter implementation
- Centralized bug fixes and improvements

#### testutil Package (Test Mocks)

Location: `pkg/infrastructure/adapter/manager/testutil/`

Provides shared mock implementations for testing adapters.

**Available mocks**:
- `MockExecutor`: Mock command executor
- `MockLogger`: Mock logger
- Helper functions: `SuccessResult()`, `FailureResult()`

**Example usage**:

```go
import "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"

func TestAdapter_Detect(t *testing.T) {
    executor := testutil.NewMockExecutor(func(_ context.Context, name string, args ...string) (*output.ExecutionResult, error) {
        return testutil.SuccessResult("/usr/bin/brew"), nil
    })
    logger := testutil.NewMockLogger()

    adapter := NewAdapter(executor, logger)
    // ... rest of test
}
```

**Benefits**:
- Consistent mock behavior across tests
- Reduces test boilerplate (~25 lines per test file)
- Easier test maintenance

### Implementing a New Package Manager Adapter

Follow these steps when adding support for a new package manager:

1. **Create adapter package**: `pkg/infrastructure/adapter/manager/newmanager/`

2. **Implement ManagerAdapter interface** using cmdutil helpers:

```go
package newmanager

import (
    "context"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/application/port/output"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/cmdutil"
)

type Adapter struct {
    executor output.CommandExecutor
    logger   output.Logger
}

func (a *Adapter) Detect(ctx context.Context) (bool, error) {
    result, err := a.executor.Execute(ctx, "which", "newmanager")
    return cmdutil.IsCommandAvailable(result, err), nil
}

func (a *Adapter) GetVersion(ctx context.Context) (string, error) {
    result, err := a.executor.Execute(ctx, "newmanager", "--version")
    if err := cmdutil.CheckResult(result, err, "get version"); err != nil {
        return "", err
    }
    return cmdutil.ExtractStdout(result), nil
}
```

3. **Write comprehensive tests** using testutil:

```go
package newmanager

import (
    "testing"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/infrastructure/adapter/manager/testutil"
)

func TestAdapter_Detect(t *testing.T) {
    executor := testutil.NewMockExecutor(func(...) (*output.ExecutionResult, error) {
        return testutil.SuccessResult("/usr/bin/newmanager"), nil
    })
    logger := testutil.NewMockLogger()

    adapter := NewAdapter(executor, logger)
    detected, err := adapter.Detect(context.Background())

    if err != nil {
        t.Fatalf("Detect() error = %v", err)
    }
    if !detected {
        t.Error("Expected newmanager to be detected")
    }
}
```

4. **Target metrics**:
   - Implementation time: < 1 day
   - Test coverage: > 90%
   - Code reuse: Use cmdutil for standard operations

### Refactoring Guidelines

When refactoring code, follow these principles:

**1. Identify Duplication**
- Look for repeated error handling patterns
- Find similar parsing logic across files
- Identify test mock duplication

**2. Extract to Shared Utilities**
- Create focused, single-purpose functions
- Add comprehensive unit tests (aim for 100%)
- Update existing code to use new utilities

**3. Validate Changes**
- All existing tests must pass
- No decrease in code coverage
- Run `make quality` before committing

**4. Document Patterns**
- Update this section with new patterns
- Add examples for complex utilities
- Reference related ADRs if applicable

**Example refactoring** (Phase 2: cmdutil extraction):
- **Before**: 5 adapters with ~92 lines of duplicated error handling
- **After**: Single `cmdutil` package with 100% test coverage
- **Impact**: Easier maintenance, consistent error messages, faster adapter development

---

## Testing Requirements

### Test Coverage Goals

| Layer | Target | Minimum |
|-------|--------|---------|
| **Domain** | 95% | 90% |
| **Application** | 90% | 85% |
| **Infrastructure** | 85% | 75% |
| **Overall** | 90% | 85% |

### Test Categories

#### 1. Unit Tests

**Domain Layer** (no mocks needed - pure functions):

```go
// pkg/domain/update/strategy_test.go
func TestStableStrategy_SelectVersion(t *testing.T) {
    strategy := NewStableStrategy()

    tests := []struct {
        name     string
        current  string
        versions []string
        want     string
        wantErr  bool
    }{
        {
            name:     "upgrade to latest stable",
            current:  "1.0.0",
            versions: []string{"1.1.0", "2.0.0-beta", "1.2.0"},
            want:     "1.2.0",  // Latest stable, not beta
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := strategy.SelectVersion(tt.current, tt.versions)
            if (err != nil) != tt.wantErr {
                t.Errorf("SelectVersion() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("SelectVersion() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Application Layer** (mock all ports):

```go
// pkg/application/update/update_all_test.go
func TestUpdateAllUseCase_Execute(t *testing.T) {
    // Mock setup
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mock_port.NewMockManagerRepository(ctrl)
    mockExec := mock_port.NewMockCommandExecutor(ctrl)
    mockLogger := mock_port.NewMockLogger(ctrl)

    // Expectations
    mockRepo.EXPECT().
        FindInstalled(gomock.Any()).
        Return([]*domain.Manager{
            {ID: "brew", Installed: true},
        }, nil)

    mockExec.EXPECT().
        Execute(gomock.Any(), "brew", "update").
        Return(&executor.Result{ExitCode: 0}, nil)

    // Test
    uc := NewUpdateAllManagersUseCase(mockRepo, mockExec, mockLogger, strategy)
    resp, err := uc.Execute(context.Background(), &dto.UpdateAllRequest{All: true})

    assert.NoError(t, err)
    assert.Equal(t, 1, len(resp.Results))
}
```

#### 2. Integration Tests

**Infrastructure Layer** (Docker-based real tests):

```go
// pkg/infrastructure/adapter/manager/homebrew/adapter_integration_test.go
// +build integration

func TestHomebrewAdapter_Update_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Setup Docker container with Homebrew
    container := setupHomebrewContainer(t)
    defer container.Terminate()

    // Create real adapter
    executor := executor.NewShellExecutor(logger)
    adapter := homebrew.NewAdapter(executor, logger)

    // Test against real Homebrew
    result, err := adapter.Update(context.Background(), UpdateOptions{DryRun: true})

    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.True(t, result.Success)
}
```

#### 3. End-to-End Tests

```bash
# tests/e2e/update_test.sh
#!/bin/bash
set -euo pipefail

# Build gz-pm
make build

# Test basic update
output=$(./build/gz-pm update --all --dry-run)
echo "$output" | grep -q "Updating package managers"

# Test JSON output
json=$(./build/gz-pm update --all --dry-run --output json)
echo "$json" | jq -e '.summary.total_managers > 0'

echo "✅ E2E tests passed"
```

### Running Tests

```bash
# Unit tests only (fast)
make test-unit

# Integration tests (requires Docker)
make test-integration

# Full implemented suite (dedicated E2E automation is not implemented yet)
go test ./...

# All tests with coverage
make test-coverage

# Specific package
go test -v ./pkg/domain/manager/...

# With race detection
go test -race ./...

# Short mode (skip integration/E2E)
go test -short ./...
```

### Test Organization

```
pkg/domain/manager/
├── entity.go           # Manager entity
├── entity_test.go      # Unit tests (pure domain logic)
└── repository.go       # Repository interface

pkg/application/update/
├── update_all.go       # Use case implementation
└── update_all_test.go  # Unit tests (mocked ports)

pkg/infrastructure/adapter/manager/homebrew/
├── adapter.go                   # Homebrew adapter
├── adapter_test.go              # Unit tests (mocked executor)
└── adapter_integration_test.go  # Integration tests (real Homebrew)
```

### Mocking

We use `gomock` for generating mocks:

```bash
# Generate mocks for interfaces
go generate ./...

# Manual mock generation
mockgen -source=pkg/application/port/output/executor.go \
        -destination=pkg/application/port/output/mock/executor_mock.go \
        -package=mock
```

**Mock Conventions**:
- Place mocks in `mock/` subdirectory
- Name: `<interface>_mock.go`
- Package: `mock`

---

## Pull Request Process

### 1. Before You Start

- **Check existing issues** - Is your feature/fix already discussed?
- **Create an issue** (for new features) - Discuss before implementing
- **Assign yourself** - Let others know you're working on it

### 2. Development Workflow

```bash
# 1. Create feature branch
git checkout -b feature/update-strategy-minor

# 2. Make changes
# Edit files, following coding standards

# 3. Run tests
make test

# 4. Lint code
make lint

# 5. Commit changes (follow Git Commit Guide)
git add .
git commit -m "feat(update): add minor update strategy

Implement minor version update strategy that only updates
patch and minor versions, avoiding major version upgrades.

- Add MinorStrategy domain service
- Update UpdateAllUseCase to support strategy selection
- Add tests for version selection logic

Model: claude-sonnet-4-5-20250929
Co-Authored-By: Claude <noreply@anthropic.com>"

# 6. Push to fork
git push origin feature/update-strategy-minor
```

### 3. Commit Message Format

**Required Format** (from `~/.claude/ctx/GIT_COMMIT_GUIDE.md`):

```
{type}({scope}): {imperative verb} {what}

{body - detailed description}

Model: claude-{model}
Co-Authored-By: Claude <noreply@anthropic.com>
```

**Types**:
- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation only
- `refactor` - Code refactoring
- `test` - Adding tests
- `chore` - Maintenance tasks

**Scopes** (mandatory):
- `domain` - Domain layer changes
- `application` - Application layer
- `infrastructure` - Infrastructure layer
- `cli` - CLI/presentation layer
- `build` - Build system, CI/CD
- `docs` - Documentation
- `test` - Test infrastructure

**Examples**:

```
feat(domain): add update strategy interface

Introduce strategy pattern for version selection, allowing
different update behaviors (latest, stable, minor, fixed).

Model: claude-sonnet-4-5-20250929
Co-Authored-By: Claude <noreply@anthropic.com>
```

```
fix(infrastructure): handle homebrew tap not found error

Add graceful error handling when homebrew tap doesn't exist.
Previously, this caused a panic. Now returns clear error message
with recovery suggestions.

Fixes #123

Model: claude-sonnet-4-5-20250929
Co-Authored-By: Claude <noreply@anthropic.com>
```

### 4. Pull Request Template

When creating a PR, use this template:

```markdown
## Description
<!-- Brief description of changes -->

## Motivation
<!-- Why are these changes needed? What problem do they solve? -->

## Changes
<!-- List key changes -->
- Added...
- Modified...
- Removed...

## Testing
<!-- How were these changes tested? -->
- [ ] Unit tests added/updated
- [ ] Integration tests added/updated
- [ ] Manual testing performed

## Checklist
- [ ] Code follows project coding standards
- [ ] Tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation updated (if applicable)
- [ ] CHANGELOG.md updated (if user-facing change)
- [ ] Commit messages follow format
- [ ] No breaking changes (or documented)

## Related Issues
<!-- Link to related issues -->
Closes #XXX
```

### 5. Review Process

1. **Automated Checks** - CI runs tests, linting, coverage
2. **Code Review** - At least 1 approval required
3. **Architecture Review** - For significant changes
4. **Documentation Review** - For user-facing changes

**Review Guidelines**:
- Be constructive and specific
- Suggest improvements, don't just criticize
- Ask questions to understand intent
- Approve when ready, request changes if needed

### 6. Merging

- **Squash and merge** for feature branches
- **Rebase and merge** for clean linear history (preferred)
- **Update CHANGELOG.md** before merging (if user-facing)

---

## Documentation

### Code Documentation

```go
// Manager represents a package manager installation.
// It contains metadata about the manager's state and configuration.
type Manager struct {
    // ID uniquely identifies the manager (e.g., "brew", "asdf")
    ID ManagerID

    // Name is the human-readable name ("Homebrew", "ASDF")
    Name string

    // Installed indicates whether the manager is currently installed
    Installed bool

    // Version is the currently installed version
    Version string
}

// Update performs a full update of the package manager.
// It returns the result of the update operation or an error if the update fails.
//
// The update process consists of:
//  1. Updating the package manager itself
//  2. Refreshing package databases
//  3. Upgrading managed packages
//
// Example:
//
//	result, err := adapter.Update(ctx, UpdateOptions{DryRun: false})
//	if err != nil {
//	    return fmt.Errorf("update failed: %w", err)
//	}
func (a *HomebrewAdapter) Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
    // Implementation
}
```

**Documentation Requirements**:
- Public types: Describe purpose and usage
- Public functions: Describe behavior, parameters, return values, examples
- Complex logic: Inline comments explaining why (not what)
- Packages: `doc.go` file with package overview

### User Documentation

When adding features, update:

- `README.md` - If it affects getting started
- `docs/specifications/use-cases/` - If new use case
- `docs/specifications/test-scenarios.md` - Add test scenarios
- `CHANGELOG.md` - For all user-facing changes

---

## Release Process

### Versioning

We follow [Semantic Versioning](https://semver.org/):

- **Major** (1.0.0): Breaking changes
- **Minor** (0.1.0): New features, backward-compatible
- **Patch** (0.0.1): Bug fixes, backward-compatible

### Release Checklist

1. **Update Version**:
   ```bash
   # Update version in version.go
   vim internal/version/version.go
   ```

2. **Update CHANGELOG.md**:
   ```markdown
   ## [1.0.0] - 2025-01-27

   ### Added
   - Feature X
   - Feature Y

   ### Changed
   - Behavior Z

   ### Fixed
   - Bug A
   - Bug B
   ```

3. **Create Git Tag**:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```

4. **GitHub Release**:
   - Go to GitHub Releases
   - Create new release from tag
   - Copy CHANGELOG section
   - Attach binaries (from CI)

---

## Getting Help

### Resources

- **Documentation**: `/docs` directory
- **Architecture**: `/ARCHITECTURE.md`
- **ADRs**: `/docs/10-architecture/adr/`
- **Specifications**: `/docs/specifications/`
- **Issues**: GitHub Issues

### Communication

- **Bug Reports**: GitHub Issues
- **Feature Requests**: GitHub Issues with `enhancement` label
- **Questions**: GitHub Discussions
- **Security Issues**: security@gizzahub.com (private)

### Common Issues

**Build Fails**:
```bash
# Check Go version
go version  # Must be 1.24+

# Clean and rebuild
make clean
make build
```

**Tests Fail**:
```bash
# Run with verbose output
go test -v ./pkg/domain/...

# Check test isolation
go test -count=1 ./...  # Disable cache
```

**Linter Errors**:
```bash
# Auto-fix where possible
golangci-lint run --fix

# See specific issues
golangci-lint run --verbose
```

---

## License

By contributing, you agree that your contributions will be licensed under the same license as the project (see `LICENSE` file).

---

## Thank You!

Thank you for contributing to gz-pm! Your efforts help make package management better for everyone.

**Quick Links**:
- [Architecture Overview](ARCHITECTURE.md)
- [Product Requirements](PRD.md)
- [Requirements Specification](REQUIREMENTS.md)
- [Specifications](docs/specifications/)
- [ADRs](docs/10-architecture/adr/)

---

**Document Version**: 1.0
**Last Updated**: 2025-01-27
**Maintainers**: See `CODEOWNERS` file
