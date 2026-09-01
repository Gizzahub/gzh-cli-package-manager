#!/bin/sh
# package-archive.sh: bundle gz-pm binary with license payload
# usage: package-archive.sh --goos GOOS --goarch GOARCH --binary PATH --output-dir DIR

set -eu
COPYFILE_DISABLE=1
export COPYFILE_DISABLE

script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
# shellcheck source-path=SCRIPTDIR
. "$script_dir/lib.sh"

usage() {
	printf 'usage: %s --goos GOOS --goarch GOARCH --binary PATH --output-dir DIR\n' "$0" >&2
}

goos=
goarch=
binary=
output_dir=

while [ $# -gt 0 ]; do
	case $1 in
	--goos | --goarch | --binary | --output-dir)
		if [ $# -lt 2 ]; then
			printf 'missing value for %s\n' "$1" >&2
			usage
			exit 2
		fi
		case $1 in
		--goos) goos=$2 ;;
		--goarch) goarch=$2 ;;
		--binary) binary=$2 ;;
		--output-dir) output_dir=$2 ;;
		esac
		shift 2
		;;
	*)
		usage
		exit 2
		;;
	esac
done

[ -n "$goos" ] && [ -n "$goarch" ] && [ -n "$binary" ] && [ -n "$output_dir" ] || {
	usage
	exit 2
}

[ -f "$binary" ] || release_die "binary not found: $binary"
[ -s "$binary" ] || release_die "binary is empty: $binary"

target="$goos/$goarch"
release_read_targets | grep -qxF "$target" || release_die "unsupported release target: $target"

stage_root=$(mktemp -d "${TMPDIR:-/tmp}/gz-pm-archive.XXXXXX")
cleanup() {
	rm -rf "$stage_root"
}
trap cleanup 0 1 2 15

wrap_dir=$(release_wrap_dir "$goos" "$goarch")
stage_dir="$stage_root/$wrap_dir"
mkdir -p "$stage_dir"

dest_binary="$stage_dir/$(release_binary_name "$goos")"
cp "$binary" "$dest_binary"
chmod 0755 "$dest_binary"

while IFS=$(printf '\t') read -r kind goos_filter _ _ source_path archive_path digest; do
	case $kind in
	'' | '#'*)
		continue
		;;
	esac
	release_payload_applies "$goos_filter" "$goos" || continue
	src="$release_repo_root/$source_path"
	[ -f "$src" ] || release_die "payload source missing: $source_path"
	if [ "$kind" = license ]; then
		actual=$(release_sha256 "$src")
		[ "$actual" = "$digest" ] || release_die "payload hash mismatch for $source_path: got $actual want $digest"
	fi
	dest="$stage_dir/$archive_path"
	mkdir -p "$(dirname "$dest")"
	cp "$src" "$dest"
	chmod 0644 "$dest"
done <"$release_payload_path"

if find "$stage_dir" \( -iname '*testify*' -o -iname '*go.yaml.in*' \) | grep -q .; then
	release_die 'test-only path present in staged archive'
fi

mkdir -p "$output_dir"
output_dir=$(CDPATH='' cd -P "$output_dir" && pwd)
archive_path="$output_dir/$(release_archive_basename "$goos" "$goarch")"
rm -f "$archive_path"

if [ "$goos" = windows ]; then
	command -v zip >/dev/null 2>&1 || release_die 'zip is required to package Windows archives'
	(
		cd "$stage_root"
		zip -r -X -q "$archive_path" "$wrap_dir"
	)
else
	tar -C "$stage_root" -czf "$archive_path" "$wrap_dir"
fi

[ -s "$archive_path" ] || release_die "archive was not created: $archive_path"
printf '%s\n' "$archive_path"
