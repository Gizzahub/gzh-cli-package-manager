# ADR-001: Standalone Package Extraction

**Date**: 2025-01-27
**Status**: Accepted
**Deciders**: Project Team
**Related**: ADR-002 (Clean Architecture), ADR-003 (Hexagonal Ports)

---

## Context

The PM (package manager) functionality currently exists within the gzh-cli monolith project (~11,662 lines across cmd/pm and internal/pm). We need to decide whether to:
1. Keep PM functionality within gzh-cli
2. Extract to standalone gzh-cli-package-manager project
3. Create shared library used by both

**Constraints**:
- gzh-cli serves multiple purposes (Git, IDE, network, PM)
- PM functionality is cohesive and self-contained
- Different release cycles desired for PM vs other tools
- PM could be valuable as standalone CLI or library

---

## Decision

**Extract PM functionality into standalone `gzh-cli-package-manager` project with binary name `pmctl`.**

**Scope**:
- All code from `cmd/pm` and `internal/pm`
- Related specifications from `specs/cli/pm`
- Integration tests from `test/integration/pm`
- Documentation specific to PM features

**Not Extracted**:
- Shared logging utilities (will be copied, not shared library)
- Git-related functionality
- IDE monitoring
- Network environment management

---

## Rationale

### Why Standalone Project?

**1. Independent Versioning and Releases**
- PM updates don't require releasing entire gzh-cli
- Can iterate faster on PM-specific features
- Users who only want PM don't need full gzh-cli

**2. Focused Scope and Clarity**
- Single responsibility: package manager orchestration
- Easier to explain: "Manage all your package managers with one tool"
- Lower cognitive load for contributors

**3. Reusability as Library**
- Can be imported by other Go projects
- Example: `import "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"`
- Enables automation scripts, custom integrations

**4. Team Scalability**
- Dedicated team can own PM project
- Clear ownership boundaries
- Parallel development without conflicts

**5. Documentation Quality**
- Focused documentation for PM use cases
- Easier to maintain specifications
- Better user onboarding (single-purpose tool)

### Why Not Shared Library?

**Alternative Considered**: Create `gzh-common` library with shared utilities.

