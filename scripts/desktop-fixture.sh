#!/bin/sh
# Writes a scratch device tree so the SDL GUI can detect Knulli on a desktop
# machine. Everything the detector reads is a plain file under ROOT, so a
# desktop run reports the same platform a device would and packages become
# actionable without any hardware.
#
# This tree is fabricated evidence for desktop verification only. Never point it
# at a real device root or at "/": installs, logs and controller mappings are
# written below ROOT, so ROOT must be a throwaway directory.
set -eu

if [ "$#" -ne 2 ]; then
  echo "usage: $0 DEVICE ROOT" >&2
  echo "  DEVICE  trimui-smart-pro | magicx-zero-28" >&2
  exit 2
fi

DEVICE=$1
ROOT=$2
MARKER=.desktop-fixture

case "$DEVICE" in
  trimui-smart-pro) RESOLUTION=1280x720 ;;
  magicx-zero-28) RESOLUTION=640x480 ;;
  *) echo "unsupported device: $DEVICE" >&2; exit 2 ;;
esac

case "$ROOT" in
  /*) ;;
  *) echo "root must be an absolute path: $ROOT" >&2; exit 2 ;;
esac
if [ "$ROOT" = "/" ] || [ -z "$ROOT" ]; then
  echo "refusing to write a desktop fixture into /" >&2
  exit 2
fi
if [ -e "$ROOT/etc/os-release" ] && [ ! -e "$ROOT/$MARKER" ]; then
  echo "refusing to replace an existing device tree at $ROOT" >&2
  echo "choose an empty directory, or remove it first if it is a scratch tree" >&2
  exit 1
fi

mkdir -p "$ROOT/etc" \
         "$ROOT/boot/boot" \
         "$ROOT/usr/share/knulli" \
         "$ROOT/usr/lib" \
         "$ROOT/lib" \
         "$ROOT/sys/class/graphics/fb0"

: >"$ROOT/$MARKER"
cat >"$ROOT/etc/os-release" <<EOF
NAME="Knulli"
ID=buildroot
OS_NAME="knulli"
OS_VERSION="scarab"
OS_DATE="2026-09-15 00:00"
EOF

# The detector reads the first whitespace-delimited value as the release name.
printf '%s\n' 'scarab 2026-09-15 00:00' >"$ROOT/usr/share/knulli/knulli.version"
printf '%s\n' "$DEVICE" >"$ROOT/boot/boot/knulli.board"

# Architecture ABI evidence. The ABI check only needs the loader path to exist;
# the glibc check greps version markers out of the libc file.
: >"$ROOT/lib/ld-linux-aarch64.so.1"
printf 'GLIBC_2.17\nGLIBC_2.34\nGLIBC_2.40\n' >"$ROOT/lib/libc.so.6"

# Runtime dependencies the catalogue manifests declare.
: >"$ROOT/usr/lib/libSDL2-2.0.so.0"
: >"$ROOT/usr/lib/libSDL2_image-2.0.so.0"
: >"$ROOT/usr/lib/libSDL2_ttf-2.0.so.0"
: >"$ROOT/lib/libresolv.so.2"
: >"$ROOT/lib/libpthread.so.0"

# Filesystem display candidate for the device. On a HiDPI desktop the SDL
# window reports pixels, so pass -resolution to pin the device size first.
printf 'U:%sp-0\n' "$RESOLUTION" >"$ROOT/sys/class/graphics/fb0/mode"

echo "desktop fixture written: $ROOT"
echo "  device:     $DEVICE"
echo "  firmware:   knulli (scarab)"
echo "  resolution: $RESOLUTION"
echo "  root holds: device files, installs, logs, controller mappings"
