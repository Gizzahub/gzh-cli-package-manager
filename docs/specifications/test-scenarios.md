# Package Manager Update Test Scenarios

> **Comprehensive test suite with 120+ scenarios**
> **Covers all edge cases, platforms, and failure modes**

## Test Categories Overview

| Category | Scenarios | Priority | Platform Coverage |
|----------|-----------|----------|-------------------|
| 1. Basic Functionality | 15 | P0 | All |
| 2. Output Formats | 8 | P0 | All |
| 3. Platform-Specific | 12 | P1 | Platform-dependent |
| 4. Error Handling | 20 | P0 | All |
| 5. Environment Detection | 10 | P1 | All |
| 6. Manager-Specific | 18 | P1 | Manager-dependent |
| 7. Dry Run vs Execution | 6 | P0 | All |
| 8. Configuration/Strategy | 8 | P1 | All |
| 9. Performance | 10 | P2 | All |
| 10. Recovery/Rollback | 8 | P1 | All |
| 11. Integration | 6 | P1 | All |
| 12. Regression | 9 | P2 | All |
| **Total** | **130** | | |

---

## 1. Basic Functionality Tests (15 scenarios)

### Test 1.1: Simple Update Success
```bash
# Setup: System with brew and npm installed
gz-pm update --all --dry-run

# Expected:
- Shows update plan for all detected managers
- Output contains manager overview
- Output contains update actions
- Exit code: 0

# Verify:
assert_contains "$output" "Updating"
assert_contains "$output" "brew"
assert_contains "$output" "npm"
```

### Test 1.2: Single Manager Update
```bash
# Setup: System with multiple managers
gz-pm update --manager brew --dry-run

# Expected:
- Updates only Homebrew
- Other managers not processed
- Exit code: 0

# Verify:
assert_contains "$output" "brew"
assert_not_contains "$output" "asdf"
assert_not_contains "$output" "npm"
```

### Test 1.3: Multiple Specific Managers
```bash
# Setup: System with brew, asdf, npm
gz-pm update --managers brew,npm --dry-run

# Expected:
- Updates only specified managers (brew, npm)
- asdf not processed
- Exit code: 0

# Verify:
assert_contains "$output" "brew"
assert_contains "$output" "npm"
assert_not_contains "$output" "asdf"
```

### Test 1.4: No Managers Found
```bash
# Setup: Clean system with no package managers
gz-pm update --all

# Expected:
- Error message about no managers found
- List of supported managers
- Installation instructions
- Exit code: 2

# Verify:
assert_contains "$output" "No supported package managers found"
assert_contains "$output" "Install from"
assert_exit_code 2
```

### Test 1.5: All Managers Up-to-Date
```bash
# Setup: System with all packages already latest
gz-pm update --all

# Expected:
- Shows "already latest" messages
- No actual downloads or installations
- Exit code: 0

# Verify:
assert_contains "$output" "already latest"
assert_not_contains "$output" "Downloaded"
```

### Test 1.6-1.15: Additional Basic Tests
- Default behavior (no flags) == `--all`
- Help flag shows usage
- Version flag shows gz-pm version
- Invalid flag shows error
- Conflicting flags handled gracefully
- Update with no network (pre-flight check fails)
- Update with low disk space (pre-flight check warns)
- Update interrupted mid-way (recovery)
- Update with package manager removed during operation
- Update logs created correctly

---

## 2. Output Format Tests (8 scenarios)

### Test 2.1: Text Output (Default)
```bash
gz-pm update --all --dry-run

# Expected:
- Human-readable text with emojis
- Sections clearly delineated
- Progress indicators visible
- Color coding (if TTY)

# Verify:
assert_contains "$output" "📦"
assert_contains "$output" "✅"
assert_contains "$output" "━━━"
```

