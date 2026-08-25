# ADR-004: Go 1.24+ Requirement

**Date**: 2025-01-27
**Status**: Superseded by [ADR-009](009-go-1.26-version-requirement.md)
**Deciders**: Project Team
**Related**: ADR-006 (No CGO Dependencies)

---

## Context

We need to decide minimum Go version for gzh-cli-package-manager. Considerations:
- Language features needed for clean code
- Compatibility with users' Go installations
- Performance improvements in newer versions
- Standard library enhancements

**Options**:
1. Go 1.21 (LTS, widespread adoption)
2. Go 1.22 (current stable at planning time)
3. Go 1.23 (current at implementation start)
4. Go 1.24+ (latest, cutting-edge features)

---

## Decision

**Require Go 1.24.0 as minimum version.**

**Justification**: Parent project (gzh-cli) already uses Go 1.24.0, and we benefit from latest language improvements.

---

## Rationale

### Why Go 1.24+?

**1. Range-over-Func (Iterators)**
- Clean iteration over custom types
- Example use case: Iterate managers
```go
// Go 1.24+ with iterators
for mgr := range repo.Managers() {
    process(mgr)
}

// vs Go 1.21
managers, err := repo.ListManagers()
for _, mgr := range managers {
    process(mgr)
}
```

**2. Improved Type Inference**
- Less verbose generic code
- Cleaner function signatures

**3. Enhanced Error Handling**
- Better error wrapping in stdlib
- Improved `errors.Join()` functionality

**4. Performance Improvements**
- Faster compilation times (~10-15% vs 1.21)
- Better runtime performance
- Improved GC in Go 1.23+

**5. Standard Library Enhancements**
- `log/slog` improvements (structured logging)
- `slices`, `maps` packages matured
- `cmp.Or()` for cleaner default value handling

**6. Compatibility with Parent Project**
- gzh-cli uses Go 1.24.0
- Shared development environment
- Same CI/CD Go version

### Why Not Older Versions?

**Go 1.21 (Rejected)**
- Missing iterators (range-over-func)
- Older `log/slog` (less features)
- Slower compilation

**Go 1.22 (Rejected)**
- Parent project already on 1.24
- Would require maintaining different version

**Go 1.23 (Considered)**
- Would be acceptable
- But 1.24 is available and provides additional improvements

---

## Consequences

### Positive ✅

1. **Modern Language Features**
   - Iterators for cleaner code
   - Better generics support
   - Improved error handling patterns

2. **Performance Benefits**
   - Faster builds (matters for large codebase)
   - Better runtime performance
   - Smaller binaries with better optimization

3. **Standard Library Improvements**
   - `log/slog` for structured logging (no external deps)
   - `slices`, `maps` packages reduce utility code
   - `cmp` package for comparisons

4. **Future-Proof**
   - Go 1.24 will be supported until at least 2026
   - Keeps us on modern toolchain
   - Easier to adopt future features

5. **Consistency with Ecosystem**
   - gzh-cli already on 1.24
   - Many Go projects moving to 1.23+
   - Good developer experience

### Negative ❌

1. **User Installation Requirements**
   - Users must have Go 1.24.0+ to build from source
   - Some users may be on older versions
   - Corporate environments may lag

   **Mitigation**:
   - Provide pre-built binaries (most users use these)
   - Document installation instructions
   - Go is easy to upgrade

2. **CI/CD Environment**
   - Must ensure CI runners have Go 1.24+
   - GitHub Actions: easy (specify in workflow)
   - Self-hosted runners: may need upgrades

   **Mitigation**:
   - GitHub Actions supports all versions
   - Docker images with Go 1.24+
   - Documented in CONTRIBUTING.md

3. **Potentially Cutting-Edge**
   - Go 1.24 is relatively new (as of 2025-01)
   - May have undiscovered bugs
   - Smaller community experience base

   **Mitigation**:
   - Go team has strong testing
   - We're not using experimental features
   - Can downgrade to 1.23 if critical issues found

### Neutral 🔄

1. **No Backward Compatibility Needed**
   - This is a new project (no legacy code)
   - Can use latest features without migration

2. **Cross-Compilation**
   - Go 1.24 cross-compiles well
   - Supports all target platforms (macOS, Linux, Windows)

---

## Implementation

### go.mod Declaration

```go
// go.mod
module github.com/gizzahub/gzh-cli-package-manager

go 1.24

require (
    github.com/spf13/cobra v1.9.1
    gopkg.in/yaml.v3 v3.0.1
    // ...
)
```

### Version Check in Makefile

