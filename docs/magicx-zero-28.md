# MagicX Zero 28 experimental test build

This target is experimental. Knulli source confirms the board and display, but it does not publish the runtime controller identity or button table. A shared A133 family does not establish application or package compatibility.

## Confirmed facts

- Knulli selects `BR2_TARGET_KNULLI_DEVICE_MAGICX_ZERO_28` and writes `magicx-zero-28` to `/boot/boot/knulli.board`: [`Config.in`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/package/system/knulli-system/Config.in#L533-L540) and [`post-image-script.sh`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/board/scripts/post-image-script.sh#L55-L66).
- The A133 board configuration is AArch64 Cortex-A53 with ARMv8 NEON: [`knulli-a133.board`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/configs/knulli-a133.board#L1-L8).
- The device tree identifies `MagicX Zero 28` and configures a 640×480 rotated framebuffer: [`magicx-zero-28.dts`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/package/boot/uboot-a133/magicx-zero-28/boot_package/magicx-zero-28.dts).
- Knulli configgen matches active controllers by GUID and name and creates `SDL_GAMECONTROLLERCONFIG`: [`controller.py`](https://github.com/knulli-cfw/knulli-linux/blob/knulli-main/package/system/knulli-configgen/configgen/configgen/controller.py#L130-L163).

No checked-in Knulli controller profile establishes the MagicX runtime SDL GUID, name, button indices, or generated mapping. The App Store records these values from SDL on the device. It does not invent them from the device tree.

## Install and controller setup

Download the `knulli-app-store-magicx-zero-28-experimental` Actions artifact. Verify `SHA256SUMS.txt`, shut down Knulli, then extract the inner ZIP into `/userdata/roms/ports`. Replace the old App Store files only when updating.

The first launch for each device/controller identity opens a dedicated setup screen before the catalogue. Choose **Use Detected Mapping**, **Test Detected Mapping**, **Customize**, or **Safe Exit**. Customize assigns Up, Down, Left, Right, Confirm, Back, Details/Diagnostics, and Exit one at a time. Every detected button gets an Accept, Retry, Start Over, or Cancel review before progress continues. Conflicts stay on the current action. Custom and tested mappings require a complete preview before atomic save.

Later launches load the saved mapping directly. Open Settings with the physical control shown beside **Settings** to run setup again, export diagnostics, reset the current mapping, or close. Hold the saved Back and Details/Diagnostics controls while launching to force setup. If no SDL GameController exists, a blocked screen shows the log and diagnostic paths and never opens the catalogue.

Mappings use this versioned file:

```text
/userdata/system/configs/knulli-app-store/controller-mappings.json
```

The key combines Knulli device ID and controller GUID. An all-zero or absent GUID falls back to normalized controller name plus device ID. A controller with neither is not persisted. Corrupt files are preserved as `.corrupt` before a replacement is written.

## Package status

- **Grout 5.1.0.0 remains blocked.** Its tagged Knulli release does not establish MagicX orientation, input dispatch, SDL ABI, or readable 640×480 behavior. Its known Zero 28 handling is in a MinUI path that the Knulli launcher does not select.
- **PlayTime 1.0.0 remains blocked.** Its tagged layout adapts to 640×480, but its dynamically linked SDL stack, mandatory accelerated renderer, and controller launch are not proven on MagicX Knulli.

These are package-specific blockers. Successful App Store navigation does not remove them, and neither package is verified.

## Test checklist

1. Record Knulli version, hardware revision, and the displayed SDL controller name/GUID.
2. Confirm the header says `MagicX Zero 28 / 640x480`, with readable letterboxed content and no clipping.
3. Confirm first launch stays on the dedicated screen. Test Use Detected Mapping, then reset and test Customize.
4. Run setup, intentionally create one conflict, retry, assign all actions, and test all actions before save.
5. Restart and confirm the saved mapping loads. Reset it, confirm setup becomes required, and confirm only this controller/device record changes.
6. Disconnect and reconnect the controller if the runtime permits it. Confirm the correct identity and mapping return.
7. Export diagnostics. Confirm controller identity, mapping source, semantic events, and validation failure are present without private data.
8. Confirm Grout and PlayTime show the MagicX device blocker and no install action.
