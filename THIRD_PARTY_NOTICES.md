# Third-Party Notices

This project is licensed under Apache License 2.0. Its compiled distributions
also contain third-party Go modules under their own licenses.

Runtime scopes below are derived from the build-specific manifest in
`release/runtime-dependencies.tsv`; test-only scopes are derived from the module
test graph. Before publishing a binary release, the release owner must regenerate
the inventory from the final module graph, verify every license, and include the
applicable license and NOTICE texts with the release artifacts. A dependency bump
is not release-ready when it introduces an unknown or incompatible license.

## Runtime provenance evidence

The evidence below is pinned to the exact module versions in
`release/runtime-dependencies.tsv`. Each SHA-256 digest matches both the file in
the Go module cache and the file served from the named upstream tag.

| Module | Release targets | Upstream tag and commit | SPDX | License / NOTICE evidence (SHA-256) |
|---|---|---|---|---|
| `github.com/spf13/cobra` v1.10.2 | Linux amd64/arm64, macOS amd64/arm64, Windows amd64 | [`v1.10.2`](https://github.com/spf13/cobra/tree/v1.10.2) at `88b30ab89da2d0d0abb153818746c5a2d30eccec` | Apache-2.0 | [`LICENSE.txt`](https://raw.githubusercontent.com/spf13/cobra/v1.10.2/LICENSE.txt) `5e3400b93bbb099e83e52bab885e7441750673c21f97988ca3f1240639b63283`; no root NOTICE file at this tag |
| `github.com/spf13/pflag` v1.0.9 | Linux amd64/arm64, macOS amd64/arm64, Windows amd64 | [`v1.0.9`](https://github.com/spf13/pflag/tree/v1.0.9) at `10438578954bba2527fe5cae3684d4532b064bbe` | BSD-3-Clause | [`LICENSE`](https://raw.githubusercontent.com/spf13/pflag/v1.0.9/LICENSE) `b8514c577c1c4b46cee454d5a882b15fa411e72c5bd7f801f241591789fce61a`; no root NOTICE file at this tag |
| `github.com/inconshreveable/mousetrap` v1.1.0 | Windows amd64 only | [`v1.1.0`](https://github.com/inconshreveable/mousetrap/tree/v1.1.0) at `4e8053ee7ef85a6bd26368364a6d27f1641c1d21` | Apache-2.0 | [`LICENSE`](https://raw.githubusercontent.com/inconshreveable/mousetrap/v1.1.0/LICENSE) `4819582701f150b28a563a6cd8ed0bf5a754e3c67b90ad38d78ba4131bf77795`; no root NOTICE file at this tag |
| `gopkg.in/yaml.v3` v3.0.1 | Linux amd64/arm64, macOS amd64/arm64, Windows amd64 | [`v3.0.1`](https://github.com/go-yaml/yaml/tree/v3.0.1) at `f6f7691b1fdeb513f56608cd2c32c51f8194bf51` | MIT AND Apache-2.0 | [`LICENSE`](https://raw.githubusercontent.com/go-yaml/yaml/v3.0.1/LICENSE) `d18f6323b71b0b768bb5e9616e36da390fbd39369a81807cca352de4e4e6aa0b`; [`NOTICE`](https://raw.githubusercontent.com/go-yaml/yaml/v3.0.1/NOTICE) `f6c2dd3a67b576eafb89b80200b8b1627230bf3821a0c14cb99a22ac19107d00` |

## Distribution evidence

- Cobra and Mousetrap use Apache-2.0. Their tagged source archives require the
  Apache license text to accompany a distribution. Neither tag has a root
  NOTICE file to carry forward.
- pflag uses BSD-3-Clause. A binary distribution must reproduce its copyright
  notice, conditions, and disclaimer in the documentation or other materials,
  and must preserve the non-endorsement condition.
- yaml.v3 contains MIT-covered ported files and Apache-2.0-covered remaining
  files. Its tagged source archive includes both `LICENSE` and `NOTICE`, so both
  are applicable release-bundle inputs.

This section records source evidence, not a final legal determination or the
release archive layout. The release owner must approve the obligations and
payload mapping before license texts are copied or release packaging changes.

## Test-only inventory

| Module | Version | Scope | License / notice |
|---|---:|---|---|
| `github.com/stretchr/testify` | v1.12.1 | Test only | MIT |
| `go.yaml.in/yaml/v3` | v3.0.5 | Test only | MIT and Apache-2.0; upstream NOTICE applies |

The authoritative license and NOTICE texts are distributed in each module's
source archive. This inventory is attribution metadata, not a replacement for
those texts in a binary release bundle.
