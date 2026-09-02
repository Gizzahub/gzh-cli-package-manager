#!/bin/sh
# validate-version.sh: validate a v-less Semantic Version for release metadata
# usage: validate-version.sh VERSION

set -eu

if [ "$#" -ne 1 ]; then
	printf 'usage: %s VERSION\n' "$0" >&2
	exit 2
fi

GZPM_VERSION_INPUT=$1
export GZPM_VERSION_INPUT

if ! awk '
function invalid() {
	exit 1
}

BEGIN {
	version = ENVIRON["GZPM_VERSION_INPUT"]
	if (version == "" || version ~ /[[:space:]]/ ||
	    version !~ /^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$/) {
		invalid()
	}

	plus = index(version, "+")
	if (plus > 0) {
		build = substr(version, plus + 1)
		base = substr(version, 1, plus - 1)
		build_count = split(build, build_ids, /\./)
		if (build == "" || build_count == 0) {
			invalid()
		}
		for (i = 1; i <= build_count; i++) {
			if (build_ids[i] !~ /^[0-9A-Za-z-]+$/) {
				invalid()
			}
		}
	} else {
		base = version
	}

	dash = index(base, "-")
	if (dash > 0) {
		prerelease = substr(base, dash + 1)
		core = substr(base, 1, dash - 1)
		prerelease_count = split(prerelease, prerelease_ids, /\./)
		if (prerelease == "" || prerelease_count == 0) {
			invalid()
		}
		for (i = 1; i <= prerelease_count; i++) {
			id = prerelease_ids[i]
			if (id !~ /^[0-9A-Za-z-]+$/) {
				invalid()
			}
			if (id ~ /^[0-9]+$/ && id ~ /^0[0-9]+$/) {
				invalid()
			}
		}
	} else {
		core = base
	}

	if (split(core, core_ids, /\./) != 3) {
		invalid()
	}
	for (i = 1; i <= 3; i++) {
		if (core_ids[i] !~ /^(0|[1-9][0-9]*)$/) {
			invalid()
		}
	}

	exit 0
}
' </dev/null; then
	printf 'invalid release version: expected SemVer without leading v\n' >&2
	exit 1
fi

printf '%s\n' "$GZPM_VERSION_INPUT"
