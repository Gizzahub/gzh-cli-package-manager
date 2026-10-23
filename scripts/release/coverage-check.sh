#!/bin/sh
# PRODUCT.md release floor. Require a report from the current candidate's tests.
set -eu
[ "$#" -eq 1 ] && [ -s "$1" ] || { echo 'coverage report missing' >&2; exit 1; }
summary=$(go tool cover -func="$1")
printf '%s\n' "$summary" | awk '
  $1 == "total:" { coverage = $3; sub(/%$/, "", coverage); found++ }
  END {
    if (found != 1 || coverage !~ /^[0-9]+(\.[0-9]+)?$/) exit 1
    printf "Release coverage: %s%% (required: 85%%)\n", coverage
    if (coverage + 0 < 85) exit 1
  }'
