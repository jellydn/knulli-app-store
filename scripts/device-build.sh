#!/bin/sh
# Builds the signed, ABI-verified device artifacts for one target.
#
# CI and a tagged release both call this script, so the cross-compile, the ABI
# gate, and the catalogue signature have exactly one home. They used to be
# copied into two workflows, where relaxing a gate in one path would have left
# the other shipping an artifact CI never validated.
#
# It expects the Debian Bookworm environment the workflows run in. The cross
# toolchain and the arm64 SDL headers are installed here when they are missing,
# so the caller does not restate a dependency list that can drift.
#
# usage: device-build.sh TARGET VERSION OUTPUT_DIR
#   TARGET      trimui-smart-pro or magicx-zero-28
#   VERSION     version stamped into the archive name
#   OUTPUT_DIR  directory for the archive, SHA256SUMS.txt, and the catalogue
#
# Everything informative is written to stderr, so the caller can read the step
# log without a hidden stdout contract.
set -eu

if [ "$#" -ne 3 ]; then
  echo "usage: $0 TARGET VERSION OUTPUT_DIR" >&2
  exit 2
fi

TARGET=$1
VERSION=$(printf '%s' "$2" | tr -cd 'A-Za-z0-9._-')
OUTPUT_DIR=$3

if [ -z "$VERSION" ]; then
  echo "version must contain a portable filename character" >&2
  exit 2
fi

case "$TARGET" in
  trimui-smart-pro|magicx-zero-28) ;;
  *) echo "unsupported device target: $TARGET" >&2; exit 2 ;;
esac

if ! command -v aarch64-linux-gnu-gcc >/dev/null 2>&1; then
  dpkg --add-architecture arm64
  rm -rf /var/lib/apt/lists/*
  apt-get update -qq
  apt-get install -y -qq binutils file gcc-aarch64-linux-gnu libc6-dev-arm64-cross libsdl2-dev:arm64 pkg-config zip
fi

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT HUP INT TERM

mkdir -p "$OUTPUT_DIR"
CATALOG="$WORK/catalog-index.json"
KEY="$WORK/catalog-signing-key"

# The index is signed with a key generated for this build, and the matching
# public key is compiled into both binaries, so a signed index cannot outlive
# the binary that embeds its key until a long-lived production key exists.
go run ./cmd/knulli-app catalogue -output "$CATALOG" -generate-signing-key "$KEY"
PUBLIC_KEY=$(tr -d ' \n' < "$KEY.pub")
if [ "${#PUBLIC_KEY}" -ne 64 ]; then
  echo "expected a 32-byte hex catalogue public key" >&2
  exit 1
fi
echo "catalogue public key $PUBLIC_KEY" >&2

LDFLAGS="-s -w -X github.com/jellydn/knulli-app-store/internal/catalog.embeddedPublicKeyHex=$PUBLIC_KEY"

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -buildvcs=false -trimpath -ldflags="$LDFLAGS" -o "$WORK/knulli-app" ./cmd/knulli-app

PKG_CONFIG_LIBDIR=/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/share/pkgconfig \
  CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
  go build -buildvcs=false -tags sdl -trimpath -ldflags="$LDFLAGS" -o "$WORK/knulli-app-ui" ./cmd/knulli-app-ui

# The GUI may link only SDL and glibc, and no glibc symbol newer than the oldest
# supported Knulli image provides.
file "$WORK/knulli-app" "$WORK/knulli-app-ui" >&2
readelf -d "$WORK/knulli-app-ui" | grep -F 'Shared library: [libSDL2-2.0.so.0]' >&2
if [ "$(readelf -d "$WORK/knulli-app-ui" | grep -c '(NEEDED)')" -ne 2 ]; then
  echo "the GUI must need exactly two shared libraries" >&2
  exit 1
fi
readelf --version-info "$WORK/knulli-app-ui" | grep -F 'Name: GLIBC_2.34' >&2

scripts/package-device.sh "$WORK/knulli-app" "$WORK/knulli-app-ui" "$CATALOG" "$OUTPUT_DIR" "$VERSION" "$TARGET" >&2
