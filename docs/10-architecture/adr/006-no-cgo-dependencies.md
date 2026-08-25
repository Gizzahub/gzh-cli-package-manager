# ADR-006: No CGO Dependencies

**Date**: 2025-01-27
**Status**: Accepted
**Deciders**: Project Team
**Related**: ADR-009 (Go 1.26+ Requirement), ADR-001 (Standalone Extraction)

---

## Context

We need to decide whether to allow CGO (C Go) dependencies in gzh-cli-package-manager. CGO allows Go programs to call C libraries and use C code.

**Considerations**:
- Cross-platform compilation requirements (macOS, Linux, Windows)
- Binary distribution simplicity
- Performance needs (SQLite, compression, crypto)
- Security and attack surface
- Build complexity and CI/CD

**Common CGO Use Cases**:
1. **SQLite** - `github.com/mattn/go-sqlite3` (CGO) vs `modernc.org/sqlite` (pure Go)
2. **Compression** - C libraries (faster) vs Go stdlib (simpler)
3. **Crypto** - Platform-specific vs Go crypto
4. **GUI** - Native bindings vs web-based TUI

---

## Decision

**Prohibit CGO dependencies. Use pure Go implementations only.**

**Build Configuration**:
```bash
# Makefile
CGO_ENABLED=0 go build -o build/gz-pm ./cmd/pm
```

**Dependency Policy**:
- ❌ Forbidden: Libraries requiring CGO
- ✅ Required: Pure Go alternatives
- ⚠️ Exception process: Requires architecture review + ADR

---

## Rationale

### Why No CGO?

**1. Cross-Compilation Simplicity**

Without CGO:
```bash
# Build for all platforms from any OS
GOOS=linux GOARCH=amd64 go build -o gz-pm-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o gz-pm-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o gz-pm-windows-amd64.exe
```

With CGO:
```bash
# Requires cross-compilation toolchains
# macOS → Linux: Need Linux C compiler
# Linux → Windows: Need MinGW
# Extremely complex CI setup
```

**2. Single Static Binary**

Without CGO:
```bash
# Truly static binary
$ ldd gz-pm
    not a dynamic executable

# Works anywhere (no libc dependencies)
$ ./build/gz-pm version  # Just works
```

With CGO:
```bash
# Dynamic linking issues
$ ldd gz-pm
    libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6
    libpthread.so.0 => /lib/x86_64-linux-gnu/libpthread.so.0

# Can fail on different distros
$ ./gz-pm
    error while loading shared libraries: libc.so.6: version GLIBC_2.34 not found
```

**3. Faster Build Times**

| Scenario | No CGO | With CGO | Difference |
|----------|--------|----------|------------|
| Clean build | 5s | 45s | **9x slower** |
| Incremental | 1s | 8s | **8x slower** |
| CI builds | 30s | 5min | **10x slower** |

**Reason**: C compilation is slower than Go compilation

**4. Simpler CI/CD**

Without CGO (GitHub Actions):
```yaml
- uses: actions/setup-go@v5
  with:
    go-version: '1.26.7'
- run: make build  # Just works
```

With CGO:
```yaml
- uses: actions/setup-go@v5
- name: Install cross-compilation toolchains
  run: |
    apt-get install -y gcc-mingw-w64  # Windows
    apt-get install -y gcc-aarch64-linux-gnu  # ARM Linux
    # Hundreds of MB of tools
- run: make build  # Complex scripts
```

**5. Distribution Simplicity**

Without CGO:
```bash
# Homebrew formula
url "https://github.com/gizzahub/gzh-cli-package-manager/releases/download/v1.0.0/gz-pm-darwin-arm64.tar.gz"
# No dependencies needed

# Go install works immediately
go install github.com/gizzahub/gzh-cli-package-manager/cmd/pm@latest
```

With CGO:
```bash
# Homebrew formula needs dependencies
depends_on "sqlite"  # Must be installed first
depends_on "gcc"     # Build-time dependency

# go install fails on some systems (missing C libs)
```

**6. Security: Smaller Attack Surface**

