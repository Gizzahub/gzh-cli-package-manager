#!/bin/sh
# Generate locally; publishing this cask to a tap is a separate approved operation.
set -eu
script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
[ "$#" -eq 2 ] || { echo "usage: $0 VERSION CHECKSUMS" >&2; exit 2; }
version=$("$script_dir/validate-version.sh" "$1")
checksums=$2
[ -f "$checksums" ] || { echo 'checksum manifest missing' >&2; exit 1; }
digest() {
  awk -v name="$1" '
    $2 == name { hash = $1; count++ }
    END {
      if (count != 1 || length(hash) != 64 || hash !~ /^[0-9a-f]+$/) exit 1
      print hash
    }' "$checksums"
}
arm=$(digest gz-pm-darwin-arm64.tar.gz) || { echo 'invalid arm64 checksum' >&2; exit 1; }
intel=$(digest gz-pm-darwin-amd64.tar.gz) || { echo 'invalid amd64 checksum' >&2; exit 1; }
cat <<EOF
cask "gz-pm" do
  arch arm: "arm64", intel: "amd64"

  version "$version"
  on_arm do
    sha256 "$arm"
  end
  on_intel do
    sha256 "$intel"
  end

  url "https://github.com/Gizzahub/gzh-cli-package-manager/releases/download/v#{version}/gz-pm-darwin-#{arch}.tar.gz"
  name "gz-pm"
  desc "Multi-package-manager update orchestrator"
  homepage "https://github.com/Gizzahub/gzh-cli-package-manager"

  binary "gz-pm-darwin-#{arch}/gz-pm"
end
EOF
