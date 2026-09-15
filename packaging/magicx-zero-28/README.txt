KNULLI APP STORE - MAGICX ZERO 28 EXPERIMENTAL BUILD

Knulli source identifies this target as magicx-zero-28, AArch64 Cortex-A53,
with a 640x480 display. A real-device diagnostic confirms Knulli Scarab,
aarch64, 640x480, and SDL GameController magicx-input with a nonzero GUID.
Physical button assignments remain runtime data.

INSTALL OR UPDATE

1. On a computer, verify the ZIP with: sha256sum -c SHA256SUMS.txt
2. Shut down Knulli.
3. Extract the versioned ZIP into /userdata/roms/ports and replace an older
   Knulli App Store launcher and knulli-app-store directory when updating.
4. Safely eject, boot, and refresh game lists or reboot.
5. Open Knulli App Store in Ports.

CONTROLLER SETUP

The first launch for each device/controller identity uses a dedicated screen
before the catalogue. Choose Use Detected Mapping, Test Detected Mapping,
Customize, or Safe Exit. Customize assigns one semantic action at a time. Each
detected button has Accept, Retry, Start Over, and Cancel choices. Conflicts
stay on the current action. Tested and custom mappings require a complete
eight-action preview before atomic save. No timeout skips setup.

Open Settings with the displayed Details/Diagnostics control. Settings can
rerun setup, export diagnostics, reset the active mapping, or close. Hold the
saved Back and Details/Diagnostics controls while launching to force setup.
Instructions do not assume physical A/B positions. A missing SDL GameController
shows a blocked screen and diagnostic locations instead of the catalogue.

Mappings are stored by Knulli device plus controller GUID under:
  /userdata/system/configs/knulli-app-store/controller-mappings.json

When GUID is absent, device plus normalized controller name is used. A mapping
with neither is not saved. Corrupt files are retained with a .corrupt suffix.

PACKAGE STATUS

Grout 5.1.0.0 remains blocked on MagicX Zero 28. PlayTime 1.0.0 is a
user-authorized experimental test package. Select Install or Adopt and confirm
the unverified warning. Its reviewed ARM64 archive uses runtime display size
and SDL GameController actions. Its accelerated renderer, launch, tracking,
repair, and uninstall still need a real-device test. Neither package is
verified.

PlayTime data stays under /userdata/system/configs/playtime. Manager state and
adoption backups are under /userdata/system/knulli-app-store. Package details
and the log identify the exact managed file when Repair is required.

TROUBLESHOOTING

Application log:
  /userdata/system/logs/knulli-app-store.log

Use Settings > Export Diagnostics to write a redacted bundle under:
  /userdata/system/knulli-app-store/diagnostics/

The log includes controller device, GUID/name when exposed, mapping source,
semantic events, setup decisions, validation failures, and display candidates.
Inspect a diagnostic bundle for private data before sharing it.
