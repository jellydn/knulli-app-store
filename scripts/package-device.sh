#!/bin/sh
set -eu

if [ "$#" -ne 5 ]; then
  echo "usage: $0 CLI UI CATALOG OUTPUT_DIR VERSION" >&2
  exit 2
fi

CLI=$1
UI=$2
CATALOG=$3
OUTPUT_DIR=$(realpath -m "$4")
VERSION=$(printf '%s' "$5" | tr -cd 'A-Za-z0-9._-')
TARGET=knulli-aarch64-glibc-sdl2

if [ -z "$VERSION" ]; then
  echo "version must contain a portable filename character" >&2
  exit 2
fi

ICON="docs/assets/cover.png"
SIG=$CATALOG.sig
for input in "$CLI" "$UI" "$CATALOG" "$SIG" "$ICON"; do
  if [ ! -f "$input" ]; then
    echo "missing input: $input" >&2
    exit 1
  fi
done

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
PORTS="$WORK/ports"
APP="$PORTS/knulli-app-store"
mkdir -p "$APP" "$PORTS/images" "$OUTPUT_DIR"

install -m 0755 "$CLI" "$APP/knulli-app"
install -m 0755 "$UI" "$APP/knulli-app-ui"
install -m 0644 "$CATALOG" "$APP/catalog-index.json"
install -m 0644 "$SIG" "$APP/catalog-index.json.sig"
install -m 0755 "packaging/knulli/Knulli App Store.sh" "$PORTS/Knulli App Store.sh"
install -m 0644 "packaging/knulli/README.txt" "$PORTS/knulli-app-store/README.txt"
# Keep notices in the already managed README; do not widen self-update paths.
printf '\nEmbedded DejaVu Sans font notice\n\n' >>"$APP/README.txt"
cat "internal/sdlui/assets/DejaVuSans-LICENSE.txt" >>"$APP/README.txt"
for notice in internal/sdlui/assets/thumbnails/*.txt; do
  printf '\nCatalogue artwork: %s\n\n' "$(basename "$notice")" >>"$APP/README.txt"
  cat "$notice" >>"$APP/README.txt"
done
install -m 0644 "$ICON" "$PORTS/images/Knulli App Store.png"

ARCHIVE="$OUTPUT_DIR/knulli-app-store-$TARGET-experimental-$VERSION.zip"
# Canonical relative paths are required by the self-updater's safe extractor.
(cd "$PORTS" && find . -type f -printf '%P\n' | LC_ALL=C sort | zip -X "$ARCHIVE" -@ >/dev/null)
(cd "$OUTPUT_DIR" && sha256sum "$(basename "$ARCHIVE")" >SHA256SUMS.txt)

printf '%s\n' "$ARCHIVE"
