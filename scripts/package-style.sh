#!/usr/bin/env bash
set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
output=${1:-"$repo_root/dist/KoreanProse.zip"}

case "$output" in
  /*) archive=$output ;;
  *) archive="$repo_root/$output" ;;
esac

if [[ -e "$archive" ]]; then
  printf 'refusing to overwrite existing archive: %s\n' "$archive" >&2
  exit 1
fi

mkdir -p "$(dirname -- "$archive")"
(
  cd "$repo_root"
  zip -X -q -r "$archive" KoreanProse -x '*/.DS_Store'
)

entries=$(unzip -Z1 "$archive")
if [[ "$entries" != *$'KoreanProse/'* ]]; then
  printf 'style archive has no KoreanProse/ directory: %s\n' "$archive" >&2
  exit 1
fi

printf '%s\n' "$archive"
