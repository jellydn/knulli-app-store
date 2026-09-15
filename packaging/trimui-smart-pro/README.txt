KNULLI APP STORE - TRIMUI SMART PRO EXPERIMENTAL BUILD

The GUI was reported functional on one TrimUI Smart Pro. This build remains
experimental until the exact Knulli release and repeatable test evidence are
recorded. Package compatibility is reviewed separately.

INSTALL

1. On your computer, verify the ZIP against SHA256SUMS.txt.
2. Extract the ZIP directly into /userdata/roms/ports on the Knulli card.
3. Confirm these paths exist:
   /userdata/roms/ports/Knulli App Store.sh
   /userdata/roms/ports/knulli-app-store/knulli-app-ui
4. Safely eject the card, start Knulli, and refresh game lists or reboot.
5. Open Knulli App Store from the Ports system.

UPDATE

Verify the new checksum, shut down Knulli, and extract the new ZIP into the
same /userdata/roms/ports directory. Allow replacement of the old launcher
and knulli-app-store files. Manager state under /userdata/system is retained.

CONTROLS

The first launch for each device/controller identity opens Controller Setup
before the catalogue. Choose Use Detected Mapping, Test Detected Mapping,
Customize, or Safe Exit. Customize reviews each detected button and offers
Accept, Retry, Start Over, and Cancel. Tested and custom mappings require a
complete eight-action preview before atomic save. No timeout skips setup.

Later launches load the saved mapping. Settings can rerun setup, export
diagnostics, or reset the active mapping. Hold the saved Back and
Details/Diagnostics controls while launching to force setup. If no SDL
GameController exists, a blocked screen shows log and diagnostic paths.

Mappings are stored by device and controller identity under:
  /userdata/system/configs/knulli-app-store/controller-mappings.json

CURRENT SAFETY STATE

Grout 5.1.0.0 and PlayTime 1.0.0 are explicit experimental test packages for
TrimUI Smart Pro. They require a risk confirmation and are not verified.
Existing copies show ADOPT; adoption inventories and backs up uncertain files
before installing the reviewed release. The other four packages are read-only.

Do not use Grout's built-in updater. Use App Store Repair and future reviewed
updates only. Grout config stays under /userdata/roms/tools/Grout. PlayTime
statistics stay under /userdata/system/configs/playtime. Manager state and
adoption backups are under /userdata/system/knulli-app-store. Refresh game
lists or reboot after package install or uninstall.

EmuDrop remains non-installable. It downloads ROMs; users must have the
required rights and comply with local copyright law.

TROUBLESHOOTING

If the app returns to EmulationStation, inspect:
  /userdata/system/logs/knulli-app-store.log

Use Settings > Export Diagnostics to export platform, catalogue, and capped
redacted logs to /userdata/system/knulli-app-store/diagnostics. It does not
include package credentials or private configuration files.

The log records each SDL and framebuffer resolution candidate. On TrimUI
Smart Pro it should select SDL renderer output 1280x720. Implausible values
such as 1280x13107 are rejected. Terminal control sequences are removed.

Do not replace Knulli's SDL library with a desktop or stock TrimUI copy.
