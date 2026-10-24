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
