# Product Requirements Document: gzh-cli-package-manager

> **Product Name**: gzh-cli-package-manager
> **CLI Binary**: `pmctl`
> **Version**: 1.0.0
> **Last Updated**: 2025-01-27
> **Status**: Draft

---

## Executive Summary

**gzh-cli-package-manager** (`pmctl`) is a standalone CLI tool that orchestrates multiple package managers through a unified interface. It solves the "package manager sprawl" problem faced by modern developers who must manage Homebrew, asdf, npm, pip, apt, and many other tools simultaneously.

### The Problem

Modern software development requires managing packages across:
- System package managers (brew, apt, pacman)
- Version managers (asdf, nvm, pyenv, rbenv)
- Language package managers (npm, pip, cargo, gem)
- JVM ecosystem (sdkman)

**Current Pain Points**:
1. **Fragmented workflows**: Different commands for each manager (`brew update && brew upgrade`, `npm update -g`, `pip install --upgrade pip`, etc.)
2. **No unified status**: Cannot see "what needs updating" across all managers
3. **Version conflicts**: Same tool installed by multiple managers (e.g., `node` via brew and asdf)
4. **Manual orchestration**: Developers write custom shell scripts to coordinate updates
5. **Environment awareness**: pip updates break conda environments, global npm conflicts with project-local

### The Solution

`pmctl` provides:
- **Unified update command**: `pmctl update --all` updates everything
- **Intelligent conflict detection**: Warns about duplicate installations
- **Environment awareness**: Detects conda, virtualenv, Docker contexts
- **Rich progress reporting**: Beautiful terminal output with actionable feedback
- **Automation-friendly**: JSON output for CI/CD integration

**Key Differentiator**: Not a replacement for package managers, but an orchestration layer that makes them work together harmoniously.

---

## Vision & Strategy

### Product Vision

"Make multi-package-manager workflows feel like using a single, well-designed tool."

### Success Metrics

**User Adoption**:
- 1,000+ active users within 6 months of v1.0
- 50+ GitHub stars within 3 months
- Featured in at least 2 developer tool round-ups

**User Satisfaction**:
- 90%+ of users report time savings
- < 5% uninstall rate
- 80%+ find duplicate detection valuable

**Technical Excellence**:
- 90%+ test coverage maintained
- < 10 critical bugs in first 6 months
- 95%+ specification compliance

---

## User Personas

### Persona 1: DevOps Engineer (Sarah)

**Background**:
- Manages 50+ servers with various Linux distributions
- Maintains development environments for 20-person team
- Automates everything with Ansible/Chef/Terraform

**Goals**:
- Keep all development tools up-to-date across team
- Ensure consistent package versions
- Automate dependency management

**Pain Points**:
- Different package managers on macOS vs Linux developers
- Manual update scripts are brittle and break often
- No visibility into what's out of date

**How pmctl Helps**:
- `pmctl status --output json` for automated inventory
- `pmctl update --all --dry-run` for preview before deployments
- Consistent commands across macOS, Ubuntu, Arch Linux

**Quote**: "I need one tool that works everywhere, not 10 different update scripts."

---

### Persona 2: Solo Full-Stack Developer (Alex)

**Background**:
- Works on 5-10 projects simultaneously
- Uses Node.js, Python, Go, Rust, Java
- macOS laptop for development

**Goals**:
- Keep development tools current without thinking about it
- Avoid version conflicts between projects
- Quick setup of new development machines

**Pain Points**:
- Forgets to update tools for months
- Wastes time debugging "it works on my machine" issues caused by old tools
- brew, npm, pip all need separate update commands

**How pmctl Helps**:
- Single `pmctl update --all` command updates everything
- Duplicate detection prevents asdf/brew conflicts
- Rich output shows what changed and why

**Quote**: "I just want my tools to stay current without the mental overhead."

---

### Persona 3: Team Lead (Marcus)

**Background**:
- Leads team of 15 developers
- Onboards 2-3 new developers per quarter
- Enforces development environment standards

**Goals**:
- Standardize team development environments
- Fast onboarding for new developers
- Reduce "environment setup" support requests

**Pain Points**:
- New developers spend days setting up environments
- Version mismatches cause integration issues
- Documentation gets out of date quickly

**How pmctl Helps**:
- `pmctl bootstrap` installs all required managers
- `pmctl export --all` creates reproducible environment configs
- `pmctl import` sets up new machine from config

**Quote**: "New devs should be productive on day 1, not day 3."

---

## Functional Requirements

