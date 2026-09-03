# gz-pm - Package Manager Control

> **Unified package manager orchestration for multi-platform development environments**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

## 🎯 What is gz-pm?

`gz-pm` (Package Manager Control) is a CLI tool that orchestrates multiple package managers across your system, providing a unified interface for updates, configuration, and management.

**Problem**: Modern development environments use multiple package managers (Homebrew, npm, pip, asdf, apt, etc.), each with different commands, update processes, and configurations.

**Solution**: gz-pm provides a single, consistent interface to manage them all.

```bash
# Instead of:
brew update && brew upgrade
asdf plugin update --all && asdf update
npm update -g
pip install --upgrade pip
cargo install-update -a

# Just run:
gz-pm update --all
```

## ✨ Features

- **Multi-Manager Support** - Homebrew, ASDF, npm, pip, cargo, apt, pacman, winget, scoop, chocolatey
- **Unified Interface** - One command to update all package managers
- **Readable Output** - Per-manager status icons and an aggregate summary
- **Environment Awareness** - Detects conda and virtualenv, and refuses to run pip
  inside a conda environment unless you pass `--pip-allow-conda`
- **Dry-Run Support** - Preview changes before executing
- **Two Output Formats** - `text` (default, human-readable) and `json` (for scripts)
- **Cross-Platform** - macOS and Linux (Ubuntu, Debian, Arch); the Windows managers ship but carry no CI coverage
- **Configuration Management** - Bootstrap package-manager setups from configuration

## 📦 Installation

### Homebrew (macOS/Linux)

Planned for the first approved release; no formula or stable tag is published yet.

```bash
brew tap gizzahub/tap
brew install gz-pm
```

### Go Install

```bash
go install github.com/gizzahub/gzh-cli-package-manager/cmd/gz-pm@latest
```

### From Source

```bash
git clone https://github.com/gizzahub/gzh-cli-package-manager.git
cd gzh-cli-package-manager
make build
make install  # Installs to $GOPATH/bin (no sudo required)
```

### Binary Releases

