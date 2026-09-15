# TrimUI Smart Pro experimental test build

This artifact is for real-device validation. It is not a statement that Knulli App Store supports the TrimUI Smart Pro.

## Established runtime facts

Authoritative Knulli sources establish these facts:

- [`knulli-a133.board`](https://github.com/knulli-cfw/knulli-linux/blob/master/configs/knulli-a133.board) selects `aarch64`, ARMv8, NEON, and Cortex-A53 for the A133 target.
- The [Knulli device page](https://knulli.org/devices/trimui/smart-pro/) identifies Allwinner A133 and PowerVR GE8300.
- Knulli's [device SDL patch](https://github.com/knulli-cfw/knulli-linux/blob/master/board/allwinner/a133/trimui-smart-pro/patches/sdl2/001-add-pvr-ge8300-mali-driver.patch) fixes the Smart Pro SDL surface at 1280×720 and uses a patched EGL/framebuffer backend.
- [`S12trimuiinput`](https://github.com/knulli-cfw/knulli-linux/blob/master/board/allwinner/a133/fsoverlay/etc/init.d/S12trimuiinput) starts `trimui_inputd`, which creates the device controller. Raw event node numbers are not a stable interface.
- Knulli's [`controller.py`](https://github.com/knulli-cfw/knulli-linux/blob/master/package/system/knulli-configgen/configgen/configgen/controller.py) generates `SDL_GAMECONTROLLERCONFIG` from EmulationStation's active mapping.
- Knulli's [`shGenerator.py`](https://github.com/knulli-cfw/knulli-linux/blob/master/package/system/knulli-configgen/configgen/configgen/generators/sh/shGenerator.py) runs Ports entries with `/bin/bash` and exports the generated controller mapping.
- The [game storage guide](https://knulli.org/play/add-games/game-storage/) defines `/userdata/roms/ports` for ports and `/userdata/system` for persistent settings.

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
- Confirm all five candidates show `READ ONLY - REVIEW REQUIRED` and cannot start an install.
- Confirm returning to EmulationStation works and a second launch also works.
- Confirm Wi-Fi disabled and enabled produce the same catalogue because this artifact reads its bundled index.
- Send the log, firmware version, observed controls, and a photo or screenshot with the result. Do not include credentials or private network data.

After this checklist passes, add linked `real-device-test` evidence for the GUI target. Package manifests still need their own package-specific hardware tests before any verified badge.
