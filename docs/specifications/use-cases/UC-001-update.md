# UC-001: Update Package Managers

## Scenario: Update all package managers and their packages

### Input

**Command**:

```bash
gz-pm update
```

**Alternative Invocations**:

```bash
gz-pm update --all                      # Explicit all (same as default)
gz-pm update --manager brew             # Single manager
gz-pm update --managers brew,asdf,npm   # Multiple specific managers
gz-pm update --all --dry-run             # Preview changes without executing
gz-pm update --all --output json        # Machine-readable JSON output
```

**Prerequisites**:

- [ ] Package managers installed (asdf, brew, npm, etc.)
- [ ] Network connectivity
- [ ] Admin permissions (for system-wide package managers like apt, pacman)
- [ ] Sufficient disk space for downloads and cache

### Expected Output

**Success Case**:

```text
🔄 Updating package managers...

📦 Homebrew
✅ brew update: Updated 23 formulae
✅ brew upgrade: Upgraded 5 packages
   - node: 20.11.0 -> 20.11.1
   - git: 2.43.0 -> 2.43.1
   - python@3.11: 3.11.7 -> 3.11.8

📦 asdf
✅ asdf update: Updated to latest version
✅ asdf plugin update --all: 8 plugins updated
✅ Available updates:
   - golang: 1.21.5 -> 1.21.6 (use: asdf install golang 1.21.6)
   - nodejs: 20.11.0 -> 20.11.1 (use: asdf install nodejs 20.11.1)

📦 Node.js (npm)
✅ npm update -g: 12 global packages updated
   - @angular/cli: 17.0.7 -> 17.0.8
   - typescript: 5.3.2 -> 5.3.3

📦 Python (pip)
✅ pip install --upgrade pip: Updated to 24.0
✅ pip list --outdated: 6 packages can be updated
   - requests: 2.31.0 -> 2.32.0
   - numpy: 1.24.3 -> 1.24.4

🎉 Package manager updates completed!
📋 Manual action needed:
   - Update asdf language versions (commands shown above)
   - Consider updating pip packages: pip install --upgrade <package>

Exit Code: 0
```

**Partial Success (Some Failures)**:

```text
🔄 Updating package managers...

📦 Homebrew
✅ brew update: Updated 15 formulae
❌ brew upgrade: Failed to upgrade 2 packages
   - postgresql: version conflict (manual intervention needed)
   - docker: insufficient disk space

📦 SDKMAN
❌ Network error: Cannot reach SDKMAN servers
💡 Check network connection and try again later

📦 Node.js (npm)
✅ npm update -g: 8 global packages updated

⚠️  Some updates failed. See details above.
🔧 Manual fixes may be required for failed updates.

Exit Code: 1
```

**No Package Managers Found**:

```text
🔍 Scanning for package managers...

❌ No supported package managers found!

💡 Supported package managers:
   - Homebrew (macOS/Linux): Install from https://brew.sh
   - asdf (Version manager): Install from https://asdf-vm.com
   - SDKMAN (Java ecosystem): Install from https://sdkman.io
   - Node.js npm: Installed with Node.js
   - Python pip: Installed with Python
   - Rust cargo: Installed with Rust
   - apt (Debian/Ubuntu): System package manager
   - pacman (Arch/Manjaro): System package manager

🚫 Nothing to update.

Exit Code: 2
```

**Dry Run Example**:

```text
🔍 [DRY RUN] Analyzing update actions...

📦 Homebrew
Would execute:
   - brew update
   - brew upgrade
Expected updates:
   - node: 20.11.0 -> 20.11.1 (24.8MB download)
   - git: 2.43.0 -> 2.43.1 (8.4MB download)

📦 asdf
Would execute:
   - asdf plugin update --all
   - asdf update
Expected updates:
   - 8 plugins would be updated

📋 Summary:
   - Managers to update: 2
   - Packages to upgrade: 5
   - Total download size: ~33MB
   - Estimated time: 2-4 minutes

⚠️  This was a dry run. No changes were made.
To execute: gz-pm update --all

Exit Code: 0
```

### Side Effects

**Files Created**:

- `~/.gz-pm/logs/update-<timestamp>.log` - Detailed update log
- `~/.gz-pm/state/last-update.json` - Last update metadata
- Package manager cache files (various locations)

**Files Modified**:

- Package manager databases updated
- Installed packages upgraded
- Package manager configuration files (if self-update occurs)

**State Changes**:

- Package manager databases refreshed
- Available package versions updated
- Some packages upgraded automatically
- System PATH potentially modified for new tool versions

### Validation

**Automated Tests**:

```bash
# Test update command (requires actual package managers)
result=$(gz-pm update 2>&1)
exit_code=$?

# Should find at least one package manager in CI/test environment
assert_not_contains "$result" "No supported package managers found"
assert_contains "$result" "Updating package managers"

# Check log file creation
assert_file_exists "$HOME/.gz-pm/logs/update-*.log"

# Test dry-run mode
dry_result=$(gz-pm update --all --dry-run 2>&1)
assert_contains "$dry_result" "DRY RUN"
assert_contains "$dry_result" "No changes were made"

# Test specific manager selection
brew_result=$(gz-pm update --manager brew 2>&1)
assert_contains "$brew_result" "Homebrew"
assert_not_contains "$brew_result" "asdf"  # Other managers not processed

# Test JSON output
json_result=$(gz-pm update --all --dry-run --output json 2>&1)
echo "$json_result" | jq . >/dev/null  # Validate JSON format
assert_exit_code 0
```

