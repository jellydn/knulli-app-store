# Release audit — 2026-11-04

The owner authorized the next normal green-main experimental release, including
the responsive/measured-footer renderer and one generic ZIP. No new physical
device result was supplied. All fourteen catalogue entries remain experimental;
historical device evidence stays attached to its exact package version.

## Per-entry decision

Checked the original publisher release APIs on 2026-11-04, including skipped
prereleases, and the Apotris publisher download page. Re-downloaded every current
manifest asset: all fourteen pinned sizes and SHA-256 values matched. New eligible
bytes were downloaded independently as described below. A GitHub release being
version-pinned is not a claim of server-enforced immutability.

| Entry | Before → release version | Decision and remaining runtime constraints |
| --- | --- | --- |
| [Grout](https://github.com/rommapp/grout/releases/tag/v5.4.1.0) | 5.4.1.0 → unchanged | Latest stable. MIT. Existing AArch64/glibc and SDL/image/ttf requirements; user-supplied RomM content/credentials, config/save-slot/cache preservation and upstream-managed runtime warnings remain. |
| [Apotris](https://akouzoukos.com/apotris/downloads) | 4.2.0 → unchanged | Publisher still links 4.2.0 GBA ZIP. AGPL-3.0 source/assets and bundled third-party notices retained. Rechecked raw tagged licence; the Gitea HTML source view redirects to sign-in, so HTML HTTP 200 alone is not source access evidence. Firmware RetroArch/mGBA required; audio and shared saves retained. No publisher checksum; existing ADR 0012 warning remains. |
| [Freedoom](https://github.com/freedoom/freedoom/releases/tag/v0.13.0) | 0.13.0 → unchanged | Latest stable. BSD-3-Clause original free WAD data and publisher checksum retained. Firmware RetroArch/prboom required; no commercial Doom data or engine bundled; config/shared saves retained. |
| [Dark & Under](https://github.com/Garage-Collective/Dark-And-Under/releases/tag/V2.0.3) | 2.0.2 → unchanged | 2.0.3 fixes a defending exploit but offers only raw HEX, not a supported ZIP/tar.gz archive. Do not repackage a third-party asset or relax archive policy. Retain original BSD-3-Clause 2.0.2 Arduboy ZIP, licence in signed review notes, RetroArch/Arduous gate and checksum warning. |
| [zitch](https://github.com/itchio/zitch/releases/tag/v0.3.0) | 0.3.0 → unchanged | Latest stable. MIT and publisher checksum retained. Original publisher PortMaster-format ZIP, not a PortMaster-hosted release. Existing PortMaster/system SDL/glibc requirements, preserved private home/accounts/games and runtime download warnings remain. No ROM server is supplied. |
| [Random Game Picker](https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.2.2) | 0.2.2 → unchanged | Latest stable. MIT, publisher checksum, existing SDL/glibc and launcher requirements retained; state directory preserved. No runtime game acquisition. |
| [RetSend](https://github.com/mxmgorin/retsend/releases/tag/v1.0.0) | 1.0.0 → unchanged | Latest stable. GPL-3.0 and publisher checksum retained; SDL/glibc required. Network transfer of user-owned files remains outside installation side effects; unowned identity/inbox data survives. |
| [Catacombs of the Damned](https://github.com/jhhoward/Arduboy3D/releases/tag/1.0) | 1.0 → unchanged | Latest release. MIT covers original code/assets; notice stays in signed review notes. Firmware RetroArch/Arduous, shared save retention and ADR 0012 checksum warning remain; no commercial data. |
| [Knulli WebUI](https://github.com/kubrick89/knulli_webui/releases/tag/1.2.2) | **1.2.1 → 1.2.2** | Eligible under existing authorized upstream-managed direct-download exception. Manual patch/source/security review below. No verified app licence or independent publisher checksum; warnings retained. Existing Python 3.7+/Bash/utilities and service-cleanup constraints remain. |
| [RAOfflineProxy](https://github.com/misantronic/RAOfflineProxy/releases/tag/v2.3.0-alpha1) | 2.3.0-alpha1 → unchanged | Latest non-prerelease-flagged release still has an alpha version name. GPL-3.0. Embedded payload is decoded as data, never executed as an installer. AArch64/glibc, CPython 3.12/pygame, hook/PID/proxy cleanup, cached credentials/awards and upstream updater warnings remain. |
| [Handheld Link Cores — mGBA Dual](https://github.com/Squirbie/handheld-link-cores/releases/tag/2026.10.29.1) | 2026.10.29.1 → unchanged | Latest release. MPL-2.0, publisher checksum, glibc **2.35** and existing RetroArch required. Only reviewed private mGBA Dual core: exclude all-cores bundle, shared integration scripts and other cores. ROMs/BIOS not supplied; config/saves/states preserved. Store GUI glibc 2.34 does not lower this package's 2.35 requirement. |
| [PocketCurator](https://github.com/tomtombombadil/PocketCurator/releases/tag/v1.1.2) | 1.1.2 → unchanged | Latest stable. Newer 1.1.13 is explicitly a prerelease/candidate; publisher says stable remains 1.1.2. MIT, checksum and verified launcher patch retained. PortMaster, permanent-ROM-deletion warning, preserved settings and upstream runtime downloader/update warning remain. |
| [PlayTime](https://github.com/unitreign/playtime/releases/tag/v1.1.1) | 1.0.0 → unchanged | 1.1.1 downloaded/inspected, but no publisher checksum is published. GitHub digest alone is allowed only for upstream-managed experimental packages; this package is not one. Do not silently change that policy or mark it upstream-managed to bypass the gate. GPL-3.0 source and AArch64/SDL/glibc 2.34 remain; statistics stay outside the managed directory. |
| [retroshelf](https://github.com/retroshelf-org/linux/releases/tag/v0.6.10) | **0.6.3 → 0.6.10** | Eligible under the existing upstream-managed direct-download exception, with publisher checksum and unchanged launcher/cleanup contract. Opaque engine, licence/source access and runtime update/migration limits remain explicit below. |

No entries were removed or promoted. No emulator, runtime, PortMaster installation
or firmware library was upgraded. Dependencies remain detection gates, not content
installed implicitly by a game. No source-only archive or remote installer was
substituted for a playable release. All original rights notices remain.

## Eligible release evidence

### WebUI 1.2.2

- Original publisher ZIP: **1,107,782 bytes**, SHA-256
  `dcfdfe3075176c99fbb2229d869512f2ca11fc543e9ae19c25e6572dae84c161`.
  Matches the [release API](https://api.github.com/repos/kubrick89/knulli_webui/releases/tags/1.2.2).
  **112 regular files, 1,438,748 extracted bytes**, no unsafe entries. Most removed
  files are Windows Zone.Identifier streams; no new native binary is bundled.
- [Tagged source](https://github.com/kubrick89/knulli_webui/tree/557d81db0b07ba338bbc907664751292d8d2b0f3)
  matches packaged Python except that the ZIP corrects VERSION from 1.2.1 to 1.2.2.
  Service bytes and the source-byte/final-hash-verified private-path patch are
  unchanged. Store paths, preserved config and removal guards do not widen.
- Reviewed credential setup, salted hashes, auth cache, config persistence and
  unchanged upload/path/input defaults. **The new prompt does not require auth**:
  empty credentials still allow requests. A reachable unconfigured service can be
  claimed by another client. Set credentials offline before first start. Basic
  auth has no TLS; use trusted private networking or loopback plus an encrypted
  tunnel, and disable unnecessary writes/remote control. Upload symlink escape
  and writable Tools scripts remain upstream-managed risks, not fixed security.
- Application-code licence remains unverified; retained console icons have their
  unchanged CC BY 4.0 notice. Direct publisher download only; no mirroring or new
  license identifier. See the original [service/config guide](knulli-webui-1.2.1.md).

### retroshelf 0.6.10

- Original KNULLI ZIP: **19,976,898 bytes**, SHA-256
  `bb42d92fe21ab2e56407d391ac3d0f5320962b01f6b80dee020ae4f42c36e4dd`.
  Matches [publisher checksums](https://github.com/retroshelf-org/linux/releases/download/v0.6.10/checksums.txt)
  and the [release API](https://api.github.com/repos/retroshelf-org/linux/releases/tags/v0.6.10).
  **Eight regular files, 43,352,008 extracted bytes**, no unsafe entries.
- Only UI, sync engine and README changed from 0.6.3. Launcher, uninstall wrapper,
  icon and both bundled SDL2_gfx copies are byte-identical. UI is AArch64 with
  GLIBC_2.17, SDL2/image/ttf/gfx, pthread, resolv and libc; engine remains static.
  Bundled gfx requires **GLIBC_2.29**, libm/SDL2/libc/loader. Added explicit `libm`
  detection to account for this existing bundled-library requirement.
- [Public release source](https://github.com/retroshelf-org/linux/tree/4646af742d1f89a32158d18f5575ff58c544daa5)
  is metadata only; the linked implementation still returns HTTP 404. Licence,
  opaque engine hook/update behavior and real database migration cannot be fully
  reviewed from source. Existing direct-download experimental exception and
  visible licence/runtime warnings remain; no hardware badge or source audit is
  invented. Close app, run upstream `--keep-data` cleanup manually, then remove
  through the Store; the Store never runs the script or deletes shared hooks.

## Lifecycle and release gates

Real pinned archives are exercised as data through `installer.Manager` in
temporary roots and a local TLS fixture server. The lifecycle probe covers all
fourteen entries: install, version/health, damaged managed entry, Repair, every
declared removal guard, removal, all declared preserve paths, unowned local data
and synthetic shared save/statistics/account/award records. The two updated apps
also install their old pinned versions before updating. No upstream executable,
launcher, Python module or installation script is run by this probe. No real
account database, firmware library, controller or ROM/save is changed.

The temporary real-archive lifecycle probe passed all fourteen entries with the
race detector on 2026-11-04 (93.762 seconds). It was removed after validation;
the retained installer and rendering regressions remain in the normal suites.

The release must pass full non-SDL/SDL race suites, both vet/Staticcheck tag sets,
coverage, catalogue determinism, workflow/script checks, and normal green-main
CI before publishing its exact artifacts. Public release verification must
confirm exactly one generic installable ZIP plus `SHA256SUMS.txt`, all seven
canonical paths, pinned package metadata, catalogue ed25519 verification against
the key embedded in that GUI, and the unchanged ABI gates. Public catalogue
signing is distinct from independent signing of the Store ZIP; update trust
remains GitHub HTTPS and the publisher's account.

## Migration and physical test gate

See [distribution/manual migration](store-distribution.md). Legacy device-named
clients cannot discover the generic filename: verify and replace all seven Store
files together with the device shut down, retaining apps/settings/state. No
duplicate device ZIPs or automatic bridge are published. New generic clients use
the common update profile. Never mix GUI/catalogue/signature bytes between builds.

Physical footer/device testing is **outstanding**. Test actual dimensions
(including TSP 1280×720), long hints in Library/More/Health/confirmation/busy/error,
full warning and recovery paging, physical letters/repeat/hotplug/Select+Y in both
existing input policies, resize/output changes where supported, readability,
performance, safe-card recovery and ES return. Changed WebUI and retroshelf need
new real-version startup/config/account-migration checks. Game play/save-load and
core netplay remain separate physical tests. No desktop result creates such evidence.
