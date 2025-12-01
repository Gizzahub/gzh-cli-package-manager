# ADR-005: Repository Pattern for Configuration Management

**Date**: 2025-01-27
**Status**: Accepted
**Deciders**: Project Team
**Related**: ADR-002 (Clean Architecture), ADR-003 (Hexagonal Ports)

---

## Context

We need to decide how to manage configuration data for gzh-cli-package-manager. Configuration includes:
- Package manager settings and preferences
- Update strategies per manager
- Bootstrap/sync templates
- User preferences (output format, logging level)

**Requirements**:
- Support local file-based storage (v1.0)
- Flexible enough for future cloud/database backends
- Testable without file system dependencies
- Thread-safe for concurrent operations
- Version migration support

**Options**:
1. Direct file I/O in use cases (no abstraction)
2. Repository pattern with interface abstraction
3. ORM/database-first approach
4. Key-value store (etcd, consul)

---

## Decision

**Adopt Repository Pattern with YAML file-based implementation for v1.0.**

**Repository Interfaces** (Domain Layer):
```go
type ConfigurationRepository interface {
    Load(ctx context.Context) (*Config, error)
    Save(ctx context.Context, cfg *Config) error
    Exists(ctx context.Context) (bool, error)
}

type ManagerConfigRepository interface {
    GetByManagerID(ctx context.Context, id ManagerID) (*ManagerConfig, error)
    SaveManagerConfig(ctx context.Context, cfg *ManagerConfig) error
    ListAll(ctx context.Context) ([]*ManagerConfig, error)
}
```

**Implementation** (Infrastructure Layer):
```go
// YAML-based implementation for v1.0
type YAMLConfigRepository struct {
    configPath string
    logger     port.Logger
    mu         sync.RWMutex  // Thread-safe access
}
```

---

## Rationale

### Why Repository Pattern?

**1. Abstraction from Storage Details**
- Use cases don't care if config is in YAML, JSON, SQLite, or cloud
- Example:
  ```go
  // Application layer - no file system knowledge
  func (uc *UpdateAllUseCase) Execute(ctx context.Context) error {
      cfg, err := uc.configRepo.Load(ctx)  // Could be YAML, DB, HTTP
      // Use cfg without knowing source
  }
  ```

**2. Testability**
- Unit tests use in-memory implementation
- No file I/O in fast test suites
- Example:
  ```go
  // Test with mock repository
  mockRepo := &InMemoryConfigRepository{
      config: &Config{Strategy: "stable"},
  }
  uc := NewUpdateAllUseCase(mockRepo, ...)
  // Test without filesystem
  ```

**3. Future Flexibility**
- v1.0: YAML files in `~/.gz-pm/`
- v1.1: SQLite for better querying
- v1.2: Cloud sync (Dropbox, Google Drive)
- v2.0: Team shared configs (etcd, consul)
- **Application layer unchanged**

**4. Clean Architecture Compliance**
- Repository interface in Domain layer (no dependencies)
- Repository implementation in Infrastructure layer
- Use cases depend on interface, not implementation

**5. Migration Support**
- Repository can handle version upgrades
- Example:
  ```go
  func (r *YAMLConfigRepository) Load(ctx context.Context) (*Config, error) {
      data, _ := os.ReadFile(r.configPath)
      var raw map[string]interface{}
      yaml.Unmarshal(data, &raw)

      // Version migration
      if raw["version"] == "1.0" {
          return migrateV1ToV2(raw)
      }
      // Load normally
  }
  ```

### Why YAML for v1.0?

**1. Human-Readable**
- Users can edit config manually
- Easy to understand structure
- Comments supported

**2. Ecosystem Compatibility**
- Standard in DevOps tooling
- gzh-cli already uses YAML
- Migration from gzh-cli easier

**3. Simple Implementation**
- `gopkg.in/yaml.v3` is mature and fast
- No complex schema management
- Straightforward serialization

**4. Version Control Friendly**
- Text format works well with git
- Diffs are readable
- Team can share configs via repo

