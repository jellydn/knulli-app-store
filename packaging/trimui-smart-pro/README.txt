KNULLI APP STORE - TRIMUI SMART PRO EXPERIMENTAL BUILD

Target: Knulli, AArch64, 1280x720. The current gabagool-backed Option B Library
uses a responsive list. A same-source development build passed native Picker
Install, Update, Repair, injected navigation and return to EmulationStation.
This is not a public-ZIP device test. Physical presses, audio and the later
idle-redraw change still need hardware checks. See the linked device evidence.
A full release does not remove experimental limits; package evidence is separate.

INSTALL OR MANUAL UPDATE

Release target: v2026.10.25.1. Hardware reports remain scoped to the artifacts
actually tested. Earlier native evidence used Picker 0.2.0, not current 0.2.1.

1. Download examples for the Smart Pro ZIP and SHA256SUMS.txt:
   https://github.com/jellydn/knulli-app-store/releases/tag/v2026.10.25.1
   The target ZIP is:
   knulli-app-store-trimui-smart-pro-experimental-v2026.10.25.1.zip
2. In the download directory, run:
   sha256sum --ignore-missing -c SHA256SUMS.txt
   Continue only if your ZIP reports OK. The other target may be absent.
3. Shut down Knulli and connect its data card to the computer.
4. Extract the ZIP directly into /userdata/roms/ports. Check these paths:
   /userdata/roms/ports/Knulli App Store.sh
   /userdata/roms/ports/knulli-app-store/knulli-app-ui
   /userdata/roms/ports/images/Knulli App Store.png
5. For an update, replace all Store release files together: launcher, icon,
   CLI, GUI, catalogue, signature, and this guide. Do not delete installed
   apps, settings, or /userdata/system/knulli-app-store.
6. Safely eject, start Knulli, and refresh game lists or reboot.
7. Open Knulli App Store in Ports.

Device builds need system SDL2, SDL2_image, and SDL2_ttf. SDL2_gfx is linked
statically. The build rejects required glibc versions above 2.34. Do not
replace Knulli's system libraries with desktop or stock TrimUI copies.

CONTROLS

The Store opens the Option B Library immediately: nearby rows remain visible,
the selected row expands beside a mint focus rail, and All/Ready/Installed
filters stay visible. The footer uses circled face-button glyphs and contextual
actions; the upper badge shows the quit chord. There is no controller setup
or custom mapping editor. Controls use Knulli's EmulationStation (ES) letters:

  D-pad up/down     Browse apps; left/right also work
  L1/R1             Filter All, Ready, and Installed
  A                 Review or confirm an action
  B                 Cancel or go back
  Y                 Open More; Y inside More exports diagnostics
  Select + Y        Hold Select, then press Y to quit

The first A press on a row opens confirmation; it does not install an app.
More has details and secondary actions. An Issue opens its health check first.
The footer shows the actions available on each screen. Completion notices
clear after three seconds; failures stay until answered.

The launcher selects the default Knulli layout, or Xbox layout when the Ports
Xbox flag exists. Bindings compensate once; SDL events are not swapped.
Update the launcher and binaries together. Legacy controller-mappings.json
files are ignored and left unchanged. No assignments are loaded or saved.

If no SDL GameController is available, a blocked screen shows diagnostic
paths. Reconnect the controller to resume. Y exports diagnostics directly
there and on an empty catalogue; Confirm also exports on the blocked screen.
Select+Y remains the quit chord.

STORE SELF-UPDATE

Select the Knulli App Store entry. Confirm Check for updates, then confirm
Update when offered. Exit and reopen after completion. Checks include
official experimental pre-releases and use this packaged target.

Self-update needs the standard /userdata/roms/ports/knulli-app-store location
and a build with update support. It verifies GitHub's asset size and SHA-256,
then replaces only the seven Store release files. Installed apps and settings
stay. This trusts GitHub and the publisher account, not an independent update
signing key. Older builds, custom locations, and text file busy failures need
the manual ZIP procedure above. Startup recovery runs before the GUI loads.
A failed rollback retains its journal and snapshots for recovery. Evidence
already deleted by an older build cannot be recovered by this mechanism.

CATALOGUE AND SAFETY

On 2026-10-25 (Asia/Singapore), the owner confirmed seven listed versions
tested on TrimUI Smart Pro only, Knulli AArch64 at 1280x720. Source:
https://ampcode.com/threads/T-01a17039-648c-7024-abe9-8785444a2b5b
Manifest statuses remain experimental: Grout 5.2.0.0, PlayTime 1.0.0,
Random Game Picker 0.2.0, RetSend 1.0.0, PocketCurator 1.1.2,
RAOfflineProxy 2.2.0-alpha1, and retroshelf 0.5.6. Passed package evidence
covers that Smart Pro matrix only. MagicX and other matrices remain untested.
Picker is now 0.2.1 and UNTESTED on all matrices. The 0.2.0 report remains
historical exact-version evidence. The other six current versions retain
Smart Pro-only evidence. All seven review statuses remain experimental.
Separate lifecycle, physical button, audio, power-loss, and Store idle-redraw
results are not supplied by this confirmation.

