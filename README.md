<p align="center">
  <img src="docs/assets/icon.svg" alt="Knulli App Store" width="128" height="128" />
</p>
<h1 align="center">Knulli App Store</h1>
<p align="center">
  <a href="https://github.com/jellydn/knulli-app-store/actions/workflows/check.yml">
    <img alt="CI" src="https://github.com/jellydn/knulli-app-store/actions/workflows/check.yml/badge.svg" />
  </a>
  <a href="https://github.com/jellydn/knulli-app-store/releases/latest">
    <img alt="Release" src="https://img.shields.io/github/v/release/jellydn/knulli-app-store?include_prereleases&sort=semver&label=release" />
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
>
> Builds are published as dated experimental pre-releases on every green push to `main`; pushing a `v*` tag cuts a full release.

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

Download the newest device build from [GitHub Releases](https://github.com/jellydn/knulli-app-store/releases), then follow the [TrimUI Smart Pro](docs/trimui-smart-pro.md) or [MagicX Zero 28](docs/magicx-zero-28.md) guide.

### App menu entries and metadata

PlayTime, Grout, and RetSend belong in **Tools**; the Store and Random Game Picker belong in **Ports**. Install and Repair create missing menu entries with a name, description, and bundled image where declared. The Store registers its own missing Ports entry on launch without shipping a shared gamelist over other games. A queued refresh is not proof that EmulationStation finished reloading; return to the frontend and restart it if needed.

After installing a Store build containing the RetSend menu fix, run **Repair** on RetSend 1.0.0. This moves its launcher from `RetSend/tools/Retsend.sh` to `RetSend/RetSend.sh`, within Knulli's default scan depth. Its binary, configuration, identity, history, and inbox remain in their existing locations. Run Repair on PlayTime and Grout to add missing menu metadata. Reopen the Store to fill its own missing metadata, then refresh the game lists or restart EmulationStation if needed.

For existing manual, scraped, or app-created entries, the Store fills only blank name, description, and image fields for the exact launcher path. Existing names and artwork, play history, and custom fields are preserved. Filling blanks does not grant ownership, so Uninstall does not remove these entries. No manual gamelist deletion is needed. See [menu ownership and migration](docs/how-to-add-a-package.md#emulationstation-menu-metadata).

### Update the Store from the Store

After installing a build with self-update support, select the **Knulli App Store** card, then **Check for updates**. Confirm the check. If a newer published build is available, select **Update** and confirm again. Exit and reopen the Store after completion.

The check includes official experimental pre-releases and uses the build's packaged target, not a device allowlist. The Store verifies GitHub's asset size and SHA-256 before extraction, replaces only its seven release files, and keeps installed apps, settings, and manager state. Failed writes roll back; startup recovers interrupted transactions before loading the signed catalogue. It does not run a downloaded installer script.

Self-update is available for device builds installed at `/userdata/roms/ports/knulli-app-store`. Custom locations and older builds need a manual ZIP update first. Network failures or missing digests leave Update unavailable. The trust root is the official GitHub repository over HTTPS, not an independent long-lived update signing key.

If self-update reports **text file busy**, the running build still uses the old in-place executable copy. It cannot install its own fix through that updater. Install a release containing the atomic-copy fix once using the manual ZIP procedure: verify the checksum, shut down Knulli, and replace all Store files from the device ZIP together. Do not delete `/userdata/system/knulli-app-store` or installed app data. See the [TrimUI procedure](docs/trimui-smart-pro.md#update-an-earlier-test-build) or [MagicX procedure](docs/magicx-zero-28.md#install-and-automatic-controls). Later self-updates use temporary-file replacement, and a failed rollback retains its journal and backups for recovery.

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

The catalogue has seven packages. All four packaged apps—Grout, PlayTime, Random Game Picker, and RetSend—can install on any Knulli device when architecture, ABI, and runtime dependencies pass. Missing device identity, an unlisted board, or a display outside the recorded scope does not block installation. **UNTESTED** means no exact device report exists; it is a warning, not an installation gate. PocketCurator, RAOfflineProxy and retroshelf retain validated release metadata for incremental review but have no installation definitions or install action. PortMaster is a featured external provider and is not installed from this catalogue.

| Package | What it does | Status | Remaining blocker |
| --- | --- | --- | --- |
| [Grout 5.2.0.0](https://github.com/rommapp/grout) | Connects a Linux retro handheld to a RomM server | Broad experimental Knulli | No 5.2.0.0 device test; Store staging disables the self-updater with a verified binary patch |
| [PlayTime 1.0.0](https://github.com/unitreign/playtime) | Tracks game play time on Knulli | Broad experimental Knulli | Exact 1.0.0 Smart Pro evidence only; other matching devices are experimental |
| [Random Game Picker 0.1.1](https://github.com/jellydn/knulli-random-game-picker) | Opens on a random existing ES game, with saved filters and scraped artwork | Any Knulli device; untested without exact evidence | New selector-first flow needs handheld retesting; earlier TSP evidence does not verify this version. Requires glibc 2.36+, SDL2/image/ttf and loopback ES API; [review](docs/catalogue-review-2026-10-19.md) |
| [RetSend 1.0.0](https://github.com/mxmgorin/retsend/releases/tag/v1.0.0) | Sends and receives files over local Wi-Fi with any LocalSend peer | Broad experimental Knulli | Official upstream Knulli ZIP; no device test. Store staging retains existing config/TLS identity and the default inbox. TLS, receive safety, quotas, signed provenance, and missing bundled notices remain open; [review](docs/retsend-upstream-1.0.0.md) |
| [PocketCurator](https://github.com/tomtombombadil/PocketCurator) v1.1.2 | Manages a user's own ROM collection from an EmulationStation-style interface | Approved; blocked | ZIP checksum and inventory verified. Executable-code downloads, PortMaster runtime acquisition, ROM-fetch policy, and write containment still need a reviewed package |
| [RAOfflineProxy](https://github.com/misantronic/RAOfflineProxy) v2.2.0-alpha1 | Provides an offline RetroAchievements proxy for supported emulators | Approved; blocked | Store can now decode the pinned embedded tar.gz without executing the installer. Still needs updater disabling, shared boot/config ownership controls, and Python/native dependency validation |
| [retroshelf](https://github.com/retroshelf-org/linux) v0.5.6 | Records and syncs game sessions | Candidate; blocked | Pinned ZIP and publisher checksum verified. Source remains unavailable; license, binary updater controls, hook ownership and keep-data uninstall need review |

The notes for PlayTime and Grout record evidence reviewed on 2026-09-15 and 2026-09-17. RetSend's upstream 1.0.0 review is dated 2026-10-19. All three candidates were inspected again on 2026-10-24: downloaded assets matched GitHub digests and archive inventories were checked without running launchers or installers. PocketCurator and retroshelf also matched publisher checksum files. Disposable-file probes confirmed PocketCurator's deletion function deletes a supplied path outside fixture userdata, and RAOfflineProxy's boot-hook writer modifies a supplied startup path outside fixture userdata. These test the write functions, not reachability through every UI path. RAOfflineProxy's native libraries require AArch64 glibc 2.17 symbols; its SQLite extension targets CPython 3.12. Removing its shell wrapper does not remove emulator-config changes that must be undone before uninstall. retroshelf v0.5.6 still applies staged code and repairs hooks through its binary; setting its launcher update flag alone does not establish safe runtime behavior. None of the three was promoted: warnings and checksum pinning cannot provide runtime containment or safe hook cleanup. The Store has no runtime sandbox and never executes upstream uninstall hooks. Upstream facts can change. The weekly [catalogue update check](https://github.com/jellydn/knulli-app-store/actions/workflows/catalogue-updates.yml) reports metadata changes for manual review; it never edits or approves a package.

Removing a package from the catalogue does not delete an existing installation or its user data. It no longer appears in the GUI catalogue. The CLI can still uninstall it from its saved installed state: `knulli-app uninstall PACKAGE_ID`.

## Current scope

- Manifest-driven packages for any Knulli device with matching architecture, a supported glibc ABI, and declared runtime libraries; device and display scope are advisory
- Experimental GUI artifacts for TrimUI Smart Pro at 1280×720 and MagicX Zero 28 at 640×480
- Exact device reports control tested badges; unknown device/display values remain unknown and do not block installation
- ZIP and `tar.gz` release archives, up to 512 MiB compressed and installed
- Writes below `/userdata` only

The GUI is functional on one community-tested TrimUI Smart Pro. A real MagicX diagnostic confirms Knulli Scarab, `aarch64`, 640×480, and SDL GameController `magicx-input`; full GUI testing is still incomplete. All packaged apps are available on untested hardware when runtime checks pass. A matching device is not verified unless the manifest has evidence for that exact package version and matrix. See the [advisory device policy](docs/adr/0009-advisory-device-compatibility.md).

ROM-download sources are outside the official catalogue policy.

## Experimental device GUI

The controller-native SDL2 GUI opens immediately on one app card at a time. **Left/right** (or up/down) browse apps; **L1/R1** filter All, Ready, and Installed. **ES A** opens confirmation for the first available action; it never installs on the first press. **B** cancels back to the card. **Y** opens More, with package details and all permitted actions; Y inside More exports diagnostics. Selecting an issue still opens its health check first. Hold **Select+Y** to quit. A detected external copy offers **Manage existing**, not Install; an empty directory is not an external copy. Only installer-approved actions are available, and tested badges do not bypass safety checks. No setup wizard or custom mapping editor is needed. Existing `controller-mappings.json` files are ignored and left untouched. The launcher selects `STORE_CONTROLLER_LAYOUT=knulli`, or `xbox` when `/var/run/ports_xbox_layout.flag` exists. Bindings compensate once for Knulli's ES a/b/x/y → SDL b/a/y/x mapping; events are not swapped. Desktop SDL remains normal, and footer letters follow the selected layout.

Current Knulli is identified through `OS_NAME="knulli"` in `/etc/os-release`, not its inherited `ID=buildroot`. The release identifier comes from `/usr/share/knulli/knulli.version`. Compatibility errors include the selected raw value, normalized value, source, and full detected device matrix. Press Y for More, then Y to export a redacted diagnostic text bundle. On an empty catalogue or a blocked-controller screen, Y still exports directly. The active log is capped at 512 KiB with one rotated copy at `/userdata/system/logs/knulli-app-store.log.1`.

Package health checks report the exact managed path and full expected and actual hash or mode. Selecting **Issue** opens this health section before the action list and gives the Repair step. Each check hashes every immutable managed file. State records the mode that the destination filesystem applies, not only the requested staging mode, and the check tolerates a wider mode from a filesystem that normalizes permissions. Only the loss of a read, write, or execute permission the installer set is an issue, so a file it requested as `0755` and the SD card reports as `0777` stays healthy. This prevents a false Repair state when a Knulli filesystem normalizes permissions. PlayTime's runtime database, hook, and other data outside its three immutable release files do not cause a health failure.

If **Manage existing** fails, the recovery screen shows the exact reason and offers **Retry manage**, **Export diagnostics**, and **Cancel**. **Force reinstall** also appears for a recoverable ownership failure or validated stale transaction, but not for an active transaction or a security, compatibility, licensing, or space failure. Force reinstall is never a normal first action. It needs two confirmations, keeps every mandatory security check, preserves declared data, and writes a complete timestamped recovery backup under `/userdata/system/knulli-app-store/recovery-backups/<package-id>/`.

Failed operation context is stored under `/userdata/system/knulli-app-store/lifecycle/`. A retry stays bound to its originating operation: failed fresh installs retry **Install**, while update, repair, management, force-reinstall, and uninstall failures retry their own operation only when the current package state still permits it. An absent package never routes to **Manage existing**. Normal compatibility and integrity checks run again on every retry.

Every green push to `main` publishes a dated experimental pre-release with the device zips and `SHA256SUMS.txt`. The same zips stay available as `check` workflow artifacts for pull-request review.

## Run tests

```sh
make check
make build
make catalogue
make catalogue-check
```

`make check` formats, vets, and runs tests. SDL GUI compile and layout tests need `libsdl2-dev` and `go test -tags sdl ./...`.

`make catalogue-check` builds the index twice and requires the two builds to be byte-identical. `build/` is gitignored, so a `git diff` on the generated index reports nothing whether or not it changed; this target is the check that can actually fail.

`make cover` runs the coverage gate: every package holding production Go files must have a test file, and the total must stay above `COVER_MIN` (70%). It is a separate target so `make check` stays the fast local loop; CI runs the gate on every pull request.

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
