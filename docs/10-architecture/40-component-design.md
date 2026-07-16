# 4. Component Design

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

### 4.1 Manager Adapter Pattern

All package manager adapters implement a common interface:

```go
// pkg/infrastructure/adapter/manager/adapter.go
package manager

type ManagerAdapter interface {
    // Detection
    Detect(ctx context.Context) (bool, error)
    GetVersion(ctx context.Context) (string, error)
    IsHealthy(ctx context.Context) (bool, error)

    // Update operations
    Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error)
    ListUpdatable(ctx context.Context) ([]Package, error)

    // Package information
    ListInstalled(ctx context.Context) ([]Package, error)
    GetPackageInfo(ctx context.Context, name string) (*PackageInfo, error)

    // Maintenance
    Clean(ctx context.Context) (*CleanResult, error)
}
```

**Adding New Manager** (Plugin Architecture):

1. Create adapter package: `pkg/infrastructure/adapter/manager/newmanager/`
2. Implement `ManagerAdapter` interface
3. Register in factory: `adapterFactory.Register("newmanager", adapter)`
4. Add detection logic: platform-specific paths
5. Write integration tests: Docker-based validation

**Example**: Adding Pacman support

```go
// pkg/infrastructure/adapter/manager/pacman/adapter.go
package pacman

type PacmanAdapter struct {
    executor port.CommandExecutor
    logger   port.Logger
}

func NewAdapter(executor port.CommandExecutor, logger port.Logger) *PacmanAdapter {
    return &PacmanAdapter{executor: executor, logger: logger}
}

func (a *PacmanAdapter) Detect(ctx context.Context) (bool, error) {
    // Check if pacman is available
    result, err := a.executor.Execute(ctx, "which", "pacman")
    return err == nil && result.ExitCode == 0, nil
}

func (a *PacmanAdapter) Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
    if opts.DryRun {
        return a.dryRunUpdate(ctx)
    }

    // sudo pacman -Syu
    result, err := a.executor.Execute(ctx, "sudo", "pacman", "-Syu", "--noconfirm")
    if err != nil {
        return nil, err
    }

    packages := parseP acmanOutput(result.Stdout)
    return &UpdateResult{Packages: packages}, nil
}
```

---

### 4.2 Update Strategy Pattern

```go
// pkg/domain/update/strategy.go
package update

type UpdateStrategy interface {
    SelectVersion(current string, available []string) (string, error)
    ShouldUpdate(current, available string) bool
}

// Implementations
type LatestStrategy struct{}      // Always latest (including beta/rc)
type StableStrategy struct{}      // Latest stable only
type MinorStrategy struct{}       // Latest minor/patch, no major
type FixedStrategy struct{}       // Show updates, don't install
```

**Strategy Selection**:

```go
func NewStrategy(name string) UpdateStrategy {
    switch name {
    case "latest":
        return &LatestStrategy{}
    case "stable":
        return &StableStrategy{}
    case "minor":
        return &MinorStrategy{}
    case "fixed":
        return &FixedStrategy{}
    default:
        return &StableStrategy{} // Default
    }
}
```

---

### 4.3 Repository Pattern

```go
// pkg/application/port/output/repository_port.go
package output

type ManagerRepository interface {
    FindAll(ctx context.Context) ([]*domain.Manager, error)
    FindInstalled(ctx context.Context) ([]*domain.Manager, error)
    FindByID(ctx context.Context, id domain.ManagerID) (*domain.Manager, error)
    Save(ctx context.Context, m *domain.Manager) error
}

type ConfigurationRepository interface {
    Load(ctx context.Context) (*domain.Config, error)
    Save(ctx context.Context, cfg *domain.Config) error
}
```

**Implementations**:

1. **YAML Repository** (production):
   - Read/write: `~/.config/pmctl/config.yml`
   - Atomic writes: temp file → rename
   - Schema validation

2. **Memory Repository** (testing):
   - In-memory storage
   - Fast, isolated tests
   - No file I/O
