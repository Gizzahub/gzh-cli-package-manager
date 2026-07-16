# 2.1 Domain Layer (pkg/domain)

> gzh-cli-package-manager 레이어 아키텍처 · [레이어 인덱스](README.md) · [아키텍처 인덱스](../README.md) · [ARCHITECTURE.md](../../../ARCHITECTURE.md)

**Responsibility**: Core business logic and entities

**No Dependencies On**:
- External libraries (except Go stdlib)
- Infrastructure details
- Frameworks (Cobra, YAML libraries)

**Components**:

```go
pkg/domain/
├── manager/                    # Package manager domain
│   ├── entity.go              # Manager, Package, Platform entities
│   ├── repository.go          # ManagerRepository interface
│   ├── service.go             # ManagerService (domain logic)
│   ├── types.go               # ManagerType, Status enums
│   └── manager_test.go        # Pure unit tests
├── update/                     # Update domain
│   ├── strategy.go            # UpdateStrategy interface + implementations
│   ├── executor.go            # UpdateExecutor interface
│   ├── version.go             # Version comparison logic
│   └── update_test.go
├── diagnostics/                # Health check domain
│   ├── duplicate_detector.go  # Duplicate binary detection
│   ├── health_checker.go      # Health check logic
│   └── diagnostics_test.go
└── error/                      # Domain-specific errors
    └── errors.go
```

**Example Entity**:

```go
// pkg/domain/manager/entity.go
package manager

type Manager struct {
    ID          ManagerID      // Unique identifier (brew, asdf, npm)
    Name        string         // Display name
    Type        ManagerType    // System, Version, Language
    Platform    Platform       // darwin, linux, windows
    Installed   bool
    Version     string
    ConfigPath  string
    Packages    []Package
}

type Package struct {
    Name             string
    CurrentVersion   string
    AvailableVersion string
    Manager          ManagerID
    SizeMB           float64
    UpdateType       UpdateType  // Major, Minor, Patch
}
```

**Example Domain Service**:

```go
// pkg/domain/update/strategy.go
package update

// Strategy interface (domain abstraction)
type UpdateStrategy interface {
    SelectVersion(current string, available []string) (string, error)
    ShouldUpdate(current, available string) bool
}

// Stable strategy (domain logic)
type StableStrategy struct{}

func (s *StableStrategy) SelectVersion(current string, available []string) (string, error) {
    // Pure business logic: filter out beta/rc, select latest stable
    stableVersions := filterStableVersions(available)
    return selectLatest(stableVersions), nil
}
```

**Testing**: 95%+ coverage, no mocks needed (pure functions).
