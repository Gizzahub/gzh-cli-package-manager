# Requirements Specification

> **Version**: 1.0.0
> **Last Updated**: 2025-01-27
> **Status**: Draft

## Document Overview

This document specifies all functional and non-functional requirements for the `gzh-cli-package-manager` (CLI binary: `gz-pm`), a standalone package manager orchestration tool extracted from gzh-cli.

**Scope**: Complete extraction of PM functionality (~11,662 lines) with clean architecture.

**Organization**: Requirements are categorized by feature area and prioritized for MVP (v1.0) release.

---

## 1. Functional Requirements

### 1.1 Update Command (UC-001)

#### REQ-UC001-001: Multi-Manager Update Support
**Priority**: P0 (Critical)
**Phase**: MVP
**Source**: [UC-001-update-enhanced.md](docs/specifications/use-cases/UC-001-update-enhanced.md)

**Description**: Support updating multiple package managers in a single command execution.

**Acceptance Criteria**:
- [ ] Detect all installed package managers automatically
- [ ] Support `--all` flag to update all detected managers
- [ ] Support `--manager <name>` for single manager update
- [ ] Support `--managers <csv>` for multiple specific managers
- [ ] Execute updates sequentially with status reporting
- [ ] Handle failures gracefully (continue with remaining managers)
- [ ] Exit code: 0 (success), 1 (partial), 2 (complete failure)

**Test Scenarios**: Test 1.1, 1.2, 1.3

---

#### REQ-UC001-002: Update Strategies
**Priority**: P0 (Critical)
**Phase**: MVP
**Source**: [UC-001-update-enhanced.md](docs/specifications/use-cases/UC-001-update-enhanced.md#L

18)

**Description**: Support multiple version update strategies to control update behavior.

**Acceptance Criteria**:
- [ ] Default strategy: `stable` (latest stable release only)
- [ ] `--strategy latest`: Update to absolute latest (including beta/rc)
- [ ] `--strategy stable`: Latest stable release (default)
- [ ] `--strategy minor`: Latest minor/patch, no major upgrades
- [ ] `--strategy fixed`: Show available updates but don't install
- [ ] Per-manager strategy override via configuration

**Configuration Example**:
```yaml
# ~/.config/gz-pm/config.yml
update:
  defaultStrategy: stable
  managerOverrides:
    brew: latest       # Homebrew always latest
    pip: minor         # Python packages conservative
    sdkman: stable     # SDKMAN stable only
```

**Test Scenarios**: Test 8.1, 8.2, 8.3

---

#### REQ-UC001-003: Dry-Run Mode
**Priority**: P0 (Critical)
**Phase**: MVP

**Description**: Preview update actions without executing them.

**Acceptance Criteria**:
- [ ] `--dry-run` flag shows planned update actions
- [ ] Display packages that would be updated with version changes
- [ ] Show disk space requirements before download
- [ ] Execution time estimate for updates
- [ ] No actual package installations or modifications
- [ ] Dry-run completes in < 30 seconds

**Test Scenarios**: Test 7.1

---

