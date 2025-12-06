# Context Documentation - gzh-cli-package-manager

This directory contains detailed context documentation extracted from CLAUDE.md for LLM optimization.

## Purpose

Keep CLAUDE.md under 300 lines while maintaining comprehensive guidance through linked context documents.

## Files

| File | Purpose | When to Read |
|------|---------|--------------|
| [architecture-guide.md](architecture-guide.md) | Architecture layers, ADRs, DI pattern | Before major changes |
| [testing-guide.md](testing-guide.md) | Test organization, coverage targets | Writing tests |
| [build-guide.md](build-guide.md) | Build commands, CGO policy | Build issues |
| [common-tasks.md](common-tasks.md) | Workflows, code style, error handling | Daily development |

## Quick Access

**New to the project?** Start here:
1. Read CLAUDE.md (quick overview)
2. Read architecture-guide.md (understand layers)
3. Read common-tasks.md (see how to work)
4. Read testing-guide.md (write tests)

**Adding a feature?**
- Check common-tasks.md for workflows
- Review architecture-guide.md for layer placement

**Build problems?**
- Check build-guide.md for troubleshooting

**Writing tests?**
- Check testing-guide.md for organization and helpers