**Manual Verification**:

1. Run on system with multiple package managers
2. Verify each package manager is properly updated
3. Check that upgrade recommendations are actionable
4. Confirm failed updates are clearly reported
5. Validate exit codes match specification

### Edge Cases

**Network Issues**:

- **Offline environment**: Graceful failure with clear error messages
- **Partial network**: Some package repos accessible, others not
- **Timeout handling**: Slow connections should timeout with retry suggestions
- **Proxy configuration**: Respect HTTP_PROXY and HTTPS_PROXY environment variables

**Permission Issues**:

- **System package managers**: apt, pacman require sudo
  - Error message: "Permission denied. Try: sudo gz-pm update --manager apt"
- **User-space installations**: Homebrew, asdf work without sudo
- **Directory permissions**: Clearly report which directories lack permissions

**Disk Space Issues**:

- **Insufficient space**: Pre-flight check should detect and warn
  - Error: "Insufficient disk space: need 500MB, available 200MB"
  - Suggestion: "Run gz-pm cleanup or free disk space"
- **Cache cleanup**: Automatic or prompted cleanup of old downloads

**Version Conflicts**:

- **Package dependency conflicts**: Report conflicting requirements
- **Breaking changes**: Warn about major version upgrades
- **Rollback instructions**: Provide commands to revert failed updates

**Platform Differences**:

- **macOS**: Primary managers = brew, asdf, npm, pip, sdkman, cargo
- **Linux (Ubuntu)**: apt, asdf, npm, pip, sdkman, cargo
- **Linux (Arch)**: pacman, yay, asdf, npm, pip, cargo
- **Windows**: choco, scoop (future support)

### Performance Expectations

**Response Time**:

- **Manager detection**: < 5 seconds
- **Database updates**: < 30 seconds per package manager
- **Package discovery**: < 10 seconds
- **Full update cycle**: 2-10 minutes depending on updates available
- **Dry-run analysis**: < 30 seconds

**Resource Usage**:

- **Memory**: < 200MB peak
- **Network**: Varies by number of package updates (10MB - 2GB+)
- **Disk**: Temporary cache space for downloads
- **CPU**: Low during waits, moderate during processing

## Command Line Interface

### Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--all` | `-a` | boolean | true | Update all detected managers |
| `--manager` | `-m` | string | - | Update single specific manager |
| `--managers` | | string | - | Comma-separated list of managers to update |
| `--dry-run` | `-n` | boolean | false | Preview changes without executing |
| `--output` | `-o` | string | text | Output format: text, json, simple |
| `--strategy` | `-s` | string | stable | Update strategy: latest, stable, minor, fixed |
| `--yes` | `-y` | boolean | false | Auto-confirm prompts |
| `--quiet` | `-q` | boolean | false | Minimal output |
| `--verbose` | `-v` | boolean | false | Detailed output |

### Exit Codes

| Code | Meaning | When Used |
|------|---------|-----------|
| `0` | Success | All updates completed successfully |
| `1` | Partial failure | Some updates failed, others succeeded |
| `2` | Complete failure | No managers found or all updates failed |
| `3` | Invalid arguments | Wrong flags or arguments provided |
| `4` | Prerequisites not met | Network, permissions, or disk space issues |

## Examples

### Update All Managers

```bash
gz-pm update --all
# or simply
gz-pm update
```

### Update Specific Manager

```bash
gz-pm update --manager brew
```

### Update Multiple Managers

```bash
gz-pm update --managers brew,asdf,npm
```

### Dry Run (Preview Changes)

```bash
gz-pm update --all --dry-run
```

### JSON Output for Scripts

```bash
gz-pm update --all --output json | jq '.managers[].status'
```

### Quiet Mode for Automation

```bash
gz-pm update --all --quiet --yes
```

### Latest Version Strategy

```bash
gz-pm update --all --strategy latest
```

## Integration Points

### Configuration File

Uses `~/.gz-pm/config.yaml` for:
- Default update strategy
- Manager-specific preferences
- Excluded packages or managers
- Output format preferences

### Logging

- **Location**: `~/.gz-pm/logs/update-<timestamp>.log`
- **Format**: Structured logging with timestamps
- **Retention**: Last 10 update sessions (configurable)

### State Management

- **Current state**: `~/.gz-pm/state/current.json`
- **Last update**: `~/.gz-pm/state/last-update.json`
- **Update history**: `~/.gz-pm/state/history.json` (last 100 updates)

## Notes

- Supports major package managers across platforms
- Non-destructive by default (database refresh + controlled upgrades)
- Manual confirmation for breaking changes (can override with `--yes`)
- Detailed logging for troubleshooting
- Respects package manager-specific configurations
- Environment-aware (conda, virtualenv detection)

---

**Use Case ID**: UC-001
**Version**: 1.0
**Status**: Approved
**Priority**: P0 (Critical)
**Related Requirements**: FR-001, NFR-001, NFR-002
**Implementation Status**: 0% (v1.0 target: 95%)
