# 2.2 Application Layer (pkg/application)

> gzh-cli-package-manager 레이어 아키텍처 · [레이어 인덱스](README.md) · [아키텍처 인덱스](../README.md) · [ARCHITECTURE.md](../../../ARCHITECTURE.md)

**Responsibility**: Use case orchestration and workflows

**Dependencies**:
- Domain layer (entities, interfaces)
- Port interfaces (defined here)

**No Dependencies On**:
- Infrastructure implementations
- Frameworks

**Components**:

```go
pkg/application/
├── bootstrap/                   # Bootstrap use cases
│   ├── install_manager.go      # InstallManagerUseCase
│   ├── check_installation.go   # CheckInstallationUseCase
│   ├── configure_manager.go    # ConfigureManagerUseCase
│   └── bootstrap_test.go       # Use case tests (mocked ports)
├── update/                      # Update use cases
│   ├── update_all.go           # UpdateAllManagersUseCase
│   ├── update_single.go        # UpdateSingleManagerUseCase
│   ├── dry_run.go              # DryRunAnalysisUseCase
│   └── update_test.go
├── sync/                        # Sync use cases
│   ├── check_sync.go           # CheckSyncStatusUseCase
│   ├── synchronize.go          # SynchronizeVersionsUseCase
│   └── sync_test.go
├── port/                        # Port interfaces
│   ├── input/                  # Input ports (use case interfaces)
│   │   ├── bootstrap_port.go
│   │   ├── update_port.go
│   │   └── sync_port.go
│   └── output/                 # Output ports (driven interfaces)
│       ├── executor_port.go    # CommandExecutor interface
│       ├── logger_port.go      # Logger interface
│       ├── notifier_port.go    # Notifier interface
│       └── repository_port.go  # Repository interfaces
└── dto/                         # Data Transfer Objects
    ├── bootstrap_dto.go
    ├── update_dto.go
    └── sync_dto.go
```

**Example Use Case**:

```go
// pkg/application/update/update_all.go
package update

import (
    "context"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/application/dto"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/application/port"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"
)

type UpdateAllManagersUseCase struct {
    managerRepo  port.ManagerRepository    // Output port
    executor     port.CommandExecutor      // Output port
    logger       port.Logger               // Output port
    strategy     domain.UpdateStrategy     // Domain service
}

func (uc *UpdateAllManagersUseCase) Execute(
    ctx context.Context,
    req *dto.UpdateAllRequest,
) (*dto.UpdateAllResponse, error) {
    // 1. Validate input
    if err := req.Validate(); err != nil {
        return nil, err
    }

    // 2. Get installed managers
    managers, err := uc.managerRepo.FindInstalled(ctx)
    if err != nil {
        return nil, err
    }

    // 3. Execute updates for each manager
    results := make([]*dto.UpdateResult, 0, len(managers))
    for _, mgr := range managers {
        result, err := uc.updateSingleManager(ctx, mgr, req.Strategy)
        if err != nil {
            uc.logger.Error("Update failed", err, logger.Field("manager", mgr.ID))
            result.Status = dto.StatusFailed
            result.Error = err
        }
        results = append(results, result)
    }

    // 4. Build response
    return &dto.UpdateAllResponse{
        Results:  results,
        Summary:  uc.buildSummary(results),
        Duration: time.Since(start),
    }, nil
}
```

**Testing**: 90%+ coverage, all ports mocked.