Without CGO:
- No C memory management bugs
- No buffer overflows from C code
- No dependency on system libc (varies by distro)
- Supply chain: Only Go code to audit

With CGO:
- Vulnerable C libraries (SQLite CVEs, zlib CVEs)
- Memory safety issues
- Platform-specific exploits

**7. Reproducible Builds**

Without CGO:
```bash
# Same Go version → identical binary
$ go build
$ sha256sum gz-pm
abc123...

# Months later, same hash
$ go build
$ sha256sum gz-pm
abc123...  # Identical
```

With CGO:
- Different results based on C compiler version
- Platform-specific libc versions
- Hard to reproduce exact builds

**8. Easier Debugging**

Without CGO:
- Go debugger (Delve) works perfectly
- Stack traces are pure Go
- No C/Go boundary issues

With CGO:
- Mixed Go/C stack traces
- Debugging C code requires gdb
- Harder to diagnose crashes

### Why Pure Go is Sufficient?

**1. Performance is Acceptable**

Our bottlenecks are I/O, not CPU:
- Network requests (Homebrew API): 100-500ms
- Shell command execution (brew update): 5-30s
- File I/O (config read): 1-5ms

Pure Go crypto/compression adds < 10ms overhead (negligible).

**2. Pure Go Alternatives Exist**

| Use Case | CGO Library | Pure Go Alternative | Trade-off |
|----------|-------------|---------------------|-----------|
| SQLite | `github.com/mattn/go-sqlite3` | `modernc.org/sqlite` | 5-10% slower, but still fast |
| Compression | C zlib | `compress/gzip` (stdlib) | Comparable performance |
| Crypto | OpenSSL | `crypto/*` (stdlib) | Excellent performance |
| JSON | C parsers | `encoding/json` (stdlib) | Fast enough for our data |

**3. No GUI Needed**
- CLI tool (no native UI bindings)
- TUI via Bubble Tea (pure Go)
- No need for GTK, Qt, Cocoa

---

## Consequences

### Positive ✅

1. **One-Command Cross-Compilation**
   ```bash
   make build-all
   # Produces:
   # - gz-pm-darwin-amd64
   # - gz-pm-darwin-arm64
   # - gz-pm-linux-amd64
   # - gz-pm-linux-arm64
   # - gz-pm-windows-amd64.exe
   # In ~30 seconds
   ```

2. **Universal Binaries**
   - Works on any Linux distro (Alpine, Ubuntu, RHEL)
   - No "libc version" issues
   - No runtime dependencies
   - Example:
     ```bash
     # Works on minimal container
     FROM scratch
     COPY gz-pm /gz-pm
     ENTRYPOINT ["/gz-pm"]
     # No base image needed!
     ```

3. **Faster CI Pipelines**
   - Builds complete in < 2 minutes
   - No toolchain installation
   - Parallel builds across platforms
   - Lower CI costs (less compute time)

4. **Easier Releases**
   ```bash
   goreleaser release --snapshot
   # Handles all platforms automatically
   # No custom cross-compilation scripts
   ```

5. **Developer Experience**
   - Any developer can build for any platform
   - No Xcode needed to build macOS binary from Linux
   - No MinGW needed to build Windows binary from macOS
   - Onboarding: `git clone && make build`

6. **Predictable Behavior**
   - Same Go version → same behavior on all platforms
   - No platform-specific C library quirks
   - Easier to debug "works on my machine" issues

### Negative ❌

