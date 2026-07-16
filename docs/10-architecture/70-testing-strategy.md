# 7. Testing Strategy

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

### 7.1 Test Coverage Goals

| Layer | Target Coverage | Test Type | Dependencies |
|-------|----------------|-----------|--------------|
| Domain | 95%+ | Unit | None (pure logic) |
| Application | 90%+ | Unit | Mocked ports |
| Infrastructure | 85%+ | Integration | Docker containers |
| Presentation | 80%+ | E2E | Full system |
| **Overall** | **90%+** | Mixed | - |

---

### 7.2 Test Organization

```
test/
├── unit/                        # Unit tests (fast, no I/O)
│   ├── domain/
│   │   ├── manager_test.go
│   │   └── update_test.go
│   ├── application/
│   │   └── update_usecase_test.go
│   └── infrastructure/
│       └── adapter_test.go
│
├── integration/                 # Integration tests (Docker)
│   ├── homebrew_test.go        # Requires brew in container
│   ├── asdf_test.go
│   └── docker/
│       ├── Dockerfile.ubuntu
│       ├── Dockerfile.alpine
│       └── docker-compose.yml
│
└── e2e/                         # End-to-end tests
    ├── update_command_test.go
    └── testdata/
```

---

### 7.3 Test Examples

**Domain Layer (Pure Unit Test)**:

```go
// test/unit/domain/update_strategy_test.go
package domain_test

func TestStableStrategy_SelectVersion(t *testing.T) {
    tests := []struct {
        name      string
        current   string
        available []string
        expected  string
    }{
        {
            name:      "filters beta versions",
            current:   "1.0.0",
            available: []string{"1.1.0", "1.2.0-beta", "2.0.0-rc1"},
            expected:  "1.1.0",  // Latest stable
        },
        {
            name:      "selects latest stable",
            current:   "1.0.0",
            available: []string{"1.0.1", "1.0.2", "1.1.0"},
            expected:  "1.1.0",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            strategy := NewStableStrategy()
            result, err := strategy.SelectVersion(tt.current, tt.available)

            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

**Application Layer (Mocked Ports)**:

```go
// test/unit/application/update_usecase_test.go
package application_test

func TestUpdateAllManagersUseCase_Execute(t *testing.T) {
    // Arrange: Create mocks
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mock_port.NewMockManagerRepository(ctrl)
    mockExecutor := mock_port.NewMockCommandExecutor(ctrl)
    mockLogger := mock_port.NewMockLogger(ctrl)

    // Setup expectations
    mockRepo.EXPECT().
        FindInstalled(gomock.Any()).
        Return([]*domain.Manager{
            {ID: "brew", Installed: true},
        }, nil)

    mockExecutor.EXPECT().
        Execute(gomock.Any(), "brew", "update").
        Return(&ExecutionResult{ExitCode: 0}, nil)

    // Create use case
    uc := NewUpdateAllManagersUseCase(mockRepo, mockExecutor, mockLogger, NewStableStrategy())

    // Act
    resp, err := uc.Execute(context.Background(), &dto.UpdateAllRequest{All: true})

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 1, len(resp.Results))
    assert.Equal(t, "brew", resp.Results[0].ManagerID)
}
```

**Infrastructure Layer (Docker Integration Test)**:

```go
// test/integration/homebrew_test.go
package integration_test

func TestHomebrewAdapter_Update(t *testing.T) {
    // Requires: Docker container with Homebrew installed
    container := setupHomebrewContainer(t)
    defer container.Terminate()

    executor := executor.NewShellExecutor(logger)
    adapter := homebrew.NewAdapter(executor, logger)

    // Test detection
    detected, err := adapter.Detect(context.Background())
    require.NoError(t, err)
    assert.True(t, detected)

    // Test update (dry-run)
    result, err := adapter.Update(context.Background(), UpdateOptions{DryRun: true})
    require.NoError(t, err)
    assert.NotNil(t, result)
}

func setupHomebrewContainer(t *testing.T) testcontainers.Container {
    req := testcontainers.ContainerRequest{
        FromDockerfile: testcontainers.FromDockerfile{
            Context:    "docker/",
            Dockerfile: "Dockerfile.ubuntu",
        },
        ExposedPorts: []string{"22/tcp"},
        WaitingFor:   wait.ForLog("ready"),
    }

    container, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    require.NoError(t, err)

    return container
}
```