### Why Not Other Options?

**Direct File I/O (Alternative 1)**
- **Rejected**: Tight coupling to file system
- **Reason**: Hard to test, impossible to add cloud sync later
- **Example Problem**:
  ```go
  // BAD - Use case directly reads files
  func (uc *UpdateAllUseCase) Execute() {
      data, _ := os.ReadFile("~/.gz-pm/config.yaml")  // Tight coupling
      // Can't test without filesystem, can't change storage
  }
  ```

**Database-First (Alternative 3)**
- **Rejected for v1.0**: Overkill for CLI tool
- **Reason**: Adds complexity, requires migration scripts
- **When to reconsider**: v1.2+ if querying becomes complex

**Key-Value Store (Alternative 4)**
- **Rejected for v1.0**: External dependency
- **Reason**: Users don't want to run etcd for a CLI tool
- **When to use**: v2.0 team/enterprise features

---

## Consequences

### Positive ✅

1. **Swappable Storage Backends**
   - v1.0: YAML files
   - v1.1: SQLite for faster lookups
   - v1.2: Cloud storage for sync
   - Application code unchanged

2. **Fast Unit Tests**
   - In-memory repository for tests
   - No file system setup/teardown
   - Tests run in < 10ms
   - Example:
     ```go
     // Test runs without files
     repo := &InMemoryConfigRepository{configs: testData}
     uc := NewBootstrapUseCase(repo, ...)
     result, _ := uc.Execute(ctx, req)
     ```

3. **Thread-Safe Config Access**
   - Repository handles locking
   - Concurrent reads/writes safe
   - Use cases don't worry about race conditions

4. **Version Migration Support**
   - Repository handles schema changes
   - Users upgrade seamlessly
   - Example: v1 YAML → v2 YAML with new fields

5. **Separation of Concerns**
   - Domain: Defines what config is needed (interface)
   - Application: Uses config via interface
   - Infrastructure: Handles file I/O, parsing

### Negative ❌

1. **Abstraction Overhead**
   - Repository interface + implementation = more files
   - Simple config load becomes multi-layer
   - Example:
     ```
     Use Case → ConfigurationRepository (interface)
               → YAMLConfigRepository (implementation)
               → yaml.Unmarshal → os.ReadFile
     ```

   **Mitigation**: Worth it for testability and future flexibility

2. **Initial Implementation Cost**
   - Must write repository interface + implementation
   - In-memory version for tests
   - YAML version for production
   - Estimated: 3-4 hours

   **Mitigation**: One-time cost, pays off in maintainability