After the first approved tag, download `gz-pm-<os>-<arch>` artifacts from
[GitHub Releases](https://github.com/gizzahub/gzh-cli-package-manager/releases).

## 🚀 Quick Start

### Update All Package Managers

```bash
# Update all detected package managers
gz-pm update --all

# Preview changes without executing
gz-pm update --all --dry-run

# Update specific managers only
gz-pm update --managers brew,asdf,npm

# JSON output for scripting
gz-pm update --all --output json
```

### Check Status

```bash
# View installed managers and their status
gz-pm status

# Detailed status with package counts
gz-pm status --verbose
```

### Per-Manager Commands (Windows)

Direct list/search against a single manager adapter (always registered;
fails clearly if the manager is not installed):

```bash
gz-pm winget list
gz-pm winget search git
gz-pm scoop list --output json
gz-pm scoop search 7zip
gz-pm chocolatey list
gz-pm chocolatey search git
```

See [Per-Manager CLI](docs/specifications/per-manager-cli.md) for output formats and error policy.

### Bootstrap a New System

```bash
# Install and configure from a config file
gz-pm bootstrap --config mysetup.yaml

# Interactive setup wizard
gz-pm bootstrap --interactive
```

### Configuration Export

Configuration export/import and package synchronization are planned for a future
release; they are not current `gz-pm` commands.

## 📋 Supported Package Managers

These ten adapters are the ones `pkg/infrastructure/adapter/registry` constructs.
Registration is unconditional -- no adapter gates on `GOOS` -- so a manager
becomes *available* when its executable is found on the host and reports a clear
error when it is not. The columns below therefore describe where each underlying
manager runs, not where `gz-pm` compiles.

| Manager | macOS | Linux | Windows |
|---------|-------|-------|---------|
| **Homebrew** | ✅ | ✅ | ❌ |
| **ASDF** | ✅ | ✅ | ❌ |
| **npm** (Node.js) | ✅ | ✅ | ⚠️ |
| **pip** (Python) | ✅ | ✅ | ⚠️ |
| **cargo** (Rust) | ✅ | ✅ | ⚠️ |
| **apt** (Debian/Ubuntu) | ❌ | ✅ | ❌ |
| **pacman** (Arch) | ❌ | ✅ | ❌ |
| **winget** (Windows) | ❌ | ❌ | ✅ |
| **scoop** (Windows) | ❌ | ❌ | ✅ |
| **chocolatey** (Windows) | ❌ | ❌ | ✅ |

The three Windows managers ship and are reachable as `gz-pm winget`,
`gz-pm scoop` and `gz-pm chocolatey`. They are **untested on Windows**: the
matrix in `.github/workflows/test.yml` runs `ubuntu-latest` and `macos-latest`
only, so nothing exercises them against a real winget, scoop or choco.

The ⚠️ marks the reverse gap. npm, pip and cargo run on Windows, but `gz-pm`
detects them by shelling out to `which` (`pkg/infrastructure/adapter/manager/`
`npm`, `pip`, `cargo`), which native Windows does not provide -- only the three
Windows adapters use `where`. On Windows those three will therefore report as
not installed even when they are, unless a `which` is on PATH (Git Bash, WSL,
or a shim). The manager works; the detection does not.

**sdkman** and **yay** are not supported. Both exist as manager IDs in the domain
layer, which is easy to mistake for support, but the registry builds no adapter
for either and `registry_test.go` fails if one appears.

## 🎨 Example Output

`gz-pm update --all --dry-run` on a macOS host with Homebrew, npm, pip and cargo
installed:

```text
🧪 Package Manager Update (DRY-RUN)

✅ Homebrew
   Duration: 0.0s
   No packages updated

✅ NPM
   Duration: 0.0s
   No packages updated

✅ Pip
   Duration: 0.0s
   No packages updated

✅ Cargo
   Duration: 0.0s
   No packages updated

📊 Summary
   Total Managers: 4
   Successful: 4
   Failed: 0
   Total Packages Updated: 0
   Total Duration: 51.4s
```

A manager that is not installed is left out of the run entirely rather than
reported -- `--all` selects from installed managers only, and a name given to
`--managers` that is not installed is dropped with a log line, so it appears in
neither the per-manager output nor `Total Managers`. The one manager that is
listed as skipped with a stated reason is pip inside a conda environment
(`--pip-allow-conda` overrides it). `--output json` emits the same result as a
single JSON document for scripting. The exact text layout is produced by
`displayUpdateText` in `cmd/gz-pm/command/update.go`; the selection rules are
`selectManagers` in `pkg/application/usecase/update/update.go`.

## 📖 Documentation

- **[Getting Started](#-installation)** - Installation and first steps
- **[Architecture](ARCHITECTURE.md)** - Technical design and structure
- **[Contributing](CONTRIBUTING.md)** - Development guide
- **[Product Requirements](PRD.md)** - Product vision and roadmap
- **[Requirements](REQUIREMENTS.md)** - Functional and non-functional requirements
- **[Specifications](docs/specifications/)** - Detailed use case specifications
- **[ADRs](docs/10-architecture/adr/)** - Architecture decision records

## 🏗️ Architecture

gz-pm follows **Clean Architecture** with **Hexagonal (Ports & Adapters)** pattern:

```
┌─────────────────────────────────────────────┐
│  CLI Commands (Presentation)                │
└──────────────┬──────────────────────────────┘
               │
┌──────────────▼──────────────────────────────┐
│  Use Cases (Application)                    │
└──────────────┬──────────────────────────────┘
               │
┌──────────────▼──────────────────────────────┐
│  Business Logic (Domain)                    │
└──────────────▲──────────────────────────────┘
               │
┌──────────────┴──────────────────────────────┐
│  Package Manager Adapters (Infrastructure)  │
└─────────────────────────────────────────────┘
```

**Key Principles**:
- Pure Go (no CGO dependencies)
- Interface-based design (easy to test and extend)
- Dependency injection (manual, explicit)
- 90%+ test coverage
- Static binaries (no runtime dependencies)

See [ARCHITECTURE.md](ARCHITECTURE.md) for details.

## 🧪 Development

### Prerequisites

- Go 1.24+ (required; Go 1.26.7 is the recommended development toolchain)
- Make
- Docker (for integration tests)

### Build from Source

```bash
# Clone repository
git clone https://github.com/gizzahub/gzh-cli-package-manager.git
cd gzh-cli-package-manager

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint

# Install locally
make install
```

### Project Structure

```
gzh-cli-package-manager/
├── cmd/gz-pm/                 # CLI entry point
├── pkg/
│   ├── domain/            # Core business logic (entities, interfaces)
│   ├── application/       # Use cases and workflows
│   └── infrastructure/    # Adapters, repositories, external integrations
├── docs/
│   ├── specifications/    # Use case specifications
│   └── 10-architecture/   # Architecture docs and ADRs
├── ARCHITECTURE.md        # Architecture overview
├── CONTRIBUTING.md        # Development guide
└── README.md             # This file
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines.

## 🛣️ Roadmap

The first release will be tagged **v0.1.0**. The leading zero is deliberate: the
public API is not frozen, several commands described in the specifications are
not built yet, and Windows has no CI coverage. A 1.0 would promise a
compatibility guarantee this project is not yet in a position to keep.

**Shipping today** -- every item below is reachable from `gz-pm --help`:

- `update` -- multi-manager orchestration, `--all` or `--managers`, `--dry-run`
- `status` -- per-manager availability, package counts, pending updates
- `bootstrap` -- set up managers from a config file or interactively
- `cleanup` -- `cache`, `orphans`, `quarantine` and `versions`
- `winget` / `scoop` / `chocolatey` -- per-manager list, search, install,
  uninstall, upgrade
- `text` and `json` output for the commands above

**Not built yet** -- named here because the specifications describe them and
that is easy to mistake for a shipped feature:

- `sync` and `export` (configuration round-trip)
- Resource pre-flight checks
- sdkman and yay adapters
- Plugin system, TUI, update scheduling, rollback

[PRD.md](PRD.md) is the roadmap of record. This section intentionally does not
restate its milestones: the copy that used to live here drifted, and ended up
marking unbuilt features as delivered.

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for:

- Development setup
- Coding standards
- Testing requirements
- Pull request process

**Quick Start for Contributors**:

```bash
# Fork and clone
git clone https://github.com/YOUR_USERNAME/gzh-cli-package-manager.git

# Create feature branch
git checkout -b feature/my-feature

# Make changes, commit following conventions
git commit -m "feat(domain): add feature X"

# Push and create PR
git push origin feature/my-feature
```

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) file.

## 🙏 Acknowledgments

- Inspired by and extracted from [gzh-cli](https://github.com/gizzahub/gzh-cli)
- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Follows [Clean Architecture](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html) principles
- Test infrastructure inspired by [testcontainers-go](https://github.com/testcontainers/testcontainers-go)

## 💬 Support

- **Bug Reports**: [GitHub Issues](https://github.com/gizzahub/gzh-cli-package-manager/issues)
- **Feature Requests**: [GitHub Issues](https://github.com/gizzahub/gzh-cli-package-manager/issues) with `enhancement` label
- **Questions**: [GitHub Discussions](https://github.com/gizzahub/gzh-cli-package-manager/discussions)
- **Security Issues**: security@gizzahub.com

## 📊 Project Status

- **Status**: Active implementation and quality hardening
- **Release Readiness**: The first release will be tagged v0.1.0. Windows carries no CI
  coverage and the release controls are still being verified; [PRD.md](PRD.md) holds the
  criteria a 1.0 would have to meet
- **Test Coverage**: Target 90%+; generate a current report with `make test-coverage`

---

**Made with ❤️ by the GizzaHub team**

[GitHub](https://github.com/gizzahub/gzh-cli-package-manager) •
[Documentation](docs/) •
[Contributing](CONTRIBUTING.md) •
[License](LICENSE)
