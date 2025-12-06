# Testing Guide - gzh-cli-package-manager

## Test Organization

```
pkg/domain/manager/
├── entity.go          # Core logic
└── entity_test.go     # Pure unit tests (no mocks)

pkg/application/update/
├── update_all.go      # Use case
└── update_all_test.go # Mocked ports (gomock)

pkg/infrastructure/adapter/manager/homebrew/
├── adapter.go                  # Implementation
├── adapter_test.go             # Unit tests (mocked executor)
└── adapter_integration_test.go # Real Homebrew in Docker
```

## Coverage Targets

| Layer | Target | Minimum |
|-------|--------|---------|
| Domain | 95% | 90% |
| Application | 90% | 85% |
| Infrastructure | 85% | 75% |
| **Overall** | **90%** | **85%** |

## Running Tests

```bash
# Fast unit tests only
go test -short ./...

# With race detection
go test -race ./...

# Integration tests (requires Docker)
go test -tags=integration ./...

# Single package with verbose output
go test -v ./pkg/domain/manager/...

# All tests
make test

# Unit tests only
make test-unit

# Integration tests
make test-integration

# Coverage report (generates coverage.html)
make test-coverage

# Test specific package
go test -v ./pkg/domain/manager/...

# Test with race detection
go test -race ./...
```

## Test Helpers from gzh-cli-core

```go
import "github.com/gizzahub/gzh-cli-core/testutil"

// Create temp directory (auto-cleanup)
tempDir := testutil.TempDir(t)

// Assertions
testutil.AssertNoError(t, err)
testutil.AssertEqual(t, expected, actual)
testutil.AssertContains(t, haystack, needle)

// Capture output
output := testutil.CaptureOutput(func() {
    fmt.Println("test")
})
```

## Testing by Layer

### Domain Layer
- Pure unit tests
- NO mocks
- NO external dependencies
- Test business logic directly

```go
func TestManagerEntity(t *testing.T) {
    manager := domain.NewManager("homebrew", "5.0.0")
    result := manager.Update()
    // Test pure logic
}
```

### Application Layer
- Mock all ports with gomock
- Test use case orchestration
- Verify port interactions

```go
func TestUpdateUseCase(t *testing.T) {
    ctrl := gomock.NewController(t)
    mockRepo := mocks.NewMockManagerRepository(ctrl)

    useCase := NewUpdateUseCase(mockRepo)
    // Test orchestration
}
```

### Infrastructure Layer
- Unit tests with mocked executor
- Integration tests with Docker
- Test adapter implementations

```go
func TestHomebrewAdapter(t *testing.T) {
    mockExecutor := mocks.NewMockExecutor()
    adapter := NewHomebrewAdapter(mockExecutor)
    // Test adapter
}
```

## Test File Size Policy

**Critical for LLM-friendly codebase**:

| Size | Status | Action |
|------|--------|--------|
| < 10KB (~300 lines) | ✅ Ideal | Maintain |
| 10-50KB | ⚠️ Warning | Consider splitting |
| > 50KB | ❌ Problem | Must split |

**Reason**: Keep files digestible for AI code analysis and human review.
