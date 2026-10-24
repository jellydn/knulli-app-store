# MagicX Zero 28 experimental test build

This target is experimental. Knulli source confirms the board and display. A real-device diagnostic also confirms Knulli Scarab, `aarch64`, 640×480, and SDL GameController name `magicx-input` with a nonzero GUID. The physical mapping remains device data. A shared A133 family alone does not establish application or package compatibility.

## Confirmed facts

- Knulli selects `BR2_TARGET_KNULLI_DEVICE_MAGICX_ZERO_28` and writes `magicx-zero-28` to `/boot/boot/knulli.board`: [`Config.in`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/package/system/knulli-system/Config.in#L533-L540) and [`post-image-script.sh`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/board/scripts/post-image-script.sh#L55-L66).
- The A133 board configuration is AArch64 Cortex-A53 with ARMv8 NEON: [`knulli-a133.board`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/configs/knulli-a133.board#L1-L8).
- The device tree identifies `MagicX Zero 28` and configures a 640×480 rotated framebuffer: [`magicx-zero-28.dts`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/package/boot/uboot-a133/magicx-zero-28/boot_package/magicx-zero-28.dts).
- Knulli configgen matches active controllers by GUID and name and creates `SDL_GAMECONTROLLERCONFIG`: [`controller.py`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/package/system/knulli-configgen/configgen/configgen/controller.py#L130-L163).

No checked-in Knulli controller profile establishes the MagicX button indices. The App Store records the runtime GUID, name, and mapping from SDL on the device. It does not invent them from the device tree.

## Install and automatic controls

Download the `knulli-app-store-magicx-zero-28-experimental` Actions artifact. Verify `SHA256SUMS.txt`, shut down Knulli, then extract the inner ZIP into `/userdata/roms/ports`. Replace the old App Store files only when updating. Confirm `/userdata/roms/ports/images/Knulli App Store.png` is present; the Ports list uses it after a game list refresh.

Every launch opens one app card immediately: **left/right** browse (up/down also work), **A** opens confirmation for its first allowed action, **B** cancels back, and **Y** opens More. More retains all permitted actions and details; Y there exports diagnostics. **L1/R1** filter All, Ready and Installed. **Select+Y** quits. There is no setup, calibration, mapping editor, or saved-assignment override.

If no SDL GameController exists, a blocked screen shows the log and diagnostic paths. Reconnecting resumes normal controls immediately and clears held chord state. Old `/userdata/system/configs/knulli-app-store/controller-mappings.json` files are ignored and left untouched, even when malformed.

The card footer names available controls with ES letters: `Browse (DPAD LEFT/DPAD RIGHT)  Confirm (A)  More (Y)  Quit (SELECT + Y)`. The launcher selects `STORE_CONTROLLER_LAYOUT=knulli` normally or `xbox` when `/var/run/ports_xbox_layout.flag` exists. Knulli maps ES a/b/x/y → SDL b/a/y/x normally and uses direct letters in Xbox mode. Bindings compensate once; SDL events are unchanged. Update the binary and launcher together. No guessed raw indices or system mapping writes are used. On the blocked screen, Confirm also exports diagnostics and Select+Y still quits.

The catalogue opens on **ALL**, showing one app, version, technical/test status, install state, summary and primary action. Its counter shows the position in the active filter. Shoulders switch **ALL**, **READY**, and **INSTALLED**, including leaving an empty filter. A finished operation reports in a notice bar and clears after three seconds; failures remain until answered. Candidates remain read-only and issues still open the health check before actions.

Every screen can be checked on a development machine first, with no handheld and no controller: `make gui GUI_DEVICE=magicx-zero-28 GUI_RESOLUTION=640x480` opens the interactive GUI from the keyboard against a scratch fixture root. `make walkthrough WALK_DEVICE=magicx-zero-28 WALK_RESOLUTION=640x480` replays the offline flows and records the opening, post-key, and operation-completion frames. Add `WALK_FLAGS=--install` for the network-backed package lifecycle flows. Neither run exercises a real GameController or touches a real `/userdata`. See [Desktop GUI verification](desktop-verification.md).

## Recording a device control report

Automatic defaults follow Knulli's logical SDL contract, not a device-specific raw fallback. A desktop walkthrough does not prove physical navigation. Record firmware, hardware revision, controller name/GUID, Ports layout mode, footer, and actual ES buttons for confirm, back, More, navigation, filtering and Select+Y quit. Repeat in default and Xbox modes. Export diagnostics with Y inside More, inspect for private data, and cite it in `docs/real-device-tests.md`. Do not add guessed raw button indices.

## Package status

- **Grout 5.2.0.0 is a user-authorized experimental test.** Knulli, AArch64, glibc 2.17 or later, and the declared SDL libraries must pass detection. Device identity and display size do not block installation. The Store disables its self-updater with an exact verified staging patch. No Grout version has MagicX real-device evidence.
- **PlayTime 1.0.0 is a user-authorized experimental test.** The reviewed ARM64 archive uses the Knulli SDL2 libraries, runtime display dimensions with a 640×480 fallback, and SDL GameController actions. Select **Install**, or **Manage existing** when detected external files show the row state **EXTERNAL**, and confirm the unverified warning. An empty destination directory is not a copy and keeps the **Install** action. Its accelerated renderer, launch, tracking, and complete life cycle are not proven on MagicX.

- **RetSend 1.0.0 is a user-authorized experimental test.** The [upstream Knulli package](retsend-upstream-1.0.0.md) requires Knulli, AArch64, glibc 2.28 or later, and system SDL2. Device identity and display size do not block installation. Store staging keeps configuration, transfer history, and TLS identity in `/userdata/system/configs/retsend`, and the default receive directory in `/userdata/roms/retsend-inbox`. Saved or selected routes can override that default. No RetSend version has MagicX real-device evidence.

Successful App Store navigation does not prove a package. Grout, PlayTime, and RetSend remain unverified on MagicX.

## Test checklist

1. Record Knulli version, hardware revision, and SDL controller name/GUID. Follow [Recording a device control report](#recording-a-device-control-report).
2. Confirm the header says `MagicX Zero 28 / 640x480`, with readable letterboxed content and no clipping.
3. Confirm the first launch opens the catalogue with no setup.
4. Check left/right card browsing, A review/confirm, B cancel, Y More, L1/R1 filtering and Select+Y in both default Knulli and Ports Xbox modes. X has no new feature. First A must not write files. B from direct confirmation must return to the card; secondary actions inside More still require confirmation.
5. Restart with an old custom mapping file present. Confirm defaults remain active and the file stays unchanged.
6. Disconnect and reconnect the controller if the runtime permits it. Confirm the correct identity and mapping return.
7. Press Y for More, then Y to export diagnostics. Confirm controller identity, mapping source, and semantic events are present without private data.
8. Confirm packaged apps show **Install** or **Manage existing** with an **UNTESTED** badge when no exact device report exists. Device/display scope must not block the action. Remove one runtime dependency only in a disposable test root and confirm installation is blocked with its exact name.
9. Install PlayTime, refresh game lists or reboot, and launch it. Track a game, close it, and relaunch it.
10. Confirm the App Store reports PlayTime healthy. Damage only a disposable managed test copy if you test Repair; confirm Repair restores it and keeps `/userdata/system/configs/playtime/`.
11. Uninstall PlayTime and confirm its managed files are gone while `/userdata/system/configs/playtime/` remains. If you managed an external copy and later repaired it, confirm uncertain original files are restored from `/userdata/system/knulli-app-store/originals/io.github.unitreign.playtime/`.
12. Cause a failed fresh install before destination writes, restart the Store, and confirm **Retry install** appears. Confirm **Manage existing** is absent when no external files exist, including when rollback left empty directories behind.

### Grout 5.2.0.0 checklist

1. Install and launch Grout at 640×480. Check all text, focus, confirm, back, and exit controls.
2. Connect to a test RomM 5.2.0 server and complete one browse or sync action.
3. Confirm the self-updater cannot obtain release metadata.
4. Run Repair and confirm `config.json`, `save_slots.json`, `.cache/`, and `logs/` remain.
5. Update a managed 5.1.0.0 test copy. Confirm the preserved data and a safe game-list refresh.
6. Test rollback and Uninstall. Confirm preserved data remains.
7. Cause a disposable adoption failure, use both force-reinstall confirmations, and inspect `/userdata/system/knulli-app-store/recovery-backups/app.romm.grout/<timestamp>/manifest.json`.
8. If Grout shows **Issue**, select it and confirm the health section gives the exact path, expected and actual hash or mode, and Repair guidance. Repair and confirm the transformed binary is healthy and the effective destination mode is accepted. A destination mode wider than the requested one, for example `0777` in place of `0755`, must not produce an issue; removing the execute bit must.

The App Store log is `/userdata/system/logs/knulli-app-store.log`. Diagnostic exports are under `/userdata/system/knulli-app-store/diagnostics/`. Installed state and original-file backups are under `/userdata/system/knulli-app-store/`. If a test fails, stop the app, inspect the installed-state JSON, and restore a retained original to its recorded path. Do not send PlayTime data in a report.
