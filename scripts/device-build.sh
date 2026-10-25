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

# Host packaging tools can be missing even when the cross-toolchain is ready.
if ! command -v zip >/dev/null 2>&1 || ! command -v file >/dev/null 2>&1 || ! command -v readelf >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq binutils file zip
fi

if ! command -v aarch64-linux-gnu-gcc >/dev/null 2>&1 ||
  ! PKG_CONFIG_LIBDIR=/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/share/pkgconfig pkg-config --exists sdl2 SDL2_image SDL2_ttf SDL2_gfx; then
  dpkg --add-architecture arm64
  rm -rf /var/lib/apt/lists/*
  apt-get update -qq
  apt-get install -y -qq binutils file gcc-aarch64-linux-gnu libc6-dev-arm64-cross libsdl2-dev:arm64 libsdl2-image-dev:arm64 libsdl2-ttf-dev:arm64 libsdl2-gfx-dev:arm64 pkg-config zip
fi

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT HUP INT TERM

mkdir -p "$OUTPUT_DIR"
CATALOG="$WORK/catalog-index.json"
KEY="$WORK/catalog-signing-key"

# The index is signed with a key generated for this build. The GUI retains the
# verification key; the CLI's unused key can be removed by the linker. Ship
# the GUI and its signed index together until a stable key policy exists.
go run ./cmd/knulli-app catalogue -output "$CATALOG" -generate-signing-key "$KEY"
PUBLIC_KEY=$(tr -d ' \n' < "$KEY.pub")
if [ "${#PUBLIC_KEY}" -ne 64 ]; then
  echo "expected a 32-byte hex catalogue public key" >&2
  exit 1
fi
echo "catalogue public key $PUBLIC_KEY" >&2

LDFLAGS="-s -w -X github.com/jellydn/knulli-app-store/internal/catalog.embeddedPublicKeyHex=$PUBLIC_KEY -X github.com/jellydn/knulli-app-store/internal/appstore.StoreVersion=$VERSION -X github.com/jellydn/knulli-app-store/internal/appstore.StoreTarget=$TARGET"

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
  go build -buildvcs=false -trimpath -ldflags="$LDFLAGS" -o "$WORK/knulli-app" ./cmd/knulli-app

PKG_CONFIG_LIBDIR=/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/share/pkgconfig \
  CGO_LDFLAGS="-Wl,-Bstatic -lSDL2_gfx -Wl,-Bdynamic -lm" \
  CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
  go build -buildvcs=false -tags sdl -trimpath -ldflags="$LDFLAGS" -o "$WORK/knulli-app-ui" ./cmd/knulli-app-ui

# Gabagool initializes SDL_image and SDL_ttf as well as SDL2. SDL2_gfx is
# linked statically: do not assume that firmware supplies it or replace system
# libraries. Keep an exact allowlist and retain the existing glibc floor.
file "$WORK/knulli-app" "$WORK/knulli-app-ui" >&2
needed=$(readelf -d "$WORK/knulli-app-ui" | sed -n 's/.*Shared library: \[\(.*\)\]/\1/p' | LC_ALL=C sort)
expected=$(printf '%s\n' ld-linux-aarch64.so.1 libSDL2-2.0.so.0 libSDL2_image-2.0.so.0 libSDL2_ttf-2.0.so.0 libc.so.6 libm.so.6 | LC_ALL=C sort)
printf 'GUI shared libraries:\n%s\n' "$needed" >&2
if [ "$needed" != "$expected" ]; then
  echo "unexpected GUI shared libraries; expected SDL2, SDL2_image, SDL2_ttf and glibc (libc/libm/loader) only" >&2
  exit 1
fi
versions=$(readelf --version-info "$WORK/knulli-app-ui")
maximum_glibc=$(printf '%s\n' "$versions" | sed -nE 's/.*Name: GLIBC_([0-9]+(\.[0-9]+)*).*/\1/p' | sort -V | tail -n 1)
if [ -z "$maximum_glibc" ] || [ "$(printf '%s\n' 2.34 "$maximum_glibc" | sort -V | tail -n 1)" != 2.34 ]; then
  echo "the GUI must require glibc no newer than 2.34; found ${maximum_glibc:-no version requirements}" >&2
  exit 1
fi
echo "maximum required glibc: $maximum_glibc" >&2

scripts/package-device.sh "$WORK/knulli-app" "$WORK/knulli-app-ui" "$CATALOG" "$OUTPUT_DIR" "$VERSION" "$TARGET" >&2
