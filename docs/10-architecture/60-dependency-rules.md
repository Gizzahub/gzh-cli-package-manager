# 6. Dependency Rules

> gzh-cli-package-manager 아키텍처 문서 · [인덱스](README.md) · [ARCHITECTURE.md](../../ARCHITECTURE.md)

### 6.1 Layer Dependencies

```
Presentation → Application → Domain ← Infrastructure
     ↓              ↓
     └──────────────┴─ Infrastructure (for DI only)
```

**Rules**:

1. **Domain Layer**:
   - ✅ CAN: Use Go stdlib
   - ❌ CANNOT: Import any other layer
   - ❌ CANNOT: Import external libraries (except testing)

2. **Application Layer**:
   - ✅ CAN: Import domain layer
   - ✅ CAN: Define port interfaces
   - ❌ CANNOT: Import infrastructure implementations
   - ❌ CANNOT: Import presentation layer

3. **Infrastructure Layer**:
   - ✅ CAN: Import domain layer (to implement interfaces)
   - ✅ CAN: Import external libraries
   - ✅ CAN: Import application ports (to implement them)
   - ❌ CANNOT: Import presentation layer

4. **Presentation Layer**:
   - ✅ CAN: Import application layer
   - ✅ CAN: Import infrastructure (for DI only)
   - ✅ CAN: Import frameworks (Cobra)
   - ❌ CANNOT: Bypass application layer (no direct adapter calls)

**Validation**:

```bash
# Check for dependency violations
go list -test -deps ./pkg/domain/... | grep -E "(infrastructure|application|cmd)"
# Should return empty (no violations)

go list -test -deps ./pkg/application/... | grep -E "(infrastructure|cmd)"
# Should return empty (no violations)
```

---

### 6.2 Import Rules

```go
// ✅ ALLOWED
// pkg/application/update/update_all.go
package update

import (
    "context"
    "github.com/gizzahub/gzh-cli-package-manager/pkg/domain/manager"  // ✅ App → Domain
    "github.com/gizzahub/gzh-cli-package-manager/pkg/application/port"  // ✅ Same layer
)

// ❌ FORBIDDEN
// pkg/domain/manager/entity.go
package manager

import (
    "github.com/gizzahub/gzh-cli-package-manager/pkg/application/update"  // ❌ Domain → App
    "github.com/spf13/cobra"  // ❌ Domain → Framework
)
```
