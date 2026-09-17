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
   /userdata/roms/ports/images/Knulli App Store.png
4. Safely eject the card, start Knulli, and refresh game lists or reboot.
5. Open Knulli App Store from the Ports system. The Ports list uses the image
   at images/Knulli App Store.png after a game list refresh.

UPDATE

Verify the new checksum, shut down Knulli, and extract the new ZIP into the
same /userdata/roms/ports directory. Allow replacement of the old launcher
and knulli-app-store files. Manager state under /userdata/system is retained.

CONTROLS

The first launch for each device/controller identity opens Controller Setup
before the catalogue. Choose Use Detected Mapping, Test Detected Mapping,
Customize, or Safe Exit. Customize reviews each detected button and offers
Select, Retry, Start Over, and Back. Tested and custom mappings require a
complete eight-action preview before atomic save. No timeout skips setup.

Later launches load the saved mapping. Settings can rerun setup, export
diagnostics, or reset the active mapping. Hold the saved Back and Settings
controls while launching to force setup. If no SDL GameController exists, a
blocked screen shows log and diagnostic paths.

The footer is the only place a screen shows controls. It names each available
action beside its binding, for example Confirm (A)  Settings (Y)  Quit
(SELECT + Y),
and the same action name always means the same action. No screen repeats a
control hint in its own text; the controller mapping summary uses the same
names. Quitting is the one control no button carries alone: hold Select and
press Y. On the blocked screen, Confirm exports diagnostics and that chord is
the way out.

Mappings are stored by device and controller identity under:
  /userdata/system/configs/knulli-app-store/controller-mappings.json

CURRENT SAFETY STATE

Grout 5.2.0.0 and PlayTime 1.0.0 have broad experimental Knulli eligibility
only when architecture, glibc ABI, SDL libraries, device identity, and display
bounds pass. Only PlayTime 1.0.0 has exact current Smart Pro evidence.
An existing copy shows EXTERNAL and offers Manage existing; that inventories
and backs up uncertain files without reinstalling the reviewed release. Only
files count, so an empty directory left by a rolled-back write still offers
Install. A card that reports a wider file mode than requested is not a health
failure. The other two packages are read-only.

The Store disables Grout's updater metadata URL and verifies the transformed
binary. Use App Store Repair and reviewed updates only. Grout config stays
under /userdata/roms/tools/Grout. PlayTime
statistics stay under /userdata/system/configs/playtime. Manager state and
adoption backups are under /userdata/system/knulli-app-store. After Manage
existing fails, Force reinstall needs two confirmations and keeps a complete
timestamped backup under recovery-backups/<package-id>/. The app asks
Knulli to reload game lists when it changes the menu; if it reports that a
restart is required, reboot to see the new entry. Manual refresh still works.

Health checks cover immutable managed release files. PlayTime runtime data does
not cause Repair. If Repair appears, package details and the application log
show the exact missing, changed, or mode-mismatched managed path.

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
