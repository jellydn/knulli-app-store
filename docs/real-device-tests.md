# Real-device test evidence

## Current scope after Picker 0.2.1 publication

[Picker 0.2.1](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.2.1) is published, with no new hardware evidence. It is **UNTESTED on all matrices**. The owner report below remains exact-version evidence for Picker 0.2.0 only; the other six unchanged current versions retain tested evidence on TrimUI Smart Pro, Knulli AArch64 at 1280×720 only. All seven `review.status` values remain **experimental**. No package has current MagicX evidence.

Release target: Store v2026.10.25.1. Hardware reports remain scoped to the artifacts actually tested. The earlier audit baseline was public v2026.10.25 at `14490119`; its signature checks and native development-build reports do not transfer to another release.

The lead independently verified Picker 0.2.1's six published assets/digests, 20 regular archive files and exact ABI. Published-byte Install, 0.2.0→0.2.1 Update, Repair and Uninstall validation passed in fixture roots with nested `state/` retained. No ARM64 runtime was executed. These are archive/installer checks, not hardware or Picker runtime evidence.

## 2026-10-25: Current owner confirmation

- Date: 2026-10-25 (Asia/Singapore).
- Source: [owner confirmation in this thread](https://ampcode.com/threads/T-01a17039-648c-7024-abe9-8785444a2b5b).
- Exact confirmation: “Yes, all seven current versions were tested on TrimUI Smart Pro only, Knulli AArch64 at 1280×720”.
- Passed package tests: Grout 5.2.0.0, PlayTime 1.0.0, Random Game Picker 0.2.0, RetSend 1.0.0, PocketCurator 1.1.2, RAOfflineProxy 2.2.0-alpha1, and retroshelf 0.5.6.

This confirmation supplies the listed package versions and the Smart Pro, Knulli, AArch64, 1280×720 matrix for passed `real-device-test` evidence. It supersedes the earlier broad owner report for those exact versions, not later releases. Picker 0.2.0 is now historical evidence, not a tested badge for 0.2.1. All seven review statuses remain **experimental**. MagicX Zero 28 and other matrices remain untested.

The owner did not supply separate lifecycle results, physical button tests, audio tests, power-loss tests, or Store idle-redraw verification. Do not treat this package confirmation as proof of those details or of a particular Store ZIP. No exact firmware build identifier was supplied by this confirmation; do not copy one from a historical report.

The reports below are historical records. Their original scope and limits remain useful, but they do not limit the newer package confirmation above.

## 2026-09-15: TrimUI Smart Pro package report

- Tester: Dung Huynh Duc
- Device context: TrimUI Smart Pro, `aarch64`, 1280×720
- Firmware context: Knulli Scarab; the earlier diagnostic reported `scarab 2026/08/19 16:06`
- Packages: Grout 5.1.0.0 and PlayTime 1.0.0
- Report: “Tested Grout and Playtime. All good.”
- Provenance: Slack report relayed through the project Puck thread

This historical report confirms a successful package test in the immediate Smart Pro context. It does not list separate install, launch, repair, adoption, or uninstall results. It does not confirm PlayTime on MagicX Zero 28. It applies to Grout 5.1.0.0, not 5.2.0.0. The owner confirmation above supplies newer evidence for the seven versions it lists; both catalogue packages remain experimental.

## 2026-10-24: Catalogue report awaiting device details

- Tester: `dunghd`
- Report: “All apps work except Pocketcurator”
- Provenance: [Store installation and device-testing thread](https://ampcode.com/threads/T-01a16b6d-1bf4-70ab-bb81-6fdb577187c1), after the v2026.10.24.2 release
- Reported working apps: Grout, PlayTime, Random Game Picker, RetSend, RAOfflineProxy and retroshelf
- Reported failure: PocketCurator; a launcher path mismatch in the Store package was reproduced separately with disposable files

This historical report did not identify the device, firmware, architecture, resolution or exact installed app versions. The current owner confirmation above supersedes it for current package evidence. This older report does not establish separate Repair, Update or Uninstall results, and it does not verify the PocketCurator path correction on hardware.

## 2026-10-24: PocketCurator SSH investigation

- Device: TrimUI Smart Pro, `aarch64`, board `a133`, visible framebuffer 1280×720.
- Firmware: `/usr/share/knulli/knulli.version` reports `beta 2026/09/21 21:25`; `/etc/os-release` identifies Knulli, OS date `20260921`, Batocera base 42.
- Installed Store GUI: `v2026.10.24.3`; PocketCurator: `1.1.2`, build `20260716-1`; PortMaster: `2026.06.23-0015`.
- Runtime: system Python 3.12.8, pygame 2.5.2, SDL 2.32.8.
- Installed-state version observations: Grout 5.2.0.0, PlayTime 1.0.0, Random Game Picker 0.1.1, RetSend 1.0.0 and RAOfflineProxy 2.2.0-alpha1. The retroshelf README reports 0.5.6. These observations are not additional functional tests.
- Provenance: [Mac-runner investigation](https://ampcode.com/threads/T-01a16c63-4ffd-76be-a50c-c62af0494ee4) and the owner's confirmation in the [Store thread](https://ampcode.com/threads/T-01a16b6d-1bf4-70ab-bb81-6fdb577187c1): “Great. It works now.”

The earlier Repair was installed: launcher SHA-256 `09ac528ecd73005e3f2befd7f1d56ffa788b8a1ca7ddf882f2988fc48c48579f`. Native log `PC_2026-10-24_132629.log` still shows `PYTHONPATH=/pocketcurator`, missing `main.py`, and Python exit 2. PortMaster's `control.txt` sources `device_info.txt`, which explicitly unsets `SCRIPT_DIR`. An on-device shell probe reproduced the old patch's failure and found the payload when the directory came directly from the launcher's `$0`.

The corrected 41-byte patch at offset 8083 uses `GAMEDIR="$(dirname "$0")/pocketcurator"` plus two padding spaces. Final launcher SHA-256: `b6c0085e0d714a9994e97945ec3e755e5f58befdc5608328e25084e34521a66c`. Only the launcher was changed, after backing up the complete app and shared Ports game list under `/userdata/system/backups/pocketcurator-20261024-script-dir/`. Settings remained byte-for-byte identical to the backup. No ROM or shared game-list edits were made.

Native launch through EmulationStation succeeded. Log `PC_2026-10-24_133817.log` records the correct payload path, 1280×720 display initialization, 17 systems, 152 ROMs and completed system-browser startup. The inspected framebuffer screenshot shows the nonblack Pocket Curator v1.1.2 system browser. No deletion features or update installation were used. One safe Escape key press was sent before testing stopped so the owner could use the device. Clean exit, relaunch, and native Install, Repair, Update, adoption, rollback and Uninstall were not completed. This historical investigation is launch evidence for the matrix above, not a complete lifecycle test.

The regression models PortMaster clearing `SCRIPT_DIR`: it fails with the old patch and passes with the correction. An actual publisher-archive fixture also passed Install and Repair while retaining custom settings. These are offline installer checks, not native lifecycle results.

Update the Store to **v2026.10.24.4 or later**, close PocketCurator, then use **Y (More) → Repair → Confirm** to apply the corrected catalogue and managed-file hash. Repair with v2026.10.24.3 restores the broken patch. The manual fix did not alter Store ownership records or its signed catalogue.

The backed-up launcher is `/userdata/system/backups/pocketcurator-20261024-script-dir/Pocket Curator.sh`; restoring it while the app is closed deliberately restores the known failure. Do not restore the whole app archive or shared game list over later owner changes.

## 2026-10-25: Option B Library and Picker 0.2.0

Source: [native Option B implementation and release thread](https://ampcode.com/threads/T-01a16e63-3dfe-7782-8ac5-a9a71a55726c). This is a record of that test, not a new device run.

- Device: TrimUI Smart Pro, actual 1280×720 output, KNULLI, `aarch64`. The final test did not record an exact firmware build identifier.
- Source: [published baseline](https://github.com/jellydn/knulli-app-store/commit/14490119b819f524a8a59919b949359092b3b014), also used by v2026.10.25. The tested development ZIP and the public signed release ZIP are different artifacts.
- Tested development ZIP: `knulli-app-store-trimui-smart-pro-experimental-library-final-20261025.zip`, 17,966,611 bytes, SHA-256 `ec01e5016f9f016f5454c50923b7cce87e801631cc3830cf147a87cfe7aaf2b7`.
- Picker: public stable 0.2.0 archive, SHA-256 `3f8a225cffcb2a89ffee166ff4f06dbf87923142309d97203e738a8200737df4`, 3,273,845 compressed bytes and 20 regular files. Store status remains experimental.

The Library opened with nearby rows, expanded selection, filters and controller glyphs. Fresh Picker installation, update from 0.1.1 to 0.2.0, and explicit Repair passed in disposable roots. All 20 files matched the archive; executable modes and fixture settings/logs remained correct. Repair restored only a deliberately changed managed README. A opened review, Start did not confirm, and B cancelled. The Installed filter and restored selection passed. The test injected 100 D-pad pulses through the existing input device and recorded exactly 100 semantic down events in 31.08 seconds. This tests SDL mapping, not physical button presses.

The Store quit chord returned to EmulationStation. Store-installed stable Picker opened through a temporary Ports wrapper; artwork, reroll, settings and Start exit were exercised. No stable Picker Play request was made. Final checks found EmulationStation running, no Store or Picker process, and all seven original baseline hashes unchanged. Production apps and firmware libraries were not replaced.

A process sample after lifecycle and navigation work showed 36,524 KiB RSS and 78.0% process-average CPU after 3:01. This is not an idle-only benchmark, FPS result or battery test. The [concerns audit](../.planning/codebase/CONCERNS.md) records the later controlled desktop redraw check.

Physical button presses, audio, stable Picker gameplay/hotkey return, MagicX hardware, and final Library Uninstall/Force reinstall were not tested. The missing exact firmware identifier prevents this report from granting a new exact-matrix badge. Public release download, hash, signature, inventory and ABI verification in the source thread is not public-ZIP device-deployment evidence.

## Historical broad owner catalogue confirmation

The owner later confirmed: “all apps are tested and available to install now”, relayed through the [concerns-audit request](https://ampcode.com/threads/T-01a16e38-02c2-77ca-badc-0eeb228477db). This superseded the earlier broad PocketCurator failure report for general user guidance. The 2026-10-25 current owner confirmation above now supersedes this broad report with current versions and the Smart Pro matrix. All seven current manifests remain actionable experimental packages when runtime checks pass; separate lifecycle results are not supplied.
