# UC-001: Update Package Managers (Enhanced Specification)

> **Target Compliance**: 95%
> **This specification defines the gold standard for update command behavior**

## Synopsis

```bash
pmctl update [--all | --manager <name> | --managers <list>] [options]
```

Update package managers and their managed packages with rich progress indication, resource management, and intelligent conflict detection.

## Command Variants

```bash
pmctl update --all                      # All detected managers (default)
pmctl update --manager brew             # Single manager
pmctl update --managers brew,asdf,npm   # Multiple specific managers
pmctl update --all --strategy latest    # Update strategy control
pmctl update --all --dry-run            # Preview changes
pmctl update --all --output json        # JSON output for scripts
pmctl update --all --check-duplicates   # Detect duplicate binaries
pmctl update --manager pip --pip-allow-conda  # Override conda check
```

## Prerequisites

- [ ] At least one supported package manager installed
- [ ] Network connectivity for package downloads
- [ ] Admin permissions for system package managers (apt, pacman, yum)
- [ ] Sufficient disk space (checked automatically)
- [ ] Adequate memory available (checked automatically)

## Expected Output Format

### Enhanced Output (Default for TTY)

```text
📦 Package Manager Update - pmctl v1.0.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔍 Performing pre-flight checks...

📊 Resource Availability Check
✅ Disk: Sufficient disk space: 45.2GB available, ~2.1GB needed
✅ Network: Network connectivity good: 4/4 repositories accessible
✅ Memory: Sufficient memory: 8192MB available

📋 Manager Overview:
MANAGER      SUPPORTED  INSTALLED  VERSION    NOTE
------------ ---------- ---------- ---------- --------------------
brew         ✅         ✅         5.0.0
asdf         ✅         ✅         0.14.0
sdkman       ✅         ✅         5.18.2
npm          ✅         ✅         10.2.3
pip          ✅         ✅         24.0
apt          🚫         ⛔         -          Linux only
pacman       🚫         ⛔         -          Arch/Manjaro only

🧪 Duplicate Installation Check:
Found 2 potential conflicts:
  • node: /usr/local/bin/node (brew), ~/.asdf/shims/node (asdf)
  • python3: /usr/bin/python3 (system), ~/.asdf/shims/python3 (asdf)
💡 Consider using single package manager per tool to avoid PATH conflicts

═══════════ 🚀 [1/5] brew — Updating ═══════════
🍺 Updating Homebrew...
⏳ brew update
   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░ 60%  Fetching formulae...

✅ brew update: Updated 23 formulae (15.2s)
✅ brew upgrade: Upgraded 5 packages (42.8s)
   • node: 20.11.0 → 20.11.1 (24.8MB)
   • git: 2.43.0 → 2.43.1 (8.4MB)
   • python@3.11: 3.11.7 → 3.11.8 (15.2MB)
   • jq: 1.6 → 1.7 (1.1MB)
   • tree: 2.1.0 → 2.1.1 (156KB)
✅ brew cleanup: Freed 245MB disk space (8.3s)

📊 brew summary: 5 packages upgraded, 245MB freed, 66.3s elapsed

═══════════ 🚀 [2/5] asdf — Updating ═══════════
🔄 Updating asdf version manager...
⏳ asdf plugin update --all
   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░ 65%  Updating golang plugin...

✅ asdf plugin update --all: 8 plugins updated (18.4s)
✅ asdf update: Updated to v0.14.0 (2.1s)

🔍 Checking for language version updates...
✅ nodejs: 20.11.0 → 20.11.1 installed (35.2s, 28.3MB)
   ✅ Post-action: npm cache clean --force (1.2s)

💡 golang: 1.21.5 already latest, skipping

✅ python: 3.11.7 → 3.11.8 installed (48.7s, 18.9MB)
   ✅ Post-action: pip install --upgrade pip (3.4s)

📊 asdf summary: 2 languages updated, 47.2MB downloaded, 108.0s elapsed

═══════════ 🚀 [3/5] sdkman — Updating ═══════════
☕ Updating SDKMAN...
✅ sdk selfupdate: Updated SDKMAN to 5.18.2 (4.2s)
✅ sdk update: Refreshed candidate metadata (2.8s)

💡 Available updates (manual installation required):
   • java: 21.0.1-oracle → 21.0.2-oracle
     Install: sdk install java 21.0.2-oracle
   • maven: 3.9.5 → 3.9.6
     Install: sdk install maven 3.9.6

📊 sdkman summary: SDKMAN updated, 2 candidates available, 7.0s elapsed

═══════════ 🚀 [4/5] npm — Updating ═══════════
🧩 Updating npm global packages...
⏳ npm update -g
   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░ 70%  Installing typescript...

✅ npm update -g: 12 global packages updated (38.2s)
   • @angular/cli: 17.0.7 → 17.0.8 (12.4MB)
   • typescript: 5.3.2 → 5.3.3 (4.8MB)
   • prettier: 3.1.0 → 3.1.1 (2.1MB)
   • eslint: 8.55.0 → 8.56.0 (8.3MB)
   • + 8 more packages

📊 npm summary: 12 packages updated, 35.2MB downloaded, 38.2s elapsed

═══════════ 🚀 [5/5] pip — Updating ═══════════
🐍 Updating pip packages...
✅ pip install --upgrade pip: Updated to 24.0 (5.4s)

⏳ Checking for outdated packages
   ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░ 75%  Analyzing dependencies...

✅ Updated 6 packages: (42.8s)
   • requests: 2.31.0 → 2.32.0 (1.2MB)
   • numpy: 1.24.3 → 1.24.4 (18.4MB)
   • pandas: 2.1.3 → 2.1.4 (15.3MB)
   • matplotlib: 3.7.2 → 3.7.3 (8.7MB)
   • scikit-learn: 1.3.2 → 1.4.0 (12.1MB)
   • pytest: 7.4.3 → 7.4.4 (892KB)

📊 pip summary: 6 packages updated, 56.6MB downloaded, 48.2s elapsed

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🎉 Package manager updates completed successfully!

📊 Summary:
   • Total managers processed: 5
   • Successfully updated: 5
   • Packages upgraded: 27
   • Total download size: 164.3MB
   • Disk space freed: 245MB
   • Conflicts detected: 2 (non-blocking)

💡 Recommended actions:
   • Update SDKMAN candidates:
     - sdk install java 21.0.2-oracle
     - sdk install maven 3.9.6
   • Consider consolidating node versions (brew vs asdf)

⏰ Update completed in 3m 42s (222s total)

Exit Code: 0
```

