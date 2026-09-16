<p align="center">
  <img src="docs/assets/icon.svg" alt="Knulli App Store" width="128" height="128" />
</p>
<h1 align="center">Knulli App Store</h1>
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
- [Current catalogue review](docs/catalogue-review-2026-09-15.md)
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

# Build a deterministic index. Device CI also writes catalog-index.json.sig.
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
| [PlayTime 1.0.0](https://github.com/unitreign/playtime) | Broad experimental Knulli | Exact 1.0.0 Smart Pro evidence only; other matching devices are experimental |
| [Grout 5.2.0.0](https://github.com/rommapp/grout) | Broad experimental Knulli | No 5.2.0.0 device test; Store staging disables the self-updater with a verified binary patch |
| [RAOfflineProxy](https://github.com/misantronic/RAOfflineProxy) v1.13.0-alpha1 | Approved; blocked | The Knulli asset is a self-extracting script; supported archive, extracted size, dependencies, narrow writes, and updater safety are unresolved |
| [PocketCurator](https://github.com/tomtombombadil/PocketCurator) v1.1.2 | Approved; blocked | The release is mutable; extracted inventory/size, narrow ROM and game-list writes, updater safety, and exact Knulli evidence are unresolved |

These notes record evidence reviewed on 2026-09-15. Upstream facts can change. The weekly [catalogue update check](https://github.com/jellydn/knulli-app-store/actions/workflows/catalogue-updates.yml) reports metadata changes for manual review; it never edits or approves a package.

## Current scope

- Manifest-driven packages for detected Knulli `aarch64` devices with a supported glibc ABI, declared runtime libraries, a known device identity, and validated display bounds
- Experimental GUI artifacts for TrimUI Smart Pro at 1280×720 and MagicX Zero 28 at 640×480
- Exact tested resolutions or reviewed experimental display bounds; no resolution is assumed
- ZIP and `tar.gz` release archives, up to 512 MiB compressed and installed
- Writes below `/userdata` only

The GUI is functional on one community-tested TrimUI Smart Pro. A real MagicX diagnostic confirms Knulli Scarab, `aarch64`, 640×480, and SDL GameController `magicx-input`; full GUI testing is still incomplete. PlayTime and Grout are available for experimental tests on any detected device that satisfies their exact architecture, ABI, dependency, and display bounds. A matching device is not verified unless the manifest has evidence for that exact package version and matrix.

ROM-download sources are outside the official catalogue policy.

## Experimental device GUI

The controller-native SDL2 GUI browses the catalogue, shows package details and separate approval/technical states, and presents only actions allowed by the installer service. A copy that already exists at the destination shows the row state **EXTERNAL**; when the package is actionable, its action becomes **Manage existing** instead of **Install**. Only files count: an empty directory skeleton left behind by a rolled-back write is not an external copy, so the row offers **Install**. On the first launch for each device/controller identity, a dedicated setup screen requires the user to use, test, or customize the detected mapping before the catalogue opens. It uses semantic actions and prefers Knulli's `SDL_GAMECONTROLLERCONFIG`. No universal physical A/B assumption remains. Each screen keeps its available controls in one footer line read from the active mapping. Calibration can instead show one actionless instruction there. Mapping summaries identify saved bindings without adding a second control hint inside the panel.

Current Knulli is identified through `OS_NAME="knulli"` in `/etc/os-release`, not its inherited `ID=buildroot`. The release identifier comes from `/usr/share/knulli/knulli.version`. Compatibility errors include the selected raw value, normalized value, source, and full detected device matrix. Use Settings to export a redacted diagnostic text bundle. The active log is capped at 512 KiB with one rotated copy at `/userdata/system/logs/knulli-app-store.log.1`.

Package health checks report the exact managed path and full expected and actual hash or mode. Selecting **Issue** opens this health section before the action list and gives the Repair step. Each check hashes every immutable managed file. State records the mode that the destination filesystem applies, not only the requested staging mode, and the check tolerates a wider mode from a filesystem that normalizes permissions. Only the loss of a read, write, or execute permission the installer set is an issue, so a file it requested as `0755` and the SD card reports as `0777` stays healthy. This prevents a false Repair state when a Knulli filesystem normalizes permissions. PlayTime's runtime database, hook, and other data outside its three immutable release files do not cause a health failure.

If **Manage existing** fails, the recovery screen shows the exact reason and offers **Retry manage**, **Export diagnostics**, and **Cancel**. **Force reinstall** also appears for a recoverable ownership failure or validated stale transaction, but not for an active transaction or a security, compatibility, licensing, or space failure. Force reinstall is never a normal first action. It needs two confirmations, keeps every mandatory security check, preserves declared data, and writes a complete timestamped recovery backup under `/userdata/system/knulli-app-store/recovery-backups/<package-id>/`.

Failed operation context is stored under `/userdata/system/knulli-app-store/lifecycle/`. A retry stays bound to its originating operation: failed fresh installs retry **Install**, while update, repair, management, force-reinstall, and uninstall failures retry their own operation only when the current package state still permits it. An absent package never routes to **Manage existing**. Normal compatibility and integrity checks run again on every retry.

GitHub Actions publishes separate `knulli-app-store-trimui-smart-pro-experimental` and `knulli-app-store-magicx-zero-28-experimental` artifacts after all checks pass.

## Run tests

```sh
make check
make build
make catalogue
git diff --exit-code -- build/catalog-index.json
```

`make check` formats, vets, and runs tests. SDL GUI compile and layout tests need `libsdl2-dev` and `go test -tags sdl ./...`.

`make gui` opens an interactive desktop GUI with no handheld and no controller: the keyboard carries the semantic actions and a scratch fixture root supplies the Knulli files platform detection reads. `make walkthrough` drives the offline flows and records the opening screen, each key result, and operation completions. Use `make walkthrough WALK_FLAGS=--install` to include the network-backed install, health, repair, and uninstall flows. The `gui-walkthrough` artifact on each pull request contains the offline evidence. See [Desktop GUI verification](docs/desktop-verification.md) for the bindings, fixture contents, flows, and device-only checks.

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
