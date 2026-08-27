# Third-Party Notices

This project is licensed under Apache License 2.0. Its compiled distributions
also contain third-party Go modules under their own licenses.

Runtime scopes below are derived from the build-specific manifest in
`release/runtime-dependencies.tsv`; test-only scopes are derived from the module
test graph. Before publishing a binary release, the release owner must regenerate
the inventory from the final module graph, verify every license, and include the
applicable license and NOTICE texts with the release artifacts. A dependency bump
is not release-ready when it introduces an unknown or incompatible license.

| Module | Version | Scope | License / notice |
|---|---:|---|---|
| `github.com/spf13/cobra` | v1.10.2 | Runtime — all release targets | Apache-2.0 |
| `github.com/inconshreveable/mousetrap` | v1.1.0 | Runtime — Windows only | Apache-2.0 |
| `github.com/spf13/pflag` | v1.0.9 | Runtime — all release targets | BSD-3-Clause |
| `github.com/stretchr/testify` | v1.12.1 | Test only | MIT |
| `gopkg.in/yaml.v3` | v3.0.1 | Runtime — all release targets | MIT and Apache-2.0; upstream NOTICE applies |
| `go.yaml.in/yaml/v3` | v3.0.5 | Test only | MIT and Apache-2.0; upstream NOTICE applies |

The authoritative license and NOTICE texts are distributed in each module's
source archive. This inventory is attribution metadata, not a replacement for
those texts in a binary release bundle.
