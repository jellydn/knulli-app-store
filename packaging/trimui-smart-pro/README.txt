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

D-pad: move through packages and actions
SDL A: select or confirm
SDL B: back or cancel
SDL Y: export a redacted diagnostic text bundle

With Knulli's default Ports layout, physical B (south) maps to SDL A and
physical A (east) maps to SDL B. A per-game Xbox-layout setting reverses
those physical prompts. The app uses Knulli's SDL_GAMECONTROLLERCONFIG;
it does not guess raw event or button numbers.

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

Press Y in the App Store to export platform, catalogue, and capped redacted
logs to /userdata/system/knulli-app-store/diagnostics. The export does not
include package credentials or private configuration files.

Do not replace Knulli's SDL library with a desktop or stock TrimUI copy.
