# TrimUI Smart Pro experimental test build

Release target: v2026.10.27.1. Hardware reports remain scoped to the artifacts actually tested. Device builds remain experimental. The gabagool-backed Library now uses stable rows and a persistent detail pane. The historical [native Option B report](real-device-tests.md#2026-10-25-option-b-library-and-picker-020) records Picker 0.2.0 Install, Update, Repair, preserved state, injected navigation and ES return on a development build from the v2026.10.25 source. It does not verify this redesign, public ZIP or Picker 0.2.2. Physical controls, audio and idle behavior still need hardware checks. Earlier package reports remain separate evidence.

## Established runtime facts

Authoritative Knulli sources establish these facts:

- [`knulli-a133.board`](https://github.com/knulli-cfw/knulli-linux/blob/master/configs/knulli-a133.board) selects `aarch64`, ARMv8, NEON, and Cortex-A53 for the A133 target.
- The [Knulli device page](https://knulli.org/devices/trimui/smart-pro/) identifies Allwinner A133 and PowerVR GE8300.
- Knulli's [device SDL patch](https://github.com/knulli-cfw/knulli-linux/blob/master/board/allwinner/a133/trimui-smart-pro/patches/sdl2/001-add-pvr-ge8300-mali-driver.patch) fixes the Smart Pro SDL surface at 1280×720 and uses a patched EGL/framebuffer backend.
- [`S12trimuiinput`](https://github.com/knulli-cfw/knulli-linux/blob/master/board/allwinner/a133/fsoverlay/etc/init.d/S12trimuiinput) starts `trimui_inputd`, which creates the device controller. Raw event node numbers are not a stable interface.
- Knulli's [`controller.py`](https://github.com/knulli-cfw/knulli-linux/blob/master/package/system/knulli-configgen/configgen/configgen/controller.py) generates `SDL_GAMECONTROLLERCONFIG` from EmulationStation's active mapping.
- Knulli's [`shGenerator.py`](https://github.com/knulli-cfw/knulli-linux/blob/master/package/system/knulli-configgen/configgen/configgen/generators/sh/shGenerator.py) runs Ports entries with `/bin/bash` and exports the generated controller mapping.
- The [game storage guide](https://knulli.org/play/add-games/game-storage/) defines `/userdata/roms/ports` for ports and `/userdata/system` for persistent settings.
- Knulli writes the exact device identifier `trimui-smart-pro` to `/boot/boot/knulli.board`; generic A133 device-tree values are not unique enough for device naming.
- Current Knulli leaves Buildroot's `ID=buildroot` in `/etc/os-release` and appends `OS_NAME="knulli"`, `OS_VERSION`, and `OS_DATE` in [`post-build-script.sh`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/board/scripts/post-build-script.sh#L231-L257). Firmware detection must use the Knulli-owned `OS_NAME`, not `ID`.
- Knulli's [`knulli-system.mk`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/package/system/knulli-system/knulli-system.mk#L88-L102) writes `/usr/share/knulli/knulli.version` as a release identifier followed by build date and time. The detector preserves the full raw line and uses its first whitespace-delimited value, including development suffixes.

The sources do not establish a general third-party glibc contract, raw button numbers, or whether this binary works with the patched PowerVR backend. CI uses Debian Bookworm. Device builds require system SDL2, SDL2_image, and SDL2_ttf; SDL2_gfx is linked statically. The build rejects required glibc versions above 2.34. Knulli's bundled A133 `trimui_inputd` also imports glibc 2.34, but one system binary is evidence, not a published compatibility guarantee. Do not replace Knulli's system libraries. See [the frontend decision](adr/0010-gabagool-knulli-frontend.md) for the runtime contract.

## Download and verify

1. Release target: [v2026.10.27.1](https://github.com/jellydn/knulli-app-store/releases/tag/v2026.10.27.1).
2. Download `knulli-app-store-trimui-smart-pro-experimental-v2026.10.27.1.zip` and `SHA256SUMS.txt` into one directory on your computer.
3. Verify the downloaded archive:

   ```sh
   sha256sum --ignore-missing -c SHA256SUMS.txt
   ```

   Continue only if your ZIP reports `OK`. The checksum file covers both device ZIPs; `--ignore-missing` skips the other target.

For commit-specific testing, a successful `check` run also has a `knulli-app-store-trimui-smart-pro-experimental` artifact. Extract the Actions artifact first, then verify its inner ZIP against its `SHA256SUMS.txt`. Do not install a MagicX build on the Smart Pro.

## Install

1. Shut down Knulli and connect its data card to a computer.
2. Extract the versioned ZIP directly into `/userdata/roms/ports`.
3. Check that `/userdata/roms/ports/Knulli App Store.sh`, `/userdata/roms/ports/knulli-app-store/knulli-app-ui`, and `/userdata/roms/ports/images/Knulli App Store.png` exist.
4. Safely eject the card and start Knulli.
5. Refresh game lists or reboot, then open **Knulli App Store** in **Ports**. The Ports list uses the packaged image after the refresh.

The launcher appends diagnostics to `/userdata/system/logs/knulli-app-store.log`.

## Update an earlier test build

For a build with self-update support at `/userdata/roms/ports/knulli-app-store`, select the **Knulli App Store** entry. Confirm **Check for updates**, then confirm **Update** when offered. Exit and reopen the Store after completion. Checks include official experimental pre-releases. The Store verifies the GitHub asset's exact size and SHA-256; this trusts GitHub and the official publisher account, not an independent update signing key.

Use manual replacement for an older build, a custom location, or a **text file busy** failure:

1. Verify the new artifact checksum, then shut down Knulli.
2. Extract the new versioned ZIP into `/userdata/roms/ports`. Replace all Store release files together, including the launcher, icon, binaries, catalogue, signature, and packaged guide.
3. Safely eject, boot, and launch the app again. Installed-package state under `/userdata/system/knulli-app-store` is not part of the artifact and remains in place.

Do not delete manager state or installed app data. The launcher runs recovery before loading the GUI. A failed rollback retains its journal and snapshots for recovery; an older build may already have deleted that evidence. See [self-update safety and recovery](security-model.md#store-self-update).

## Experimental package operations

On 2026-10-25 (Asia/Singapore), the [owner confirmed](https://ampcode.com/threads/T-01a17039-648c-7024-abe9-8785444a2b5b) seven versions tested on TrimUI Smart Pro only, Knulli AArch64 at 1280×720. That report covers Picker 0.2.0, not current 0.2.2. The other six current [catalogue versions](../README.md#catalogue-and-safety) retain passed package evidence for that matrix only. Picker 0.2.2 is **UNTESTED** on all matrices; MagicX and other matrices remain untested for every package. All seven manifest statuses remain experimental. Separate lifecycle results, physical presses, audio, power-loss behavior, and Store idle-redraw verification are not established by this confirmation. Firmware, architecture, ABI, and required libraries still gate installation. Missing device identity or display data does not block installation. Only matching package-version and device-matrix evidence grants a tested badge.

Grout 5.2.0.0, PlayTime 1.0.0, and upstream RetSend 1.0.0 remain experimental packages. The [current confirmation](real-device-tests.md#2026-10-25-current-owner-confirmation) supersedes the September report and earlier broad report for current tested badges. Select **Install** for a new copy. If the App Store detects files an earlier write did not own, the row shows **EXTERNAL** and the action becomes **Manage existing**. An empty destination directory left by a rolled-back write holds no files to manage, so it stays **Install**. Details explain that this inventories the copy and records safe ownership without reinstalling it. Exact release matches become manager-owned. Changed and unknown files stay unmanaged and are backed up under `/userdata/system/knulli-app-store/originals/<package-id>/` before a later Repair can replace them. State is stored under `/userdata/system/knulli-app-store/installed/`.

Random Game Picker **0.2.2** is published but **UNTESTED** on all device matrices. The 0.2.0 Smart Pro report remains historical exact-version evidence only. Tools and Emulators application systems are now excluded from discovery, selection and Play revalidation. Launcher, settings schema and exact-path ES launch behavior are unchanged. It still requires glibc ≥2.34, firmware SDL2/image/ttf/gfx, and the loopback EmulationStation API. It preserves its entire `state/` directory, including settings and logs. See the [stable picker contract](random-game-picker-go-contract.md) and [0.2.2 release](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.2.2).

The manager restores execute mode only on reviewed paths. PlayTime needs `playtime` and `playtime.sh` to be executable so Knulli can start the launcher and its local binary. Grout needs `Grout.sh` and `grout` for the same reason. RetSend 1.0.0 uses `RetSend.sh` and `tools/retsend/retsend` below its `RetSend` directory. Run Repair on an older Store installation to move the launcher from `tools/Retsend.sh` to `RetSend.sh`. Its [verified staging patch](retsend-upstream-1.0.0.md) retains the existing configuration and default inbox paths. No downloaded script is executed during installation.

PocketCurator, RAOfflineProxy, and retroshelf use the [upstream-managed model](../README.md#upstream-managed-apps). They are actionable experimental packages, not read-only candidates. Close them before Store operations. The Store does not sandbox their runtime writes, downloads, or self-updates. PocketCurator can permanently delete ROMs; back up first. Its corrected launcher has native launch evidence, not a complete lifecycle test. For an older launch-path failure, update the Store to v2026.10.24.4 or later, then use **More → Repair**. See the [device investigation](real-device-tests.md#2026-10-24-pocketcurator-ssh-investigation).

Before removing RAOfflineProxy, stop the proxy, disable autostart, and run `/userdata/system/raofflineproxy/bin/raofflineproxy remove-boot-hook` yourself. Follow any remaining Store cleanup messages for game hooks or proxy settings. Before removing retroshelf, close it and run `/userdata/roms/tools/retroshelf/bin/retroshelf-uninstall.sh --keep-data` yourself. The Store's read-only checks block removal while declared hooks or service state remain. They do not cover every custom path. The Store never runs these cleanup commands. RAOfflineProxy lacks an independent publisher checksum; retroshelf's license is unverified. Read the package warnings before confirmation.

Use **Repair** to re-download, verify, and restore managed files. Use **Uninstall** to remove manager-owned files while preserving declared data. Exact pre-existing release files are removed after adoption; changed package files are restored because their ownership is uncertain. If an operation fails, the transaction rolls back. A retained backup can be restored by copying it back to the path recorded in the installed-state JSON. Do not edit that state by hand while the app is running.

A full health check hashes every immutable managed release file. The GUI caches results by installed state; reopen the Store to recheck changes made outside it. PlayTime's database, generated hook, and other runtime data do not make the install unhealthy. The installed state records each destination file's observed mode because a target filesystem can normalize the requested mode. A wider mode is normal there: the check fails only when a file loses an owner read, write, or execute permission recorded at installation. Select **Issue** to open the health section directly. It shows the exact path, full expected and actual hash or mode, and Repair guidance.

If an operation fails, the Store records its safe retry target under `/userdata/system/knulli-app-store/lifecycle/`. A failed fresh PlayTime install with no external files shows **Retry install**, never **Retry manage**, and a destination that only holds empty directories counts as having no external files. Failed update, Repair, Manage existing, Force reinstall, and Uninstall operations return to their own action after restart only when current state and compatibility still allow it. Each retry repeats all mandatory checks.

- Grout configuration: `/userdata/roms/tools/Grout/config.json`, `save_slots.json`, `.cache/`, and `logs/`.
- PlayTime statistics and configuration: `/userdata/system/configs/playtime/`.
- RetSend configuration, transfer history, and TLS identity: `/userdata/system/configs/retsend/`; received files: `/userdata/roms/retsend-inbox/`.
- App Store log: `/userdata/system/logs/knulli-app-store.log`.

The App Store writes concise UTC timestamped events for startup, platform and catalogue decisions, package actions, download verification, extraction, transactions, backup, rollback, and completion. The active log is capped at 512 KiB and one prior file is retained as `knulli-app-store.log.1`. URLs lose credentials, query strings, and fragments; common token and password fields are redacted.

Resolution selection records every candidate with its source, dimensions, validation result, and rejection reason. The GUI prefers the SDL renderer output, then the SDL window size and current display mode. Valid framebuffer mode data is a fallback; `fb0/virtual_size` is last because it describes an allocation and can be taller than the visible display. Dimensions outside 320×200 through 7680×4320 or outside a 1:2 through 3.5:1 aspect ratio are rejected. With no valid source, resolution remains unknown; this does not block package compatibility. Terminal control sequences from older launchers are removed from new logs and exports.

Press Y for More, then Y to create a diagnostic text bundle under `/userdata/system/knulli-app-store/diagnostics/`. The screen shows the exact output path. Empty catalogues and blocked-controller screens still export directly. The bundle contains detected platform fields, public package IDs and review states, and the bounded redacted logs. It does not include Grout credentials, PlayTime data, ROMs, or private configuration.

Grout has no supported updater-disable setting. The Store therefore changes the reviewed updater metadata URL to a reserved `.invalid` URL during staging and verifies the transformed binary SHA-256. Use only Store update and repair. The manager adds a `gamelist.xml` entry only when the exact launcher path is absent. Existing entries can receive missing name, description, and image fields without changing nonblank metadata or taking ownership. It removes only an unchanged entry that its state proves it created. Shared, pre-existing, and modified entries remain. The GUI combines committed menu changes into one loopback `/reloadgames` request on exit, not one per install. Idle, cancelled and failed-only sessions send none. A refused refresh is logged as restart required without undoing committed operations. Accepted means queued, not completed. One-shot CLI commands still refresh immediately.

## Controls

Every launch opens the Library immediately: **up/down** browse (left/right also work), **A** opens confirmation for the selected row's first allowed action, **B** cancels back, **Y** opens More. More retains all allowed actions and details; Y there exports diagnostics. **L1/R1** filter All, Ready and Installed. **Select+Y** quits. There is no setup wizard or custom mapping editor. Old `/userdata/system/configs/knulli-app-store/controller-mappings.json` files no longer affect controls and are left untouched.

If SDL exposes no GameController, the controller-unavailable screen blocks navigation, automatically exports diagnostics when possible, and shows the log and export paths. Reconnecting resumes the catalogue without setup.

The footer uses circled face-button glyphs with contextual action names and navigation/filter hints; the status area above it shows the quit chord. The launcher selects `STORE_CONTROLLER_LAYOUT=knulli` by default, compensating for Knulli's ES a/b/x/y → SDL b/a/y/x mapping in bindings only. With `/var/run/ports_xbox_layout.flag` present it selects `xbox` and direct SDL letters. SDL events are not swapped again and no system mappings are written. Update the launcher and binary together. On the blocked screen, Confirm also exports diagnostics and Select+Y still quits.

The Library opens on **ALL**, with equal-height rows and a mint outline around focus. A persistent detail pane shows version, separate technical/test and install states, and wrapped summary text; the footer names the primary action. **ALL**, **READY**, and **INSTALLED** filters stay visible, including a way out of an empty filter. One responsive 480-high logical canvas expands with output aspect ratio (minimum width 640). A finished operation reports in a notice bar and clears itself after three seconds; failures remain until answered. Candidates remain read-only and issues still open the health check before actions.

Every screen can be checked on a development machine first, with no handheld and no controller: `make gui` opens the interactive GUI from the keyboard against a scratch fixture root. Arrow keys browse, Enter reviews/confirms, Escape cancels, Y opens More, Page Up/Down filter, and Tab+Y quits. `make walkthrough` records the opening, post-key, and operation-completion frames; each pull request publishes that evidence as the `gui-walkthrough` artifact. Add `WALK_FLAGS=--install` for the network-backed package lifecycle flows. Neither run exercises a real GameController or touches a real `/userdata`, so the checklist below still needs the device. See [Desktop GUI verification](desktop-verification.md).

## Device test checklist

- Record the exact Knulli release and device hardware revision.
- Confirm the app appears in Ports and opens at 1280×720 without replacing system libraries.
- Confirm text, selection, trust state, and package details are readable with no clipping.
- Confirm D-pad navigation, select, back, and exit with Knulli's default Ports layout.
- Confirm circled face labels and contextual footer actions match the active mapping, with Select+Y above the footer. Record the log if no controller appears.
- Confirm up/down (and left/right) browse equal-height Library rows with a mint outline and matching detail pane; L1/R1 filter All, Ready and Installed, including leaving an empty filter. A must open confirmation without writing files; B must return to the selected row. Y must open More, where secondary actions still require confirmation. Check that completion notices do not cover action buttons. Multiple committed menu changes must trigger one refresh on exit, not one per operation.
- Press ES Y for More, then Y to export diagnostics, and inspect the bundle for platform and controller decisions. Repeat the face-button checks in both default and Ports Xbox-layout modes. Do not send a bundle if manual inspection finds private data.
- Confirm the header shows `TRIMUI SMART PRO / 1280X720`. Unknown boards must show `UNKNOWN DEVICE`, and failed runtime-size detection must identify its fallback.
- Confirm the six unchanged package versions show `DEVICE TESTED` only for the owner-confirmed Smart Pro, Knulli AArch64, 1280×720 matrix. Picker 0.2.2 must show `UNTESTED` on every matrix; other matrices show `UNTESTED` for all packages without matching evidence. RAOfflineProxy, PocketCurator, and retroshelf must offer Install or Manage existing when runtime checks pass, with upstream-managed warnings. Missing device or display evidence must not block these actions.
- Confirm returning to EmulationStation works and a second launch also works.
- Confirm Wi-Fi disabled and enabled produce the same catalogue because this artifact reads its bundled index.
- Send the log, firmware version, observed controls, and a photo or screenshot with the result. Do not include credentials or private network data.

After this checklist passes, add linked `real-device-test` evidence for the GUI target. Package manifests still need their own package-specific hardware tests before any verified badge.

### PlayTime 1.0.0

1. Install, refresh game lists or reboot, and launch PlayTime.
2. Confirm the launcher and statistics function, then close and reopen it.
3. Change one statistic, run Repair, and confirm the statistic remains.
4. Run Uninstall and confirm managed files are gone while `/userdata/system/configs/playtime/` remains.
5. If PlayTime existed before this test, use **Manage existing** and confirm its statistics remain.
6. Cause a download or checksum failure before writes. Restart the Store and confirm **Retry install** is shown while the destination is absent.

### Grout 5.2.0.0

1. Install, refresh game lists or reboot, and launch Grout. Confirm no self-update action can obtain metadata.
2. Connect to a test RomM server and confirm one core browse or sync operation. Do not include credentials in the report.
3. Run Repair and confirm `config.json`, `save_slots.json`, `.cache/`, and `logs/` remain.
4. Run Uninstall and confirm managed files are gone while the preserved paths remain.
5. If Grout existed before this test, use **Manage existing** and confirm credentials and configuration remain.

6. Test update from 5.1.0.0 and confirm all four preserved paths remain. Test a failed update and confirm rollback restores 5.1.0.0 exactly.
7. Cause a disposable **Manage existing** failure. Confirm the recovery screen shows the exact reason, **Retry manage**, **Force reinstall**, diagnostics, and cancel. Complete both force confirmations and inspect the timestamped backup manifest.
8. Select any **Issue** state and confirm Details opens the health section with the exact path, expected and actual value, and Repair guidance. After Repair, confirm no false mode issue remains on the device filesystem.

Report install, launch, core function, update, rollback, Repair, Uninstall, adoption, and force-reinstall results separately for each package. A passing Smart Pro result does not verify another device.

## Firmware-fix reproduction

1. Update the App Store files from the new artifact and start it on current Knulli.
2. Select Grout or PlayTime. Confirm compatibility says the Knulli identity came from `/etc/os-release:OS_NAME` and calls the decision experimental.
3. Confirm **Install** or **Manage existing** is available when the header shows TrimUI Smart Pro and runtime 1280×720. The diagnostic log should select `SDL renderer output`; any `1280x13107` display or virtual-framebuffer candidate must show `valid=false` with a rejection reason.
4. Press Y for More, then Y to export diagnostics and inspect the platform line. It should show raw firmware `knulli`, normalized firmware `knulli`, source `/etc/os-release:OS_NAME`, and the current release identifier from `/usr/share/knulli/knulli.version`.
5. If an action remains unavailable, send the diagnostic bundle after checking it for private data. Unknown firmware must remain blocked.
