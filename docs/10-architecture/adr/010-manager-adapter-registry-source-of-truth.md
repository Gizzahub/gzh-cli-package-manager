# ADR-010: Manager Adapter Registry Source of Truth

**Date**: 2026-09-01  
**Status**: Proposed — design prepared; implementation approval pending  
**Deciders**: package-manager maintainer (design review pending)  
**Related**: TASK-121, TASK-120, [ADR-002](002-clean-architecture.md),
[ADR-003](003-hexagonal-ports-adapters.md)

## Context

The CLI composition root and the in-memory detecting repository each construct a
manager adapter registry. TASK-120 exposed the risk: APT and Pacman were already
available to detection but were missing from the CLI map. The immediate wiring fix
added both entries to `cmd/gz-pm/main.go`, but the two construction sites still have
independent lists.

The two lists currently contain the same ten registered IDs, but their declaration
orders differ. Go map iteration order is deliberately unspecified, so declaration
order is not a runtime contract. The contract that can drift is the supported ID
set, the constructor associated with each ID, and the consumer path that receives
the resulting adapter.

The domain also declares `sdkman` and `yay` IDs. They are not registered in either
runtime map and therefore remain unsupported until a separate support decision and
adapter implementation exist.

## Inventory

| Manager ID | CLI composition root | Detection repository | Constructor |
|---|---|---|---|
| `apt` | `cmd/gz-pm/main.go:newManagerAdapters` | `pkg/infrastructure/repository/memory/detecting_manager.go:registerAdapters` | `apt.NewAdapter` |
| `brew` | same | same | `homebrew.NewAdapter` |
| `pacman` | same | same | `pacman.NewAdapter` |
| `npm` | same | same | `npm.NewAdapter` |
| `pip` | same | same | `pip.NewAdapter` |
| `cargo` | same | same | `cargo.NewAdapter` |
| `asdf` | same | same | `asdf.NewAdapter` |
| `winget` | same | same | `winget.NewAdapter` |
| `scoop` | same | same | `scoop.NewAdapter` |
| `choco` | same | same | `chocolatey.NewAdapter` |

The CLI registry is passed to the update use case and command-layer per-manager
operations. The detecting repository's registry is used for detection and status
queries. Both constructors receive the same executor and logger supplied by their
composition owner. Neither registry owns privilege escalation; that behavior stays
inside the existing adapter contracts.

## Decision

The preferred implementation is an infrastructure-owned factory in a separate
subpackage, for example `pkg/infrastructure/adapter/registry`. The package may
import the adapter interface and concrete adapter subpackages, then return
`map[manager.ManagerID]adapter.Adapter`. Keeping the factory outside
`pkg/infrastructure/adapter/manager` avoids an import cycle: every concrete adapter
already imports that package for the shared interface.

The CLI composition root and detecting repository should consume this factory rather
than maintain their own constructor lists. The factory's key set and constructor
mapping become the single source of truth. A focused drift test should assert:

1. every supported runtime ID has a non-nil adapter;
2. the registry contains no duplicate or unexpected IDs; and
3. both consumers receive the same key set.

If an implementation review finds that the factory would violate package ownership
or create an import cycle, the fallback is a declarative supported-manager matrix in
the infrastructure layer plus a test that compares each consumer's key set and
constructor coverage. The fallback must preserve the same invariants and must not
move adapter construction into the presentation layer.

## Dependency and Behavior Boundaries

- The dependency direction remains domain → application → infrastructure →
  presentation; the repository must not import `cmd/gz-pm` or command packages.
- Registry consolidation must not add managers, alter manager IDs, or change
  adapter command parsing, privilege behavior, or dry-run semantics.
- Map declaration or iteration order is not observable behavior. If output needs a
  stable order, the consumer must sort IDs explicitly rather than relying on map
  order.
- `sdkman` and `yay` remain out of the registry until their support is separately
  approved and implemented.

## Deferred Implementation and Verification

Implementation is a separate follow-up after this design review. The follow-up must
extract the factory, wire both consumers, and add key-set/non-nil constructor tests.
It must then run the existing focused repository and CLI tests, the adapter tests,
`make test-unit`, and `make lint`. Native package-manager update commands and
privilege-sensitive E2E remain outside this refactor.

## Consequences

Centralizing construction makes a newly supported manager visible in one place and
prevents the TASK-120 class of CLI/detection drift. The infrastructure registry
package will import all concrete adapters, so constructor ownership is explicit and
the package remains an outer-layer concern. A small amount of wiring code moves from
the composition root and repository, but no public domain or application contract
changes.