Firmware, architecture, ABI, and runtime libraries must pass checks. Unknown
device identity or display size does not block installation. UNTESTED means
no matching package-version and device-matrix evidence exists. The current
confirmation supersedes older package badge restrictions, not the limits of
historical GUI reports. It does not verify every current Store GUI behavior.

The Store verifies pinned HTTPS downloads before staging regular files. It
never runs a downloaded installer. Store writes stay below /userdata and use
journals and snapshots for rollback. Installed apps are NOT sandboxed.
PortMaster is external; the Store does not install its catalogue.

An existing copy offers Manage existing. Exact matches can become owned;
uncertain files stay unmanaged and are backed up before later replacement.
Empty directories do not count as external copies. Repair restores pinned
files and keeps declared data. Uninstall removes owned files and restores
originals where applicable. Close apps before Store operations.

Grout's updater URL is disabled by a verified staging patch. Its settings
stay under /userdata/roms/tools/Grout. PlayTime data stays under
/userdata/system/configs/playtime. RetSend keeps configuration and TLS identity
under /userdata/system/configs/retsend and its default inbox under
/userdata/roms/retsend-inbox. User-selected routes can override that inbox.
Repair moves an older RetSend launcher to RetSend/RetSend.sh.

Random Game Picker 0.2.1 needs glibc 2.34 or later, firmware SDL2/image/ttf/gfx,
and the loopback EmulationStation API. Settings and logs in state/ stay.
It is UNTESTED on every device matrix. It fixes settings limits/malformed
fields, idle redraw, per-scan path caching and ABI/provenance checks. Launcher,
settings schema and exact-path ES launch behavior are unchanged. Preserve the
entire state/ directory. Its runtime requirements are separate from the Store's.
https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.2.1

PocketCurator, RAOfflineProxy, and retroshelf are upstream-managed. They
control later setup, runtime writes, downloads, and self-updates. Store Repair
restores pinned code, not upstream-updated code, and keeps declared data.
PocketCurator needs PortMaster and can permanently delete ROMs. Back up first.
Use only ROMs you have the right to use and servers you trust. For an older
PocketCurator launcher-path failure, update the Store to v2026.10.24.4 or
later, close PocketCurator, then use More > Repair and confirm.

RAOfflineProxy needs Python 3.12 and pygame. No separate publisher checksum
exists; its pinned hash was checked against GitHub's asset digest. Before
removal, stop the proxy, disable autostart, then run this command yourself:
  /userdata/system/raofflineproxy/bin/raofflineproxy remove-boot-hook
Follow remaining Store cleanup messages for game hooks and proxy settings.
Keep its config/cache; upstream's own Uninstall action deletes data.

retroshelf needs an account and Wi-Fi. Its license is unverified; the Store
downloads directly from the publisher without redistribution. Close it, then
run this command yourself before Store removal:
  /userdata/roms/tools/retroshelf/bin/retroshelf-uninstall.sh --keep-data
Read-only checks block removal while declared hooks or service state remain.
They do not cover every custom path. The Store never runs cleanup scripts.

TROUBLESHOOTING

Application log: /userdata/system/logs/knulli-app-store.log
Diagnostics: /userdata/system/knulli-app-store/diagnostics/
Manager state and backups: /userdata/system/knulli-app-store/

Press Y for More, then Y to export diagnostics. Inspect the export for private
data before sharing it. Exports contain platform, controller, catalogue, and
redacted log data, not package settings or ROMs.

A full health check hashes immutable managed files. The GUI caches results;
reopen the Store to detect changes made outside it. Wider filesystem modes
are accepted; lost recorded owner permissions are an issue. Select Issue for
the exact path and expected/actual hash or mode. Declared runtime data does
not cause a health failure.

Force reinstall is limited to recoverable management failures or validated
stale transactions. It needs two confirmations and retains a complete backup
under recovery-backups/<package-id>/. It cannot bypass mandatory safety checks.
Active transactions block recovery actions. Do not edit state while running.

A menu refresh accepted by Knulli is queued, not completed. Return to
EmulationStation and refresh or restart it if an entry is missing.

Device instructions and evidence:
https://github.com/jellydn/knulli-app-store/blob/main/docs/trimui-smart-pro.md