1. **Slightly Slower Performance (Some Cases)**
   - Pure Go SQLite ~5-10% slower than CGO SQLite
   - Impact: Negligible (we don't use DB in v1.0)
   - Mitigation: Profile first, optimize only if needed

2. **Limited Library Choices**
   - Can't use popular CGO libraries
   - Example: `go-sqlite3` is most popular, but we use `modernc.org/sqlite`
   - Mitigation: Pure Go ecosystem is mature

3. **Future Constraints**
   - If we need GPU acceleration (unlikely)
   - If we need native OS APIs (can use syscall package)
   - Mitigation: Exception process exists (requires ADR)

4. **Can't Use System Libraries**
   - No direct OpenSSL calls (must use Go crypto)
   - No system SQLite (must bundle pure Go version)
   - Mitigation: Go stdlib crypto is excellent

### Neutral 🔄

1. **Binary Size**
   - Pure Go binaries: ~10-15MB (includes runtime)
   - CGO binaries: ~8-12MB (+ system dependencies)
   - Trade-off: Slightly larger, but self-contained

2. **Language Purity**
   - Entire codebase is Go (no C)
   - Easier to onboard Go developers
   - But can't leverage C ecosystem

---

## Implementation

### Build Configuration

```makefile
# Makefile
.PHONY: build
build:
	CGO_ENABLED=0 go build \
		-ldflags="-s -w" \
		-o build/gz-pm \
		./cmd/pm

.PHONY: build-all
build-all:
	# macOS
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o dist/gz-pm-darwin-amd64 ./cmd/pm
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -o dist/gz-pm-darwin-arm64 ./cmd/pm

	# Linux
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o dist/gz-pm-linux-amd64 ./cmd/pm
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o dist/gz-pm-linux-arm64 ./cmd/pm

	# Windows
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/gz-pm-windows-amd64.exe ./cmd/pm

.PHONY: test
test:
	CGO_ENABLED=0 go test -v ./...
```

### CI Configuration

```yaml
# .github/workflows/build.yml
name: Build

on: [push, pull_request]

jobs:
  build:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}

    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.26.7'

    - name: Build
      run: make build
      env:
        CGO_ENABLED: 0

    - name: Test
      run: make test
      env:
        CGO_ENABLED: 0
```

### Dependency Check Script

```bash
#!/bin/bash
# scripts/check-cgo.sh
# Ensure no CGO dependencies sneak in

echo "Checking for CGO dependencies..."

# List all dependencies
DEPS=$(go list -deps ./...)

# Check each dependency for CGO usage
for dep in $DEPS; do
    if go list -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' "$dep" 2>/dev/null | grep -q .; then
        echo "❌ VIOLATION: $dep requires CGO"
        exit 1
    fi
done

echo "✅ No CGO dependencies found"
```

### Pre-commit Hook

```bash
# .git/hooks/pre-commit
#!/bin/bash
./scripts/check-cgo.sh || {
    echo "Commit blocked: CGO dependency detected"
    exit 1
}
```

---

## Pure Go Alternatives

### Database (v1.1+ if needed)

```go
// INSTEAD OF:
// import _ "github.com/mattn/go-sqlite3"  // Requires CGO

// USE:
import _ "modernc.org/sqlite"  // Pure Go

// Usage is identical (both use database/sql)
db, err := sql.Open("sqlite", "gz-pm.db")
```

**Performance**: modernc.org/sqlite is 5-10% slower, but still > 10,000 ops/sec (more than enough).

### Compression (if needed)

```go
// Go stdlib is sufficient
import (
    "compress/gzip"
    "compress/zlib"
)

// No need for C compression libraries
```

### Crypto

```go
// Go stdlib crypto is excellent
import (
    "crypto/sha256"
    "crypto/aes"
    "crypto/rand"
)

// No need for OpenSSL bindings
```

---

## Exception Process

If a future requirement **absolutely needs** CGO:

1. **Document why pure Go is insufficient**
   - Performance benchmarks
   - Feature gaps
   - Alternatives considered

2. **Write new ADR**
   - ADR-XXX: Exception for [specific library]
   - Must justify breaking this rule

3. **Architecture review**
   - Team consensus required
   - Security review (attack surface)

4. **Update build system**
   - Document cross-compilation requirements
   - Update CI/CD
   - Provide Docker build environment

**Examples that might justify exception**:
- GPU acceleration for ML (unlikely for package manager)
- Platform-specific APIs not available via syscall (rare)
- Critical performance bottleneck (must prove with benchmarks)

---

## Validation

### Build Validation

```bash
# Verify binary has no dynamic dependencies
ldd build/gz-pm
# Expected: "not a dynamic executable"

# Verify CGO is disabled
go version -m build/gz-pm | grep CGO
# Expected: CGO_ENABLED=0
```

### Cross-Compilation Test

```bash
# Build for all platforms (should succeed without extra tools)
make build-all

# Verify all binaries exist
ls -lh dist/
# Should show:
# gz-pm-darwin-amd64
# gz-pm-darwin-arm64
# gz-pm-linux-amd64
# gz-pm-linux-arm64
# gz-pm-windows-amd64.exe
```

### Dependency Check

```bash
# Run CGO check in CI
./scripts/check-cgo.sh
# Expected: ✅ No CGO dependencies found
```

### Success Criteria

- [ ] `CGO_ENABLED=0` in all Makefiles
- [ ] All builds succeed without C compiler
- [ ] Cross-compilation works from any platform
- [ ] CI builds complete in < 5 minutes
- [ ] Binaries work on minimal Linux (Alpine)
- [ ] No dynamic library dependencies
- [ ] Dependency check script passes

---

## Case Studies

### Example 1: SQLite (If Needed in Future)

**Scenario**: v1.1 wants to use SQLite for config storage.

**CGO Approach** (rejected):
```go
import _ "github.com/mattn/go-sqlite3"  // Requires CGO
// Breaks cross-compilation
// Adds libc dependency
```

**Pure Go Approach** (accepted):
```go
import _ "modernc.org/sqlite"  // Pure Go
// Works everywhere
// Slightly slower (acceptable)
```

### Example 2: Compression

**Scenario**: Need to compress config files.

**CGO Approach** (rejected):
```go
// Use C zlib for 10% better performance
import "github.com/klauspost/compress/zlib"  // Uses CGO
```

**Pure Go Approach** (accepted):
```go
import "compress/gzip"  // Go stdlib
// Performance difference negligible for our data sizes
```

### Example 3: Platform Detection

**Scenario**: Detect platform-specific details.

**No CGO needed**:
```go
import "runtime"

func getPlatform() Platform {
    switch runtime.GOOS {
    case "darwin":
        return PlatformMacOS
    case "linux":
        return PlatformLinux
    case "windows":
        return PlatformWindows
    }
}

// No C needed, pure Go runtime package
```

---

## Alternatives Considered

### Alternative 1: Allow CGO Selectively

**Proposal**: Allow CGO only for "critical" dependencies.

**Rejected because**:
- Slippery slope (what's "critical"?)
- Complicates build system
- Hard to maintain (some builds work, others don't)

### Alternative 2: Provide Two Builds

**Proposal**:
- `gz-pm` (pure Go, default)
- `gz-pm-cgo` (with CGO, optional)

**Rejected because**:
- User confusion ("which version?")
- Doubles maintenance burden
- Doubles CI time
- No clear benefit (pure Go is sufficient)

### Alternative 3: Use CGO Only for Optional Features

**Proposal**: Core is pure Go, plugins can use CGO.

**Rejected for v1.0**:
- We don't have plugin system yet
- Premature optimization
- Can reconsider in v2.0

---

## References

- **Go Build Modes**: https://pkg.go.dev/cmd/go#hdr-Build_modes
- **CGO Documentation**: https://pkg.go.dev/cmd/cgo
- **Cross-Compilation Guide**: https://go.dev/doc/install/source#environment
- **Pure Go SQLite**: https://pkg.go.dev/modernc.org/sqlite
- **Static Binaries**: https://mt165.co.uk/blog/static-link-go/

---

## Revision History

| Date | Changes | Author |
|------|---------|--------|
| 2025-01-27 | Initial version | Claude Code |

---

## Status Notes

**Current Status**: Accepted

**Enforcement**:
- Makefile enforces `CGO_ENABLED=0`
- Pre-commit hook checks dependencies
- CI validates static binaries
- Code review checklist includes CGO check

**Future Review**: v1.2 (Month 5) - Evaluate if any features genuinely require CGO. If not, reaffirm this decision.
