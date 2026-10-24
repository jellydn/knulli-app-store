# Real-device test evidence

## 2026-09-15: TrimUI Smart Pro package report

- Tester: Dung Huynh Duc
- Device context: TrimUI Smart Pro, `aarch64`, 1280×720
- Firmware context: Knulli Scarab; the earlier diagnostic reported `scarab 2026/08/19 16:06`
- Packages: Grout 5.1.0.0 and PlayTime 1.0.0
- Report: “Tested Grout and Playtime. All good.”
- Provenance: Slack report relayed through the project Puck thread

This report confirms a successful package test in the immediate Smart Pro context. It does not list separate install, launch, repair, adoption, or uninstall results. It does not confirm PlayTime on MagicX Zero 28. It applies to Grout 5.1.0.0, not the current 5.2.0.0 release. PlayTime 1.0.0 can show a device-tested badge only for this exact Smart Pro matrix. Both catalogue packages remain broad experimental packages elsewhere.

## 2026-10-24: Catalogue report awaiting device details

- Tester: `dunghd`
- Report: “All apps work except Pocketcurator”
- Provenance: [Store installation and device-testing thread](https://ampcode.com/threads/T-01a16b6d-1bf4-70ab-bb81-6fdb577187c1), after the v2026.10.24.2 release
- Reported working apps: Grout, PlayTime, Random Game Picker, RetSend, RAOfflineProxy and retroshelf
- Reported failure: PocketCurator; a launcher path mismatch in the Store package was reproduced separately with disposable files

The report does not identify the device, firmware, architecture, resolution or exact installed app versions. These details have been requested. Keep tested badges unchanged until the test matrix is confirmed. This report does not establish separate Repair, Update or Uninstall results, and it does not verify the PocketCurator path correction on hardware.

## 2026-10-24: PocketCurator SSH investigation

- Device: TrimUI Smart Pro, `aarch64`, board `a133`, visible framebuffer 1280×720.
- Firmware: `/usr/share/knulli/knulli.version` reports `beta 2026/09/21 21:25`; `/etc/os-release` identifies Knulli, OS date `20260921`, Batocera base 42.
- Installed Store GUI: `v2026.10.24.3`; PocketCurator: `1.1.2`, build `20260716-1`; PortMaster: `2026.06.23-0015`.
- Runtime: system Python 3.12.8, pygame 2.5.2, SDL 2.32.8.
- Installed-state version observations: Grout 5.2.0.0, PlayTime 1.0.0, Random Game Picker 0.1.1, RetSend 1.0.0 and RAOfflineProxy 2.2.0-alpha1. The retroshelf README reports 0.5.6. These observations are not additional functional tests.
- Provenance: [Mac-runner investigation](https://ampcode.com/threads/T-01a16c63-4ffd-76be-a50c-c62af0494ee4) and the owner's confirmation in the [Store thread](https://ampcode.com/threads/T-01a16b6d-1bf4-70ab-bb81-6fdb577187c1): “Great. It works now.”

The earlier Repair was installed: launcher SHA-256 `09ac528ecd73005e3f2befd7f1d56ffa788b8a1ca7ddf882f2988fc48c48579f`. Native log `PC_2026-10-24_132629.log` still shows `PYTHONPATH=/pocketcurator`, missing `main.py`, and Python exit 2. PortMaster's `control.txt` sources `device_info.txt`, which explicitly unsets `SCRIPT_DIR`. An on-device shell probe reproduced the old patch's failure and found the payload when the directory came directly from the launcher's `$0`.

The corrected 41-byte patch at offset 8083 uses `GAMEDIR="$(dirname "$0")/pocketcurator"` plus two padding spaces. Final launcher SHA-256: `b6c0085e0d714a9994e97945ec3e755e5f58befdc5608328e25084e34521a66c`. Only the launcher was changed, after backing up the complete app and shared Ports game list under `/userdata/system/backups/pocketcurator-20261024-script-dir/`. Settings remained byte-for-byte identical to the backup. No ROM or shared game-list edits were made.

Native launch through EmulationStation succeeded. Log `PC_2026-10-24_133817.log` records the correct payload path, 1280×720 display initialization, 17 systems, 152 ROMs and completed system-browser startup. The inspected framebuffer screenshot shows the nonblack Pocket Curator v1.1.2 system browser. No deletion features or update installation were used. One safe Escape key press was sent before testing stopped so the owner could use the device. Clean exit, relaunch, and native Install, Repair, Update, adoption, rollback and Uninstall were not completed. Keep lifecycle-tested badges unchanged; this is launch evidence for the exact matrix above.

The regression models PortMaster clearing `SCRIPT_DIR`: it fails with the old patch and passes with the correction. An actual publisher-archive fixture also passed Install and Repair while retaining custom settings. These are offline installer checks, not native lifecycle results.

Update the Store to **v2026.10.24.4 or later**, close PocketCurator, then use **Y (More) → Repair → Confirm** to apply the corrected catalogue and managed-file hash. Repair with v2026.10.24.3 restores the broken patch. The manual fix did not alter Store ownership records or its signed catalogue.

The backed-up launcher is `/userdata/system/backups/pocketcurator-20261024-script-dir/Pocket Curator.sh`; restoring it while the app is closed deliberately restores the known failure. Do not restore the whole app archive or shared game list over later owner changes.