### Partial Success with Detailed Errors

```text
📦 Package Manager Update - pmctl v1.0.0
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔍 Performing pre-flight checks...

📊 Resource Availability Check
✅ Disk: Sufficient disk space: 12.4GB available
⚠️  Network: Degraded connectivity: 3/4 repositories accessible
✅ Memory: Sufficient memory: 4096MB available

═══════════ 🚀 [1/4] brew — Updating ═══════════
🍺 Updating Homebrew...
✅ brew update: Updated 15 formulae (12.4s)
❌ brew upgrade: Failed to upgrade 2/7 packages

Successful upgrades (5):
   ✅ git: 2.43.0 → 2.43.1 (8.4MB, 15.2s)
   ✅ jq: 1.6 → 1.7 (1.1MB, 4.8s)
   ✅ tree: 2.1.0 → 2.1.1 (156KB, 2.1s)
   ✅ wget: 1.21.4 → 1.21.5 (2.3MB, 6.3s)
   ✅ curl: 8.4.0 → 8.5.0 (4.2MB, 8.9s)

Failed upgrades (2):
   ❌ postgresql@14: Version conflict detected
      • Current: 14.9 (via Homebrew)
      • Available: 16.1 (major version upgrade with breaking changes)
      • Migration required: Data directory incompatible
      • Fix: brew unlink postgresql@14 && brew install postgresql@16
            pg_upgrade --old-datadir /usr/local/var/postgres \
                      --new-datadir /usr/local/var/postgres-16

   ❌ docker: Insufficient disk space
      • Requires: 1.2GB
      • Available: 800MB
      • Fix: brew cleanup (will free ~500MB)
            Or free additional disk space manually

📊 brew summary: 5/7 packages upgraded, 2 failed, 48.7s elapsed

═══════════ ⚠️ [2/4] sdkman — SKIP ═══════════
❌ Network error: Cannot reach SDKMAN servers

Error details:
   • Host: get.sdkman.io
   • DNS resolution: Failed (NXDOMAIN)
   • Timeout: 30 seconds
   • Network interface: en0 (active)

Troubleshooting steps:
   1. Check network connectivity: ping 8.8.8.8
   2. Check DNS resolution: nslookup get.sdkman.io
   3. Check firewall settings
   4. Verify proxy configuration: echo $HTTP_PROXY
   5. Retry: pmctl update --manager sdkman

Skipping sdkman updates for this session.

═══════════ 🚀 [3/4] npm — Updating ═══════════
🧩 Updating npm global packages...
✅ npm update -g: 8 global packages updated (32.4s, 28.3MB)

═══════════ ⚠️ [4/4] pip — SKIP ═══════════
⚠️  Conda environment detected: /opt/miniconda3/envs/myproject

Conda/pip conflict warning:
   • Active conda environment: myproject
   • Using pip in conda can cause dependency conflicts
   • Package versions may become inconsistent
   • Conda's dependency solver may break

Recommended approach:
   • Use conda/mamba for package management:
     conda update --all
   • Or deactivate conda before using pip:
     conda deactivate && pmctl update --manager pip

Override (not recommended):
   pmctl update --manager pip --pip-allow-conda

Skipping pip updates for this session.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️  Package manager updates partially completed.

📊 Summary:
   • Total managers processed: 4
   • Successfully updated: 2 (brew partial, npm complete)
   • Failed: 1 (sdkman - network issues)
   • Skipped: 1 (pip - environment conflict)
   • Packages upgraded: 13
   • Manual fixes required: 2

🔧 Required manual fixes:

1. PostgreSQL version conflict (brew):
   brew unlink postgresql@14 && brew install postgresql@16

2. Docker disk space (brew):
   brew cleanup
   # Or free 400MB+ manually

3. Network connectivity (sdkman):
   Check firewall/DNS and retry: pmctl update --manager sdkman

4. Conda environment (pip):
   Use conda instead: conda update --all
   Or override: pmctl update --manager pip --pip-allow-conda

💡 Quick retry for failed managers:
   pmctl update --managers sdkman

⏰ Update completed in 1m 45s (partial)

Exit Code: 1
```

