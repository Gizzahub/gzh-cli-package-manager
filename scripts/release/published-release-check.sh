#!/bin/sh
# Refuse artifact replacement on a release that is already public.
set -eu
[ "$#" -eq 2 ] || { echo "usage: $0 OWNER/REPO TAG" >&2; exit 2; }
repository=$1
tag=$2
case "$repository" in */*) ;; *) echo 'repository must be OWNER/REPO' >&2; exit 2;; esac
case "$tag" in v*) ;; *) echo 'release tag must start with v' >&2; exit 2;; esac
script_dir=$(CDPATH='' cd -P "$(dirname "$0")" && pwd)
"$script_dir/validate-version.sh" "${tag#v}" >/dev/null
api_exit=0
response=$(gh api --include "repos/$repository/releases/tags/$tag") || api_exit=$?
status=$(printf '%s\n' "$response" | awk 'NR == 1 { sub(/\r$/, ""); if ($1 ~ /^HTTP\//) print $2 }')
body=$(printf '%s\n' "$response" | awk 'body { print } /^\r?$/ { body = 1 }')
case "$status:$api_exit" in
200:0)
  if printf '%s\n' "$body" | jq -se 'length == 1 and (.[0] | type == "object" and .draft == true)' >/dev/null; then
    echo 'Existing release is a draft; upload permitted.'
    exit 0
  fi
  echo 'Refusing upload: release is published or draft state is invalid.' >&2
  exit 1
  ;;
404:*)
  # Validate the API response as well; a proxy HTML error is not absence proof.
  if [ "$api_exit" -ne 0 ] && printf '%s\n' "$body" | jq -se 'length == 1 and (.[0] | type == "object" and .message == "Not Found")' >/dev/null; then
    echo 'Release does not exist; draft creation permitted.'
    exit 0
  fi
  ;;
esac
echo "Refusing upload: release lookup failed (HTTP ${status:-unknown}, exit $api_exit)." >&2
exit 1
