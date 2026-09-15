KNULLI APP STORE - MAGICX ZERO 28 EXPERIMENTAL BUILD

Knulli source identifies this target as magicx-zero-28, AArch64 Cortex-A53,
with a 640x480 display. The App Store and controller setup require real-device
testing. The runtime controller GUID, name, and raw buttons are not published.

INSTALL OR UPDATE

1. On a computer, verify the ZIP with: sha256sum -c SHA256SUMS.txt
2. Shut down Knulli.
3. Extract the versioned ZIP into /userdata/roms/ports and replace an older
   Knulli App Store launcher and knulli-app-store directory when updating.
4. Safely eject, boot, and refresh game lists or reboot.
5. Open Knulli App Store in Ports.

CONTROLLER SETUP

The first startup uses Knulli's SDL_GAMECONTROLLERCONFIG when available.
Press any controller button during the eight-second startup prompt to begin
setup, or wait to keep automatic mapping. Assign Up, Down, Left, Right,
Confirm, Back, Details/Diagnostics, and Exit. Conflicting buttons are rejected.
Test every action before the mapping is saved.

Open Settings with the displayed Details/Diagnostics control. Settings can
rerun setup, export diagnostics, reset the active controller to automatic
mapping, or close. Instructions show semantic actions and SDL labels; they do
not assume physical A/B positions.

Mappings are stored by Knulli device plus controller GUID under:
  /userdata/system/configs/knulli-app-store/controller-mappings.json

When GUID is absent, device plus normalized controller name is used. A mapping
with neither is not saved. Corrupt files are retained with a .corrupt suffix.

PACKAGE STATUS

Grout 5.1.0.0 and PlayTime 1.0.0 remain blocked on MagicX Zero 28. Grout lacks
evidence for Knulli MagicX orientation, input handling, SDL ABI, and 640x480 UI.
PlayTime has adaptive 640x480 layout code, but its dynamic SDL ABI, accelerated
renderer, and MagicX controller launch are not verified. A133 alone is not
compatibility evidence. Neither package is verified.

TROUBLESHOOTING

Application log:
  /userdata/system/logs/knulli-app-store.log

Use Settings > Export Diagnostics to write a redacted bundle under:
  /userdata/system/knulli-app-store/diagnostics/

The log includes controller device, GUID/name when exposed, mapping source,
semantic events, setup decisions, validation failures, and display candidates.
Inspect a diagnostic bundle for private data before sharing it.
