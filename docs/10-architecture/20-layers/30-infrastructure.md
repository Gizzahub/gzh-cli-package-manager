# 2.3 Infrastructure Layer (pkg/infrastructure)

> gzh-cli-package-manager 레이어 아키텍처 · [레이어 인덱스](README.md) · [아키텍처 인덱스](../README.md) · [ARCHITECTURE.md](../../../ARCHITECTURE.md)

**Responsibility**: External system integrations

**Dependencies**:
- Domain layer (implements domain interfaces)
- Application layer (implements output ports)
- External libraries (allowed here)

**Components**:

```go
pkg/infrastructure/
├── adapter/
│   ├── manager/                 # Package manager adapters
│   │   ├── homebrew/
│   │   │   ├── adapter.go      # Homebrew implementation
│   │   │   ├── detector.go     # Detection logic
│   │   │   ├── parser.go       # Output parsing
│   │   │   └── adapter_test.go # Integration tests
│   │   ├── asdf/
│   │   │   ├── adapter.go
│   │   │   ├── plugin_manager.go
│   │   │   └── adapter_test.go
│   │   ├── npm/
│   │   ├── pip/
│   │   ├── apt/
│   │   └── factory.go          # Manager factory
│   ├── executor/                # Command execution
│   │   ├── shell_executor.go   # Real shell execution
│   │   ├── mock_executor.go    # Mock for testing
│   │   └── executor_test.go
│   └── filesystem/              # File operations
│       ├── config_loader.go
│       ├── state_persister.go
│       └── filesystem_test.go
├── repository/                  # Repository implementations
│   ├── memory/                  # In-memory (for testing)
│   │   └── manager_repository.go
│   └── yaml/                    # YAML-based persistence
│       ├── config_repository.go
│       ├── state_repository.go
│       └── yaml_test.go
└── platform/                    # Platform detection
    ├── detector.go
    ├── darwin/
    ├── linux/
    ├── windows/
    └── detector_test.go
```

**Example Adapter**:

```go
// pkg/infrastructure/adapter/manager/homebrew/adapter.go
package homebrew

import (
    "context"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/application/port"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

type HomebrewAdapter struct {
    executor port.CommandExecutor
    logger   port.Logger
}

// Implements ManagerAdapter interface
func (a *HomebrewAdapter) Detect(ctx context.Context) (bool, error) {
    result, err := a.executor.Execute(ctx, "which", "brew")
    return err == nil && result.ExitCode == 0, nil
}

func (a *HomebrewAdapter) GetVersion(ctx context.Context) (string, error) {
    result, err := a.executor.Execute(ctx, "brew", "--version")
    if err != nil {
        return "", err
    }
    return parseBrewVersion(result.Stdout), nil
}

func (a *HomebrewAdapter) Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
    // 1. brew update (refresh package database)
    if err := a.brewUpdate(ctx); err != nil {
        return nil, err
    }

    // 2. brew upgrade (upgrade packages)
    packages, err := a.brewUpgrade(ctx, opts.DryRun)
    if err != nil {
        return nil, err
    }

    // 3. brew cleanup (free disk space)
    freedMB, err := a.brewCleanup(ctx)
    if err != nil {
        a.logger.Warn("Cleanup failed", logger.Field("error", err))
    }

    return &UpdateResult{
        Packages: packages,
        FreedMB:  freedMB,
    }, nil
}

func parseBrewVersion(output string) string {
    // Homebrew 4.2.0
    parts := strings.Fields(output)
    if len(parts) >= 2 {
        return parts[1]
    }
    return "unknown"
}
```

**Testing**: 85%+ coverage, Docker-based integration tests.
