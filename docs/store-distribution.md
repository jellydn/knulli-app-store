# Store distribution and legacy migration

Current version: [v2026.11.04](https://github.com/jellydn/knulli-app-store/releases/tag/v2026.11.04),
including the current install/migration guide and desktop screenshots in its
source documentation. Download the generic ZIP and `SHA256SUMS.txt` from that
same release. Tagged publication does not promote the experimental packages or
prove hardware compatibility. Physical footer/device testing remains outstanding.
See [ADR 0013](adr/0013-compatibility-profile-distribution.md), the
[per-entry release audit](release-audit-2026-11-04.md) and
[desktop capture limits](desktop-verification.md#release-screenshots).

## One installable ZIP

Both successful-main and tag workflows build/collect exactly one
`knulli-app-store-knulli-aarch64-glibc-sdl2-experimental-VERSION.zip`.
There is no device matrix and no new-release legacy device ZIP. Actions uses
`knulli-app-store-knulli-aarch64-glibc-sdl2-experimental` (or `-tag` for tags).
Extract the Actions transport archive first; install only its inner Store ZIP.
The collector rejects missing, duplicate, legacy and wrong-profile ZIPs. Release
reruns refuse to mix existing unexpected assets; they do not delete old assets.

The owner approved **one installable ZIP plus SHA256SUMS.txt** on 2026-11-03:
two uploaded files, not two distributions. Publication through normal green-main
checks was authorized on 2026-11-04. Keep the checksum file separate from the installable ZIP.
The updater still requires the GitHub HTTPS API's exact size and SHA-256.
The signed catalogue and its signature already belong inside the ZIP.

The catalogue signature authenticates the catalogue against the key embedded
in its matching GUI. It does **not** independently sign the Store ZIP. Store
updates retain the existing trust in GitHub HTTPS and the publisher's account;
an unsigned checksum file adds integrity checking, not a separate authority.
Missing API digests fail closed. Independent Store signing needs a separate
key/rollout decision and is not introduced by this naming migration.

## Compatibility is not screen size

The profile describes Linux AArch64, glibc and SDL2 on KNULLI, not universal
hardware support. `scripts/device-build.sh VERSION OUTPUT_DIR` cross-compiles
with the existing Bookworm toolchain. It checks the exact direct GUI dependencies:
the AArch64 loader, libc, libm, SDL2, SDL2_image and SDL2_ttf. SDL2_gfx is linked
statically. Required glibc symbols must be no newer than 2.34; inspect each
actual build for its floor. The CLI remains CGO-free. Transitive SDL dependencies
and the firmware display driver still need a real compatible system.

Do not replace firmware libraries. KNULLI detection, package firmware/ABI/runtime
gates, controller mapping and exact hardware evidence remain unchanged. Drawable
size controls layout only. The initial UI target is 320×240; a desktop render
does not verify a 320×240 handheld, rotation, audio, GPU or physical controls.

Both former launchers had the same behavior. The shared launcher still selects
KNULLI's ES letter policy, or direct Xbox letters when the Ports flag exists.
It does not select a build or infer controls from the board name.

## Legacy migration

**Use verified manual replacement for existing device-named installations.**
This changes distribution paths neither on the card nor inside the ZIP:

1. Download the official generic ZIP and its checksum file. Verify with
   `sha256sum -c SHA256SUMS.txt`; continue only on `OK`. Inspect the source release
   and retain its exact URL, size and SHA-256. Local device-test artifacts are
   unpublished test bytes, not an official release.
2. Back up the Store files and `/userdata/system/knulli-app-store`. Finish or
   recover any open transaction before migration. Shut down KNULLI and connect
   the card to a computer. Do not extract over a running Store.
3. Extract into `/userdata/roms/ports`, replacing **all seven files together**:
   `Knulli App Store.sh`, `images/Knulli App Store.png`, and `knulli-app-store/`
   `knulli-app`, `knulli-app-ui`, `catalog-index.json`,
   `catalog-index.json.sig`, `README.txt`. Never mix a key-bearing GUI with
   another build's catalogue/signature.
4. Keep other apps, configuration, state and backups. Safely eject, boot and
   launch. The launcher runs recovery before the GUI loads its catalogue.
5. Export diagnostics and verify the new version, controls and installed items.
   New binaries discover only the generic profile. Rebuilt code also accepts
   the two legacy target strings as metadata aliases; that does not patch old
   binaries already installed on cards.

Automatic updates still stage and verify before the existing seven-file
transaction, use atomic replacement, retain failed rollback evidence and require
exit/reopen. No catalogue package paths or manager-state format change.

### Automatic bridge limit

Old source filters the device-named prefix, takes the first matching asset in
API order, skips its own build/tag and numerically older/equal tags, and reads
only 20 releases. It cannot discover a generic name. Adding another device-named
copy to a release already containing that prefix does not reliably select the
new copy. Replacing historical assets would alter published bytes and is not
authorized. A new release with device copies would violate the approved rule.

Read-only inspection on 2026-11-03 found **all 20 visible official releases
already contain both device ZIPs**. Thus there is no verified automatic bridge
destination in that window. Clients at the latest tag also cannot use a
same-tag bridge. Do not claim seamless automatic migration.

`scripts/prepare-legacy-bridge.sh VERIFIED_GENERIC_ZIP NEW_OUTPUT_DIR` can prepare
byte-identical compatibility copies and checksums **locally only**. It neither
authenticates its input nor uploads anything and is excluded from all release
collectors. Verify the source first. It is preparation for a separately approved
rollout only, not a working automatic bridge for the current release inventory.
Before any such rollout, establish an eligible visible tag, unambiguous first
asset selection, source integrity and client-version eligibility. If that
requires altering historical assets or publishing device ZIPs, obtain an explicit
policy exception. Verified manual migration requires neither exception.

## Physical device-test checklist

Use the same checksum-verified public ZIP on each intended compatible device,
or record the exact scope of a separately approved unpublished test build. Record
ZIP SHA-256, version, board, firmware/build, architecture, glibc, SDL libraries,
renderer/output, controller name/GUID and Ports letter mode. Keep older reports
scoped to their original bytes; no new hardware pass is implied here.

- Cold launch and return to ES; no loader, library, signature or firmware errors.
- At 1280×720 and 640×480, check list/detail, contained art and readable hints.
  On available compatible small/square/portrait hardware, test compact layout;
  320×240 is a target, not a claim that such hardware has passed.
- Confirm/back/More, filter/category, L1/R1 warning paging, repeat, hotplug and
  Select+Y in both default and Ports Xbox modes. Start must not confirm.
- Page long Health/More/warning and both recovery confirmations to their last
  line and back without changing the action/filter or starting work.
- After manual legacy replacement, verify installed apps/settings/state are
  unchanged. Test generic-to-generic update, restart and recoverable interruption
  only on a backed-up test card and separately authorized release endpoint.
- Check real framebuffer rotation, acceleration/software fallback, performance
  and idle behavior. Desktop tests cannot establish these results.
