<h1 align="center">Welcome to knulli-app-store 👋</h1>
<p align="center">
  <a href="https://github.com/jellydn/knulli-app-store/actions/workflows/check.yml">
    <img alt="CI" src="https://github.com/jellydn/knulli-app-store/actions/workflows/check.yml/badge.svg" />
  </a>
  <img alt="Go" src="https://img.shields.io/badge/Go-%3E%3D1.27-00ADD8.svg" />
  <a href="https://knulli.org/">
    <img alt="Knulli" src="https://img.shields.io/badge/Knulli-community-blue.svg" />
  </a>
  <a href="LICENSE">
    <img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg" />
  </a>
  <a href="https://twitter.com/jellydn">
    <img alt="Twitter: jellydn" src="https://img.shields.io/twitter/follow/jellydn.svg?style=social" />
  </a>
</p>

> Safe community package manager for [Knulli](https://knulli.org/) utilities, themes, and integrations. Reviewable catalogue, transactional installer, controller-only SDL2 interface.

## ✨ Features

- Reviewable catalogue with separate community approval and technical status
- Transactional installer that never runs a remote install script
- Immutable HTTPS releases with exact size and SHA-256 checks
- Writes below `/userdata` only, with snapshots, a crash journal, and rollback
- Controller-only SDL2 GUI for TrimUI Smart Pro (1280×720) and MagicX Zero 28 (640×480)
- Static `aarch64` CLI for review, install, repair, and uninstall
- PortMaster as a featured external provider. This repo does not copy that catalogue

## 🏠 [Homepage](https://github.com/jellydn/knulli-app-store)

### 📖 Guides

- [TrimUI Smart Pro](docs/trimui-smart-pro.md)
- [MagicX Zero 28](docs/magicx-zero-28.md)
- [Security model](docs/security-model.md)
- [Architecture](docs/architecture.md)
- [How to add a package](docs/how-to-add-a-package.md)
- [Manifest review](docs/manifest-review.md)
- [Real-device tests](docs/real-device-tests.md)
- [Architecture decisions](docs/adr/)

## Prerequisites

- Go >= 1.27
- `libsdl2-dev` for the experimental GUI
- A reviewed Knulli `aarch64` device for on-device use

## Install

Download the newest device build from [GitHub Actions](https://github.com/jellydn/knulli-app-store/actions/workflows/check.yml), then follow the [TrimUI Smart Pro](docs/trimui-smart-pro.md) or [MagicX Zero 28](docs/magicx-zero-28.md) guide.

Build the CLI locally:

```sh
make check
make build
make catalogue
```

Build the initial device CLI without CGO:

```sh
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -o build/knulli-app-aarch64 ./cmd/knulli-app
```

## Usage

```sh
# Review one manifest without installing it.
go run ./cmd/knulli-app validate catalogue/packages/io.github.unitreign.playtime.json

# Build a deterministic index that release automation can sign.
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

The catalogue has four packages. Community approval records provenance separately from technical and real-device status.

| Package | Status | Remaining blocker |
| --- | --- | --- |
| [PlayTime 1.0.0](https://github.com/unitreign/playtime) | Smart Pro tested; MagicX experimental | The Smart Pro report did not itemize lifecycle steps; no MagicX package result |
| [Grout 5.1.0.0](https://github.com/rommapp/grout) | Verified on Smart Pro | The report was positive but did not itemize lifecycle steps; built-in updater must not be used |
| [RAOfflineProxy](https://github.com/misantronic/RAOfflineProxy) v1.13.0-alpha1 | Approved; blocked | The Knulli asset is a self-extracting script; supported archive, extracted size, dependencies, narrow writes, and updater safety are unresolved |
| [PocketCurator](https://github.com/tomtombombadil/PocketCurator) v1.1.2 | Approved; blocked | The release is mutable; extracted inventory/size, narrow ROM and game-list writes, updater safety, and exact Knulli evidence are unresolved |

These notes record evidence reviewed on 2026-09-15. Upstream facts can change. The weekly [catalogue update check](https://github.com/jellydn/knulli-app-store/actions/workflows/catalogue-updates.yml) reports metadata changes for manual review; it never edits or approves a package.

## Current scope

- Manifest-driven packages for reviewed Knulli `aarch64` devices; the original installer test matrix is H700
- Experimental GUI artifacts for TrimUI Smart Pro at 1280×720 and MagicX Zero 28 at 640×480
- One or more explicit resolutions per approved manifest; no resolution is assumed
- ZIP and `tar.gz` release archives, up to 512 MiB compressed and installed
- Writes below `/userdata` only

The GUI is functional on one community-tested TrimUI Smart Pro. A real MagicX diagnostic confirms Knulli Scarab, `aarch64`, 640×480, and SDL GameController `magicx-input`; full GUI testing is still incomplete. Grout remains unsupported on MagicX. PlayTime remains experimental there.

ROM-download sources are outside the official catalogue policy.

## Experimental device GUI

The controller-native SDL2 GUI browses the catalogue, shows package details and separate approval/technical states, and presents only actions allowed by the installer service. A copy that already exists at the destination shows the row state **EXTERNAL**; when the package is actionable, its action becomes **Manage existing** instead of **Install**. On the first launch for each device/controller identity, a dedicated setup screen requires the user to use, test, or customize the detected mapping before the catalogue opens. It uses semantic actions and prefers Knulli's `SDL_GAMECONTROLLERCONFIG`. No universal physical A/B assumption remains.

Current Knulli is identified through `OS_NAME="knulli"` in `/etc/os-release`, not its inherited `ID=buildroot`. The release identifier comes from `/usr/share/knulli/knulli.version`. Compatibility errors include the selected raw value, normalized value, source, and full detected device matrix. Use Settings to export a redacted diagnostic text bundle. The active log is capped at 512 KiB with one rotated copy at `/userdata/system/logs/knulli-app-store.log.1`.

Package health checks report the exact managed path and failed content or mode check. State records the mode that the destination filesystem applies, not only the requested staging mode. This prevents a false Repair state when a Knulli filesystem normalizes permissions. PlayTime's runtime database, hook, and other data outside its three immutable release files do not cause a health failure.

GitHub Actions publishes separate `knulli-app-store-trimui-smart-pro-experimental` and `knulli-app-store-magicx-zero-28-experimental` artifacts after all checks pass.

## Run tests

```sh
make check
make build
make catalogue
git diff --exit-code -- build/catalog-index.json
```

`make check` formats, vets, and runs tests. SDL GUI compile and layout tests need `libsdl2-dev` and `go test -tags sdl ./...`.

## Author

👤 **Huynh Duc Dung**

- Website: https://productsway.com/
- Twitter: [@jellydn](https://twitter.com/jellydn)
- Github: [@jellydn](https://github.com/jellydn)

## 🤝 Contributing

Contributions, issues and feature requests are welcome.

Feel free to check [issues page](https://github.com/jellydn/knulli-app-store/issues). You can also take a look at the [contributing guide](CONTRIBUTING.md).

Package pull requests need a complete review of license, immutable release asset, checksum, Knulli compatibility, install paths, preserved files, network use, and every write path. A README statement alone is not compatibility evidence.

## Show your support

[![kofi](https://img.shields.io/badge/Ko--fi-F16061?style=for-the-badge&logo=ko-fi&logoColor=white)](https://ko-fi.com/dunghd)
[![paypal](https://img.shields.io/badge/PayPal-00457C?style=for-the-badge&logo=paypal&logoColor=white)](https://paypal.me/dunghd)
[![buymeacoffee](https://img.shields.io/badge/Buy_Me_A_Coffee-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/dunghd)

Give a ⭐️ if this project helped you!

## 📝 License

Copyright © 2026 [Huynh Duc Dung](https://github.com/jellydn).

This project is [MIT](LICENSE) licensed.