### Test 2.2: JSON Output Format
```bash
gz-pm update --all --dry-run --output json

# Expected:
- Valid JSON structure
- All update details in structured format
- Parseable with jq

# Verify:
echo "$output" | jq . >/dev/null
assert_exit_code 0
assert_contains "$(echo "$output" | jq -r '.summary.total_managers')" "[0-9]+"
```

### Test 2.3: Simple Output Format
```bash
gz-pm update --all --dry-run --output simple

# Expected:
- Plain text without emojis or colors
- No progress bars
- CI/CD friendly

# Verify:
assert_not_contains "$output" "📦"
assert_not_contains "$output" "━━━"
```

### Test 2.4: Auto-Detection (Non-TTY)
```bash
# Pipe output (non-TTY)
gz-pm update --all --dry-run | cat

# Expected:
- Automatically uses simple format
- No ANSI codes
- No progress bars

# Verify:
output=$(gz-pm update --all --dry-run | cat)
assert_not_contains "$output" $'\033'  # No ANSI escape codes
```

### Test 2.5-2.8: Additional Output Tests
- NO_COLOR environment variable respected
- Verbose flag adds detail
- Quiet flag reduces output
- Output consistency across formats (same data)

---

## 3. Platform-Specific Tests (12 scenarios)

### Test 3.1: macOS Package Managers
```bash
# Setup: macOS system
# Platform: darwin (x86_64 or arm64)
gz-pm update --all --dry-run

# Expected:
- Detects: brew, asdf, npm, pip, sdkman, cargo
- Marks as unsupported: apt, pacman, yum
- Exit code: 0

# Verify (macOS only):
assert_contains "$output" "brew.*✅"
assert_contains "$output" "apt.*🚫"
```

### Test 3.2: Linux (Ubuntu) Package Managers
```bash
# Setup: Ubuntu system
# Platform: linux (x86_64 or aarch64)
gz-pm update --all --dry-run

# Expected:
- Detects: apt, asdf, npm, pip, cargo
- brew: supported but may not be installed
- pacman: unsupported

# Verify (Ubuntu only):
assert_contains "$output" "apt.*✅"
assert_contains "$output" "pacman.*🚫"
```

### Test 3.3: Linux (Arch) Package Managers
```bash
# Setup: Arch Linux system
gz-pm update --all --dry-run

# Expected:
- Detects: pacman, yay (if installed), asdf, npm, pip
- apt: unsupported

# Verify (Arch only):
assert_contains "$output" "pacman.*✅"
assert_contains "$output" "apt.*🚫"
```

### Test 3.4-3.12: Additional Platform Tests
- Apple Silicon (arm64) vs Intel (x86_64) detection
- Homebrew installation path differences (Intel vs ARM)
- Linux distro detection (Ubuntu vs Arch vs Fedora)
- Windows (future): choco, scoop detection
- Cross-platform: asdf, npm, pip work everywhere
- Permission requirements per platform (sudo needed)
- System vs user package separation (per platform)
- Package manager availability checks
- Platform-specific error messages

---

## 4. Error Handling Tests (20 scenarios)

### Test 4.1: Network Connectivity Issues
```bash
# Simulate: Block network access
# (Use Docker network isolation or iptables)
gz-pm update --manager brew

# Expected:
- Clear error about network failure
- Retry suggestions
- Graceful degradation (other managers continue)
- Exit code: 1 (partial failure)

# Verify:
assert_contains "$output" "Network error"
assert_contains "$output" "Check.*connectivity"
assert_contains "$output" "Retry"
```

### Test 4.2: DNS Resolution Failure
```bash
# Simulate: Invalid DNS (point to 127.0.0.1)
gz-pm update --manager sdkman

# Expected:
- DNS resolution failure reported
- Troubleshooting steps shown
- nslookup command suggested

# Verify:
assert_contains "$output" "DNS resolution failed"
assert_contains "$output" "nslookup"
```

