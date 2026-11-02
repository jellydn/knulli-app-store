# Free games review — 2026-11-02

## Scope and decision

The owner approved native free games and legally redistributable homebrew ROMs. [ADR 0011](adr/0011-reviewed-free-games.md) records this change while keeping PortMaster external-only. The new `game` type reuses the existing package lifecycle; it is not an emulator installer or a general ROM-download source. Games must have verified rights for all content, including candidates and upstream-managed games.

Discovery source: [spruceUI/Ports-and-Free-Games](https://github.com/spruceUI/Ports-and-Free-Games/tree/a981c7601bb4e47a28985b9ec6860311ae5e609e). Both `game.json` and `game_v2.json` were inspected. Spruce devices and minimum firmware versions are not KNULLI compatibility evidence. The [Nursery workflow](https://github.com/spruceUI/Ports-and-Free-Games/blob/a981c7601bb4e47a28985b9ec6860311ae5e609e/.github/workflows/game_nursery_release.yml) produces `.7z` files, deletes/recreates the selected release tag, and packages spruce-specific paths. None of those assets are used.

## Added: Freedoom: Phase 1 + Phase 2

One manifest, `io.github.freedoom.freedoom`, installs the original publisher's two-game bundle. It does not download the same bundle twice or split ownership of its files.

- [Publisher release v0.13.0](https://github.com/freedoom/freedoom/releases/tag/v0.13.0), source commit `cfb8644b1a8dc7d7d2177e6a892ccaa2922bdaae`.
- [Original ZIP](https://github.com/freedoom/freedoom/releases/download/v0.13.0/freedoom-0.13.0.zip): 24,143,781 bytes; SHA-256 `3f9b264f3e3ce503b4fb7f6bdcb1f419d93c7b546f4df3e874dd878db9688f59`.
- Independently calculated digest matches the [publisher's PGP-signed checksum text](https://github.com/freedoom/freedoom/releases/download/v0.13.0/freedoom-0.13.0-CHECKSUM). Signing-key identity was not independently authenticated. `release.immutable` means pinned bytes, not a claim of GitHub-enforced immutability.
- [BSD-3-Clause permission](https://github.com/freedoom/freedoom/blob/cfb8644b1a8dc7d7d2177e6a892ccaa2922bdaae/COPYING.adoc) covers the game's content. Retain COPYING, CREDITS, CREDITS-MUSIC and all manuals. No proprietary Doom IWAD, BIOS, or engine executable is supplied.
- Archive: one directory and ten regular files, 61,716,528 regular-file bytes. No unsafe paths, duplicates, links, or special entries. Only the single `freedoom-0.13.0/` prefix is stripped; no binary patches, custom mappings, or executable modes.

| Installed file | Bytes |
| --- | ---: |
| COPYING.txt | 1,644 |
| CREDITS-MUSIC.txt | 9,388 |
| CREDITS.txt | 15,872 |
| NEWS.html | 33,827 |
| README.html | 34,415 |
| freedoom-manual-en.pdf | 1,333,497 |
| freedoom-manual-es.pdf | 1,347,608 |
| freedoom-manual-fr.pdf | 1,357,453 |
| freedoom1.wad | 28,795,076 |
| freedoom2.wad | 28,787,748 |

Both game files have valid bounded IWAD directories. Phase 1 SHA-256 is `7323bcc168c5a45ff10749b339960e98314740a734c30d4b9f3337001f9e703d`; Phase 2 is `a8772e088847032510d97ba2312406a6998f21cbab44d4ff10696faa9c0ecd4b`.

### KNULLI runtime and paths

Source review uses KNULLI distribution commit `ab3a6809286b3cff1648fb8c96a0200bc869beb6`:

- [System configuration](https://github.com/knulli-cfw/distribution/blob/ab3a6809286b3cff1648fb8c96a0200bc869beb6/package/batocera/emulationstation/batocera-es-system/es_systems.yml) declares the `prboom` system, `.wad` support, and the libretro PrBoom core. Both files are IWADs, not mods that need commercial data.
- [Core build recipe](https://github.com/knulli-cfw/distribution/blob/ab3a6809286b3cff1648fb8c96a0200bc869beb6/package/batocera/emulators/retroarch/libretro/libretro-prboom/libretro-prboom.mk) pins PrBoom commit `6ec854969fd9dec33bb2cab350f05675d1158969`, includes an AArch64 build path, and installs `/usr/lib/libretro/prboom_libretro.so`. The Store detects that core separately from `/usr/bin/retroarch`; missing either blocks installation. Host architecture and ABI remain AArch64/glibc. No native instruction-set claim is made about WAD data.
- [Core source](https://github.com/libretro/libretro-prboom/blob/6ec854969fd9dec33bb2cab350f05675d1158969/src/d_main.c) recognizes both Freedoom filenames. It uses internal default PrBoom data when a separate `prboom.wad` is absent, so nothing is copied from shared firmware data into the private directory.
- Store destination: `/userdata/roms/prboom/Freedoom`. Both launch files sit one directory below the system root. The sole managed menu entry is `./Freedoom/freedoom1.wad` in `/userdata/roms/prboom/gamelist.xml`. Phase 2 uses normal system scanning; refresh game lists and browse PrBoom/Freedoom. A custom “parse gamelists only” configuration may require manual metadata for Phase 2. No scanner result is claimed from a physical device.
- [KNULLI configgen](https://github.com/knulli-cfw/distribution/blob/ab3a6809286b3cff1648fb8c96a0200bc869beb6/package/batocera/core/batocera-configgen/configgen/configgen/generators/libretro/libretroConfig.py) sets savefile and savestate directories to `/userdata/saves/prboom` and supplies RetroPad controller defaults. The [core](https://github.com/libretro/libretro-prboom/blob/6ec854969fd9dec33bb2cab350f05675d1158969/libretro/libretro.c) uses per-WAD `freedoom1`/`freedoom2` subdirectories for its saves and configuration; it falls back to the content directory if the save directory is unavailable. Store operations never write these shared save paths. Runtime-created local saves/configuration remain unowned; `prboom.cfg` is explicitly preserved.
- Removal deletes owned release files and only an unchanged Store-created Phase 1 menu entry. Saves, settings, pre-existing metadata, other WADs, shared `prboom.wad`, and other installations remain. Empty directories may remain. Close the game before Store operations.
- Network: GitHub for installation; no runtime downloader or updater in the payload. Ordinary single-player gameplay needs no network. KNULLI supplies the engine, rendering, audio, configuration and controller mapping.

[PortMaster Doom Engines](https://github.com/PortsMaster/PortMaster-New/blob/15b1a91425f915ec604cc0bf062bd5c657e53c83/ports/doomengines/port.json) also bundles Freedoom. It owns its separate Ports launcher and `doomengines` tree. This Store package uses neither that archive nor those paths. Users who already play Freedoom there do not need a second copy. Grout was not duplicated. The separately reviewed zitch integration is outside this game addition.

## Exclusions and deferred entries

These are findings from the reviewed shortlist, not a claim that every other source entry is illegal or incompatible.

| Entry or group | Reason not added |
| --- | --- |
| Cannonball, Resonance, Sonic Mania | Source metadata says additional commercial files are required. Hidden status does not make those files acceptable. |
| Doom1 | Shareware/proprietary data, not the reviewed freely licensed replacement. No commercial Doom files are included. |
| Jazz Jackrabbit shareware, Rick Dangerous, Streets of Rage Remake, Super Mario War | No verified rights to all bundled commercial-IP assets established in this review. Free download and an open engine license are insufficient. |
| Sumset, Sudoku Thrice | Bundled notices grant inclusion specifically in spruce Game Nursery, not permission for this Store. See the pinned source notices under Game Boy Color/Roms/GBC/licenses. |
| Tobu Tobu Girl Deluxe | Original project declares MIT code and CC-BY-4.0 assets, but its GitHub repository has no published release assets. Do not substitute a Nursery ZIP or an expiring itch.io download link. |
| GB-Wordyl | GitHub releases inspected through v1.0.6 have no playable release assets; upstream directs ROM downloads to itch.io. No supported pinned publisher archive reviewed. |
| Super Tilt Bro | GitHub releases API returned no releases. The current source also describes a RAINBOW mapper not supported by ordinary emulators; that does not prove the older Nursery ROM incompatible, but prevents assuming current source is a compatible substitute. |
| Super Sudoku | Publisher repository has version-named ROMs in `releases/`, but no GitHub releases or tags. Raw ROMs/source snapshots are not accepted release ZIP/tar.gz assets under the unchanged release contract. |
| Sure Instinct | MIT source is available; playable versioned ZIPs are offered through itch.io. No stable publisher release URL plus required checksum evidence was established here. |
| Anarch, 2048 | Inspected spruce launchers hardcode `/mnt/SDCARD/RetroArch`; Anarch bundles an ARM32 core. No compatible original-publisher KNULLI release was established. Do not infer AArch64 support. |
| Other native ports, ROMs, EasyRPG, J2ME, Game Tank, ZX Spectrum, C64, Arduboy and eBooks | Not promoted: complete original-release, rights and KNULLI runtime/lifecycle review was not completed. eBooks are outside this game addition. No blanket licensing or compatibility conclusion is made. |

## Verification limits

New tests cover licensed game acceptance, unlicensed/blank-license game rejection (including upstream-managed and candidate cases), and the PrBoom prerequisite. A small offline archive fixture covers the new manifest's install/repair/removal behavior, non-executable WADs, retained local/shared saves, unrelated firmware and PortMaster data, and menu ownership.

The actual publisher archive also passed install, repair, health/hash and removal checks in a disposable fixture root, including existing menu ownership and a menu failure that leaves no installed WADs. This validates the real pinned download/extraction path and installed bytes; it does not execute a downloaded installer or launch a game on KNULLI.

An x86_64 build of KNULLI's pinned PrBoom source loaded each original IWAD with an empty system directory (no external `prboom.wad`). A temporary libretro harness ran 400 cycles per game with simulated Start/A input: each produced 403 video callbacks, with 109 distinct frame hashes for Phase 1 and 322 for Phase 2, and created its per-WAD save/configuration directory. The harness initially omitted the logging interface required by this core; adding that callback resolved its null-function crash. This is a host smoke check, not a KNULLI frontend or device result, and does not prove completed levels or in-game save/reload.

Physical device startup, menu scanning, controller input, audio, performance and in-game save/reload still require testing. Neither phase earns a tested badge. The publisher checksum signature was not authenticated against an independently trusted signing key.

JSON Schema checks passed for the new manifest, 16 license/status/upstream-managed game cases, the existing utility license exception, and unknown-type rejection. A catalog-wide schema probe also found an existing unrelated mismatch: `io.github.squirbie.handheld-link-cores` has an empty `compatibility.minimum_version`, while the schema requires a nonempty value. That entry was not changed; the Go catalog validator accepts it.
