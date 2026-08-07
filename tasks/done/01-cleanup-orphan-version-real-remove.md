# cleanup orphans/versions — real remove (safety-first)

- status: done
- priority: P3
- effort: M
- source: gzh-cli issue 25 residual (transferred 2026-08-07)
- repo: gzh-cli-package-manager

## Why

Issue 25 MVP already ships:

- `cleanup cache scan|clean`
- `cleanup quarantine purge`
- `cleanup orphans list` / `versions list` (**heuristic candidates only**)

What remains is **actually removing** orphan/old-version packages with manager-specific
safety rules. That is product-sensitive (destructive) and does not belong in the
gzh-cli git provider workstream.

## Scope

1. Define per-manager orphan semantics (at least one manager fully: e.g. homebrew
   `brew leaves` / apt auto-installed, or document “heuristic only” for Windows PMs).
2. `cleanup orphans remove [--dry-run] [--manager]` — default dry-run; require confirm
   for live delete.
3. `cleanup versions remove [--dry-run]` — keep newest N versions (configurable).
4. Tests with mock adapters/executors (no live package managers required).
5. Update `docs/specifications/use-cases/UC-006-cleanup.md`.

## Out of scope

- Cross-manager dependency graphs
- Full quarantine file-backup restore (separate task if needed)
- `gz` binary default-shipping of `pm` (`pm_external` tag)

## Acceptance

- [x] dry-run is default for remove paths
- [x] at least one manager has real remove wired through adapter/executor
- [x] unit tests for dry-run vs live path
- [x] clear error when manager not detected / not supported

## References

- `cmd/pm/command/cleanup.go`
- `pkg/domain/cleanup/interfaces.go` (`OrphanDetector`, `VersionScanner`, `Executor`)
- Parent history: gzh-cli `tasks/issue/25-advanced-cleanup-strategies-implementation.md`


## Resolution (2026-08-07)

- `AdapterCleanupExecutor` + `orphans remove` / `versions remove`
- dry-run defaults **true**; live with `--dry-run=false`
- Uses `adapterm.Installer` (winget/scoop/chocolatey, …)
- Heuristic candidates only (not full dependency graphs) — documented
