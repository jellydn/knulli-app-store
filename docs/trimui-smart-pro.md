# TrimUI Smart Pro experimental test build

The GUI was reported functional on one TrimUI Smart Pro. Builds remain experimental while the exact Knulli release, hardware revision, and repeatable test evidence are collected. Package compatibility is reviewed separately.

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

The sources do not establish the general SDL2 package version, a third-party glibc contract, raw button numbers, or whether this binary works with the patched PowerVR backend. CI uses Debian Bookworm. The binary needs `libSDL2-2.0.so.0` and glibc symbols through 2.34. Knulli's bundled A133 `trimui_inputd` also imports glibc 2.34, but one system binary is evidence, not a published compatibility guarantee.

## Download and verify

1. Open the successful `check` workflow run for the commit to test.
2. Download the `knulli-app-store-trimui-smart-pro-experimental` artifact.
3. Extract the Actions artifact on a computer. It contains one versioned ZIP and `SHA256SUMS.txt`.
4. Verify it:

   ```sh
   sha256sum -c SHA256SUMS.txt
   ```

## Install

1. Shut down Knulli and connect its data card to a computer.
2. Extract the versioned ZIP directly into `/userdata/roms/ports`.
3. Check that `/userdata/roms/ports/Knulli App Store.sh` and `/userdata/roms/ports/knulli-app-store/knulli-app-ui` exist.
4. Safely eject the card and start Knulli.
5. Refresh game lists or reboot, then open **Knulli App Store** in **Ports**.

The launcher appends diagnostics to `/userdata/system/logs/knulli-app-store.log`.

## Update an earlier test build

1. Verify the new artifact checksum, then shut down Knulli.
2. Extract the new versioned ZIP into `/userdata/roms/ports` and allow it to replace the old launcher and `knulli-app-store` files.
3. Safely eject, boot, and launch the app again. Installed-package state under `/userdata/system/knulli-app-store` is not part of the artifact and remains in place.

## Experimental package operations

Only Grout 5.1.0.0 and PlayTime 1.0.0 are actionable. Grout is verified for this Smart Pro matrix. PlayTime remains experimental because MagicX is untested. Select **Install** for a new copy. If the App Store detects an external copy, the row shows **EXTERNAL** and the action becomes **Manage existing**. Details explain that this inventories the copy and records safe ownership without reinstalling it. Exact release matches become manager-owned. Changed and unknown files stay unmanaged and are backed up under `/userdata/system/knulli-app-store/originals/<package-id>/` before a later Repair can replace them. State is stored under `/userdata/system/knulli-app-store/installed/`.

The manager restores execute mode only on reviewed paths. PlayTime needs `playtime` and `playtime.sh` to be executable so Knulli can start the launcher and its local binary. Grout needs `Grout.sh` and `grout` for the same reason. No downloaded script is executed during installation.

Use **Repair** to re-download, verify, and restore managed files. Use **Uninstall** to remove manager-owned files while preserving declared data. Exact pre-existing release files are removed after adoption; changed package files are restored because their ownership is uncertain. If an operation fails, the transaction rolls back. A retained backup can be restored by copying it back to the path recorded in the installed-state JSON. Do not edit that state by hand while the app is running.

Health checks compare only immutable managed release files. PlayTime's database, generated hook, and other runtime data do not make the install unhealthy. The installed state records each destination file's observed mode because a target filesystem can normalize the requested mode. If Repair appears, the details panel and log identify every missing, changed, or mode-mismatched managed path.

- Grout configuration: `/userdata/roms/tools/Grout/config.json`, `save_slots.json`, `.cache/`, and `logs/`.
- PlayTime statistics and configuration: `/userdata/system/configs/playtime/`.
- App Store log: `/userdata/system/logs/knulli-app-store.log`.

The App Store writes concise UTC timestamped events for startup, platform and catalogue decisions, package actions, download verification, extraction, transactions, backup, rollback, and completion. The active log is capped at 512 KiB and one prior file is retained as `knulli-app-store.log.1`. URLs lose credentials, query strings, and fragments; common token and password fields are redacted.