### Dry Run Example

```text
📦 Package Manager Update - pmctl v1.0.0 [DRY RUN]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️  DRY RUN MODE: No changes will be made

🔍 Analyzing update actions...

═══════════ 📋 [1/3] brew — Analysis ═══════════
Would execute:
   1. brew update
   2. brew upgrade

Expected updates:
   ✓ node: 20.11.0 → 20.11.1
     - Download size: 24.8MB
     - Disk space: +12MB
   ✓ git: 2.43.0 → 2.43.1
     - Download size: 8.4MB
     - Disk space: +4MB
   ✓ python@3.11: 3.11.7 → 3.11.8
     - Download size: 15.2MB
     - Disk space: +8MB

Cleanup:
   ✓ brew cleanup would free ~245MB

Estimated time: 45-90 seconds

═══════════ 📋 [2/3] asdf — Analysis ═══════════
Would execute:
   1. asdf plugin update --all
   2. asdf update

Expected updates:
   ✓ 8 plugins would be updated
   ✓ asdf: 0.13.1 → 0.14.0

Language version updates available:
   ✓ nodejs: 20.11.0 → 20.11.1 (28.3MB)
   ✓ python: 3.11.7 → 3.11.8 (18.9MB)
   - golang: 1.21.5 (already latest)

Estimated time: 90-150 seconds

═══════════ 📋 [3/3] npm — Analysis ═══════════
Would execute:
   1. npm update -g

Expected updates:
   ✓ 12 global packages
     - Total download: ~35MB

Estimated time: 30-45 seconds

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 Dry Run Summary:
   • Managers to update: 3
   • Packages to upgrade: 27
   • Total download size: ~130MB
   • Disk space freed: ~245MB
   • Estimated time: 3-5 minutes

⚠️  This was a dry run. No changes were made.

To execute these updates:
   pmctl update --all

To execute specific managers only:
   pmctl update --managers brew,npm

Exit Code: 0
```

