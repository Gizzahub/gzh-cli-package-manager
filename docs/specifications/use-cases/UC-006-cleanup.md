# UC-006: Cleanup (cache scan/clean, quarantine purge)

**Status**: Implemented (minimal executable slice)
**Commands**: `gz-pm cleanup …`
**Last Updated**: 2026-08-07

## Intent

Provide safe, inspectable cleanup for package-manager side-effects:

1. Discover known cache directories and report size/entry counts.
2. Clear those caches with mandatory dry-run support.
3. Purge expired quarantine records by retention policy.

This is **not** a package uninstall/reinstall orchestrator. Quarantine file
backup/restore remains a follow-up.

## Subcommands

| Command | Behavior | Mutates disk? |
|---------|----------|---------------|
| `cleanup cache scan [--manager ID]` | Resolve known paths under `$HOME` (and `npm_config_cache`), walk tree, record `CacheInfo` | No |
| `cleanup cache status` | Print last scan results (process-local store) | No |
| `cleanup cache clean [--manager ID] [--dry-run]` | Scan then clear directory **contents** | Yes (unless dry-run) |
| `cleanup quarantine list [--manager ID]` | List quarantine records | No |
| `cleanup quarantine expired [--retention N]` | List records older than N days | No |
| `cleanup quarantine purge [--retention N] [--dry-run]` | Delete expired quarantine records | Yes (metadata; unless dry-run) |

## Known cache managers

| Manager ID | Default path (relative to home) | Notes |
|------------|----------------------------------|-------|
| homebrew | `Library/Caches/Homebrew` (darwin), `.cache/Homebrew` (linux) | First GOOS match wins |
| npm | `.npm` | Overridden by `npm_config_cache` |
| pip | `.cache/pip` | |
| cargo | `.cargo/registry/cache` | |
| yarn | `.cache/yarn` | |
| go | `go/pkg/mod/cache/download` | |
| pnpm | `Library/pnpm/store` (darwin), `.local/share/pnpm/store` (linux) | |

Missing paths are skipped (not errors). Unreadable nodes inside a tree are skipped.

## Flags

| Flag | Default | Scope |
|------|---------|-------|
| `--manager` | `""` (all) | cache + quarantine list |
| `--dry-run` | `false` | clean, purge |
| `--retention` | `30` | expired, purge |

## Error policy

- Invalid `--retention` (≤ 0) → fail with `ErrInvalidRetentionDays`.
- Clean failures per path are collected in `Summary.Errors`; overall error is
  `ErrCacheClearFailed` when any path fails.
- No fabricated status: empty scan/list prints an explicit “none found” message.

## Safety

- `--dry-run` always runs **before** any delete path (scan-only for sizing).
- Clean removes **children** of the cache directory when listing succeeds; it
  does not require root/sudo.
- No `sh -c` shell interpolation.

## Test coverage

- Path resolution unit tests (`ResolveCachePaths`)
- Scan/clean with injectable `FileSystem` (`MapFileSystem` + temp OS FS)
- Quarantine purge dry-run vs execute
- CLI wiring tests for `cache scan`, `cache clean --dry-run`, `quarantine purge`

## Out of scope (follow-ups)

- Persistent quarantine/cache store across CLI process restarts
- Orphan package detection (`cleanup orphans`)
- Multi-version cleanup (`cleanup versions`)
- Actual package uninstall + file backup for quarantine