**Rejected Because**:
- Increased maintenance burden (3 repos instead of 2)
- Versioning complexity (breaking changes affect both consumers)
- Premature abstraction (YAGNI - You Aren't Gonna Need It)
- Simple copy of utilities is sufficient for now

**Future Path**: If shared code grows significantly (>1,000 lines), revisit shared library approach.

---

## Consequences

### Positive ✅

1. **Clear Product Identity**
   - `pmctl` is discoverable standalone tool
   - Can build dedicated community around PM orchestration
   - Easier marketing/positioning

2. **Faster Innovation**
   - Add package managers without gzh-cli release cycle
   - Experiment with new features (TUI, plugins)
   - Respond quickly to package manager API changes

3. **Better Testing**
   - Isolated test suite runs faster
   - Docker integration tests don't affect gzh-cli CI
   - Coverage metrics specific to PM functionality

4. **Reduced Complexity**
   - New contributors understand smaller codebase
   - Issues are scoped to PM domain
   - Debugging is more straightforward

5. **Flexible Distribution**
   - Homebrew formula for just `pmctl`
   - Go install without gzh-cli dependencies
   - Smaller binary size (~15MB vs ~33MB)

### Negative ❌

1. **Increased Repository Overhead**
   - Two repos to maintain (gzh-cli + pmctl)
   - Duplicate CI/CD configuration
   - Separate issue tracking

   **Mitigation**: Use GitHub templates, share workflows

2. **Code Duplication**
   - Logger interface copied to pmctl
   - Config utilities duplicated
   - ~200 lines of shared code

   **Mitigation**: Acceptable for now, extract to library if grows >1,000 lines

3. **User Migration Required**
   - Existing `gz pm` users must transition to `pmctl`
   - Documentation updates needed
   - Support two tools during transition period

   **Mitigation**:
   - Maintain `gz pm` in gzh-cli as thin wrapper (calls pmctl)
   - Comprehensive migration guide
   - 6-month deprecation timeline

4. **Cross-Project Coordination**
   - Changes affecting both projects need coordination
   - Shared logger interface must stay compatible
   - Breaking changes require care

   **Mitigation**: Semantic versioning, clear deprecation policies

### Neutral 🔄

1. **Partial Compatibility with gzh-cli**
   - Core functionality maintained
   - Configuration format compatible
   - JSON output structure preserved
   - Command naming changed (`pmctl` vs `gz pm`)

2. **Build/Release Process Changes**
   - Separate goreleaser configuration
   - Independent GitHub Actions
   - Distinct release schedules

---

## Implementation Notes

### Migration Path

**Phase 1: Extract (Week 3)**
1. Copy `cmd/pm` → `cmd/pmctl`
2. Copy `internal/pm` → `pkg/{domain,application,infrastructure}` (refactored)
3. Copy tests → `test/`
4. Copy specs → `docs/specifications/`

**Phase 2: Refactor (Week 4-7)**
1. Apply Clean Architecture (see ADR-002)
2. Remove gzh-cli dependencies
3. Update tests for standalone execution
4. Create new CLI entry point

**Phase 3: Validation (Week 8)**
1. Integration tests pass on all platforms
2. Specification compliance verified (95%+)
3. Performance benchmarks meet targets
4. Documentation complete

**Phase 4: Release (Week 9)**
1. Beta release (v0.9.0)
2. User testing (10+ beta testers)
3. Address critical feedback
4. v1.0.0 release

### Compatibility Layer in gzh-cli

gzh-cli will maintain `gz pm` as thin wrapper:

```go
// gzh-cli/cmd/pm/pm.go
func Execute() {
    // Check if pmctl is installed
    if pmctlPath := exec.LookPath("pmctl"); pmctlPath != "" {
        // Delegate to pmctl
        cmd := exec.Command(pmctlPath, os.Args[2:]...)
        cmd.Run()
    } else {
        // Prompt to install pmctl
        fmt.Println("PM functionality moved to pmctl")
        fmt.Println("Install: brew install pmctl")
    }
}
```

**Deprecation Timeline**:
- Month 1-3: Both tools coexist, `gz pm` delegates to `pmctl`
- Month 4-6: Warning message when using `gz pm`
- Month 7+: `gz pm` removed from gzh-cli

---

## Validation

### Success Criteria

- [ ] pmctl works standalone (no gzh-cli dependency)
- [ ] All 120+ test scenarios pass
- [ ] 90%+ code coverage maintained
- [ ] Binary size < 20MB
- [ ] Installation methods work (brew, go install, binary)
- [ ] Migration guide successfully used by 10+ users
- [ ] Performance comparable to original (±10%)

### Risks

**Risk 1: Low Adoption**
- **Impact**: High
- **Mitigation**: Excellent UX, show value immediately, community building

**Risk 2: Feature Divergence**
- **Impact**: Medium
- **Mitigation**: Maintain compatibility layer, clear communication

**Risk 3: Ecosystem Fragmentation**
- **Impact**: Low
- **Mitigation**: pmctl can still be used as library by gzh-cli if needed

---

## Alternatives Considered

### Alternative 1: Keep in gzh-cli Monolith
**Rejected**: Violates Single Responsibility Principle, slower release cycle, harder to test in isolation

### Alternative 2: Shared Library Approach
**Rejected**: Premature abstraction, increased complexity for uncertain benefit

### Alternative 3: Complete Rewrite in Different Language
**Rejected**: Go is appropriate, rewrite wastes validated code, team knows Go

---

## References

- **PRD**: `/PRD.md` - Product vision for standalone tool
- **REQUIREMENTS**: `/REQUIREMENTS.md` - NFR-COMPAT-002 discusses compatibility
- **ARCHITECTURE**: `/ARCHITECTURE.md` - Clean architecture enables extraction
- **Original Discussion**: gzh-cli Planning Session 2025-01-27

---

## Revision History

| Date | Changes | Author |
|------|---------|--------|
| 2025-01-27 | Initial version | Claude Code |

---

## Status Notes

**Current Status**: Accepted and in implementation (Week 1 documentation phase)

**Future Review**: After v1.0 release (Week 9), evaluate if shared library is needed based on actual code duplication metrics.