### FR-001: Multi-Manager Update Orchestration

**Priority**: P0 (MVP Blocker)

**User Story**: As a developer, I want to update all my package managers and their packages with a single command, so I don't have to remember and execute different update procedures for each tool.

**Acceptance Criteria**:
- Execute `pmctl update --all` updates all detected managers
- Support selective updates: `pmctl update --manager brew`
- Show progress for each manager with clear status
- Handle failures gracefully (continue with remaining managers)
- Complete typical update in < 5 minutes (excluding downloads)

**Dependencies**: Manager detection (FR-002)

**Related Requirements**: REQ-UC001-001, REQ-UC001-002, REQ-UC001-003

---

### FR-002: Automatic Manager Detection

**Priority**: P0 (MVP Blocker)

**User Story**: As a developer, I want pmctl to automatically discover which package managers I have installed, so I don't need to configure anything manually.

**Acceptance Criteria**:
- Detect brew, asdf, npm, pip, apt, pacman, yay, sdkman, nvm, rbenv, pyenv automatically
- Support macOS, Linux (Ubuntu/Arch/Fedora), Windows/WSL2
- Cache detection results for 24 hours
- `--refresh` flag to force re-detection
- Detection completes in < 5 seconds

**Technical Approach**:
- Check standard installation paths
- Execute `which <manager>` or equivalent
- Parse version strings from `<manager> --version`

**Related Requirements**: REQ-STATUS-001

---

### FR-003: Update Strategies

**Priority**: P0 (MVP Blocker)

**User Story**: As a cautious developer, I want to control how aggressively packages are updated, so I can avoid breaking changes in production environments.

**Acceptance Criteria**:
- Default strategy: `stable` (latest stable releases only)
- `--strategy latest`: Include beta/rc versions
- `--strategy minor`: Patch and minor updates only, no major versions
- `--strategy fixed`: Show available updates but don't install
- Per-manager strategy overrides via config file

**Example**:
```bash
# Update everything to latest stable
pmctl update --all --strategy stable

# Update Homebrew aggressively, pip conservatively
# Config: brew=latest, pip=minor
pmctl update --all
```

**Related Requirements**: REQ-UC001-002

---

### FR-004: Dry-Run Preview

**Priority**: P0 (MVP Blocker)

**User Story**: As a cautious developer, I want to see what would be updated before actually updating, so I can avoid surprises.

**Acceptance Criteria**:
- `--dry-run` flag shows planned changes without executing
- Display: packages to update, version changes, download sizes
- Execution time estimate
- Dry-run completes in < 30 seconds
- No actual package installations

**Output Example**:
```
🔍 Performing pre-flight checks...
📦 Planned Updates (dry run):

brew:
  • node: 20.11.0 → 20.11.1 (24.8MB)
  • git: 2.43.0 → 2.43.1 (8.4MB)

npm:
  • typescript: 5.3.2 → 5.3.3 (2.1MB)

Total: 3 packages, 35.1MB download
Estimated time: 2m 30s

Run without --dry-run to apply changes
```

**Related Requirements**: REQ-UC001-003

---

### FR-005: Enhanced Terminal Output

**Priority**: P1 (High - MVP Recommended)

**User Story**: As a developer, I want beautiful, informative output that clearly shows what's happening and what I need to do, so I can quickly understand the update status.

**Acceptance Criteria**:
- Unicode box drawing for section separators
- Emoji indicators: ✅ success, ❌ error, ⚠️ warning, 💡 info
- Color coding: green (success), red (error), yellow (warning)
- Progress tracking: `[3/5]` manager count
- Version changes: `package: old → new (size)`
- Summary with statistics and recommendations
- 95%+ compliance with specification examples

**Design Principles**:
- Scannable: Developers should find key info in < 5 seconds
- Actionable: Errors include specific fix instructions
- Informative: Show why decisions were made

**Related Requirements**: REQ-UC001-004

---

### FR-006: Machine-Readable JSON Output

**Priority**: P1 (High - MVP Recommended)

**User Story**: As a DevOps engineer, I want JSON output for integration with CI/CD pipelines and automation tools.

**Acceptance Criteria**:
- `--output json` produces valid JSON
- Schema-validated structure
- Contains all data from text output
- Parseable with `jq` and standard tools
- Supports piping to other commands

**Use Cases**:
- CI/CD: Fail builds if critical updates available
- Monitoring: Track package versions across fleet
- Reporting: Generate update compliance reports

**Related Requirements**: REQ-UC001-005

---

### FR-007: Duplicate Binary Detection

