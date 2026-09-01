#!/bin/sh
# checksums.sh: write SHA-256 checksums for gz-pm release archives
# usage: checksums.sh --input-dir DIR --output PATH

set -eu

script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
# shellcheck source-path=SCRIPTDIR
. "$script_dir/lib.sh"

usage() {
	printf 'usage: %s --input-dir DIR --output PATH\n' "$0" >&2
}

input_dir=
output_path=

while [ $# -gt 0 ]; do
	case $1 in
	--input-dir | --output)
		if [ $# -lt 2 ]; then
			printf 'missing value for %s\n' "$1" >&2
			usage
			exit 2
		fi
		case $1 in
		--input-dir) input_dir=$2 ;;
		--output) output_path=$2 ;;
		esac
		shift 2
		;;
	*)
		usage
		exit 2
		;;
	esac
done

[ -n "$input_dir" ] && [ -n "$output_path" ] || {
	usage
	exit 2
}
[ -d "$input_dir" ] || release_die "input directory not found: $input_dir"

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/gz-pm-checksums.XXXXXX")
cleanup() {
	rm -rf "$tmp_dir"
}
trap cleanup 0 1 2 15

find "$input_dir" \( -name 'gz-pm-*.tar.gz' -o -name 'gz-pm-*.zip' \) -type f |
	LC_ALL=C sort >"$tmp_dir/archives"

[ -s "$tmp_dir/archives" ] || release_die "no gz-pm archives found under $input_dir"

: >"$tmp_dir/names"
: >"$tmp_dir/checksums"
while IFS= read -r archive; do
	base=$(basename "$archive")
	if grep -qxF "$base" "$tmp_dir/names"; then
		release_die "duplicate archive basename: $base"
	fi
	printf '%s\n' "$base" >>"$tmp_dir/names"
	printf '%s  %s\n' "$(release_sha256 "$archive")" "$base" >>"$tmp_dir/checksums"
done <"$tmp_dir/archives"

release_read_targets >"$tmp_dir/targets"
: >"$tmp_dir/expected"
while IFS= read -r target; do
	release_archive_basename "${target%/*}" "${target#*/}" >>"$tmp_dir/expected"
done <"$tmp_dir/targets"
LC_ALL=C sort -o "$tmp_dir/expected" "$tmp_dir/expected"
LC_ALL=C sort -o "$tmp_dir/actual" "$tmp_dir/names"
if ! diff -u "$tmp_dir/expected" "$tmp_dir/actual"; then
	release_die 'checksum input set does not match release targets'
fi

mkdir -p "$(dirname "$output_path")"
cp "$tmp_dir/checksums" "$output_path"
printf '%s\n' "$output_path"
