KNULLI APP STORE - EXPERIMENTAL AARCH64 / GLIBC / SDL2 BUILD

Guide updated for v2026.11.04:
https://github.com/jellydn/knulli-app-store/releases/tag/v2026.11.04
This tagged Store release does not promote package status or verify hardware.
All fourteen catalogue entries remain experimental. WebUI 1.2.2 and retroshelf
0.6.10 retain their runtime/security warnings and need new-version device tests.

One distribution profile: knulli-aarch64-glibc-sdl2. This is not a universal
binary. It requires KNULLI, Linux AArch64, the AArch64 glibc loader, and system
SDL2, SDL2_image and SDL2_ttf. SDL2_gfx is linked statically. The build rejects
required glibc symbols newer than 2.34. Do not replace firmware libraries.
Actual firmware renderer, controller and physical readability need device tests.
Device identity and exact display remain diagnostic/test evidence, not layouts
or release channels. The initial UI minimum is 320x240.

INSTALL OR MANUAL UPDATE

Download from an official release:
https://github.com/jellydn/knulli-app-store/releases

The single installable asset is named:
  knulli-app-store-knulli-aarch64-glibc-sdl2-experimental-VERSION.zip
SHA256SUMS.txt is a separate integrity file, not another device distribution.
The signed catalogue and signature are inside the ZIP with their matching GUI.

1. Download the ZIP and SHA256SUMS.txt into one directory. Verify:
     sha256sum -c SHA256SUMS.txt
   Continue only if your ZIP reports OK.
2. Back up the Store files and /userdata/system/knulli-app-store. Shut down
   KNULLI and connect its data card to your computer.
3. Extract into /userdata/roms/ports. Replace all seven Store files together:
   Knulli App Store.sh, images/Knulli App Store.png, and knulli-app-store/
   knulli-app, knulli-app-ui, catalog-index.json, catalog-index.json.sig,
   README.txt. Never mix the GUI, catalogue or signature from different builds.
4. Keep installed apps, settings and installer state. Safely eject, start
   KNULLI, refresh game lists if needed, and open the Store in Ports.

LEGACY DEVICE-NAMED INSTALLATIONS

The generic ZIP replaces either former TrimUI Smart Pro or MagicX Zero 28
distribution at the same paths. Once installed, updates use the common profile.
Old binaries cannot discover the new name by themselves. Verified manual ZIP
replacement above works without a bridge. The 2026-11-03 inspection found both
device ZIPs in all 20 then-visible releases; no automatic bridge was verified.
Old clients select the first matching asset and skip their current/older tags.
Do not assume an automatic bridge was published or replace historical assets.
See the migration guide:
https://github.com/jellydn/knulli-app-store/blob/main/docs/store-distribution.md

CONTROLS AND LAYOUT

  D-pad up/down     Browse packages; choose actions inside More
  D-pad left/right Change category; choose actions inside More
  L1/R1            Filter status in Library; read text during review
  A                Review or confirm; Start never confirms
  B                Cancel or go back; a lone Back never quits
  Y                More; Y inside More exports diagnostics
  Select + Y       Hold Select, then press Y to quit

The launcher uses KNULLI's ES controller letters, or direct Xbox letters when
the Ports Xbox flag exists. SDL events are not swapped again. No custom
assignments are loaded or saved. Reconnect a missing controller to resume.
Confirm also exports diagnostics on the blocked screen.

Layout follows SDL drawable dimensions: two panes where 640x480 logical space
fits, otherwise a single-column list and fixed summary. Short outputs use
native text. The header shows the current category/status; the footer owns the
controls. Measured key chips and complete hint labels wrap without dropping
actions. Footer/status height is reserved before body layout, which can show
fewer list rows. Review screens page complete warnings/recovery text with a visible
line range. Paging does not change filters/actions or execute an operation.
Recovery retains two confirmations. Resize preserves selection and operations.

SELF-UPDATE AND SAFETY

At the standard install location, select Knulli App Store, confirm Check for
updates, then confirm Update. Exit and reopen afterward. Official experimental
pre-releases are included. The updater verifies the HTTPS GitHub asset URL,
size and SHA-256 before extraction. This trusts GitHub and the publisher's
account, not an independent Store signing key. The catalogue signature is not
a signature over the Store ZIP. Missing digests or unsafe inventories fail.

The launcher recovers incomplete transactions before loading the GUI and its
catalogue. Atomic replacement and journal rollback preserve other apps/settings.
Custom locations or builds without self-update metadata need manual replacement.
Keep journals/backups after a failed recovery. Do not edit state while running.

The Store never executes downloaded installers. Writes stay below /userdata;
installed apps are NOT sandboxed. Read each package's runtime/rights warning.
Manage existing inventories pre-existing files. Repair restores pinned code and
keeps declared data; Uninstall removes only owned files and restores originals
where applicable. Upstream-managed apps can write/update/download at runtime.
The Store never runs upstream cleanup scripts; complete any removal checks
yourself. PortMaster remains external. Exact package/version/device evidence,
not this shared ZIP, determines the tested badge.

LOGS AND PHYSICAL VERIFICATION

Log: /userdata/system/logs/knulli-app-store.log
Diagnostics: /userdata/system/knulli-app-store/diagnostics/
State and backups: /userdata/system/knulli-app-store/

Use More then Y to export diagnostics; inspect for private data before sharing.
Test cold launch, real display, physical controls/repeat/hotplug/Select+Y,
complete warning/recovery text, ES return and performance on intended hardware.
Desktop checks do not establish device compatibility. A menu refresh accepted
by KNULLI is queued, not completed; refresh/restart ES if an entry is missing.

Current desktop-rendered screenshots and reproducible CPU interval commands:
https://github.com/jellydn/knulli-app-store/blob/v2026.11.04/README.md
https://github.com/jellydn/knulli-app-store/blob/v2026.11.04/docs/idle-cpu-benchmark.md
The sampler reads an existing Linux PID without launching or changing the app.
It needs sh, awk, sleep, getconf and /proc; check firmware tool availability.
No new handheld CPU, physical-control, audible-audio or Picker gameplay/hotkey
result is supplied by this release. Those tests remain owner follow-ups.