## Side Effects

### Files Created

- `~/.pmctl/logs/update-<timestamp>.log` - Detailed timestamped update log
- `~/.pmctl/state/update-<timestamp>.json` - Update session results (JSON)
- `~/.pmctl/cache/` - Package manager cache files
- `/tmp/pmctl-*.tmp` - Temporary download and processing files (auto-cleaned)

### Files Modified

- Package manager databases (brew, apt, pacman, etc.)
- Installed package binaries and libraries
- Package manager configuration files (if self-update occurs)
- System PATH (if new tools installed)
- Shell RC files (if tools modify them)

### State Changes

- Package databases refreshed with latest metadata
- Outdated packages upgraded to newer versions
- Package caches cleaned and optimized
- Environment variables updated for new tool versions
- Symlinks updated (especially for asdf, sdkman)

## Validation

### Automated Tests

```bash
# Test enhanced output format
result=$(pmctl update --all --dry-run 2>&1)
assert_contains "$result" "Package Manager Update - pmctl"
assert_contains "$result" "Resource Availability Check"
assert_contains "$result" "Manager Overview"
assert_contains "$result" "Dry Run Summary"

# Test progress indication
result=$(pmctl update --manager brew 2>&1)
assert_contains "$result" "▓"  # Progress bar character
assert_contains "$result" "⏳"  # Progress indicator

# Test resource checks
result=$(pmctl update --all 2>&1)
assert_contains "$result" "Disk:"
assert_contains "$result" "Network:"
assert_contains "$result" "Memory:"

# Test duplicate detection
result=$(pmctl update --all --check-duplicates 2>&1)
assert_contains "$result" "Duplicate Installation Check"

# Test error reporting
result=$(pmctl update --manager nonexistent 2>&1)
exit_code=$?
assert_exit_code 3  # Invalid arguments
assert_contains "$result" "Unknown package manager"

# Test JSON output structure
json=$(pmctl update --all --dry-run --output json 2>&1)
echo "$json" | jq . >/dev/null  # Validate JSON
echo "$json" | jq -r '.summary.total_managers' | grep -qE '^[0-9]+$'
echo "$json" | jq -r '.managers[0].name' | grep -q '.'
```

### Compliance Criteria

**Output Format (95% target)**:
- [ ] Section banners with Unicode box drawing (═══════════)
- [ ] Emoji indicators (📦 🚀 ✅ ❌ ⚠️ 💡 ⏰)
- [ ] Progress bars during operations
- [ ] Detailed version changes with download sizes
- [ ] Resource availability checks (disk, network, memory)
- [ ] Duplicate binary detection and reporting
- [ ] Comprehensive summary with statistics
- [ ] Actionable error messages with fix commands
- [ ] Estimated and actual timing information

**Functional Behavior (90% target)**:
- [ ] All documented flags work correctly
- [ ] Exit codes match specification exactly
- [ ] Dry-run predictions match actual execution
- [ ] Error recovery and partial success handling
- [ ] Environment conflict detection (conda, virtualenv)
- [ ] Network timeout and retry logic
- [ ] Permission error handling with sudo suggestions

**Performance (90% target)**:
- [ ] Pre-flight checks complete in < 10 seconds
- [ ] Manager detection in < 5 seconds
- [ ] Progress updates every 2-3 seconds
- [ ] Memory usage < 500MB
- [ ] Proper cleanup of temporary files

## Edge Cases

### Resource Constraints

**Insufficient Disk Space**:
- Pre-flight check detects shortage
- Shows required vs available space
- Suggests cleanup commands
- Prevents download start if critical

**Network Issues**:
- DNS resolution failures → Clear error with DNS check steps
- Firewall blocking → Suggests checking firewall rules
- Proxy authentication → Respects HTTP_PROXY environment
- Partial connectivity → Continues with accessible repos

