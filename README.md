# Knulli App Store

Knulli App Store is an early, safety-focused package manager for community Knulli utilities, themes, and integrations. It combines a reviewable catalogue, a transactional headless installer, and an experimental controller UI that uses the same installer interface.

## Current scope

- Manifest-driven packages for reviewed Knulli `aarch64` devices; the original installer test matrix is H700
- Experimental GUI artifacts for TrimUI Smart Pro at 1280×720 and MagicX Zero 28 at 640×480
- One or more explicit resolutions per approved manifest; no resolution is assumed
- ZIP and `tar.gz` release archives, up to 512 MiB compressed and installed
- Writes below `/userdata` only

The experimental GUI is functional on one community-tested TrimUI Smart Pro. MagicX Zero 28 support is source-backed but untested. This is not package evidence. Grout 5.1.0.0 and PlayTime 1.0.0 remain actionable only on Smart Pro and are blocked on MagicX. Neither package is verified.

PortMaster is a featured external provider. Users open it through Knulli's official install and launch mechanism. This repository does not copy the PortMaster catalogue or implement another PortMaster installer. ROM-download sources, including EmuDrop, are outside the official catalogue policy.
EmuDrop is one explicit community-approved exception. It remains non-installable and displays a copyright warning. This exception does not permit other ROM-download sources.

## Why the installer comes first

Package operations can damage user data even when a user interface looks safe. The command-line core makes compatibility checks, download verification, extraction, rollback, and uninstall behavior testable without SDL or a device display. See [Architecture](docs/architecture.md) and [Security model](docs/security-model.md).

## Build and check

Go 1.19 or newer is required.

```sh
make check
make build
make catalogue
```

Build the initial device target without CGO:

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o build/knulli-app-aarch64 ./cmd/knulli-app
```

## Command-line use

```sh
# Review one manifest without installing it.
go run ./cmd/knulli-app validate catalogue/packages/io.github.unitreign.playtime.json

# Build a deterministic index that can be signed by release automation.
go run ./cmd/knulli-app catalogue -output build/catalog-index.json

# Install an actionable local manifest. Flags fill values that detection cannot read.
sudo ./build/knulli-app install \
  -firmware knulli -firmware-version VERSION \
  -arch aarch64 -device trimui-smart-pro -resolution 1280x720 \
  package.json

sudo ./build/knulli-app repair package.json
sudo ./build/knulli-app update package-v2.json
sudo ./build/knulli-app uninstall org.example.package
```

`-root` redirects all device paths into another directory. Tests use it for a complete temporary-filesystem workflow. It is also useful for offline inspection; it must not point at an untrusted tree with symlinked path components.

The installer never runs remote install scripts. It only copies regular files from a verified archive, preserves declared configuration, and writes one optional EmulationStation menu record.

## Catalogue status

The catalogue contains two experimental packages and four non-actionable candidates. Community approval records provenance separately from technical and real-device status.

| Package | Status | Remaining blocker |
| --- | --- | --- |
| EmuDrop | Community approved | No license; launch-time updater bypasses manager verification and rollback; ROM/copyright risk |
| PlayTime 1.0.0 | Experimental test | No package-specific Smart Pro result or minimum Knulli version evidence |
| Grout 5.1.0.0 | Experimental test | No linked Smart Pro result or minimum Knulli version; built-in updater cannot be disabled |
| RAOfflineProxy | Not approved | Knulli asset is an unrestricted install script with no publisher checksum |
| ETK Tool | Not approved | Release is script-based; system-space behavior is outside the initial policy |
| PocketCurator | Not approved | A checksum exists, but the repository has no declared license; paths and device behavior are not tested |

These notes record evidence reviewed on 2026-09-15. Upstream facts can change. Follow the [manifest review workflow](docs/manifest-review.md) before promotion.

## Experimental device GUI

The controller-native SDL2 GUI browses the catalogue, shows package details and separate approval/technical states, and presents only actions allowed by the installer service. It uses semantic actions and prefers Knulli's `SDL_GAMECONTROLLERCONFIG`. A controller-only setup flow can save mappings by Knulli device and SDL controller identity. No universal physical A/B assumption remains.

Current Knulli is identified through `OS_NAME="knulli"` in `/etc/os-release`, not its inherited `ID=buildroot`. The release identifier comes from `/usr/share/knulli/knulli.version`. Compatibility errors include the selected raw value, normalized value, source, and full detected device matrix. Use Settings to export a redacted diagnostic text bundle. The active log is capped at 512 KiB with one rotated copy at `/userdata/system/logs/knulli-app-store.log.1`.

The GUI validates SDL renderer, window, display-mode, and framebuffer resolution candidates before compatibility checks. It records the selected source and rejects corrupted or implausible dimensions instead of trusting `fb0/virtual_size`.

GitHub Actions publishes separate `knulli-app-store-trimui-smart-pro-experimental` and `knulli-app-store-magicx-zero-28-experimental` artifacts after all checks pass. See the [TrimUI Smart Pro guide](docs/trimui-smart-pro.md) and [MagicX Zero 28 guide](docs/magicx-zero-28.md).
