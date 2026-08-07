# Per-Manager CLI (winget / scoop / chocolatey)

**Status**: Implemented (list + search + install/uninstall/upgrade + sources/buckets)
**Related issues**: gzh-cli `tasks/issue/22`, `23`, `24`
**Binary**: `gz-pm` (also available as `gz pm …` when embedded)

## Intent

Unified commands (`status`, `update`) already orchestrate Windows managers via adapters.
Per-manager commands expose **manager-native** list/search/install/uninstall through those
same adapters — they wrap the native CLI; they do not reimplement package management.

## Command structure

```text
gz-pm <manager> <subcommand> [args] [--output text|json] [--dry-run]
```

| Manager CLI name | Adapter ID | Native binary |
|------------------|------------|---------------|
| `winget`         | `winget`   | `winget`      |
| `scoop`          | `scoop`    | `scoop`       |
| `chocolatey`     | `choco`    | `choco`       |

### Common subcommands

| Subcommand  | Args | Flags | Behavior |
|-------------|------|-------|----------|
| `list`      | none | `--output` | Detect → `ListPackages` via adapter |
| `search`    | `<query>` (required) | `--output` | Detect → `Search` via adapter |
| `install`   | `<id>` (required) | `--dry-run` | Detect → `Installer.Install` |
| `uninstall` | `<id>` (required) | `--dry-run` | Detect → `Installer.Uninstall` |
| `upgrade`   | `[id]` optional | `--all`, `--dry-run` | Detect → `Adapter.Update` |

### Manager-specific subcommands

| Manager | Subcommand | Args | Behavior |
|---------|------------|------|----------|
| winget | `source list` | none | `SourceLister.ListSources` → parse `winget source list` |
| scoop | `bucket list` | none | `BucketManager.ListBuckets` |
| scoop | `bucket add` | `<name> [url]` | `BucketManager.AddBucket` |
| scoop | `bucket remove` / `rm` | `<name>` | `BucketManager.RemoveBucket` |

Examples:

```bash
gz-pm winget list
gz-pm winget search git
gz-pm winget install Git.Git --dry-run
gz-pm winget uninstall Git.Git --dry-run
gz-pm winget upgrade --all --dry-run
gz-pm winget source list

gz-pm scoop list --output json
gz-pm scoop search 7zip
gz-pm scoop install git --dry-run
gz-pm scoop uninstall git --dry-run
gz-pm scoop upgrade --all
gz-pm scoop bucket list
gz-pm scoop bucket add extras
gz-pm scoop bucket remove extras

gz-pm chocolatey list
gz-pm chocolatey search git -o json
gz-pm chocolatey install git --dry-run
gz-pm chocolatey uninstall git --dry-run
gz-pm chocolatey upgrade --all --dry-run
```

### Dry-run policy

- `install` / `uninstall`: adapter skips the native install/uninstall command; CLI prints
  `Dry-run: would install|uninstall …`.
- `upgrade`: uses existing `UpdateOptions.DryRun` on `Adapter.Update` (no native upgrade).
- No real UAC elevation is performed. Chocolatey elevation-related executor errors are
  rewritten with a clear “run as Administrator” hint.

### Optional capability interfaces

```go
type Installer interface {
  Install(ctx, pkgID string, dryRun bool) error
  Uninstall(ctx, pkgID string, dryRun bool) error
}

type SourceLister interface {
  ListSources(ctx) ([]Source, error)
}

type BucketManager interface {
  ListBuckets(ctx) ([]Bucket, error)
  AddBucket(ctx, name, url string) error
  RemoveBucket(ctx, name string) error
}
```

Implemented by:

| Adapter | Installer | SourceLister | BucketManager | Searcher |
|---------|-----------|--------------|---------------|----------|
| winget | ✅ | ✅ | — | ✅ |
| scoop | ✅ | — | ✅ | ✅ |
| chocolatey | ✅ (+ UAC wrap) | — | — | ✅ |

### Out of scope / residual

- Progress bars / install streaming output
- Real UAC elevation prompts
- Manifest display for Scoop
- Live Windows integration tests (unit tests use injectable `CommandExecutor` mocks)
- Advanced orphan dependency graphs (see cleanup orphans/versions best-effort)

## Output formats

### text (default)

```text
winget list — 2 package(s)
  Git  2.43.0
  VisualStudioCode  1.85.0 → 1.86.0
```

Empty result:

```text
No packages found (scoop search).
```

### json

```json
{
  "manager": "chocolatey",
  "action": "list",
  "count": 1,
  "packages": [
    {
      "name": "git",
      "current_version": "2.43.0",
      "manager": "choco"
    }
  ]
}
```

Source list JSON:

```json
{
  "count": 1,
  "sources": [{ "name": "winget", "arg": "https://…" }]
}
```

Bucket list JSON:

```json
{
  "count": 1,
  "buckets": [{ "name": "main", "source": "https://…" }]
}
```

## Error policy

| Condition | Exit | Message pattern |
|-----------|------|-----------------|
| Adapters not injected | non-zero | `<manager>: adapters not initialized` |
| Adapter missing for ID | non-zero | `<manager>: adapter not registered` |
| Manager not installed / not on PATH | non-zero | `<manager> is not available on this system (not installed or not in PATH)` |
| Detect call fails | non-zero | `<manager>: detect failed: …` |
| List/search/install/uninstall/upgrade fails | non-zero | `<manager> <action> failed: …` |
| Search/install without required args | non-zero | cobra `ExactArgs` / `RangeArgs` validation |
| Unknown `--output` | non-zero | `unknown output format "…" (supported: text, json)` |
| Capability not implemented | non-zero | `<manager> does not support <capability>` |
| Chocolatey elevation-related failure | non-zero | original error + `(hint: … re-run as Administrator)` |

Notes:

- Manager subcommands **always register** on every platform (including non-Windows).
- Runtime path always goes through the adapter: **Detect first**, then action.
- Clear “not available” is preferred over silent no-op when the binary is missing.
- Commands use `RunE` (errors bubble to Cobra); no hidden swallow of failures.

## Architecture

```text
cmd/pm/command (presentation)
    → adapterm.Adapter / Searcher / Installer / SourceLister / BucketManager
        → CommandExecutor (mockable)
            → native winget | scoop | choco
```

- `SetManagerAdapters` injects the same adapter map used by `update`.
- Unit tests inject `testutil.MockExecutor`; no live Windows required.

## Acceptance (issues 22–24)

| AC | Status |
|----|--------|
| Spec: command structure / output / error policy | ✅ this document |
| `gz-pm winget` list/search/install/uninstall/upgrade/source list + tests | ✅ mock executor |
| `gz-pm scoop` list/search/install/uninstall/upgrade/bucket + tests | ✅ mock executor |
| `gz-pm chocolatey` list/search/install/uninstall/upgrade + UAC wrap + tests | ✅ mock executor |

Progress bars and real UAC elevation remain out of scope.
