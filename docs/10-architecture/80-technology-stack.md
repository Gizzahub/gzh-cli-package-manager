# 8. Technology Stack

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

### 8.1 Core Technologies

- **Language**: Go 1.24.0+ (pure Go, no CGO); Go 1.26.7 is the recommended development toolchain
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **Configuration**: YAML ([gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3))
- **Logging**: [uber-go/zap](https://github.com/uber-go/zap) (structured logging)

### 8.2 Development Tools

- **Testing**: [testify](https://github.com/stretchr/testify), [gomock](https://github.com/golang/mock)
- **Linting**: [golangci-lint](https://golangci-lint.run/)
- **Formatting**: `gofumpt`, `goimports`
- **Integration Testing**: [testcontainers-go](https://golang.testcontainers.org/)

### 8.3 External Dependencies

```go
// go.mod
module github.com/gizzahub/gzh-cli-package-manager

go 1.24

require (
    github.com/spf13/cobra v1.9.1
    gopkg.in/yaml.v3 v3.0.1
    go.uber.org/zap v1.27.0
    github.com/fatih/color v1.18.0
    github.com/stretchr/testify v1.9.0
)
```

**No Heavy Dependencies**:
- No ORM (file-based config)
- No database (YAML persistence)
- No web framework (CLI only)
