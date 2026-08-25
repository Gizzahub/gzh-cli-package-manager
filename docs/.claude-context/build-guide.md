# Build Guide - gzh-cli-package-manager

## Build & Development

```bash
# Build binary (outputs to bin/gz-pm)
make build

# Build for all platforms (macOS, Linux, Windows)
make build-all

# Install to $GOPATH/bin (no sudo required)
make install

# Run in development mode
make dev ARGS="--version"

# Watch for changes and rebuild (requires entr)
make watch
```

## Code Quality

```bash
# Format code (required before commits)
make fmt

# Run go vet
make vet

# Run linter (golangci-lint required)
make lint

# Auto-fix linting issues
golangci-lint run --fix

# Run all quality checks (fmt + lint + test)
make quality

# Full CI validation (includes coverage)
make ci-local
```

## Project Information

```bash
# Display version, git commit, build date
make version

# Display build configuration
make info

# Clean build artifacts
make clean
```

## CGO Policy

- **CGO_ENABLED=0** enforced
- Must use pure Go alternatives
- Check with: `./scripts/check-cgo.sh`

## Binary Output

- **Binary name**: `gz-pm`
- **Output location**: `build/gz-pm`
- **Platforms**: macOS, Linux (Ubuntu/Arch), Windows (WSL2)

## Quick Reference

```bash
# Most common commands
make build           # Build binary
make test           # Run unit tests
make quality        # Full quality suite (fmt + lint + test)
make clean          # Clean artifacts

# Binary location after build
./build/gz-pm version

# Directory structure
cmd/pm/              # CLI entry point (Presentation)
pkg/domain/          # Business logic (NO external deps)
pkg/application/     # Use cases (orchestration)
pkg/infrastructure/  # Adapters, repos (external integrations)
internal/            # Private utilities
docs/                # Documentation
```
