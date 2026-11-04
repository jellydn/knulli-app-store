#!/bin/sh
# Both publish paths fail closed on legacy, duplicate or wrong-profile ZIPs.
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 INPUT_DIR NEW_OUTPUT_DIR" >&2
  exit 2
fi

SOURCE=$1
OUTPUT=$(realpath -m "$2")
archives=$(find "$SOURCE" -type f -name '*.zip' -print)
if [ -z "$archives" ] || [ "$(printf '%s\n' "$archives" | wc -l)" -ne 1 ]; then
  echo "expected exactly one compatibility-profile Store ZIP" >&2
  exit 1
fi
name=$(basename "$archives")
case "$name" in
  knulli-app-store-knulli-aarch64-glibc-sdl2-experimental-?*.zip) ;;
  *) echo "unexpected Store ZIP: $name" >&2; exit 1 ;;
esac

# A fresh directory prevents stale assets from entering a later publication.
mkdir "$OUTPUT"
cp "$archives" "$OUTPUT/$name"
(cd "$OUTPUT" && sha256sum "$name" >SHA256SUMS.txt)