### Test 4.3: Permission Denied
```bash
# Setup: Run as non-privileged user
# Platform: Linux
gz-pm update --manager apt

# Expected:
- Permission error message
- Suggests using sudo
- Exit code: 4 (prerequisites not met)

# Verify:
assert_contains "$output" "Permission denied"
assert_contains "$output" "sudo gz-pm"
```

### Test 4.4: Insufficient Disk Space
```bash
# Simulate: System with very low disk space (<500MB)
gz-pm update --manager brew

# Expected:
- Pre-flight check detects low disk space
- Shows required vs available
- Cleanup suggestions
- Exit code: 4

# Verify:
assert_contains "$output" "Insufficient disk space"
assert_contains "$output" "available"
assert_contains "$output" "cleanup"
```

### Test 4.5-4.20: Additional Error Tests
- Package manager not responding (timeout)
- Corrupted package download (checksum fail)
- Version conflict (dependency incompatibility)
- Lock file exists (another process running)
- Invalid package manager name
- Broken symlinks in PATH
- Package signature verification failure
- Proxy authentication required
- Firewall blocking specific ports
- SSL certificate validation failure
- Package repository unavailable (404)
- Interrupted download recovery
- Disk quota exceeded
- Package cache corruption
- Concurrent gz-pm processes

---

## 5. Environment Detection Tests (10 scenarios)

### Test 5.1: Conda Environment Active
```bash
# Setup: Activate conda environment
conda activate myproject
gz-pm update --manager pip

# Expected:
- Detects conda environment
- Warns about pip/conda conflicts
- Suggests using conda/mamba instead
- Provides override flag
- Exit code: 0 (skips pip with warning)

# Verify:
assert_contains "$output" "Conda environment detected"
assert_contains "$output" "conda.*mamba"
assert_contains "$output" "--pip-allow-conda"
```

### Test 5.2: Python Virtual Environment Active
```bash
# Setup: Python virtual environment
python -m venv venv
source venv/bin/activate
gz-pm update --manager pip --dry-run

# Expected:
- Detects virtual environment
- Adjusts pip strategy (local vs global)
- Uses correct pip executable

# Verify:
assert_contains "$output" "Virtual environment"
pip_path=$(which pip)
assert_contains "$pip_path" "venv"
```

### Test 5.3: Multiple Node Version Managers
```bash
# Setup: Both nvm and asdf with node
gz-pm update --all --check-duplicates

# Expected:
- Detects node from multiple sources
- Shows PATH conflicts
- Recommends consolidation

# Verify:
assert_contains "$output" "Duplicate Installation Check"
assert_contains "$output" "node.*brew.*asdf"
```

### Test 5.4-5.10: Additional Environment Tests
- asdf with .tool-versions file (respect version pinning)
- sdkman with multiple Java versions
- rbenv vs rvm detection
- pyenv vs system Python
- Docker container environment detection
- CI/CD environment detection (disable interactivity)
- SSH session vs local terminal (TTY detection)

---

## 6. Package Manager Specific Tests (18 scenarios)

### Test 6.1: Homebrew - Formula and Cask
```bash
# macOS only
gz-pm update --manager brew

# Verify:
- Formula updates (CLI tools)
- Cask updates (GUI apps) if applicable
- Cleanup after updates
- Keg-only packages handled correctly
```

### Test 6.2: Homebrew - Tap Management
```bash
# Verify:
- Third-party taps updated
- Tap conflicts handled
- Deprecated taps warned
```

### Test 6.3: ASDF - Plugin Updates
```bash
gz-pm update --manager asdf

# Verify:
- Plugin updates executed first
- Language version updates available shown
- Post-install hooks executed
```

### Test 6.4-6.18: Additional Manager Tests
- ASDF: .tool-versions compatibility filters
- ASDF: Post-install hooks (npm for node, pip for python)
- npm: Global vs local package handling
- npm: Permission issues with global installs
- pip: User vs system site-packages
- pip: requirements.txt integration
- apt: Update vs upgrade distinction
- apt: PPA repository handling
- pacman: AUR helper (yay) integration
- cargo: Registry update
- sdkman: Candidate installation
- go: Module cache handling
- rustup: Toolchain updates
- rbenv/rvm: Ruby version management
- pyenv: Python version management

