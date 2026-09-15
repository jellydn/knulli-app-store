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

## Controls

- D-pad: move through packages or available actions.
- SDL A: select and confirm.
- SDL B: back and cancel.

With Knulli's default Ports layout, physical B (south) maps to SDL A and physical A (east) maps to SDL B. If the per-game Xbox layout is enabled, the physical labels change. The footer reports whether SDL accepted a controller mapping.

## Device test checklist

- Record the exact Knulli release and device hardware revision.
- Confirm the app appears in Ports and opens at 1280×720 without replacing system libraries.
- Confirm text, selection, trust state, and package details are readable with no clipping.
- Confirm D-pad navigation, select, back, and exit with Knulli's default Ports layout.
- Confirm the footer says `CONTROLLER: READY`; record the log if it does not.
- Confirm the header shows `TRIMUI SMART PRO / 1280X720`. Unknown boards must show `UNKNOWN DEVICE`, and failed runtime-size detection must identify its fallback.
- Confirm all six candidates show `READ ONLY - REVIEW REQUIRED` and cannot start an install.
- Confirm EmuDrop shows the ROM copyright warning and EmuDrop, PlayTime, and Grout show `APPROVED / CANDIDATE`.
- Confirm returning to EmulationStation works and a second launch also works.
- Confirm Wi-Fi disabled and enabled produce the same catalogue because this artifact reads its bundled index.
- Send the log, firmware version, observed controls, and a photo or screenshot with the result. Do not include credentials or private network data.

After this checklist passes, add linked `real-device-test` evidence for the GUI target. Package manifests still need their own package-specific hardware tests before any verified badge.