Resolution selection records every candidate with its source, dimensions, validation result, and rejection reason. The GUI prefers the SDL renderer output, then the SDL window size and current display mode. Valid framebuffer mode data is a fallback; `fb0/virtual_size` is last because it describes an allocation and can be taller than the visible display. Dimensions outside 320×200 through 7680×4320 or outside a 1:2 through 3.5:1 aspect ratio are rejected. No valid source means compatibility remains blocked. Terminal control sequences from older launchers are removed from new logs and exports.

Use Settings > Export Diagnostics to create a text bundle under `/userdata/system/knulli-app-store/diagnostics/`. The screen shows the exact output path. The bundle contains detected platform fields, public package IDs and review states, and the bounded redacted logs. It does not include Grout credentials, PlayTime data, ROMs, or private configuration.

Do not use Grout's built-in updater during this test. It has no supported disable setting and operates outside App Store rollback. Future Grout updates must use a newly reviewed manifest. The manager adds a `gamelist.xml` entry only when the exact launcher path is absent. It removes only an unchanged entry that its state proves it created. Shared, pre-existing, and modified entries remain. After a committed change, it asks Knulli's loopback `/reloadgames` endpoint to queue a refresh. If Knulli does not accept the request, the GUI says **Restart required**; it never claims that a queued refresh completed.

## Controls

The first launch for each device/controller identity opens a dedicated setup screen before the catalogue. Choose **Use Detected Mapping**, **Test Detected Mapping**, **Customize**, or **Safe Exit**. Customize assigns one semantic action at a time, then reviews each detected button with Accept, Retry, Start Over, and Cancel choices. All custom and tested mappings require an eight-action preview before atomic save. Later launches load the saved mapping directly. Settings can reopen setup, export diagnostics, or reset the active mapping. Hold the saved Back and Details/Diagnostics controls while launching for a deliberate setup override. Physical button positions are not assumed.

If SDL exposes no GameController, the setup screen stays blocked, automatically exports diagnostics when possible, and shows the log and export paths. It does not open the catalogue with unknown controls.

## Device test checklist

- Record the exact Knulli release and device hardware revision.
- Confirm the app appears in Ports and opens at 1280×720 without replacing system libraries.
- Confirm text, selection, trust state, and package details are readable with no clipping.
- Confirm D-pad navigation, select, back, and exit with Knulli's default Ports layout.
- Confirm the footer shows the active semantic controls and mapping source; record the log if no controller appears.
- Open Settings with its displayed physical label, export diagnostics, and inspect the bundle for platform and mapping decisions. Do not send it if manual inspection finds private data.
- Confirm the header shows `TRIMUI SMART PRO / 1280X720`. Unknown boards must show `UNKNOWN DEVICE`, and failed runtime-size detection must identify its fallback.
- Confirm Grout shows `VERIFIED`, PlayTime shows its Smart Pro test state, and the other two packages remain read-only.
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

### Grout 5.1.0.0

1. Install, refresh game lists or reboot, and launch Grout without using its updater.
2. Connect to a test RomM server and confirm one core browse or sync operation. Do not include credentials in the report.
3. Run Repair and confirm `config.json`, `save_slots.json`, `.cache/`, and `logs/` remain.
4. Run Uninstall and confirm managed files are gone while the preserved paths remain.
5. If Grout existed before this test, use **Manage existing** and confirm credentials and configuration remain.

Report install, launch, core function, Repair, Uninstall, and adoption results separately for each package. The current report says both package tests were good but does not itemize these steps. PlayTime stays experimental until its MagicX matrix is tested.

## Firmware-fix reproduction

1. Update the App Store files from the new artifact and start it on current Knulli.
2. Select Grout or PlayTime. Confirm compatibility says the Knulli identity came from `/etc/os-release:OS_NAME` and calls the decision experimental.
3. Confirm **Install** or **Manage existing** is available when the header shows TrimUI Smart Pro and runtime 1280×720. The diagnostic log should select `SDL renderer output`; any `1280x13107` display or virtual-framebuffer candidate must show `valid=false` with a rejection reason.
4. Export diagnostics from Settings and inspect the platform line. It should show raw firmware `knulli`, normalized firmware `knulli`, source `/etc/os-release:OS_NAME`, and the current release identifier from `/usr/share/knulli/knulli.version`.
5. If an action remains unavailable, send the diagnostic bundle after checking it for private data. Unknown firmware must remain blocked.
