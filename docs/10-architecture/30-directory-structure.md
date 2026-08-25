# 3. Directory Structure

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

```
gzh-cli-package-manager/
├── cmd/
│   └── gz-pm/                   # CLI entry point
│       ├── main.go              # DI setup, main function
│       ├── command/             # Cobra commands
│       ├── formatter/           # Output formatters
│       └── validator/           # Input validation
│
├── pkg/                         # Public packages
│   ├── domain/                  # Domain Layer (NO external deps)
│   │   ├── manager/            # Package manager domain
│   │   ├── update/             # Update domain
│   │   ├── diagnostics/        # Diagnostics domain
│   │   └── error/              # Domain errors
│   │
│   ├── application/             # Application Layer
│   │   ├── bootstrap/          # Bootstrap use cases
│   │   ├── update/             # Update use cases
│   │   ├── sync/               # Sync use cases
│   │   ├── port/               # Port interfaces
│   │   │   ├── input/          # Input ports (use case interfaces)
│   │   │   └── output/         # Output ports (driven interfaces)
│   │   └── dto/                # Data Transfer Objects
│   │
│   └── infrastructure/          # Infrastructure Layer
│       ├── adapter/
│       │   ├── manager/        # Manager adapters (brew, asdf, etc.)
│       │   ├── executor/       # Command execution
│       │   └── filesystem/     # File operations
│       ├── repository/         # Repository implementations
│       │   ├── memory/         # In-memory (testing)
│       │   └── yaml/           # YAML persistence
│       └── platform/           # Platform detection
│
├── internal/                    # Private implementation
│   ├── logger/                 # Logging interface + implementations
│   ├── config/                 # Configuration management
│   └── cli/                    # CLI utilities
│
├── test/                        # Test code
│   ├── unit/                   # Unit tests (by layer)
│   ├── integration/            # Integration tests
│   │   ├── docker/            # Docker-based tests
│   │   │   ├── Dockerfile.ubuntu
│   │   │   ├── Dockerfile.alpine
│   │   │   └── Dockerfile.fedora
│   │   └── testdata/          # Test fixtures
│   └── e2e/                    # End-to-end tests
│
├── docs/                        # Documentation
│   ├── 00-overview/
│   ├── 10-architecture/
│   │   └── adr/                # Architecture Decision Records
│   ├── 20-requirements/
│   ├── 30-specifications/
│   ├── 40-api/
│   └── 50-development/
│
├── scripts/                     # Build and dev scripts
│   ├── build.sh
│   ├── test.sh
│   └── release.sh
│
├── README.md
├── ARCHITECTURE.md              # This file
├── REQUIREMENTS.md
├── PRD.md
├── CONTRIBUTING.md
├── Makefile
├── go.mod
└── go.sum
```

**File Size Policy** (LLM-Friendly):
- **Ideal**: < 300 lines (< 10KB)
- **Maximum**: 500 lines (< 20KB)
- **Split if**: > 500 lines or multiple responsibilities
