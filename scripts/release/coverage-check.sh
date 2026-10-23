#!/bin/sh
# PRODUCT.md release floor. Require a report from the current candidate's tests.
set -eu
[ "$#" -eq 1 ] && [ -s "$1" ] || { echo 'coverage report missing' >&2; exit 1; }
go tool cover -func="$1" >/dev/null
awk '
  NR > 1 { total += $2; if ($3 > 0) covered += $2 }
  END {
    if (total == 0) exit 1
    printf "Release coverage: %.4f%% (required: 85%%)\n", 100 * covered / total
    if (covered * 100 < total * 85) exit 1
  }' "$1"
