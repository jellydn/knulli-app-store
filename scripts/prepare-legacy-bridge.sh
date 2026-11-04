#!/bin/sh
# Local preparation only. Never called by new-release workflows or uploads.
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 VERIFIED_GENERIC_ZIP NEW_OUTPUT_DIR" >&2
  exit 2
fi

ARCHIVE=$1
OUTPUT=$(realpath -m "$2")
name=$(basename "$ARCHIVE")
prefix=knulli-app-store-knulli-aarch64-glibc-sdl2-experimental-
case "$name" in
  "$prefix"?*.zip) ;;
  *) echo "expected the verified compatibility-profile ZIP" >&2; exit 1 ;;
esac
version=${name#"$prefix"}
version=${version%.zip}
case "$version" in
  *[!A-Za-z0-9._-]*) echo "invalid bridge version" >&2; exit 1 ;;
esac

unzip -tqq "$ARCHIVE"
mkdir "$OUTPUT"
for target in trimui-smart-pro magicx-zero-28; do
  cp "$ARCHIVE" "$OUTPUT/knulli-app-store-$target-experimental-$version.zip"
done
(cd "$OUTPUT" && sha256sum ./*.zip >SHA256SUMS.txt)
echo "Prepared byte-identical legacy aliases locally; nothing uploaded." >&2
echo "Do not include these in a new release. See docs/store-distribution.md." >&2
