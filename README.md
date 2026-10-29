# Knulli App Store

A community package manager for [Knulli](https://knulli.org/) utilities, themes, and integrations. Install, manage, update, repair, and remove reviewed packages with a controller or the CLI.

The Store verifies downloads and journals file changes for rollback. It never runs downloaded installers. Installed apps are **not sandboxed**.

## Devices and release

Release target: [v2026.10.27](https://github.com/jellydn/knulli-app-store/releases/tag/v2026.10.27). Includes Picker 0.2.2, which excludes Tools and Emulators applications. Hardware reports remain scoped to the artifacts actually tested. Device targets remain **experimental**:

| Target | Display | Evidence |
| --- | --- | --- |
| [TrimUI Smart Pro](docs/trimui-smart-pro.md) | 1280×720 | Historical native Option B development-build checks with Picker 0.2.0: Install, Update, Repair, injected navigation and ES return. No Picker 0.2.2 hardware gameplay evidence; physical presses and audio remain unverified. |
| [MagicX Zero 28](docs/magicx-zero-28.md) | 640×480 | Device diagnostic confirms Knulli, AArch64, display, and SDL controller; full GUI testing remains incomplete. |

**Knulli is the only implemented firmware profile.** Device GUI builds require AArch64, glibc, and system SDL2, SDL2_image, and SDL2_ttf; SDL2_gfx is linked statically. Do not replace firmware libraries.

Download examples for v2026.10.27:

- `knulli-app-store-trimui-smart-pro-experimental-v2026.10.27.zip`
- `knulli-app-store-magicx-zero-28-experimental-v2026.10.27.zip`
- `SHA256SUMS.txt`

Successful `main` builds also publish dated experimental pre-releases. A release version alone does not establish hardware evidence.

## Install and use

1. Download the target ZIP and checksum file into one directory on your computer:

   ```sh
   sha256sum --ignore-missing -c SHA256SUMS.txt
   ```

   Continue only if your ZIP reports `OK`.
2. Shut down Knulli and connect its data card to your computer.
3. Extract the ZIP into `/userdata/roms/ports`. Check for:
   - `Knulli App Store.sh`
   - `knulli-app-store/knulli-app-ui`
   - `images/Knulli App Store.png`
4. Safely eject, start Knulli, refresh game lists or reboot, then open **Knulli App Store** in **Ports**.
5. Select an app, read its warnings, and confirm the action.

An existing copy offers **Manage existing**: matching release files become owned without reinstalling; uncertain files remain unmanaged and are backed up before replacement. Empty directories do not count as external copies. **Repair** restores pinned code and keeps declared data. **Uninstall** removes owned files, restores originals where applicable, and keeps declared data. Close apps before Store operations.

The GUI requests one menu refresh on exit if package operations changed the game list, not after each install. One-shot CLI commands still refresh immediately. Menu refresh **accepted** means queued, not completed. Refresh or restart EmulationStation if an entry is missing.

### Update the Store

At the standard `/userdata/roms/ports/knulli-app-store` location, a build with self-update support offers a **Knulli App Store** entry. Confirm **Check for updates**, then **Update**; exit and reopen afterward. Checks include official experimental pre-releases and use the packaged target. Asset size and SHA-256 come from GitHub, not an independent update signing key.

For an older build, custom location, or **text file busy** failure, verify the ZIP and shut down Knulli before extracting it. Replace launcher, icon, binaries, catalogue, signature, and packaged guide together. Do not delete installed apps, settings, or `/userdata/system/knulli-app-store`. See [self-update safety and recovery](docs/security-model.md#store-self-update).

## Controls

The **Library** uses a stable package list beside a persistent detail pane. A mint outline marks focus; **All / Ready / Installed** filters stay visible. Test evidence and installation state are separate, and long titles and summaries wrap in the detail pane. Controller glyphs and keyboard hints name the active controls. This Zitch-led presentation revision starts with v2026.10.27.1. No setup wizard or custom mapping editor is needed.

| Action | Knulli controller | Desktop keyboard |
| --- | --- | --- |
| Browse apps | D-pad up/down; left/right also work | Arrow keys |
| Filter All, Ready, Installed | L1/R1 | Page Up/Page Down |
| Review or confirm | A | Enter |
| Cancel or go back | B | Escape |
| Open More; export diagnostics inside More | Y | Y |
| Quit | Hold Select, then press Y | Hold Tab or F5, then press Y |

The first A press opens confirmation, not installation. **More** contains details and secondary actions; **Issue** opens health first. Y exports diagnostics directly on empty-catalogue and blocked-controller screens. Face labels follow the active Knulli Ports layout; the launcher compensates once for default or Xbox mode. Legacy `controller-mappings.json` files are ignored and left unchanged. See [frontend design](docs/adr/0010-gabagool-knulli-frontend.md) and [device reports](docs/real-device-tests.md).

To test a separately prepared, unpublished build without replacing the production Store, use the [isolated device-test procedure](docs/ui-redesign-device-test.md). Normal release ZIPs are not isolated test packages.

## Catalogue and safety

On 2026-10-25 (Asia/Singapore), the [owner confirmed](https://ampcode.com/threads/T-01a17039-648c-7024-abe9-8785444a2b5b) seven package versions tested on TrimUI Smart Pro only, Knulli AArch64 at 1280×720. That report covers Picker 0.2.0, not the newly published 0.2.2. The other six current versions retain passed package evidence on that matrix only, not separate lifecycle results. **Picker 0.2.2 is UNTESTED on all matrices.** MagicX Zero 28 and other matrices remain untested for every package. All seven manifest review statuses remain **experimental**:

| Package | Current manifest version | Purpose |
| --- | --- | --- |
| Grout | 5.2.0.0 | Connect to a RomM server |
| PlayTime | 1.0.0 | Track game play time |
| Random Game Picker | 0.2.2 | Select from existing EmulationStation games |
| RetSend | 1.0.0 | Transfer files over local Wi-Fi with LocalSend peers |
| PocketCurator | 1.1.2 | Manage your ROM collection; requires PortMaster |
| RAOfflineProxy | 2.2.0-alpha1 | Offline casual RetroAchievements; requires Python 3.12 and pygame |
| retroshelf | 0.5.6 | Sync game sessions; requires an account and Wi-Fi |

Firmware, architecture, ABI, and runtime dependencies still gate installation. Device identity and display scope are advisory. **UNTESTED** means no passed report matches the exact package version and device matrix. See the [evidence and its limits](docs/real-device-tests.md#2026-10-25-current-owner-confirmation). [Picker 0.2.2](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.2.2) excludes Tools and Emulators application systems before gamelist parsing, from selection and during Play revalidation. Path-only candidates use boundary-aware application-root checks. It adds no hardware gameplay evidence. Its launcher, settings schema, `state/` preservation and exact-path ES launch contract are unchanged. It still requires glibc ≥2.34, firmware SDL2/image/ttf/gfx, and the loopback EmulationStation API. Its requirements are separate from the Store's.

PortMaster is an external provider: the Store does not install its catalogue. ROM-download sources are outside official catalogue policy. Use only ROMs you have the right to use and servers you trust.

- Reviewed manifests pin HTTPS release URLs, exact sizes, and SHA-256 hashes. Device builds verify a signed bundled catalogue.
- Archives are bounded to 512 MiB compressed and installed; unsafe paths, links, and non-regular entries are rejected.
- Store writes stay below `/userdata`, rejecting existing symlinks in write paths. Downloads are verified and staged before journaled mutations; installed state is written last.
- Declared data remains. Menu entries are removed only when Store-created and unchanged; pre-existing metadata is not overwritten.
- Candidates are read-only. Community approval, technical status, and exact-device tested badges are separate.

### Upstream-managed apps

**PocketCurator, RAOfflineProxy, and retroshelf** are actionable experimental packages, not read-only candidates. The Store verifies pinned installation bytes, but does not control later setup, runtime writes, downloads, or self-updates. Repair restores pinned code and keeps declared data.

- PocketCurator can permanently delete ROMs: back up first. For an older launcher-path failure, update the Store to v2026.10.24.4 or later, close PocketCurator, then use **More → Repair**. See the [native launch investigation](docs/real-device-tests.md#2026-10-24-pocketcurator-ssh-investigation).
- RAOfflineProxy lacks an independent publisher checksum; its pinned hash was checked against GitHub's asset digest. This trusts GitHub and the publisher's account.
- retroshelf's license is unverified; the Store downloads directly from its publisher without redistribution.
- RAOfflineProxy and retroshelf require manual hook/service cleanup before removal. Read the Store's cleanup messages and the [RAOfflineProxy](catalogue/packages/io.github.misantronic.raofflineproxy.json) / [retroshelf](catalogue/packages/org.retroshelf.knulli.json) manifest warnings. Declared read-only checks block removal; the Store never runs cleanup scripts.

See [package manifests](catalogue/packages/), [package contribution guidance](docs/how-to-add-a-package.md), [RetSend review](docs/retsend-upstream-1.0.0.md), and the [security model](docs/security-model.md) for preserved paths and boundaries.

The daily [catalogue update check](https://github.com/jellydn/knulli-app-store/actions/workflows/catalogue-updates.yml) runs at 12:00 PM Singapore time. It verifies eligible experimental release proposals, tests changed metadata, and drafts a PR for human review; ambiguous releases and patch-dependent packages remain manual. It never approves or merges a package. After human merge to `main`, successful normal checks automatically publish an experimental prerelease; full releases require a `v*` tag. See [contribution guidance](CONTRIBUTING.md) for permissions and validation.

## Build and test

Use **Go 1.27.3** with **Staticcheck 0.8.2** to match CI and local hooks. The analyzer supports the updated Go export format. GUI builds and `make check` also need SDL development files. On Debian or Ubuntu:

```sh
sudo apt-get install libsdl2-dev libsdl2-image-dev libsdl2-ttf-dev libsdl2-gfx-dev
make check             # Format, vet both tag sets, race tests
make build             # Static host CLI: build/knulli-app
make catalogue         # Validate manifests and build the index
make catalogue-check   # Require two byte-identical index builds
make cover             # Test files in production packages; ≥70% total coverage
go test -race -tags sdl ./...
make gui               # Keyboard GUI with a scratch device root
make gui GUI_DEVICE=magicx-zero-28 GUI_RESOLUTION=640x480
make walkthrough       # Offline flow and screen evidence
```

`build/` is ignored; do not commit generated output. Git diff cannot replace `make catalogue-check`. Desktop fixtures do not prove physical controls or native device behavior. See [desktop verification](docs/desktop-verification.md); add `WALK_FLAGS=--install` for network-backed lifecycle flows.

### CLI examples

```sh
./build/knulli-app --help
./build/knulli-app validate catalogue/packages/io.github.unitreign.playtime.json
# On Knulli, with permission to write below /userdata:
./build/knulli-app install package.json
./build/knulli-app adopt package.json       # Manage existing
./build/knulli-app repair package.json
./build/knulli-app update package-v2.json
./build/knulli-app uninstall org.example.package
```

Platform overrides require known facts. `-root` redirects device paths to a trusted fixture root for tests.

## Limits and troubleshooting

- Native GUI behavior, physical controls, and package lifecycle results need their own evidence; earlier reports do not verify every current build or another device.
- Installed apps have no runtime sandbox. Cleanup checks cover declared paths, not every custom configuration.
- The catalogue uses a per-build signing key; independent catalogue updates need a long-lived key strategy. Self-update has a separate GitHub trust boundary.
- Local root access and hostile processes racing filesystem checks remain outside installer protections.
- Removing a package from the catalogue does not remove its files or data. The CLI can uninstall it by its saved package ID.

Inspect `/userdata/system/logs/knulli-app-store.log`; use **More → Y** to export diagnostics under `/userdata/system/knulli-app-store/diagnostics/`. Logs are bounded and redacted, but inspect exports before sharing. Recovery limits are in the [security model](docs/security-model.md).

## Contribute and learn more

- [Contributing](CONTRIBUTING.md) and [manifest review](docs/manifest-review.md)
- [Architecture](docs/architecture.md), [decisions](docs/adr/), and [codebase map](.planning/codebase/ARCHITECTURE.md)
- [Issues](https://github.com/jellydn/knulli-app-store/issues)

Report security issues privately to the owner; do not post credentials, ROMs, or private data.

## License

[MIT](LICENSE) · Copyright © 2026 [Huynh Duc Dung](https://github.com/jellydn). Installed packages have their own licenses.