**Priority**: P2 (Medium - Post-MVP)

**User Story**: As a developer, I want to know when the same tool is installed by multiple package managers, so I can avoid path conflicts and confusion.

**Acceptance Criteria**:
- `--check-duplicates` flag enables detection
- Scan: `/usr/local/bin`, `~/.asdf/shims`, `/usr/bin`, `~/go/bin`
- Report binaries managed by multiple managers
- Provide recommendations (e.g., "use asdf for node, uninstall from brew")
- Non-blocking (updates continue with warnings)

**Related Requirements**: REQ-UC001-006

---

### FR-008: Environment Awareness

**Priority**: P1 (High - MVP Recommended)

**User Story**: As a Python developer, I want pmctl to detect when I'm in a conda environment, so it doesn't break my environment with pip updates.

**Acceptance Criteria**:
- Detect conda/mamba environments (check `$CONDA_DEFAULT_ENV`)
- Detect Python virtualenvs (check `$VIRTUAL_ENV`)
- Detect Docker containers (check `/.dockerenv`)
- Warn before pip updates in conda
- Provide override: `--pip-allow-conda`

**Related Requirements**: REQ-UC001-007

---

### FR-009: Bootstrap Missing Managers

**Priority**: P1 (High - MVP Recommended)

**User Story**: As a new team member, I want to install all required package managers automatically, so I can get productive quickly.

**Acceptance Criteria**:
- `pmctl bootstrap` installs missing managers
- Support: homebrew, asdf, nvm, rbenv, pyenv, sdkman
- Platform-specific installation methods
- Dependency resolution (install dependencies first)
- Post-install configuration (add to PATH, set defaults)
- Dry-run support

**Example**:
```bash
# New machine setup
pmctl bootstrap  # Installs homebrew, asdf
pmctl update --all --dry-run  # Preview what would update
pmctl update --all  # Actually update
```

**Related Requirements**: REQ-BOOTSTRAP-001, REQ-BOOTSTRAP-002

---

### FR-010: Status and Health Check

**Priority**: P0 (MVP Blocker)

**User Story**: As a developer, I want to quickly see which package managers are installed and their health status.

**Acceptance Criteria**:
- `pmctl status` shows all managers with versions
- Indicate: supported, installed, version, package count
- Platform-specific availability
- Health checks: valid binary, valid config, network connectivity
- Cached results (24-hour TTL) with `--refresh` override

**Related Requirements**: REQ-STATUS-001

---

## Non-Functional Requirements

### NFR-001: Performance

**Response Time Goals**:
- Manager detection: < 5 seconds
- Status check: < 2 seconds (with cache)
- Dry-run: < 30 seconds
- Single manager update: 30s - 5min (network-dependent)

**Resource Constraints**:
- Memory: < 500MB peak usage
- CPU: Efficient (not a CPU-bound operation)
- Disk: Clean temp files after operations

**Rationale**: Developers will abandon slow tools. Sub-second responses for common operations maintain flow state.

**Related Requirements**: REQ-NFR-PERF-001, REQ-NFR-PERF-002

---

### NFR-002: Reliability

**Error Handling**:
- Graceful degradation on network failures
- Retry logic: 3 attempts with exponential backoff
- Transaction-like updates (rollback on failure)
- Lock files prevent concurrent updates

**Data Safety**:
- Never delete user data without confirmation
- Backup configs before modifications
- Atomic file operations
- Verify download checksums

**Rationale**: Package manager tool must be rock-solid. Data loss or corruption destroys trust.

**Related Requirements**: REQ-NFR-RELIABILITY-001, REQ-NFR-RELIABILITY-002

---

### NFR-003: Usability

**Error Messages**:
- Clear, actionable language
- Suggest specific fixes
- Include context (what, why, how to fix)
- Link to documentation

**Example**:
```
❌ docker: Insufficient disk space (need 1.2GB, available: 800MB)
   • Fix: brew cleanup or free disk space
   • Run: df -h to check disk usage
   • Documentation: https://docs.pmctl.dev/disk-space
```

**Help Documentation**:
- Comprehensive `--help` for all commands
- Man pages on Unix systems
- Shell completion (bash, zsh, fish)

**Rationale**: Great tools disappear into workflows. Poor UX creates support burden.

**Related Requirements**: REQ-NFR-USABILITY-001, REQ-NFR-USABILITY-002

---

### NFR-004: Security

**Privilege Management**:
- Detect when sudo is required
- Never auto-elevate privileges
- Prompt for confirmation before sudo
- Document why privileges needed

