# Per-Manager CLI (winget / scoop / chocolatey)

**Status**: Implemented (MVP — list + search)
**Related issues**: gzh-cli `tasks/issue/22`, `23`, `24`
**Binary**: `gz-pm` (also available as `gz pm …` when embedded)

## Intent

Unified commands (`status`, `update`) already orchestrate Windows managers via adapters.
Per-manager commands expose **manager-native** list/search through those same adapters —
they wrap the native CLI; they do not reimplement package management.

## Command structure

```text
gz-pm <manager> <subcommand> [args] [--output text|json]
```

| Manager CLI name | Adapter ID | Native binary |
|------------------|------------|---------------|
| `winget`         | `winget`   | `winget`      |
| `scoop`          | `scoop`    | `scoop`       |
| `chocolatey`     | `choco`    | `choco`       |

### Subcommands (MVP)

| Subcommand | Args | Behavior |
|------------|------|----------|
| `list`     | none | Detect → `ListPackages` via adapter |
| `search`   | `<query>` (required) | Detect → `Search` via adapter |

Examples:

```bash
gz-pm winget list
gz-pm winget search git
gz-pm scoop list --output json
gz-pm scoop search 7zip
gz-pm chocolatey list
gz-pm chocolatey search git -o json
```

### Out of scope for this MVP

- `install` / `uninstall` / `upgrade` / `bucket` / UAC elevation / progress bars
- Live Windows integration tests (unit tests use injectable `CommandExecutor` mocks)

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

## Error policy

| Condition | Exit | Message pattern |
|-----------|------|-----------------|
| Adapters not injected | non-zero | `<manager>: adapters not initialized` |
| Adapter missing for ID | non-zero | `<manager>: adapter not registered` |
| Manager not installed / not on PATH | non-zero | `<manager> is not available on this system (not installed or not in PATH)` |
| Detect call fails | non-zero | `<manager>: detect failed: …` |
| List/search native command fails | non-zero | `<manager> list/search failed: …` |
| Search without query | non-zero | cobra `ExactArgs(1)` validation |
| Unknown `--output` | non-zero | `unknown output format "…" (supported: text, json)` |
| Search not implemented by adapter | non-zero | `<manager> does not support search` |

Notes:

- Manager subcommands **always register** on every platform (including non-Windows).
- Runtime path always goes through the adapter: **Detect first**, then list/search.
- Clear “not available” is preferred over silent no-op when the binary is missing.
- Commands use `RunE` (errors bubble to Cobra); no hidden swallow of failures.

## Architecture

```text
cmd/pm/command (presentation)
    → adapterm.Adapter / adapterm.Searcher
        → CommandExecutor (mockable)
            → native winget | scoop | choco
```

- `SetManagerAdapters` injects the same adapter map used by `update`.
- Winget / Scoop / Chocolatey implement optional `adapterm.Searcher`.
- Unit tests inject `testutil.MockExecutor`; no live Windows required.

## Acceptance (issues 22–24)

| AC | Status |
|----|--------|
| Spec: command structure / output / error policy | ✅ this document |
| `gz-pm winget <subcommand>` ≥1 + tests | ✅ `list`, `search` + mock executor tests |
| `gz-pm scoop <subcommand>` ≥1 + tests | ✅ `list`, `search` + mock executor tests |
| `gz-pm chocolatey <subcommand>` ≥1 + tests | ✅ `list`, `search` + mock executor tests |

Advanced scope (sources/buckets/UAC/progress) remains open for follow-up.
