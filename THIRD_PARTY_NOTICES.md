# Third-Party Notices

This project is licensed under Apache License 2.0. Its compiled distributions
also contain third-party Go modules under their own licenses.

The dependency versions below are derived from `go.mod` at release preparation
time. Before publishing a binary release, the release owner must regenerate this
inventory from the final module graph, verify every license, and include the
applicable license and NOTICE texts with the release artifacts. A dependency
bump is not release-ready when it introduces an unknown or incompatible license.

| Module | Version | License / notice |
|---|---:|---|
| `github.com/spf13/cobra` | v1.10.2 | Apache-2.0 |
| `github.com/inconshreveable/mousetrap` | v1.1.0 | Apache-2.0 |
| `github.com/spf13/pflag` | v1.0.9 | BSD-3-Clause |
| `github.com/stretchr/testify` | v1.12.1 | MIT |
| `gopkg.in/yaml.v3` | v3.0.1 | MIT and Apache-2.0; upstream NOTICE applies |
| `go.yaml.in/yaml/v3` | v3.0.5 | MIT and Apache-2.0; upstream NOTICE applies |

The authoritative license and NOTICE texts are distributed in each module's
source archive. This inventory is attribution metadata, not a replacement for
those texts in a binary release bundle.
