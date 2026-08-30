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
sdk selfupdate && sdk update

# Just run:
gz-pm update --all
```

## ✨ Features

- **Multi-Manager Support** - Homebrew, ASDF, npm, pip, apt, pacman, sdkman, cargo, and more
- **Unified Interface** - One command to update all package managers
- **Rich Progress Output** - Visual progress bars, color-coded status, detailed summaries
- **Smart Conflict Detection** - Identifies duplicate binaries and version manager conflicts
- **Environment Awareness** - Detects conda, virtualenv, and adjusts behavior accordingly
- **Dry-Run Support** - Preview changes before executing
- **Multiple Output Formats** - Human-readable (default), JSON (for scripts), simple (for CI/CD)
- **Cross-Platform** - macOS, Linux (Ubuntu, Arch, Debian), Windows (planned)
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

| Manager | macOS | Linux | Windows | Status |
|---------|-------|-------|---------|--------|
| **Homebrew** | ✅ | ✅ | ❌ | Stable |
| **ASDF** | ✅ | ✅ | ❌ | Stable |
| **npm** (Node.js) | ✅ | ✅ | ✅ | Stable |
| **pip** (Python) | ✅ | ✅ | ✅ | Stable |
| **cargo** (Rust) | ✅ | ✅ | ✅ | Stable |
| **sdkman** (JVM) | ✅ | ✅ | ❌ | Stable |
| **apt** (Debian/Ubuntu) | ❌ | ✅ | ❌ | Stable |
| **pacman** (Arch) | ❌ | ✅ | ❌ | Stable |
| **yay** (AUR) | ❌ | ✅ | ❌ | Beta |
| **choco** (Windows) | ❌ | ❌ | 🔜 | Planned |
| **scoop** (Windows) | ❌ | ❌ | 🔜 | Planned |

## 🎨 Example Output

```text
📦 Package Manager Update - gz-pm v1.0.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔍 Performing pre-flight checks...

📊 Resource Availability Check
✅ Disk: Sufficient disk space: 45.2GB available, ~2.1GB needed
✅ Network: Network connectivity good: 4/4 repositories accessible
✅ Memory: Sufficient memory: 8192MB available

═══════════ 🚀 [1/3] brew — Updating ═══════════
🍺 Updating Homebrew...
✅ brew update: Updated 23 formulae (15.2s)
✅ brew upgrade: Upgraded 5 packages (42.8s)
   • node: 20.11.0 → 20.11.1 (24.8MB)
   • git: 2.43.0 → 2.43.1 (8.4MB)
   • python@3.11: 3.11.7 → 3.11.8 (15.2MB)

═══════════ 🚀 [2/3] asdf — Updating ═══════════
🔄 Updating asdf version manager...
✅ asdf plugin update --all: 8 plugins updated (18.4s)
✅ nodejs: 20.11.0 → 20.11.1 installed (35.2s)

═══════════ 🚀 [3/3] npm — Updating ═══════════
🧩 Updating npm global packages...
✅ npm update -g: 12 global packages updated (38.2s)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🎉 Package manager updates completed successfully!

📊 Summary:
   • Managers updated: 3/3
   • Packages upgraded: 27
   • Download size: 164.3MB
   • Disk freed: 245MB

⏰ Completed in 3m 42s
```

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

### v1.0 (Current) - MVP
- ✅ Multi-manager update orchestration
- ✅ Enhanced output with progress indication
- ✅ Resource pre-flight checks
- ✅ Duplicate binary detection
- ✅ Dry-run support
- ✅ JSON output for automation

### v1.1 (Month 3)
- 🔜 Bootstrap command (setup from config)
- 🔜 Sync command (config-driven updates)
- 🔜 Export command (save current config)
- 🔜 SQLite-based configuration
- 🔜 Plugin system for custom managers

### v1.2 (Month 5)
- 🔜 TUI (Terminal UI) with Bubble Tea
- 🔜 Cloud config sync (Dropbox, Google Drive)
- 🔜 Update scheduling (cron integration)
- 🔜 Rollback capability

### v2.0 (Month 9)
- 🔜 Team/enterprise features (shared configs)
- 🔜 Windows support (choco, scoop)
- 🔜 REST API for integration
- 🔜 Prometheus metrics

See [PRD.md](PRD.md) for detailed roadmap.

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
- **Release Readiness**: The project is not yet declared v1.0-ready; automated checks, platform coverage, and release controls require verification before release
- **Test Coverage**: Target 90%+; generate a current report with `make test-coverage`

---

**Made with ❤️ by the GizzaHub team**

[GitHub](https://github.com/gizzahub/gzh-cli-package-manager) •
[Documentation](docs/) •
[Contributing](CONTRIBUTING.md) •
[License](LICENSE)
