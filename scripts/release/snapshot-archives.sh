#!/bin/sh
# snapshot-archives.sh: cross-build and package gz-pm snapshot archives
# usage: snapshot-archives.sh --output-dir DIR

set -eu

script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
# shellcheck source-path=SCRIPTDIR
. "$script_dir/lib.sh"

usage() {
	printf 'usage: %s --output-dir DIR\n' "$0" >&2
}

output_dir=
case $# in
2)
	[ "$1" = --output-dir ] || {
		usage
		exit 2
	}
	output_dir=$2
	;;
*)
	usage
	exit 2
	;;
esac

[ -n "$output_dir" ] || {
	usage
	exit 2
}

"$script_dir/targets-check.sh" >/dev/null
"$script_dir/payload-check.sh" >/dev/null

mkdir -p "$output_dir"
output_dir=$(CDPATH='' cd -P "$output_dir" && pwd)
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/gz-pm-snapshot-build.XXXXXX")
cleanup() {
	rm -rf "$build_dir"
}
trap cleanup 0 1 2 15

targets_file="$build_dir/targets"
release_read_targets >"$targets_file"

while IFS= read -r target; do
	goos=${target%/*}
	goarch=${target#*/}
	binary_name=gz-pm-$goos-$goarch
	if [ "$goos" = windows ]; then
		binary_name=$binary_name.exe
	fi
	binary_path="$build_dir/$binary_name"
	if ! (
		cd "$release_repo_root"
		GOWORK=off GOFLAGS='-mod=readonly' CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
			go build -trimpath -o "$binary_path" ./cmd/gz-pm
	); then
		release_die "snapshot build failed for $target"
	fi
	"$script_dir/package-archive.sh" \
		--goos "$goos" \
		--goarch "$goarch" \
		--binary "$binary_path" \
		--output-dir "$output_dir" >/dev/null
done <"$targets_file"

"$script_dir/checksums.sh" --input-dir "$output_dir" --output "$output_dir/checksums.txt" >/dev/null
printf '%s\n' "$output_dir"