```makefile
# Makefile
.PHONY: check-go-version
check-go-version:
	@GO_VERSION=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	REQUIRED="1.24"; \
	if [ "$$(printf '%s\n' "$$REQUIRED" "$$GO_VERSION" | sort -V | head -n1)" != "$$REQUIRED" ]; then \
		echo "Error: Go $$REQUIRED or higher required, found $$GO_VERSION"; \
		exit 1; \
	fi

build: check-go-version
	go build -o build/gz-pm ./cmd/pm
```

### CI/CD Configuration

```yaml
# .github/workflows/test.yml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'  # Explicit version
    - run: make test
```

### Documentation

**README.md**:
```markdown
## Requirements

- Go 1.24.0 or higher

### Installing Go

**macOS**:
\`\`\`bash
brew install go@1.24
\`\`\`

**Linux**:
\`\`\`bash
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
\`\`\`

**Verify**:
\`\`\`bash
go version  # Should show go1.24 or higher
\`\`\`
```

---

## Language Features We Use

### Iterators (Go 1.23+, refined in 1.24)

```go
// pkg/domain/manager/repository.go
type ManagerRepository interface {
    // Iterator pattern for streaming managers
    All(ctx context.Context) iter.Seq[*Manager]
}

// Usage
for mgr := range repo.All(ctx) {
    if mgr.Installed {
        fmt.Println(mgr.Name)
    }
}
```

### Structured Logging (log/slog - Go 1.21+, improved in 1.24)

```go
// internal/logger/structured_logger.go
import "log/slog"

type StructuredLogger struct {
    logger *slog.Logger
}

func (l *StructuredLogger) Info(msg string, fields ...Field) {
    l.logger.Info(msg, fieldsToAttrs(fields)...)
}
```

### Slices Package

```go
import "slices"

// Filter stable versions
stableVersions := slices.DeleteFunc(versions, func(v string) bool {
    return strings.Contains(v, "beta") || strings.Contains(v, "rc")
})

// Sort managers by priority
slices.SortFunc(managers, func(a, b *Manager) int {
    return cmp.Compare(a.Priority, b.Priority)
})
```

### Comparison Package (cmp)

```go
import "cmp"

// Default values with cmp.Or
version := cmp.Or(userVersion, defaultVersion, "latest")

// Compare with cmp.Compare
func compareVersions(a, b string) int {
    return cmp.Compare(semver.Parse(a), semver.Parse(b))
}
```

---

## Validation

### Version Enforcement

**Build-time check**:
```bash
go version
# Fails if Go < 1.24
```

**Runtime check** (optional, for library usage):
```go
// pkg/version/check.go
func init() {
    version := runtime.Version()
    if !strings.HasPrefix(version, "go1.24") && !strings.HasPrefix(version, "go1.25") {
        log.Fatal("Go 1.24+ required")
    }
}
```

### Success Criteria

- [ ] go.mod specifies `go 1.24`
- [ ] Makefile checks Go version before build
- [ ] CI/CD uses Go 1.24
- [ ] README documents requirement
- [ ] All developers using Go 1.24+

---

## Migration Path (If Needed)

If critical bug found in Go 1.24:

1. **Downgrade to 1.23**:
   ```bash
   # Update go.mod
   go 1.23

   # Remove 1.24-specific features
   # - Avoid latest stdlib additions
   # - Test thoroughly
   ```

2. **Community Communication**:
   - Issue announcement
   - Update README
   - Tag new release

**Likelihood**: Very low (Go has excellent stability)

---

## Alternatives Considered

### Alternative 1: Go 1.21 (LTS)
**Rejected**: Missing modern features, parent project uses 1.24

### Alternative 2: Go 1.22
**Rejected**: Parent project already on 1.24, no benefit to being behind

### Alternative 3: Flexible "1.21+"
**Rejected**: Makes it harder to use new features, code becomes conservative

---

## References

- **Go 1.24 Release Notes**: https://go.dev/doc/go1.24
- **Go 1.23 Release Notes**: https://go.dev/doc/go1.23
- **Go Version Policy**: https://go.dev/doc/devel/release
- **Parent Project** (gzh-cli): Uses Go 1.24.0 in go.mod

---

## Revision History

| Date | Changes | Author |
|------|---------|--------|
| 2025-01-27 | Initial version | Claude Code |

---

## Status Notes

**Current Status**: Accepted

**Go Version Guarantee**: We will support Go 1.24+ for entire v1.x lifecycle. Major version bump (v2.0) would be required to change minimum Go version.

**Upgrade Policy**: We may adopt newer Go versions (1.25, 1.26) but will maintain backward compatibility with 1.24 until v2.0.
