#!/bin/sh

set -eu

script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
repo_root=$(CDPATH='' cd -P "$script_dir/../.." && pwd)
manifest_path="$repo_root/release/runtime-dependencies.tsv"

usage() {
	printf 'usage: %s [--check]\n' "$0" >&2
}

case $# in
0)
	mode=print
	;;
1)
	case $1 in
	--check)
		mode=check
		;;
	*)
		usage
		exit 2
		;;
	esac
	;;
*)
	usage
	exit 2
	;;
esac

runtime_deps_tmp=$(mktemp -d "${TMPDIR:-/tmp}/gz-pm-runtime-deps.XXXXXX")

cleanup() {
	rm -rf "$runtime_deps_tmp"
}
trap cleanup 0 1 2 15

generate_manifest() {
	output_path=$1
	target_index=0

	{
		printf '%s\n' '# runtime-dependencies/v1'
		printf '%s\n' '# main-package: ./cmd/gz-pm'
		printf '%s\n' '# columns: goos<TAB>goarch<TAB>module<TAB>version'
	} >"$output_path"

	# Keep this matrix aligned with .github/workflows/build.yml and .make/build.mk.
	for target in \
		linux/amd64 \
		linux/arm64 \
		darwin/amd64 \
		darwin/arm64 \
		windows/amd64
	do
		target_index=$((target_index + 1))
		goos=${target%/*}
		goarch=${target#*/}
		raw_path="$runtime_deps_tmp/$target_index.raw"
		rows_path="$runtime_deps_tmp/$target_index.rows"
		sorted_path="$runtime_deps_tmp/$target_index.sorted"

		if ! (
			cd "$repo_root"
			GOWORK=off GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
				go list -mod=readonly -deps \
				-f '{{if and .Module (not .Module.Main)}}{{.Module.Path}}{{"\t"}}{{.Module.Version}}{{end}}' \
				./cmd/gz-pm
		) >"$raw_path"; then
			printf 'runtime dependency inventory failed for %s\n' "$target" >&2
			exit 1
		fi

		awk -F '\t' -v goos="$goos" -v goarch="$goarch" '
			NF == 0 { next }
			NF != 2 || $1 == "" || $2 == "" {
				printf "invalid runtime module row for %s/%s: %s\n", goos, goarch, $0 > "/dev/stderr"
				invalid = 1
				next
			}
			{ print goos "\t" goarch "\t" $1 "\t" $2 }
			END { if (invalid) exit 1 }
		' "$raw_path" >"$rows_path"

		LC_ALL=C sort -u "$rows_path" >"$sorted_path"
		cat "$sorted_path" >>"$output_path"
	done
}

generated_manifest="$runtime_deps_tmp/runtime-dependencies.tsv"
generate_manifest "$generated_manifest"

case $mode in
print)
	cat "$generated_manifest"
	;;
check)
	if [ ! -f "$manifest_path" ]; then
		printf 'runtime dependency manifest not found: %s\n' "$manifest_path" >&2
		exit 1
	fi
	diff -u "$manifest_path" "$generated_manifest"
	;;
esac
