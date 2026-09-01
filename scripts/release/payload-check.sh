#!/bin/sh
# payload-check.sh: verify license payload hashes and inventory coverage
# usage: payload-check.sh

set -eu

script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
# shellcheck source-path=SCRIPTDIR
. "$script_dir/lib.sh"

[ -f "$release_payload_path" ] || release_die "payload manifest not found: $release_payload_path"
[ -f "$release_inventory_path" ] || release_die "runtime inventory not found: $release_inventory_path"

forbidden=$(awk -F '\t' '
	/^#/ || NF == 0 { next }
	$3 ~ /testify/ || $3 ~ /go\.yaml\.in\/yaml/ {
		print $3
	}
' "$release_payload_path") || true

if [ -n "$forbidden" ]; then
	printf 'test-only module listed in runtime payload:\n%s\n' "$forbidden" >&2
	exit 1
fi

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/gz-pm-payload-check.XXXXXX")
cleanup() {
	rm -rf "$tmp_dir"
}
trap cleanup 0 1 2 15

: >"$tmp_dir/payload.modules"
: >"$tmp_dir/archive.paths"
awk -F '\t' '
	/^#/ || NF == 0 { next }
	NF != 7 || $1 == "" || $2 == "" || $5 == "" || $6 == "" || $7 == "" {
		printf "invalid payload row: %s\n", $0 > "/dev/stderr"
		invalid = 1
		next
	}
	{
		print $0
	}
	END { if (invalid) exit 1 }
' "$release_payload_path" >"$tmp_dir/payload.rows"

while IFS=$(printf '\t') read -r kind goos_filter module version source_path archive_path digest; do
	abs_source="$release_repo_root/$source_path"
	[ -f "$abs_source" ] || release_die "payload source missing: $source_path"
	[ -s "$abs_source" ] || release_die "payload source empty: $source_path"
	case $kind in
	project)
		[ "$digest" = '-' ] || release_die "project payload must use sha256 -: $source_path"
		;;
	license)
		actual=$(release_sha256 "$abs_source")
		if [ "$actual" != "$digest" ]; then
			release_die "payload hash mismatch for $source_path: got $actual want $digest"
		fi
		[ "$module" != '-' ] && [ "$version" != '-' ] || release_die "license payload missing module: $source_path"
		printf '%s\t%s\t%s\n' "$goos_filter" "$module" "$version" >>"$tmp_dir/payload.modules"
		;;
	*)
		release_die "unknown payload kind: $kind"
		;;
	esac
	printf '%s\n' "$archive_path" >>"$tmp_dir/archive.paths"
done <"$tmp_dir/payload.rows"

if grep -E 'testify|go\.yaml\.in' "$tmp_dir/archive.paths" >/dev/null 2>&1; then
	release_die 'test-only path leaked into payload archive paths'
fi

awk -F '\t' '
	/^#/ || NF == 0 { next }
	NF < 4 || $1 == "" || $2 == "" || $3 == "" || $4 == "" {
		printf "invalid inventory row: %s\n", $0 > "/dev/stderr"
		invalid = 1
		next
	}
	{ print $1 "\t" $3 "\t" $4 }
	END { if (invalid) exit 1 }
' "$release_inventory_path" | LC_ALL=C sort -u >"$tmp_dir/inventory.modules"

status=0
while IFS=$(printf '\t') read -r goos module version; do
	matched=0
	while IFS=$(printf '\t') read -r goos_filter payload_module payload_version; do
		if [ "$payload_module" = "$module" ] && [ "$payload_version" = "$version" ] &&
			{ [ "$goos_filter" = '*' ] || [ "$goos_filter" = "$goos" ]; }; then
			matched=1
			break
		fi
	done <"$tmp_dir/payload.modules"
	if [ "$matched" -eq 0 ]; then
		printf 'inventory module missing from payload: %s %s %s\n' "$goos" "$module" "$version" >&2
		status=1
	fi
done <"$tmp_dir/inventory.modules"

while IFS=$(printf '\t') read -r goos_filter module version; do
	matched=0
	while IFS=$(printf '\t') read -r goos inventory_module inventory_version; do
		if [ "$inventory_module" = "$module" ] && [ "$inventory_version" = "$version" ] &&
			{ [ "$goos_filter" = '*' ] || [ "$goos_filter" = "$goos" ]; }; then
			matched=1
			break
		fi
	done <"$tmp_dir/inventory.modules"
	if [ "$matched" -eq 0 ]; then
		printf 'payload module missing from inventory: %s %s %s\n' "$goos_filter" "$module" "$version" >&2
		status=1
	fi
done <"$tmp_dir/payload.modules"

if [ "$status" -ne 0 ]; then
	exit 1
fi

printf 'release payload hashes and inventory coverage ok\n'
