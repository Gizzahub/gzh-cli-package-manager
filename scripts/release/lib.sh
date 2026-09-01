#!/bin/sh
# lib.sh: shared helpers for gz-pm release archive scripts
# usage: . "$(dirname "$0")/lib.sh"

release_script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
release_repo_root=$(CDPATH='' cd -P "$release_script_dir/../.." && pwd)
release_targets_path="$release_repo_root/release/targets.tsv"
release_payload_path="$release_repo_root/release/payload-files.tsv"
release_inventory_path="$release_repo_root/release/runtime-dependencies.tsv"
release_workflow_path="$release_repo_root/.github/workflows/build.yml"
release_build_mk_path="$release_repo_root/.make/build.mk"
export release_repo_root release_targets_path release_payload_path
export release_inventory_path release_workflow_path release_build_mk_path

release_die() {
	printf '%s\n' "$*" >&2
	exit 1
}

release_sha256() {
	file=$1
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$file" | awk '{print $1}'
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$file" | awk '{print $1}'
	else
		openssl dgst -sha256 "$file" | awk '{print $NF}'
	fi
}

release_read_targets() {
	awk -F '\t' '
		/^#/ || NF == 0 { next }
		NF < 2 || $1 == "" || $2 == "" {
			printf "invalid release target row: %s\n", $0 > "/dev/stderr"
			invalid = 1
			next
		}
		{ print $1 "/" $2 }
		END { if (invalid) exit 1 }
	' "$release_targets_path"
}

release_archive_basename() {
	goos=$1
	goarch=$2
	if [ "$goos" = windows ]; then
		printf 'gz-pm-%s-%s.zip\n' "$goos" "$goarch"
	else
		printf 'gz-pm-%s-%s.tar.gz\n' "$goos" "$goarch"
	fi
}

release_wrap_dir() {
	printf 'gz-pm-%s-%s\n' "$1" "$2"
}

release_binary_name() {
	if [ "$1" = windows ]; then
		printf 'gz-pm.exe\n'
	else
		printf 'gz-pm\n'
	fi
}

release_payload_applies() {
	goos_filter=$1
	goos=$2
	[ "$goos_filter" = '*' ] || [ "$goos_filter" = "$goos" ]
}
