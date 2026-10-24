# Random Game Picker — experimental review, 2026-10-19

The owner requested a public first release and App Store inclusion for handheld testing. [Issue 62](https://github.com/jellydn/knulli-app-store/issues/62) records the intake. This is maintainer provenance, not community or hardware verification.

## Exact release and archive

- Project: [jellydn/knulli-random-game-picker](https://github.com/jellydn/knulli-random-game-picker).
- Version: [0.1.0-alpha.1](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.1.0-alpha.1), published by the successful [release workflow](https://github.com/jellydn/knulli-random-game-picker/actions/runs/39193670740).
- Asset: `random-game-picker-0.1.0-alpha.1-knulli-aarch64.zip`, GitHub asset ID 655107287.
- SHA-256: `b37fbfcea1d38345f26a9a65995a5fc2bfbe8ae3f2267265bad947c694f66607`.
- Compressed size: 1,088,216 bytes. Installed regular-file size: 2,818,747 bytes.
- Independently downloaded ZIP and publisher `SHA256SUMS.txt` agree. Versioned URL, exact size and digest pin the accepted bytes; GitHub asset replacement cannot silently change an installation.

All ten archive entries are regular files. Inspection rejected duplicate names, absolute paths, traversal and non-regular modes. The inventory is `RandomGamePicker.sh`, `random-game-picker`, `LICENSE`, `README.md`, `THIRD_PARTY.md`, `assets/DejaVuSans.ttf`, and four dependency notices under `licenses/`. Only the launcher and binary need execute mode. There is no archive strip prefix or bundled state directory.

Project license is MIT. The font, pugixml, JSON headers and static C++ runtime retain their notices, including the GCC runtime exception. SDL libraries are system dependencies, not replaced or bundled.

`aarch64-linux-gnu-readelf -l -d --version-info random-game-picker` on the downloaded payload reports ELF64 AArch64, interpreter `/lib/ld-linux-aarch64.so.1`, and highest `GLIBC_2.36`. NEEDED entries are `libSDL2-2.0.so.0`, `libSDL2_image-2.0.so.0`, `libSDL2_ttf-2.0.so.0`, `libm.so.6`, `libc.so.6`, and `ld-linux-aarch64.so.1`. libstdc++, libgcc and pugixml are static. The manifest requires glibc 2.36 and detected SDL2/image/ttf/libc. This inspection is not runtime ABI proof on a device.

## Paths, network and launch

The Store installs only `/userdata/roms/ports/RandomGamePicker`, with one exact menu entry `./RandomGamePicker/RandomGamePicker.sh` in the Ports gamelist. It does not own the shared Ports directory or other ports. Users with a nonstandard configured Ports root must use manual installation in that root instead.

The launcher resolves its own location and sets `PICKER_STATE_DIR` to its `state` subdirectory. Runtime writes are settings JSON, its temporary replacement and a replaced per-session `last-run.log`. `state` is preserved across lifecycle operations. No remote code, updater, shell-built emulator command or ROM download is present. The picker reads ES configuration, gamelists, ROM existence and scraped artwork without modifying them. An emulator launched by ES retains its normal save/history behavior.

Network is declared because the app uses sockets, but the only endpoint is fixed loopback `127.0.0.1:1234`. Requests have bounded size and time, without DNS, redirects or retries. Live games are reconciled with ES; Ports, collections, missing/hidden games and ambiguous paths are excluded. Play revalidates the catalogue, closes SDL/input, sends one raw-path POST to ES, then exits. HTTP acceptance is not proof of play. Whether ES processes that request correctly after a running Port exits is still an open hardware gate. No nested configgen fallback hides a failed handoff.

## Validation and limits

Desktop tests exercise the actual SDL application at 1280×720 and 640×480: PNG/JPEG artwork, corrupt-art fallback, filters, 100 rerolls at each size, empty recent pool/reset, and window teardown before one exact fake-ES POST containing spaces, quotes and shell metacharacters. A supplemental capture waits for rendered pixels; an early first-frame CI image was blank and is not visual evidence. None of these tests start an emulator or prove physical controller mapping.

The Store package-specific fixture covers both declared devices: rejecting older glibc, install, rejected repeated install, repair, update, health, uninstall, preserved settings/logs, other-port/menu preservation, and rollback on a malformed shared gamelist. It uses synthetic payload bytes with adjusted digests; a separate local archive check verifies the published bytes. A temporary validation test also installed, checked health, repaired and uninstalled the actual downloaded ZIP through the real `installer.Manager`, with the original manifest digest and sizes unchanged; settings survived. It passed and was removed, so permanent tests remain offline. The AArch64 binary was not executed. Existing archive/security tests remain unchanged.

Local commands passed: `make check`, `go test -tags sdl ./...`, `make catalogue`, `make catalogue-check`, `make cover`, and `staticcheck` with and without `-tags sdl`. Coverage was 84.2%, above the 70% floor. Two catalogue builds matched byte for byte. Device CI rebuilds and signs the catalogue as usual; it does not modify or rebuild the pinned picker ZIP.

The entry stays `experimental`, limited to `trimui-smart-pro` and `magicx-zero-28` with explicit 1280×720 and 640×480 resolutions. No real-device report is invented. Hardware launch, ES return/history/Quick Resume, physical controls, audio/display restoration, firmware ABI and 10,000-game performance remain unverified. Persistent indexing and Surprise Mode are absent in this alpha. Follow the upstream [device report checklist](https://github.com/jellydn/knulli-random-game-picker/blob/v0.1.0-alpha.1/docs/testing.md) before any promotion.

## Alpha.2 — TSP button correction

The owner tested alpha.1 on TSP and reported A/B and X/Y reversed. The reviewed KNULLI configgen source explains the failure: default ES `a/b/x/y` becomes SDL `b/a/y/x`; the picker dispatched SDL names directly. Alpha.2 translates logical SDL events back to ES labels, without raw device indices. The launcher reads the existing `/var/run/ports_xbox_layout.flag`, which configgen also uses, to keep Xbox-layout mode unswapped. It never creates or modifies that flag. Direct desktop execution without the Port launcher retains SDL labels.

The manifest now pins [alpha.2](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.1.0-alpha.2): SHA-256 `92142b1cfcd05aaf5ff9fc9cf3e7bca571645c2fac1281cb0e4712fe7587a219`, 1,088,619 compressed bytes and 2,820,223 installed bytes. The downloaded ZIP agrees with the publisher checksum. All ten entries, modes, licenses, interpreter, NEEDED entries and glibc floor were inspected again. Paths, network behavior and preserved state are unchanged. Update both `RandomGamePicker.sh` and the binary; replacing only the binary does not select the corrected KNULLI mode.

The upstream [release CI](https://github.com/jellydn/knulli-random-game-picker/actions/runs/39195463092) passed. A new SDL controller-event regression fails on unchanged alpha.1 because ES A exits, then passes on alpha.2 for A pick/play, B back, X reroll without rescanning, and Y rescan in **both** default KNULLI and Xbox-layout modes. Existing keyboard, two-size, artwork, 100-reroll and single-submit tests also passed. This is synthetic SDL input, not a physical-controller driver test. The owner's report proves the observed alpha.1 issue, not the corrected build or full launch matrix. Hardware retest remains required and the entry stays experimental.

The full local Store checks passed again, including race/SDL/static checks, 84.2% coverage and catalogue determinism. A temporary real-asset validation installed the published alpha.1, updated to the published alpha.2 with exact original digests/sizes, confirmed the corrected launcher and healthy binary inventory, then repaired and uninstalled while preserving alpha.1 settings. It passed and was removed; no AArch64 binary was executed in the fixture.

## Stable 0.1.0 — owner-confirmed TSP retest

The owner confirmed that the corrected alpha.2 works on TSP and requested the first stable release. Upstream [0.1.0](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.1.0) is now a non-prerelease, published after successful host/device tests by [release CI](https://github.com/jellydn/knulli-random-game-picker/actions/runs/39197423071). This is stable release status for the current feature set, not completion of the full PRD or an independent device-verification report. The exact firmware, full launch/return matrix, MagicX and large-library performance remain unverified. The Store entry therefore stays `experimental`.

The downloaded stable ZIP matches publisher `SHA256SUMS.txt`: SHA-256 `ab7e8fb9da6702fca8befd1fff7df60f32fb0ac6a5ca6439181e0d86bd30a5e2`, 1,088,560 compressed bytes and 2,820,305 installed bytes. All ten unique archive entries are safe relative regular files; inventory and license layout are unchanged. Inspection of the downloaded binary confirms the same AArch64 interpreter, dependencies and glibc 2.36 floor. No install path, permission, network endpoint or state-preservation rule was expanded.

A temporary validation used the real Store CLI and the unchanged published alpha.2 and stable manifests/assets in separate TSP and MagicX filesystem fixtures. Install alpha.2 → update stable → damage/repair binary → uninstall passed on both fixtures. Exact stable binary bytes were checked after update and repair; user settings survived each step and uninstall removed the managed binary/launcher. ES was absent, so the CLI correctly reported restart required rather than a successful menu reload. This validates package lifecycle, not execution of an AArch64 application or real controller/emulator behavior.

## Stable 0.1.1 — selector-first startup

The owner requested the next release and a simpler Store flow. Upstream [0.1.1](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.1.1) opens directly on a random game after discovery, with saved system/favorites filters. A plays, X rerolls, Y opens settings and Y inside settings rescans. B back/quit and Start quit are unchanged. It also fixes the saved system being reset before discovery. The earlier alpha.2 TSP report does not verify this version; the manifest stays experimental without a new device-tested flag.

The published ZIP and publisher checksum agree: SHA-256 `2e71a068eed7c2f1cdecef95ccd2ec3342c98d57e7b94c14edcd6415182abbb3`, 1,089,242 compressed bytes and 2,821,654 installed regular-file bytes. Inventory, installation paths, dependencies, network endpoints and preserved state are unchanged. The [release workflow](https://github.com/jellydn/knulli-random-game-picker/actions/runs/39218814428) passed host, device and publication jobs. Both SDL layouts and screen sizes cover startup Play, Y settings/rescan, saved filters, empty/failed/delayed discovery, 100 rerolls per size and single-submit handoff. These are fixture checks, not handheld or emulator execution proof.

All seven Store packages remain visible. Five reviewed experimental packages are testable when compatibility checks pass. PlayTime 1.0.0 has the existing exact TSP device-tested evidence; the Grout report is for older 5.1.0.0, not current 5.2.0.0. PocketCurator and RAOfflineProxy remain read-only because their package safety review is incomplete, not merely because a device-tested badge is absent. This catalogue update does not change that policy or implement a new Store UI.

Real published 0.1.0 install → 0.1.1 update → damaged-binary repair → uninstall passed through the Store CLI on both device filesystem fixtures. Exact updated/repaired binary and launcher bytes matched the published archive; saved filters, logs and an unrelated Port/menu entry survived. No AArch64 binary was executed. Archive entry modes and the downloaded ELF dependencies/glibc floor were inspected again. Local race, SDL, catalogue determinism and both static checks passed; coverage was 84.3%. The catalogue summary and warning begin with short text so the current two-line panels show the purpose and untested status without an incomplete negative statement.
