# Bounded updates

`gz-pm update --bounded` is an opt-in, resumable update mode for explicitly
selected mise tools and Homebrew formulae. It never expands an empty selection
into a manager-wide update.

Use a durable state path and no more than three selected packages per run:

```sh
gz-pm update --managers mise --bounded --batch-size 3 \
  --state-file "$HOME/Library/Application Support/gz-pm/bounded.json" \
  --max-duration 15m --min-free-gib 100 --max-swap-gib 10
```

The state file records complete, failed, and deferred packages. A later run
with the same state path resumes deferred work. POSIX hosts hold an advisory
lock while a run is active. A prior `running` or `failed` entry requires
human inspection and an explicit state reset; it is never retried automatically.

Exit status `3` means work was deferred because a resource, batch, or duration
limit prevented a new package from starting. It does not interrupt a package
already handed to its package manager. Normal update failures retain the usual
nonzero failure result.

On macOS each package start checks free disk and, when `--max-swap-gib` is set,
memory pressure and swap usage. An unavailable measurement fails closed. Linux
checks disk; Windows rejects bounded updates because its lock and resource
measurement contract is not implemented.

Homebrew selections live in the update preferences file:

```yaml
version: 1
managers:
  brew:
    strategy: fixed # report-only until the allowlist is deliberately enabled
    packages: [jq, ripgrep]
  mise:
    strategy: micro
    tools: [go, pnpm]
```

Use `--dry-run` with a `fixed` policy; a non-dry bounded run rejects it so a
report-only policy can never advance the persistent state. For Homebrew, each selected package runs as `brew upgrade <package>`. Homebrew
may install or update dependencies required by that package; gz-pm reports the
direct selected package and cannot precompute every dependency change. The
existing `fixed` strategy does not perform package upgrades.