---

## 7. Dry Run vs Execution Tests (6 scenarios)

### Test 7.1: Dry Run Prediction Accuracy
```bash
# Capture dry-run output
dry_result=$(gz-pm update --all --dry-run --output json)

# Execute actual update
real_result=$(gz-pm update --all --output json)

# Verify:
# Actual package counts match predictions (±2)
# Download size estimates reasonable (±20%)
# Manager list identical
```

### Test 7.2: No Changes When Up-to-Date (Dry Run)
```bash
# System already updated
gz-pm update --all --dry-run

# Verify:
assert_contains "$output" "already latest"
assert_contains "$output" "No changes"
```

### Test 7.3-7.6: Additional Dry Run Tests
- Dry-run doesn't create state files
- Dry-run doesn't modify package databases
- Dry-run time estimate accuracy
- Dry-run detects errors (network, permissions)

---

## 8. Configuration and Strategy Tests (8 scenarios)

### Test 8.1: Update Strategy - Latest
```bash
gz-pm update --all --strategy latest

# Expected:
- Updates to absolute latest versions
- Includes beta/rc versions where applicable
```

### Test 8.2: Update Strategy - Stable
```bash
gz-pm update --all --strategy stable

# Expected:
- Updates to latest stable versions only
- Excludes beta/rc versions
```

### Test 8.3: Update Strategy - Minor
```bash
gz-pm update --all --strategy minor

# Expected:
- Updates to latest patch/minor versions
- No major version upgrades
```

### Test 8.4-8.8: Additional Strategy Tests
- Fixed strategy (no updates)
- Per-manager strategy configuration
- Global default strategy from config file
- Strategy override via command line
- Version pinning from config file

---

## 9. Performance and Resource Tests (10 scenarios)

### Test 9.1: Large Package Set
```bash
# Setup: System with 100+ packages
time gz-pm update --all --dry-run

# Expected:
- Completes within 60 seconds
- Memory usage < 500MB

# Verify:
duration=$(measure_duration)
assert_less_than "$duration" 60
```

### Test 9.2-9.10: Additional Performance Tests
- Parallel manager updates (no conflicts)
- Serial manager updates (for reliability)
- Progress indication refresh rate
- Resource monitoring during execution
- Cleanup of temporary files
- Log file size management (rotation)
- Network bandwidth limiting
- CPU usage reasonable (<50% average)
- Scalability with manager count

---

## 10. Recovery and Rollback Tests (8 scenarios)

### Test 10.1: Interrupted Update Recovery
```bash
# Start update, kill mid-way
gz-pm update --all &
PID=$!
sleep 30
kill $PID

# Restart
gz-pm update --all

# Expected:
- Continues from safe state
- No corrupted package databases
```

### Test 10.2-10.8: Additional Recovery Tests
- Failed package installation (others continue)
- Network interruption recovery
- Disk full during download
- State file corruption recovery
- Lock file cleanup after crash
- Rollback capability for failed updates
- Partial state validation
- Resume interrupted downloads

---

## 11. Integration Tests (6 scenarios)

### Test 11.1: Full Workflow
```bash
# Complete workflow:
gz-pm status                    # Before state
gz-pm update --all --dry-run    # Plan
gz-pm update --all              # Execute
gz-pm status                    # After state

# Verify state changes consistent
```

### Test 11.2-11.6: Additional Integration Tests
- Multi-user environment (shared vs user packages)
- Config file integration (read preferences)
- Export after update (config generation)
- Bootstrap new system after update
- Status check accuracy after updates

---

## 12. Regression Tests (9 scenarios)

