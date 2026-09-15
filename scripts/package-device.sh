#!/bin/sh
set -eu

if [ "$#" -ne 6 ]; then
  echo "usage: $0 CLI UI CATALOG OUTPUT_DIR VERSION TARGET" >&2
  exit 2
fi

CLI=$1
UI=$2
CATALOG=$3
OUTPUT_DIR=$(realpath -m "$4")
VERSION=$(printf '%s' "$5" | tr -cd 'A-Za-z0-9._-')
TARGET=$6

if [ -z "$VERSION" ]; then
  echo "version must contain a portable filename character" >&2
  exit 2
fi

case "$TARGET" in
  trimui-smart-pro|magicx-zero-28) ;;
  *) echo "unsupported package target: $TARGET" >&2; exit 2 ;;
esac

ICON="docs/assets/cover.png"
for input in "$CLI" "$UI" "$CATALOG" "$ICON"; do
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
install -m 0755 "packaging/$TARGET/Knulli App Store.sh" "$PORTS/Knulli App Store.sh"
install -m 0644 "packaging/$TARGET/README.txt" "$PORTS/knulli-app-store/README.txt"
install -m 0644 "$ICON" "$PORTS/images/Knulli App Store.png"

ARCHIVE="$OUTPUT_DIR/knulli-app-store-$TARGET-experimental-$VERSION.zip"
(cd "$PORTS" && find . -type f -print | LC_ALL=C sort | zip -X "$ARCHIVE" -@ >/dev/null)
(cd "$OUTPUT_DIR" && sha256sum "$(basename "$ARCHIVE")" >SHA256SUMS.txt)

printf '%s\n' "$ARCHIVE"