3. **Potential Over-Engineering**
   - Config might stay simple (YAML forever)
   - Repository abstraction unused
   - Risk: "YAGNI" (You Aren't Gonna Need It)

   **Mitigation**:
   - Pattern is standard in Clean Architecture
   - Testability alone justifies it
   - Team can add backends if needed

### Neutral 🔄

1. **Configuration Format**
   - YAML for v1.0
   - Can coexist with JSON, TOML via adapters
   - Format choice independent of repository pattern

2. **File Location**
   - v1.0: `~/.gz-pm/config.yaml`
   - v1.1+: Could be `~/.config/gz-pm/` (XDG)
   - Repository hides location from application

---

## Implementation

### Repository Interface (Domain Layer)

```go
// pkg/domain/config/repository.go
package config

import "context"

// ConfigurationRepository manages application configuration
type ConfigurationRepository interface {
    // Load reads configuration from storage
    Load(ctx context.Context) (*Config, error)

    // Save writes configuration to storage
    Save(ctx context.Context, cfg *Config) error

    // Exists checks if configuration file exists
    Exists(ctx context.Context) (bool, error)

    // Delete removes configuration (for reset/uninstall)
    Delete(ctx context.Context) error
}

// ManagerConfigRepository manages per-manager configurations
type ManagerConfigRepository interface {
    // GetByManagerID retrieves config for specific manager
    GetByManagerID(ctx context.Context, id ManagerID) (*ManagerConfig, error)

    // SaveManagerConfig persists manager-specific settings
    SaveManagerConfig(ctx context.Context, cfg *ManagerConfig) error

    // ListAll returns all manager configurations
    ListAll(ctx context.Context) ([]*ManagerConfig, error)
}
```

### Domain Entities

```go
// pkg/domain/config/entity.go
package config

type Config struct {
    Version           string                    // Schema version (e.g., "1.0")
    DefaultStrategy   string                    // "stable", "latest", etc.
    OutputFormat      string                    // "enhanced", "simple", "json"
    LogLevel          string                    // "info", "debug", "warn"
    Managers          map[ManagerID]*ManagerConfig
    LastUpdated       time.Time
}

type ManagerConfig struct {
    ID              ManagerID
    Enabled         bool
    Strategy        string    // Override default strategy
    AutoUpdate      bool
    CustomOptions   map[string]interface{}
}
```

### YAML Repository Implementation (Infrastructure Layer)

```go
// pkg/infrastructure/repository/config/yaml_repository.go
package config

import (
    "context"
    "os"
    "path/filepath"
    "sync"

    "gopkg.in/yaml.v3"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/config"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/application/port"
)

type YAMLConfigRepository struct {
    configPath string
    logger     port.Logger
    mu         sync.RWMutex  // Protects concurrent access
}

func NewYAMLConfigRepository(configPath string, logger port.Logger) *YAMLConfigRepository {
    return &YAMLConfigRepository{
        configPath: configPath,
        logger:     logger,
    }
}

func (r *YAMLConfigRepository) Load(ctx context.Context) (*config.Config, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Check file exists
    if _, err := os.Stat(r.configPath); os.IsNotExist(err) {
        return r.defaultConfig(), nil  // Return defaults if no file
    }

    // Read file
    data, err := os.ReadFile(r.configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    // Parse YAML
    var cfg config.Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    // Version migration
    cfg = r.migrateIfNeeded(cfg)

    return &cfg, nil
}

func (r *YAMLConfigRepository) Save(ctx context.Context, cfg *config.Config) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Ensure directory exists
    dir := filepath.Dir(r.configPath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create config dir: %w", err)
    }

    // Marshal to YAML
    data, err := yaml.Marshal(cfg)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }

    // Write atomically (write to temp, then rename)
    tempPath := r.configPath + ".tmp"
    if err := os.WriteFile(tempPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    if err := os.Rename(tempPath, r.configPath); err != nil {
        return fmt.Errorf("failed to rename config: %w", err)
    }

    r.logger.Info("Configuration saved", "path", r.configPath)
    return nil
}

func (r *YAMLConfigRepository) Exists(ctx context.Context) (bool, error) {
    _, err := os.Stat(r.configPath)
    if os.IsNotExist(err) {
        return false, nil
    }
    if err != nil {
        return false, err
    }
    return true, nil
}

func (r *YAMLConfigRepository) defaultConfig() *config.Config {
    return &config.Config{
        Version:         "1.0",
        DefaultStrategy: "stable",
        OutputFormat:    "enhanced",
        LogLevel:        "info",
        Managers:        make(map[config.ManagerID]*config.ManagerConfig),
    }
}

func (r *YAMLConfigRepository) migrateIfNeeded(cfg config.Config) config.Config {
    // Version migration logic
    if cfg.Version == "0.9" {
        // Migrate from v0.9 to v1.0
        cfg.Version = "1.0"
        // Add new fields with defaults
    }
    return cfg
}
```

### In-Memory Repository (Testing)

```go
// pkg/infrastructure/repository/config/memory_repository.go
package config

import (
    "context"
    "sync"

    "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/config"
)

type InMemoryConfigRepository struct {
    config *config.Config
    mu     sync.RWMutex
}

func NewInMemoryConfigRepository(initialConfig *config.Config) *InMemoryConfigRepository {
    return &InMemoryConfigRepository{
        config: initialConfig,
    }
}

func (r *InMemoryConfigRepository) Load(ctx context.Context) (*config.Config, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Return copy to prevent mutations
    return copyConfig(r.config), nil
}

func (r *InMemoryConfigRepository) Save(ctx context.Context, cfg *config.Config) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    r.config = copyConfig(cfg)
    return nil
}

func (r *InMemoryConfigRepository) Exists(ctx context.Context) (bool, error) {
    return r.config != nil, nil
}

func copyConfig(cfg *config.Config) *config.Config {
    // Deep copy implementation
    copied := *cfg
    copied.Managers = make(map[config.ManagerID]*config.ManagerConfig, len(cfg.Managers))
    for k, v := range cfg.Managers {
        managerCfg := *v
        copied.Managers[k] = &managerCfg
    }
    return &copied
}
```

### Usage in Application Layer

```go
// pkg/application/bootstrap/bootstrap_all.go
package bootstrap

type BootstrapAllUseCase struct {
    configRepo   port.ConfigurationRepository  // Port interface
    managerRepo  port.ManagerRepository
    logger       port.Logger
}

func (uc *BootstrapAllUseCase) Execute(ctx context.Context, req *dto.BootstrapRequest) error {
    // Load configuration
    cfg, err := uc.configRepo.Load(ctx)
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Use configuration
    strategy := cmp.Or(req.Strategy, cfg.DefaultStrategy, "stable")

    // ... bootstrap logic ...

    // Save updated configuration
    cfg.LastUpdated = time.Now()
    if err := uc.configRepo.Save(ctx, cfg); err != nil {
        return fmt.Errorf("failed to save config: %w", err)
    }

    return nil
}
```

### Dependency Injection

```go
// cmd/gz-pm/main.go
func main() {
    // Determine config path
    configPath := os.Getenv("PMCTL_CONFIG_PATH")
    if configPath == "" {
        home, _ := os.UserHomeDir()
        configPath = filepath.Join(home, ".gz-pm", "config.yaml")
    }

    // Create repository
    configRepo := config.NewYAMLConfigRepository(configPath, logger)

    // Inject into use cases
    bootstrapUC := bootstrap.NewBootstrapAllUseCase(
        configRepo,  // ConfigurationRepository port
        managerRepo,
        logger,
    )

    // ...
}
```

---

## Configuration File Format

### v1.0 YAML Structure

```yaml
# ~/.gz-pm/config.yaml
version: "1.0"

# Global settings
defaultStrategy: stable
outputFormat: enhanced
logLevel: info

# Manager-specific configurations
managers:
  homebrew:
    enabled: true
    strategy: stable
    autoUpdate: false
    customOptions:
      caskUpgrade: true
      greedy: true

  asdf:
    enabled: true
    strategy: latest
    autoUpdate: false

  npm:
    enabled: true
    strategy: stable
    customOptions:
      global: true

# Metadata
lastUpdated: 2025-01-27T10:30:00Z
```

### Future v1.1 SQLite Schema (Planned)

```sql
-- For faster querying in v1.1+
CREATE TABLE config (
    key TEXT PRIMARY KEY,
    value TEXT,
    type TEXT,  -- 'string', 'bool', 'json'
    updated_at TIMESTAMP
);

CREATE TABLE manager_configs (
    manager_id TEXT PRIMARY KEY,
    enabled BOOLEAN,
    strategy TEXT,
    auto_update BOOLEAN,
    custom_options TEXT,  -- JSON
    updated_at TIMESTAMP
);
```

---

## Testing Strategy

### Unit Tests (Application Layer)

```go
func TestBootstrapUseCase_LoadsConfig(t *testing.T) {
    // Arrange
    mockConfig := &config.Config{
        DefaultStrategy: "stable",
        OutputFormat:    "enhanced",
    }
    mockRepo := config.NewInMemoryConfigRepository(mockConfig)
    uc := bootstrap.NewBootstrapAllUseCase(mockRepo, managerRepo, logger)

    // Act
    req := &dto.BootstrapRequest{Strategy: ""}  // Should use default
    err := uc.Execute(context.Background(), req)

    // Assert
    assert.NoError(t, err)
    // Verify "stable" strategy was used
}
```

### Integration Tests (Infrastructure Layer)

```go
func TestYAMLConfigRepository_RoundTrip(t *testing.T) {
    // Arrange
    tempDir := t.TempDir()
    configPath := filepath.Join(tempDir, "config.yaml")
    repo := config.NewYAMLConfigRepository(configPath, logger)

    originalConfig := &config.Config{
        Version:         "1.0",
        DefaultStrategy: "latest",
        Managers:        make(map[config.ManagerID]*config.ManagerConfig),
    }

    // Act - Save
    err := repo.Save(context.Background(), originalConfig)
    assert.NoError(t, err)

    // Act - Load
    loadedConfig, err := repo.Load(context.Background())
    assert.NoError(t, err)

    // Assert
    assert.Equal(t, originalConfig.Version, loadedConfig.Version)
    assert.Equal(t, originalConfig.DefaultStrategy, loadedConfig.DefaultStrategy)
}
```

---

## Migration Path

### v1.0 → v1.1 (YAML → SQLite)

```go
// pkg/infrastructure/repository/config/migrator.go
type ConfigMigrator struct {
    yamlRepo   *YAMLConfigRepository
    sqliteRepo *SQLiteConfigRepository
}

func (m *ConfigMigrator) MigrateToSQLite(ctx context.Context) error {
    // Load from YAML
    cfg, err := m.yamlRepo.Load(ctx)
    if err != nil {
        return err
    }

    // Save to SQLite
    if err := m.sqliteRepo.Save(ctx, cfg); err != nil {
        return err
    }

    // Backup old YAML
    os.Rename(m.yamlRepo.configPath, m.yamlRepo.configPath+".backup")

    return nil
}
```

---

## Validation

### Architecture Conformance

```bash
# Verify domain layer only has interface
grep -r "yaml.Unmarshal\|json.Unmarshal" pkg/domain/ && echo "VIOLATION" || echo "OK"

# Verify application layer uses interface only
grep -r "YAMLConfigRepository\|SQLiteConfigRepository" pkg/application/ && echo "VIOLATION" || echo "OK"
```

### Success Criteria

- [ ] ConfigurationRepository interface in domain layer
- [ ] YAML implementation in infrastructure layer
- [ ] In-memory implementation for tests
- [ ] All use cases use repository interface (not direct file I/O)
- [ ] Unit tests run without file system
- [ ] Configuration persists across runs
- [ ] Thread-safe concurrent access

---

## Alternatives Considered

### Alternative 1: Direct File I/O in Use Cases
**Rejected**: Violates Clean Architecture, impossible to test without filesystem

### Alternative 2: Global Configuration Singleton
**Rejected**: Hard to test, thread-safety issues, tight coupling

### Alternative 3: Viper Configuration Library
**Considered**: Viper provides rich config management
**Rejected for v1.0**: Adds external dependency, repository pattern gives us same flexibility with less complexity
**Reconsider**: v1.2+ if config becomes very complex

---

## References

- **Repository Pattern**: Martin Fowler's Patterns of Enterprise Application Architecture
- **Clean Architecture**: Robert C. Martin (Domain interfaces, Infrastructure implementations)
- **YAML v3**: https://pkg.go.dev/gopkg.in/yaml.v3
- **ADR-002**: Clean Architecture layers
- **ADR-003**: Port interfaces and adapters

---

## Revision History

| Date | Changes | Author |
|------|---------|--------|
| 2025-01-27 | Initial version | Claude Code |

---

## Status Notes

**Current Status**: Accepted

**Implementation Plan**:
- Week 4: Domain layer (Config entity + repository interface)
- Week 5: Infrastructure layer (YAML repository implementation)
- Week 6: Testing (in-memory repository + tests)
- Week 7: Integration (wire into use cases)

**Future Enhancements**:
- v1.1: SQLite implementation for faster querying
- v1.2: Cloud sync support (Dropbox, Google Drive adapters)
- v2.0: Team shared configs (etcd, consul adapters)