**Network Security**:
- HTTPS for all downloads
- TLS certificate validation
- Proxy support (HTTP_PROXY, HTTPS_PROXY)
- Verify package signatures

**Rationale**: Package managers are high-value attack targets. Security must be built-in, not bolted-on.

**Related Requirements**: REQ-NFR-SECURITY-001, REQ-NFR-SECURITY-002

---

### NFR-005: Compatibility

**Platform Support**:
- macOS: 12.0+ (Monterey and later)
- Linux: Ubuntu 20.04+, Arch, Fedora 36+, Alpine
- Windows: WSL2 only (native Windows in future)
- Architecture: amd64, arm64 (Apple Silicon)

**Go Version**:
- Minimum: Go 1.26.0
- No CGO dependencies (pure Go)
- Cross-compilation support

**Partial Compatibility with gzh-cli**:
- Core functionality maintained
- Configuration file format compatible
- JSON output structure preserved
- Command naming evolved (`pmctl` vs `gz pm`)

**Rationale**: Developers use diverse platforms. Cross-platform support is table stakes.

**Related Requirements**: REQ-NFR-COMPAT-001, REQ-NFR-COMPAT-002

---

### NFR-006: Maintainability

**Code Quality**:
- Test coverage: ≥ 90% overall
- Cyclomatic complexity: < 15 per function
- File size: < 500 lines (< 10KB for LLM-friendliness)
- Clean Architecture layers with no dependency violations

**Documentation**:
- All public APIs documented (godoc)
- 5+ Architecture Decision Records
- Comprehensive README
- Contributing guide
- Plugin development guide

**Rationale**: This is a long-term project (5+ year lifespan). Maintainability determines whether it thrives or becomes legacy.

**Related Requirements**: REQ-NFR-MAINTAINABILITY-001, REQ-NFR-MAINTAINABILITY-002

---

## User Journey: First-Time User

### Scenario: Alex sets up a new MacBook

**Step 1: Installation** (30 seconds)
```bash
brew install pmctl
# OR
go install github.com/gizzahub/gzh-cli-package-manager/cmd/pmctl@latest
```

**Step 2: Status Check** (5 seconds)
```bash
pmctl status
```

**Output**:
```
📋 Package Manager Status:
MANAGER      SUPPORTED  INSTALLED  VERSION    PACKAGES
------------ ---------- ---------- ---------- --------
brew         ✅         ✅         4.2.0      12
asdf         ✅         ⛔         N/A        N/A
npm          ✅         ✅         10.2.4     8
pip          ✅         ✅         24.0       35

💡 Tip: Install asdf for better version management
Run: pmctl bootstrap --manager asdf
```

**Step 3: Bootstrap Missing Managers** (2 minutes)
```bash
pmctl bootstrap --manager asdf
```

**Step 4: First Update** (3 minutes)
```bash
pmctl update --all --dry-run  # Preview
pmctl update --all  # Execute
```

**Total Time**: ~6 minutes from zero to fully updated

**User Feedback**: "This is so much easier than my old update script!"

---

## Competitive Landscape

### Direct Competitors

**None**: No direct competitors exist for multi-package-manager orchestration.

### Indirect Competitors / Alternatives

1. **Shell Scripts**: Developers write custom `update.sh`
   - **Pros**: Customized, no dependencies
   - **Cons**: Brittle, platform-specific, no error handling

2. **Ansible/Chef/Puppet**: Configuration management tools
   - **Pros**: Powerful, infrastructure-scale
   - **Cons**: Overkill for laptop, steep learning curve

3. **Homebrew Bundle**: Manages multiple Homebrew packages
   - **Pros**: Integrated with brew
   - **Cons**: macOS-only, brew-centric, doesn't coordinate other managers

4. **Topgrade**: Multi-tool updater (Rust)
   - **Pros**: Similar concept
   - **Cons**: Different design philosophy, less environment awareness

### Competitive Advantages

1. **Clean Architecture**: Maintainable, testable, extensible
2. **Environment Awareness**: Conda detection, virtualenv support
3. **Rich Output**: 95% spec compliance, beautiful UX
4. **Go-based**: Single binary, fast, cross-platform
5. **Specification-First**: Comprehensive docs before code

---

## Success Criteria

### Launch Criteria (v1.0)

**Functionality**:
- [ ] All P0 requirements implemented and tested
- [ ] 80%+ P1 requirements implemented
- [ ] 90%+ test coverage achieved

