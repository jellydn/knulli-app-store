# Knulli App Store

## What

[Knulli App Store](https://github.com/jellydn/knulli-app-store) is a safe community package manager for [Knulli](https://knulli.org/) utilities, themes, and integrations. It has a reviewable catalogue, a transactional installer, and a controller-only SDL2 interface.

The catalogue has four packages. [Grout 5.1.0.0](https://github.com/rommapp/grout) is verified only on TrimUI Smart Pro. [PlayTime 1.0.0](https://github.com/unitreign/playtime) has a successful Smart Pro report but stays experimental because MagicX Zero 28 is not yet package-tested. [RAOfflineProxy](https://github.com/misantronic/RAOfflineProxy) and [PocketCurator](https://github.com/tomtombombadil/PocketCurator) are approved for future tests but are blocked until their technical review is complete.

## Why

Package operations can damage user data. The installer verifies immutable releases, limits writes, preserves declared data, records ownership, and rolls back failed operations. It never runs a remote install script. See the [security model](docs/security-model.md) and [architecture](docs/architecture.md).

## How

Download the newest device build from [GitHub Actions](https://github.com/jellydn/knulli-app-store/actions/workflows/check.yml), then follow the [TrimUI Smart Pro](docs/trimui-smart-pro.md) or [MagicX Zero 28](docs/magicx-zero-28.md) installation guide. Package evidence is in the [review guide](docs/manifest-review.md) and [real-device test record](docs/real-device-tests.md). Contributors can use [How to add a package](docs/how-to-add-a-package.md).

## Current scope

- Manifest-driven packages for reviewed Knulli `aarch64` devices; the original installer test matrix is H700
- Experimental GUI artifacts for TrimUI Smart Pro at 1280×720 and MagicX Zero 28 at 640×480
- One or more explicit resolutions per approved manifest; no resolution is assumed
- ZIP and `tar.gz` release archives, up to 512 MiB compressed and installed
- Writes below `/userdata` only

The GUI is functional on one community-tested TrimUI Smart Pro. A real MagicX diagnostic confirms Knulli Scarab, `aarch64`, 640×480, and SDL GameController `magicx-input`; full GUI testing is still incomplete. Grout remains unsupported on MagicX. PlayTime remains experimental there.

PortMaster is a featured external provider. Users open it through Knulli's official install and launch mechanism. This repository does not copy the PortMaster catalogue or implement another PortMaster installer. ROM-download sources are outside the official catalogue policy.

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

# Take ownership of a copy that already exists at the destination, without reinstalling it.
sudo ./build/knulli-app adopt package.json

sudo ./build/knulli-app repair package.json
sudo ./build/knulli-app update package-v2.json
sudo ./build/knulli-app uninstall org.example.package
```

`-root` redirects all device paths into another directory. Tests use it for a complete temporary-filesystem workflow. It is also useful for offline inspection; it must not point at an untrusted tree with symlinked path components.

After a committed menu change the command prints `game list refresh accepted` when Knulli accepted the reload request, or `restart required to update game list` when it did not. Accepted means queued, not completed.

The installer never runs remote install scripts. It only copies regular files from a verified archive, preserves declared configuration, and writes one optional EmulationStation menu record.

## Catalogue status

The catalogue contains one verified package, one experimental package, and two approved but non-actionable candidates. Community approval records provenance separately from technical and real-device status.

| Package | Status | Remaining blocker |
| --- | --- | --- |
| PlayTime 1.0.0 | Smart Pro tested; MagicX experimental | The Smart Pro report did not itemize lifecycle steps; no MagicX package result |
| Grout 5.1.0.0 | Verified on Smart Pro | The report was positive but did not itemize lifecycle steps; built-in updater must not be used |
| RAOfflineProxy v1.13.0-alpha1 | Approved; blocked | The Knulli asset is a self-extracting script; supported archive, extracted size, dependencies, narrow writes, and updater safety are unresolved |
| PocketCurator v1.1.2 | Approved; blocked | The release is mutable; extracted inventory/size, narrow ROM and game-list writes, updater safety, and exact Knulli evidence are unresolved |

These notes record evidence reviewed on 2026-09-15. Upstream facts can change. The weekly [catalogue update check](https://github.com/jellydn/knulli-app-store/actions/workflows/catalogue-updates.yml) reports metadata changes for manual review; it never edits or approves a package. Follow the [manifest review workflow](docs/manifest-review.md) before promotion.

## Experimental device GUI

The controller-native SDL2 GUI browses the catalogue, shows package details and separate approval/technical states, and presents only actions allowed by the installer service. A copy that already exists at the destination shows the row state **EXTERNAL**; when the package is actionable, its action becomes **Manage existing** instead of **Install**. On the first launch for each device/controller identity, a dedicated setup screen requires the user to use, test, or customize the detected mapping before the catalogue opens. It uses semantic actions and prefers Knulli's `SDL_GAMECONTROLLERCONFIG`. No universal physical A/B assumption remains.

Current Knulli is identified through `OS_NAME="knulli"` in `/etc/os-release`, not its inherited `ID=buildroot`. The release identifier comes from `/usr/share/knulli/knulli.version`. Compatibility errors include the selected raw value, normalized value, source, and full detected device matrix. Use Settings to export a redacted diagnostic text bundle. The active log is capped at 512 KiB with one rotated copy at `/userdata/system/logs/knulli-app-store.log.1`.

Package health checks now report the exact managed path and failed content or mode check. State records the mode that the destination filesystem applies, not only the requested staging mode. This prevents a false Repair state when a Knulli filesystem normalizes permissions. PlayTime's runtime database, hook, and other data outside its three immutable release files do not cause a health failure.

The GUI validates SDL renderer, window, display-mode, and framebuffer resolution candidates before compatibility checks. It records the selected source and rejects corrupted or implausible dimensions instead of trusting `fb0/virtual_size`.

GitHub Actions publishes separate `knulli-app-store-trimui-smart-pro-experimental` and `knulli-app-store-magicx-zero-28-experimental` artifacts after all checks pass. See the [TrimUI Smart Pro guide](docs/trimui-smart-pro.md) and [MagicX Zero 28 guide](docs/magicx-zero-28.md).