#### REQ-UC001-004: Enhanced Output Formatting
**Priority**: P1 (High)
**Phase**: MVP
**Source**: [UC-001-update-enhanced.md](docs/specifications/use-cases/UC-001-update-enhanced.md#L36)

**Description**: Provide rich, human-readable output with progress indication.

**Acceptance Criteria**:
- [ ] Section banners with Unicode box drawing: `═══════════ 🚀 [1/5] brew — Updating ═══════════`
- [ ] Emoji indicators: ✅ (success), ❌ (failure), ⚠️ (warning), 💡 (info), 🔄 (progress)
- [ ] Color-coded output: green (success), red (error), yellow (warning)
- [ ] Version changes format: `package: old_version → new_version (size)`
- [ ] Progress tracking with current/total: `[3/5]`
- [ ] Summary section with statistics and recommendations
- [ ] Execution time tracking: `⏰ Update completed in 3m 42s`

**Output Format Specification**: 95% compliance with spec examples

**Test Scenarios**: Test 2.1, 2.3

---

#### REQ-UC001-005: JSON Output Format
**Priority**: P1 (High)
**Phase**: MVP

**Description**: Machine-readable JSON output for automation and CI/CD integration.

**Acceptance Criteria**:
- [ ] `--output json` flag produces valid JSON
- [ ] JSON contains all data from text output
- [ ] Parseable with `jq` and standard JSON tools
- [ ] Schema-validated structure

**JSON Schema**:
```json
{
  "timestamp": "2025-01-27T10:30:00Z",
  "managers": [
    {
      "name": "brew",
      "status": "success|partial|failed",
      "packages_updated": 5,
      "packages": [
        {
          "name": "node",
          "old_version": "20.11.0",
          "new_version": "20.11.1",
          "size_mb": 24.8
        }
      ],
      "errors": []
    }
  ],
  "summary": {
    "total_managers": 5,
    "successful": 5,
    "failed": 0,
    "packages_upgraded": 27,
    "download_size_mb": 52.1,
    "disk_freed_mb": 245,
    "duration_seconds": 222
  }
}
```

**Test Scenarios**: Test 2.2, 2.3

---

#### REQ-UC001-006: Duplicate Binary Detection
**Priority**: P2 (Medium)
**Phase**: v1.1
**Source**: [UC-001-update-enhanced.md](docs/specifications/use-cases/UC-001-update-enhanced.md#L50)

**Description**: Detect and report duplicate binary installations across managers.

**Acceptance Criteria**:
- [ ] `--check-duplicates` flag enables duplicate detection
- [ ] Scan common binary paths: `/usr/local/bin`, `~/.asdf/shims`, etc.
- [ ] Report binaries managed by multiple package managers
- [ ] Provide recommendations for resolving conflicts
- [ ] Non-blocking (updates continue with warnings)

**Example Output**:
```
🧪 Duplicate Installation Check:
Found 2 potential conflicts:
  • node: /usr/local/bin/node (brew), ~/.asdf/shims/node (asdf)
  • python3: /usr/bin/python3 (system), ~/.asdf/shims/python3 (asdf)

💡 Recommended actions:
  • Consider switching node to single manager to avoid conflicts
```

**Test Scenarios**: Test 5.3

---

#### REQ-UC001-007: Environment Awareness
**Priority**: P1 (High)
**Phase**: MVP
**Source**: [UC-001-update-enhanced.md](docs/specifications/use-cases/UC-001-update-enhanced.md#L159)

**Description**: Detect and handle special environments (conda, virtualenv, Docker).

**Acceptance Criteria**:
- [ ] Detect active conda/mamba environments
- [ ] Warn about pip updates in conda environments
- [ ] Provide override flag: `--pip-allow-conda`
- [ ] Detect Python virtual environments
- [ ] Detect Docker container execution
- [ ] Adjust update strategy based on environment

**Example Behavior**:
```
═══════════ ⚠️ [4/4] pip — SKIP ═══════════
⚠️  Conda environment detected: /opt/miniconda3/envs/myproject
   • pip updates in conda environments can cause dependency conflicts
   • Use conda/mamba for package management instead
   • Override with: gz-pm update --manager pip --pip-allow-conda
```

**Test Scenarios**: Test 5.1, 5.2

---

### 1.2 Status Command

#### REQ-STATUS-001: Manager Detection
**Priority**: P0 (Critical)
**Phase**: MVP

**Description**: Detect and report installation status of all supported package managers.

**Acceptance Criteria**:
- [ ] Detect installed package managers automatically
- [ ] Report manager version information
- [ ] Show supported vs installed status
- [ ] Platform-specific detection (macOS, Linux, Windows/WSL)
- [ ] Cache detection results (24-hour TTL)
- [ ] `--refresh` flag to bypass cache

**Output Example**:
```
📋 Package Manager Status:
MANAGER      SUPPORTED  INSTALLED  VERSION    PACKAGES
------------ ---------- ---------- ---------- --------
brew         ✅         ✅         4.2.0      47
asdf         ✅         ✅         0.14.0     12
npm          ✅         ✅         10.2.4     35
pip          ✅         ✅         24.0       89
apt          🚫         ⛔         N/A        N/A (macOS only)
```

---

#### REQ-STATUS-002: Health Diagnostics
**Priority**: P2 (Medium)
**Phase**: v1.1

**Description**: Diagnose package manager health and configuration issues.

**Acceptance Criteria**:
- [ ] Check manager binary paths
- [ ] Validate configuration files
- [ ] Detect common misconfigurations
- [ ] Report permission issues
- [ ] Suggest fixes for problems

---

### 1.3 Bootstrap Command

#### REQ-BOOTSTRAP-001: Manager Installation
**Priority**: P1 (High)
**Phase**: MVP

**Description**: Automatically install missing package managers.

**Acceptance Criteria**:
- [ ] Support bootstrapping: homebrew, asdf, nvm, rbenv, pyenv, sdkman
- [ ] Platform-specific installation methods
- [ ] Dependency resolution (install dependencies first)
- [ ] Dry-run support for installation preview
- [ ] Configuration after installation
- [ ] Validation of successful installation

**Supported Managers**:
- **Homebrew**: macOS and Linux
- **ASDF**: Multi-platform version manager
- **NVM**: Node.js version manager
- **Rbenv**: Ruby version manager
- **Pyenv**: Python version manager
- **SDKMAN**: Java/JVM ecosystem manager

---

#### REQ-BOOTSTRAP-002: Post-Install Configuration
**Priority**: P1 (High)
**Phase**: MVP

**Description**: Configure package managers after installation.

**Acceptance Criteria**:
- [ ] Add to PATH in shell configuration
- [ ] Set default versions for version managers
- [ ] Configure package manager repositories
- [ ] Run post-install hooks
- [ ] Verify configuration success

---

### 1.4 Sync Command

#### REQ-SYNC-001: Version Synchronization
**Priority**: P2 (Medium)
**Phase**: v1.1

**Description**: Synchronize versions between version managers and package managers.

**Acceptance Criteria**:
- [ ] Detect version mismatches: nvm ↔ npm, pyenv ↔ pip, rbenv ↔ gem
- [ ] Sync strategies: vm_priority, pm_priority, latest
- [ ] Auto-fix option with backup
- [ ] User confirmation before changes
- [ ] Rollback on failure

**Sync Pairs**:
- NVM ↔ NPM
- Pyenv ↔ Pip
- Rbenv ↔ Gem
- ASDF ↔ Multiple plugins

---

### 1.5 Export/Import Commands

#### REQ-EXPORT-001: Package List Export
**Priority**: P2 (Medium)
**Phase**: v1.1

**Description**: Export currently installed packages to configuration files.

**Acceptance Criteria**:
- [ ] Export all managers: `gz-pm export --all`
- [ ] Export specific manager: `gz-pm export --manager brew`
- [ ] Output formats: YAML, JSON
- [ ] Include version information
- [ ] Include manager-specific metadata

---

#### REQ-IMPORT-001: Package Installation from Config
**Priority**: P2 (Medium)
**Phase**: v1.1

**Description**: Install packages from exported configuration files.

**Acceptance Criteria**:
- [ ] Read configuration from `~/.config/gz-pm/*.yml`
- [ ] Install missing packages
- [ ] Respect version constraints
- [ ] Handle dependency resolution
- [ ] Report installation success/failures

---

### 1.6 Cache Management

#### REQ-CACHE-001: Cache Operations
**Priority**: P2 (Medium)
**Phase**: v1.1

**Description**: Manage package manager caches to free disk space.

**Acceptance Criteria**:
- [ ] `gz-pm cache status`: Show cache sizes
- [ ] `gz-pm cache clean`: Clean all caches
- [ ] `gz-pm cache clean --manager brew`: Clean specific manager
- [ ] Dry-run support
- [ ] Report space freed

**Supported Manager Caches**:
- go, npm, yarn, pnpm, pip, poetry, cargo, brew, apt, dnf, yum, pacman

---

## 2. Non-Functional Requirements

### 2.1 Performance

#### REQ-NFR-PERF-001: Response Time
**Priority**: P1 (High)
**Phase**: MVP

**Requirements**:
- Manager detection: < 5 seconds
- Single manager update: 30 seconds - 5 minutes (network-dependent)
- All managers update: 2-15 minutes (varies by package count)
- Dry-run analysis: < 30 seconds
- Status check: < 2 seconds (with cache)

---

#### REQ-NFR-PERF-002: Resource Usage
**Priority**: P1 (High)
**Phase**: MVP

**Requirements**:
- Memory usage: < 500MB peak
- CPU: Moderate during downloads, efficient during processing
- Network: Respect rate limits, support proxy configuration
- Disk: Temporary space cleaned after operations

---

### 2.2 Compatibility

#### REQ-NFR-COMPAT-001: Platform Support
**Priority**: P0 (Critical)
**Phase**: MVP

**Requirements**:
- **macOS**: 12.0+ (Monterey and later)
- **Linux**: Ubuntu 20.04+, Arch Linux, Fedora 36+, Alpine Linux
- **Windows**: WSL2 only (native Windows in future)
- **Architecture**: amd64, arm64 (Apple Silicon)

---

#### REQ-NFR-COMPAT-002: Go Version
**Priority**: P0 (Critical)
**Phase**: MVP

**Requirements**:
- Minimum: Go 1.24.0 (Go 1.26.7 recommended for development and regular CI)
- No CGO dependencies (pure Go)
- Cross-compilation support for all platforms

---

### 2.3 Usability

#### REQ-NFR-USABILITY-001: Error Messages
**Priority**: P1 (High)
**Phase**: MVP

**Requirements**:
- Clear, actionable error messages
- Suggest fixes for common problems
- Include context: what failed, why, how to fix
- Avoid technical jargon when possible
- Provide documentation links for complex issues

**Example**:
```
❌ brew upgrade: Failed to upgrade postgresql
   • Current: 14.9 (via Homebrew)
   • Available: 16.1 (breaking changes)
   • Fix: brew unlink postgresql@14 && brew install postgresql@16
   • Documentation: planned; no public documentation URL is published yet
```

---

#### REQ-NFR-USABILITY-002: Help Documentation
**Priority**: P1 (High)
**Phase**: MVP

**Requirements**:
- Comprehensive `--help` for all commands
- Examples in help output
- Man pages for Unix systems
- Shell completion (bash, zsh, fish)

---

### 2.4 Reliability

#### REQ-NFR-RELIABILITY-001: Error Recovery
**Priority**: P1 (High)
**Phase**: MVP

**Requirements**:
- Graceful degradation on network failures
- Retry logic for transient errors (3 attempts)
- Transaction-like updates (rollback on failure)
- State recovery after interruption
- Lock files to prevent concurrent updates

---

#### REQ-NFR-RELIABILITY-002: Data Safety
**Priority**: P0 (Critical)
**Phase**: MVP

**Requirements**:
- Never delete user data without confirmation
- Backup configuration before modifications
- Atomic file operations (write to temp, then rename)
- Validate checksums for downloads
- Verify package signatures where available

---

### 2.5 Security

#### REQ-NFR-SECURITY-001: Privilege Escalation
**Priority**: P0 (Critical)
**Phase**: MVP

**Requirements**:
- Detect when sudo is required
- Never auto-elevate privileges
- Prompt user for confirmation before sudo operations
- Document why elevated privileges are needed

---

#### REQ-NFR-SECURITY-002: Network Security
**Priority**: P1 (High)
**Phase**: MVP

**Requirements**:
- HTTPS for all package downloads
- TLS certificate validation
- Proxy support (HTTP_PROXY, HTTPS_PROXY environment variables)
- Respect system certificate store

---

### 2.6 Maintainability

#### REQ-NFR-MAINTAINABILITY-001: Code Quality
**Priority**: P1 (High)
**Phase**: MVP

**Requirements**:
- Test coverage: ≥ 90% overall
  - Domain layer: ≥ 95%
  - Application layer: ≥ 90%
  - Infrastructure layer: ≥ 85%
- Cyclomatic complexity: < 15 per function
- File size: < 500 lines per file (< 10KB recommended)
- No dependency violations between layers

---

#### REQ-NFR-MAINTAINABILITY-002: Documentation
**Priority**: P1 (High)
**Phase**: MVP

**Requirements**:
- All public APIs documented with godoc
- Architecture Decision Records for major decisions (5+ ADRs)
- Comprehensive README with quick start
- Developer contributing guide
- Plugin development guide (adding new package managers)

---

## 3. Requirement Traceability Matrix

| Requirement | Specification | Test Scenarios | Implementation | Status |
|-------------|---------------|----------------|----------------|--------|
| REQ-UC001-001 | UC-001 §2.1 | Test 1.1, 1.2, 1.3 | cmd/update/update.go | 🟡 Planned |
| REQ-UC001-002 | UC-001 §3.1 | Test 8.1, 8.2, 8.3 | pkg/domain/update/strategy.go | 🟡 Planned |
| REQ-UC001-003 | UC-001 §2.2 | Test 7.1 | pkg/application/update/usecase.go | 🟡 Planned |
| REQ-UC001-004 | UC-001 §4.1 | Test 2.1, 2.3 | cmd/gz-pm/formatter/enhanced.go | 🟡 Planned |
| REQ-UC001-005 | UC-001 §4.2 | Test 2.2 | cmd/gz-pm/formatter/json.go | 🟡 Planned |
| REQ-UC001-006 | UC-001 §5.1 | Test 5.3 | pkg/domain/diagnostics/duplicates.go | 🟡 Planned |
| REQ-UC001-007 | UC-001 §6.1 | Test 5.1, 5.2 | pkg/infrastructure/detector/environment.go | 🟡 Planned |

**Legend**:
- ✅ Implemented
- 🟡 Planned
- 🔴 Blocked

---

## 4. Requirement Priorities

### P0 - Critical (MVP Blockers)
Must be implemented for v1.0 release:
- REQ-UC001-001: Multi-Manager Update
- REQ-UC001-002: Update Strategies
- REQ-UC001-003: Dry-Run Mode
- REQ-STATUS-001: Manager Detection
- REQ-BOOTSTRAP-001: Manager Installation
- All NFR-COMPAT requirements
- All NFR-SECURITY requirements

### P1 - High (MVP Recommended)
Strongly recommended for v1.0:
- REQ-UC001-004: Enhanced Output
- REQ-UC001-005: JSON Output
- REQ-UC001-007: Environment Awareness
- REQ-BOOTSTRAP-002: Post-Install Config
- All NFR-PERF requirements
- All NFR-USABILITY requirements
- All NFR-RELIABILITY requirements
- All NFR-MAINTAINABILITY requirements

### P2 - Medium (Post-MVP)
Planned for v1.1:
- REQ-UC001-006: Duplicate Detection
- REQ-STATUS-002: Health Diagnostics
- REQ-SYNC-001: Version Synchronization
- REQ-EXPORT-001: Package Export
- REQ-IMPORT-001: Package Import
- REQ-CACHE-001: Cache Management

### P3 - Low (Future)
Future enhancements (v2.0+):
- Plugin system for custom package managers
- GUI interface
- Cloud configuration sync
- Team collaboration features

---

## 5. Out of Scope (Explicitly Not Included)

The following are **not** included in v1.0:

1. **Native Windows Support**: Only WSL2 supported initially
2. **GUI Interface**: CLI only
3. **Package Creation**: Only consumption, not authoring packages
4. **Repository Hosting**: Relies on existing package repositories
5. **Telemetry**: No usage tracking or analytics
6. **Auto-Updates**: User must manually update gz-pm itself

---

## 6. Acceptance Criteria Summary

Before v1.0 release:
- [ ] All P0 requirements implemented and tested
- [ ] 80%+ P1 requirements implemented
- [ ] 90%+ overall test coverage achieved
- [ ] All critical NFRs met (performance, security, compatibility)
- [ ] Documentation complete (README, ARCHITECTURE, ADRs, API docs)
- [ ] Integration tests pass on all supported platforms
- [ ] Specification compliance: 95%+ for enhanced output format
- [ ] Beta testing completed with 10+ external users
- [ ] Migration guide from gzh-cli validated

---

## 7. References

- **Specifications**: `/docs/specifications/`
- **Test Scenarios**: `/docs/specifications/test-scenarios.md`
- **Architecture**: `/ARCHITECTURE.md`
- **ADRs**: `/docs/10-architecture/adr/`
- **Original gzh-cli specs**: [Gizzahub/gzh-cli](https://github.com/Gizzahub/gzh-cli)

---

**Document Control**:
- **Author**: Claude Code (AI-assisted)
- **Reviewers**: TBD
- **Approval**: TBD
- **Next Review**: 2025-02-01