**Memory Pressure**:
- Monitors available memory
- Warns if < 1GB available
- Adjusts parallel operations

### Environment Conflicts

**Conda/Mamba Detection**:
```text
⚠️  Conda environment detected: /opt/miniconda3/envs/myproject
   • Use conda/mamba for package management instead
   • Override with: pmctl update --manager pip --pip-allow-conda
```

**Virtual Environment**:
```text
💡 Virtual environment active: /home/user/project/venv
   • pip updates will affect this environment only
   • Deactivate for system-wide updates
```

**Multiple Version Managers**:
```text
🧪 Duplicate Installation Check:
Found potential conflicts:
  • node: /usr/local/bin/node (brew), ~/.asdf/shims/node (asdf)
  • Recommendation: Choose one primary manager for node
```

### Platform-Specific Behavior

**macOS**:
- Homebrew Cask support
- Apple Silicon vs Intel detection
- Xcode Command Line Tools check

**Linux (Ubuntu/Debian)**:
- apt requires sudo → Clear permission error
- System vs user package separation
- PPA repository handling

**Linux (Arch/Manjaro)**:
- pacman vs yay (AUR helper)
- System package conflicts with user managers
- Pacman lock file handling

### Version Conflicts

**Breaking Changes**:
```text
❌ postgresql@14: Version conflict detected
   • Current: 14.9 (via Homebrew)
   • Available: 16.1 (major version with breaking changes)
   • Migration required: Data directory incompatible
   • Fix: [detailed migration steps]
```

**Dependency Conflicts**:
```text
⚠️  Dependency conflict detected:
   • package-a requires library-x >= 2.0
   • package-b requires library-x < 2.0
   • Resolution: Update package-b or hold package-a
```

## Performance Expectations

| Operation | Target | Acceptable Range |
|-----------|--------|------------------|
| Pre-flight checks | < 10s | 5-15s |
| Manager detection | < 5s | 2-8s |
| Single manager update | 30s-5min | Varies by packages |
| All managers update | 2-15min | Varies heavily |
| Dry-run analysis | < 30s | 10-45s |
| Progress update frequency | 2-3s | 1-5s |
| Memory usage (peak) | < 500MB | 200-800MB |

## JSON Output Schema

```json
{
  "pmctl_version": "1.0.0",
  "timestamp": "2025-01-27T10:30:00Z",
  "operation": "update",
  "dry_run": false,
  "pre_flight": {
    "disk": {"status": "ok", "available_gb": 45.2, "required_gb": 2.1},
    "network": {"status": "ok", "reachable_repos": 4, "total_repos": 4},
    "memory": {"status": "ok", "available_mb": 8192}
  },
  "managers": [
    {
      "name": "brew",
      "supported": true,
      "installed": true,
      "version": "5.0.0",
      "status": "success",
      "packages_upgraded": 5,
      "download_size_mb": 49.6,
      "disk_freed_mb": 245,
      "duration_seconds": 66.3,
      "details": [
        {"package": "node", "from": "20.11.0", "to": "20.11.1", "size_mb": 24.8},
        {"package": "git", "from": "2.43.0", "to": "2.43.1", "size_mb": 8.4}
      ]
    }
  ],
  "summary": {
    "total_managers": 5,
    "successful": 5,
    "failed": 0,
    "skipped": 0,
    "packages_upgraded": 27,
    "total_download_mb": 164.3,
    "disk_freed_mb": 245,
    "total_duration_seconds": 222,
    "conflicts": {
      "count": 2,
      "details": [
        {"binary": "node", "paths": ["/usr/local/bin/node", "~/.asdf/shims/node"]}
      ]
    }
  },
  "exit_code": 0
}
```

---

**Use Case ID**: UC-001-enhanced
**Version**: 1.0
**Status**: Approved
**Compliance Target**: 95%
**Priority**: P0 (Critical)
**Related**: UC-001-update.md (basic spec), ADR-007 (Enhanced Output)
**Implementation Status**: 0% (Week 7 presentation layer target)