**Quality**:
- [ ] Zero known critical bugs
- [ ] 95%+ specification compliance (enhanced output)
- [ ] Integration tests pass on macOS, Ubuntu, Arch

**Documentation**:
- [ ] README complete with quick start
- [ ] ARCHITECTURE.md documents design
- [ ] 5+ ADRs written
- [ ] API documentation generated
- [ ] Migration guide from gzh-cli

**User Validation**:
- [ ] 10+ beta testers complete onboarding
- [ ] 80%+ beta testers rate "satisfied" or "very satisfied"
- [ ] < 3 "critical" severity issues from beta

---

## Release Roadmap

### v1.0 (MVP) - Week 9
**Theme**: Core Functionality

**Features**:
- Multi-manager update orchestration
- Automatic manager detection
- Update strategies (latest, stable, minor)
- Dry-run preview
- Enhanced terminal output
- JSON output format
- Environment awareness (conda, virtualenv)
- Bootstrap missing managers
- Status and health check

**Platforms**: macOS, Ubuntu, Arch Linux

---

### v1.1 - Month 3
**Theme**: Advanced Features

**Features**:
- Duplicate binary detection
- Version synchronization (nvm ↔ npm, etc.)
- Package export/import
- Cache management
- Health diagnostics
- Shell completion (bash, zsh, fish)

---

### v1.2 - Month 5
**Theme**: Team Collaboration

**Features**:
- Shared configuration profiles
- Team environment templates
- Compliance reporting
- Update scheduling
- Notification integrations (Slack, email)

---

### v2.0 - Month 9
**Theme**: Ecosystem Expansion

**Features**:
- Plugin system for custom package managers
- Native Windows support (non-WSL)
- GUI interface (optional)
- Cloud configuration sync
- Advanced analytics and insights

---

## Risks & Mitigation

### Risk 1: Package Manager API Changes
**Impact**: High | **Probability**: Medium

**Description**: Homebrew, apt, npm may change CLI output format, breaking parsing.

**Mitigation**:
- Version-specific parsers
- Comprehensive integration tests
- Fallback to previous parser versions
- Community contribution for new versions

---

### Risk 2: Platform Fragmentation
**Impact**: Medium | **Probability**: High

**Description**: Supporting macOS, Ubuntu, Arch, Fedora, Alpine increases complexity.

**Mitigation**:
- Platform abstraction layer in architecture
- Docker-based integration tests for each platform
- Community testing on diverse systems

---

### Risk 3: Security Vulnerabilities
**Impact**: Critical | **Probability**: Low

**Description**: Compromised package downloads or privilege escalation bugs.

**Mitigation**:
- Security audit before v1.0
- Checksum verification for downloads
- No auto-elevation of privileges
- Regular dependency updates

---

### Risk 4: Low Adoption
**Impact**: High | **Probability**: Medium

**Description**: Developers continue using shell scripts instead of pmctl.

**Mitigation**:
- Exceptional UX (beautiful output, great docs)
- Show value immediately (first run should impress)
- Community building (blog posts, talks, demos)
- Integration with popular tools (mise, devenv)

---

## Open Questions

1. **Should we support Windows natively** (non-WSL) in v1.0 or defer to v2.0?
   - **Decision**: Defer to v2.0 (WSL2 sufficient for MVP)

2. **Should we build a plugin system** for custom package managers?
   - **Decision**: v2.0 feature (architecture supports it, but not priority)

3. **Should we offer a hosted service** for configuration sync?
   - **Decision**: Not in scope for v1.x (CLI-first product)

4. **How do we handle breaking changes** in package managers (e.g., Homebrew 5.0)?
   - **Decision**: Version-specific adapters + community contributions

---

## Appendix

### Glossary

- **Package Manager**: Tool that installs, updates, removes software (brew, apt, npm)
- **Version Manager**: Tool that manages multiple versions of a tool (asdf, nvm, rbenv)
- **Orchestration**: Coordinating multiple tools to work together
- **Dry-Run**: Preview operation without executing
- **Bootstrap**: Initial setup/installation

### References

- **Requirements**: `/REQUIREMENTS.md`
- **Architecture**: `/ARCHITECTURE.md`
- **Specifications**: `/docs/specifications/`
- **Original gzh-cli**: [Gizzahub/gzh-cli](https://github.com/Gizzahub/gzh-cli)

---

**Document Control**:
- **Author**: Claude Code (AI-assisted)
- **Product Owner**: TBD
- **Reviewers**: TBD
- **Approval**: TBD
- **Next Review**: 2025-02-03
