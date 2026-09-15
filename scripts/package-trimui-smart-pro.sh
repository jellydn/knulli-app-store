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

if [ -z "$VERSION" ]; then
  echo "version must contain a portable filename character" >&2
  exit 2
fi

for input in "$CLI" "$UI" "$CATALOG"; do
  if [ ! -f "$input" ]; then
    echo "missing input: $input" >&2
    exit 1
  fi
done

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
PORTS="$WORK/ports"
APP="$PORTS/knulli-app-store"
mkdir -p "$APP" "$OUTPUT_DIR"

install -m 0755 "$CLI" "$APP/knulli-app"
install -m 0755 "$UI" "$APP/knulli-app-ui"
install -m 0644 "$CATALOG" "$APP/catalog-index.json"
install -m 0755 "packaging/trimui-smart-pro/Knulli App Store.sh" "$PORTS/Knulli App Store.sh"
install -m 0644 packaging/trimui-smart-pro/README.txt "$PORTS/knulli-app-store/README.txt"

ARCHIVE="$OUTPUT_DIR/knulli-app-store-trimui-smart-pro-experimental-$VERSION.zip"
(cd "$PORTS" && find . -type f -print | LC_ALL=C sort | zip -X "$ARCHIVE" -@ >/dev/null)
(cd "$OUTPUT_DIR" && sha256sum "$(basename "$ARCHIVE")" >SHA256SUMS.txt)

printf '%s\n' "$ARCHIVE"
