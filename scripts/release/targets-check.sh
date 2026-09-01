#!/bin/sh
# targets-check.sh: fail if workflow, build.mk, and inventory targets drift
# usage: targets-check.sh

set -eu

script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
# shellcheck source-path=SCRIPTDIR
. "$script_dir/lib.sh"

tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/gz-pm-targets-check.XXXXXX")
cleanup() {
	rm -rf "$tmp_dir"
}
trap cleanup 0 1 2 15

sort_list() {
	LC_ALL=C sort -u
}

from_tsv="$tmp_dir/targets.tsv.txt"
from_workflow="$tmp_dir/workflow.txt"
from_build_mk="$tmp_dir/build.mk.txt"
from_inventory="$tmp_dir/inventory.txt"

release_read_targets | sort_list >"$from_tsv"

awk '
	$1 == "-" && $2 == "goos:" {
		goos = $3
		pending = 1
		next
	}
	pending && $1 == "goarch:" {
		if (goos == "" || $2 == "") {
			printf "invalid workflow matrix pair: goos=%s goarch=%s\n", goos, $2 > "/dev/stderr"
			invalid = 1
		} else {
			print goos "/" $2
		}
		pending = 0
		goos = ""
	}
	END { if (invalid) exit 1 }
' "$release_workflow_path" | sort_list >"$from_workflow"

awk '
	/^build-all:/ { in_target = 1; next }
	in_target && /^[A-Za-z_][A-Za-z0-9_-]*:/ { in_target = 0 }
	in_target {
		goos = ""
		goarch = ""
		if (match($0, /GOOS=[A-Za-z0-9]+/)) {
			goos = substr($0, RSTART + 5, RLENGTH - 5)
		}
		if (match($0, /GOARCH=[A-Za-z0-9]+/)) {
			goarch = substr($0, RSTART + 7, RLENGTH - 7)
		}
		if (goos != "" && goarch != "") {
			print goos "/" goarch
		}
	}
' "$release_build_mk_path" | sort_list >"$from_build_mk"

awk -F '\t' '
	/^#/ || NF == 0 { next }
	NF < 2 || $1 == "" || $2 == "" {
		printf "invalid inventory row: %s\n", $0 > "/dev/stderr"
		invalid = 1
		next
	}
	{ print $1 "/" $2 }
	END { if (invalid) exit 1 }
' "$release_inventory_path" | sort_list >"$from_inventory"

status=0
for label_pair in \
	"release/targets.tsv:$from_tsv" \
	".github/workflows/build.yml:$from_workflow" \
	".make/build.mk:$from_build_mk" \
	"release/runtime-dependencies.tsv:$from_inventory"
do
	label=${label_pair%%:*}
	path=${label_pair#*:}
	if [ ! -s "$path" ]; then
		printf 'no release targets parsed from %s\n' "$label" >&2
		status=1
		continue
	fi
	if ! diff -u "$from_tsv" "$path"; then
		printf 'release target set drifted from release/targets.tsv: %s\n' "$label" >&2
		status=1
	fi
done

if ! grep -q 'release_read_targets' "$script_dir/runtime-deps.sh"; then
	printf 'inventory helper does not iterate release_read_targets\n' >&2
	status=1
fi
if ! grep -q 'release/targets.tsv' "$script_dir/lib.sh"; then
	printf 'release helper library does not read release/targets.tsv\n' >&2
	status=1
fi

if [ "$status" -ne 0 ]; then
	exit 1
fi

printf 'release targets aligned:\n'
cat "$from_tsv"
