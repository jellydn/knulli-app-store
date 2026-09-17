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
   Confirm images/Knulli App Store.png is present beside the launcher.
4. Safely eject, boot, and refresh game lists or reboot.
5. Open Knulli App Store in Ports. The Ports list uses that image after a
   game list refresh.

CONTROLLER SETUP

The first launch for each device/controller identity uses a dedicated screen
before the catalogue. Choose Use Detected Mapping, Test Detected Mapping,
Customize, or Safe Exit. Customize assigns one semantic action at a time. Each
detected button has Select, Retry, Start Over, and Back choices. Conflicts
stay on the current action. Tested and custom mappings require a complete
eight-action preview before atomic save. No timeout skips setup.

Open Settings with the control the footer shows beside SETTINGS. Settings can
rerun setup, export diagnostics, reset the active mapping, or close. Hold the
saved Back and Settings controls while launching to force setup. Instructions
do not assume physical A/B positions. A missing SDL GameController shows a
blocked screen and diagnostic locations instead of the catalogue.

The footer is the only place a screen shows controls. It names each available
action beside its binding, for example Confirm (A)  Back (B)  Settings (Y),
and the same action name always means the same action. No screen repeats a
control hint in its own text; the controller mapping summary uses the same
names. On the blocked screen, Confirm exports diagnostics and Back leaves.

Mappings are stored by Knulli device plus controller GUID under:
  /userdata/system/configs/knulli-app-store/controller-mappings.json

When GUID is absent, device plus normalized controller name is used. A mapping
with neither is not saved. Corrupt files are retained with a .corrupt suffix.

PACKAGE STATUS

Grout 5.2.0.0, PlayTime 1.0.0, and RetSend 0.9.1 are user-authorized
experimental tests only when architecture, glibc ABI, SDL libraries, device
identity, and 640x480 pass.
Select Install, or Manage existing for detected external files, and confirm
the warning. An empty destination directory is not a copy, and a card that
reports a wider file mode than requested is not a health failure. The Store
disables Grout's self-updater with a verified staging patch. RetSend needs
local Wi-Fi and keeps its configuration and TLS identity under
/userdata/system/configs/retsend, with received files in
/userdata/roms/retsend-inbox. None of the three has MagicX real-device evidence.

PlayTime data stays under /userdata/system/configs/playtime. Manager state and
adoption backups are under /userdata/system/knulli-app-store. Force reinstall
appears only after Manage existing fails, needs two confirmations, and stores
a timestamped recovery backup. Package details
and the log identify the exact managed file when Repair is required. The app
asks Knulli to reload game lists when it changes the menu; reboot if it reports
that a restart is required.

TROUBLESHOOTING

Application log:
  /userdata/system/logs/knulli-app-store.log

Use Settings > Export Diagnostics to write a redacted bundle under:
  /userdata/system/knulli-app-store/diagnostics/

The log includes controller device, GUID/name when exposed, mapping source,
semantic events, setup decisions, validation failures, and display candidates.
Inspect a diagnostic bundle for private data before sharing it.