### Test 12.1: Version Detection Accuracy
```bash
# Verify version parsing for all managers
# Test edge cases: pre-release, custom builds
# Check version comparison logic
```

### Test 12.2-12.9: Additional Regression Tests
- Configuration file format compatibility
- Backward compatible command syntax
- State file schema migration
- Log file format stability
- Exit code consistency
- Error message stability (for parsing)
- JSON schema stability
- Platform detection accuracy
- Manager detection heuristics

---

## Test Execution Framework

### Automated Test Runner

```bash
#!/bin/bash
# test-runner.sh - Execute test scenarios

set -euo pipefail

PASS=0
FAIL=0
SKIP=0

run_test() {
    local category="$1"
    local test_name="$2"
    local test_func="$3"

    echo "Running [$category] $test_name"

    if $test_func; then
        echo "✅ PASS: $test_name"
        ((PASS++))
    else
        echo "❌ FAIL: $test_name"
        ((FAIL++))
    fi
}

skip_test() {
    local reason="$1"
    echo "⏭️  SKIP: $reason"
    ((SKIP++))
}

# Test assertions
assert_contains() {
    local haystack="$1"
    local needle="$2"
    if echo "$haystack" | grep -q "$needle"; then
        return 0
    else
        echo "  Expected to find: $needle"
        return 1
    fi
}

assert_exit_code() {
    local expected="$1"
    local actual="$?"
    if [ "$actual" -eq "$expected" ]; then
        return 0
    else
        echo "  Expected exit code: $expected, got: $actual"
        return 1
    fi
}

# Run test suites
run_basic_functionality_tests
run_output_format_tests
run_platform_specific_tests
run_error_handling_tests
run_environment_detection_tests
run_manager_specific_tests
run_dry_run_tests
run_configuration_tests
run_performance_tests
run_recovery_tests
run_integration_tests
run_regression_tests

# Summary
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Results:"
echo "  ✅ Passed: $PASS"
echo "  ❌ Failed: $FAIL"
echo "  ⏭️  Skipped: $SKIP"
echo "  Total: $((PASS + FAIL + SKIP))"

if [ $FAIL -eq 0 ]; then
    echo "🎉 All tests passed!"
    exit 0
else
    echo "⚠️  Some tests failed"
    exit 1
fi
```

### Docker Test Environments

```dockerfile
# Dockerfile.test-ubuntu
FROM ubuntu:22.04

RUN apt-get update && apt-get install -y \
    curl git build-essential \
    python3 python3-pip \
    nodejs npm

# Install asdf
RUN git clone https://github.com/asdf-vm/asdf.git ~/.asdf --branch v0.14.0

# Install gz-pm to Go bin directory
RUN mkdir -p /root/go/bin
COPY gz-pm /root/go/bin/gz-pm
RUN chmod +x /root/go/bin/gz-pm
ENV PATH="/root/go/bin:${PATH}"

CMD ["bash"]
```

### CI/CD Integration

```yaml
# .github/workflows/test-scenarios.yml
name: Test Scenarios

on: [push, pull_request]

jobs:
  test-scenarios:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
        test-category:
          - basic-functionality
          - output-formats
          - platform-specific
          - error-handling

    runs-on: ${{ matrix.os }}

    steps:
    - uses: actions/checkout@v4

    - name: Setup Go
      uses: actions/setup-go@v5
      with:
        go-version: '1.26.7'

    - name: Build gz-pm
      run: make build

    - name: Run test scenarios
      run: |
        ./scripts/test-runner.sh --category ${{ matrix.test-category }}

    - name: Upload test results
      if: always()
      uses: actions/upload-artifact@v3
      with:
        name: test-results-${{ matrix.os }}-${{ matrix.test-category }}
        path: test-results/
```

---

**Document Version**: 1.0
**Last Updated**: 2025-01-27
**Total Test Scenarios**: 130+
**Coverage Target**: 90%+
**Automation Level**: 85%+ automated
