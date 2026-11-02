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

## Complete source inventory — follow-up review

The follow-up inspected all **85 game/runtime entries** at the same spruce commit (including hidden entries and `game_v2.json`), plus the three app entries and fourteen eBooks for scope. It read the game metadata, bundled notices and native launchers. The table below accounts for every game/runtime entry. A missing rights grant or a missing reviewed release is a blocker, not a finding that the game is illegal. A compatible emulator alone does not clear release provenance, asset rights or save-path review.

**No additional game cleared every gate.** Freedoom's two source entries remain one Store package. No Nursery asset was imported, no candidate with unclear asset rights was added, and no checksum exception was broadened. In particular, a GitHub-generated digest is not a publisher checksum for an ordinary game-data package; the existing exception is limited to authorized upstream-managed experimental packages. These exclusions can be reviewed again when the missing evidence is available.

Source links below refer to the pinned [spruce tree](https://github.com/spruceUI/Ports-and-Free-Games/tree/a981c7601bb4e47a28985b9ec6860311ae5e609e). Bundled notices are discovery evidence, not independent publisher authorization. Original-publisher checks are listed separately below.

| Source group / every entry | Blocking evidence or result |
| --- | --- |
| Arduboy: Ardynia; CastleBoy; Catacombs of the Damned; Dark and Under; Nineteen44 | Notices show MIT/BSD code or conditional whole-game distribution (Ardynia); they do not by themselves authenticate an original release. KNULLI does declare Arduboy with the Arduous core, so these are **not** excluded as an unsupported system. Catacombs' publisher release 1.0 supplies `Catacombs.arduboy`, but no publisher checksum asset/text was found in the inspected release. The other entries have no fully reviewed publisher archive/checksum pair. No spruce HEX/ZIP is substituted. CastleBoy/Nineteen44 also require care over demake asset rights. |
| Commodore 64: Balls Like A Frog; C64anabalt; Doc Cosmos; Endless Forms Most Beautiful; Fire!; Lester; LuftrauserZ; Mario's Cement Factory; Old Tower; Ooze; Robot Jet Action; Robots Rumble; Super Bread Box 64; Tenebra; Yoomp! 64; You Have To Win The Game | Every bundled notice says only that the title was hosted online for free, with an opt-out email. That is not a rights grant for code and assets. No original-publisher rights plus pinned archive/checksum contract was established. Official conversions and free download prices do not supply it. |
| Doom: Doom1; Freedoom1; Freedoom2 | Doom1 is proprietary/shareware. Freedoom1 and Freedoom2 are already supplied together by the reviewed original v0.13.0 archive. No duplicate package. |
| EasyRPG: Dandelion; Dead Again; Grimm's Hollow; Ib; Lavender; OFF; Return of Touhou Mother; Spring Eternal; Touhou Mother; Yume Nikki | Notices link archive.org, itch.io, Steam or fan translations rather than granting complete redistribution rights. Ib's notice expressly restricts redistribution of images/music. No complete publisher release/rights contract for the game and bundled RPG Maker assets was established. EasyRPG support is not the blocker by itself. |
| Game Boy Advance: Apotris | Publisher offers versioned GBA ZIPs, including v4.2.0. The download page has no publisher checksum, and GitHub lists no releases. Thus a self-calculated digest would not satisfy the required checksum evidence. Do not replace the GBA build with a PortMaster archive. |
| Game Boy Advance: Sips | Music attribution exists, but the notice does not establish all game/audio rights or a pinned publisher archive/checksum pair. |
| Game Boy Color: Metamorphosis Collection DEMO | No complete bundled rights notice or reviewed original-publisher versioned archive/checksum contract established. A free demo is not redistribution permission. |
| Game Boy Color: Nine Lives Sudoku; Sudoku Thrice; Sumset | Notices grant inclusion specifically in spruce Game Nursery. No grant to this Store was established. |
| Game Boy: GB-Wordyl; Tobu Tobu Girl Deluxe | Known open-source projects, but inspected publisher GitHub releases contain no playable assets. Do not use Nursery copies or source snapshots in place of a reviewed playable release. See the detailed findings below. |
| Game Boy: The Bouncing Ball | Notice cites CC-BY-SA-4.0 and author download pages, but no authenticated version-pinned publisher ZIP/tar.gz plus checksum pair was established. |
| Game Boy: The Mayor of Sanctuary | Notice identifies the author and termination of a prior distributor's license, not a grant to this Store. No complete rights/release contract established. |
| Game Tank: A Very Hard Game; Accursed Fiend; Brick Game; Cubicle Knight; Sivani; Tetris | No GameTank system/runtime is declared in the reviewed KNULLI system matrix. These spruce entries install their emulator under `Emu/GAMETANK`; that is not KNULLI integration. Brick Game additionally records permission to spruce only. MIT notices for other entries do not resolve runtime/release requirements. |
| J2ME: FreeJ2ME | A runtime bundle plus Java and three midlets, not one reviewed game release. The bundled core/runtime and Bomber 2, jtris and Crossword Solver need separate publisher/rights/ABI review; spruce's device list is not KNULLI evidence. No runtime or extra games imported. |
| NES: From Below | Bundled notice records permission specifically to the spruce team with free-access conditions. No independent Store distribution grant or reviewed publisher archive/checksum pair established. |
| NES: Squatch Quest Demo | Notice describes a prerelease demo and commercial full-game link, not a full redistribution grant. No reviewed publisher archive/checksum contract established. |
| NES: Super Tilt Bro | Publisher releases API lists no releases. Current mapper support also needs review; see below. |
| Ports: 2048; Anarch; CaveStory; Dinothawr; Mr. Boom; Pong (Gong) | Spruce launchers call `/mnt/SDCARD/RetroArch` and private supplied cores; these are not KNULLI launch paths. Dinothawr has no publisher GitHub release assets; the advertised historical downloads are HTTP Windows bundles. Mr. Boom publishes source tarballs, not a reviewed KNULLI runtime package; KNULLI already declares its own MrBoom system/core. No Store package should claim those shared firmware paths or just install a dummy launcher. CaveStory's engine license does not establish data rights. |
| Ports: Cannonball; Resonance; Sonic Mania | Metadata explicitly requires additional commercial files. Excluded, including the hidden entries. |
| Ports: Celeste Classic | Original ccleste releases inspected through v1.4.0 supply Windows ZIP and 3DS assets, not a supported KNULLI AArch64 package. Spruce launcher sends signals to `joystickinput`; do not copy it. |
| Ports: Digger | Publisher v0.5 supplies source, Windows/macOS and Debian/Ubuntu package bundles, not a reviewed directly extractable KNULLI runtime/launcher. No remote build/install script or nested system package extraction is introduced. |
| Ports: Dino Jump; Planet Merge; nKaruga | Spruce launchers use spruce processes, library paths or `/mnt/SDCARD` save/runtime paths. No reviewed original-publisher KNULLI archive with complete asset rights and required checksum established. |
| Ports: Hydra Castle Labyrinth | Port publisher v0.1.0 now has ARM64 ZIPs, so absence of ARM64 builds is **not** claimed. No publisher checksum asset/text or complete original game-data rights was established. Spruce's GPL engine notice alone does not license all data. |
| Ports: L'Abbaye des Morts | Inspected publisher releases supply an x86_64 glibc-2.35 archive (v2.0.4), not a reviewed KNULLI AArch64 package. The spruce launcher also depends on `joystickinput` and spruce paths. |
| Ports: Project Starfighter; Rocks'n'Diamonds | Open engines do not make spruce bundles portable. Launchers depend on spruce libraries/processes/save paths. No reviewed original-publisher directly extractable KNULLI release with all bundled-content rights and checksum evidence established. Do not substitute PortMaster builds. |
| Ports: Jazz Jackrabbit; Rick Dangerous; Streets of Rage Remake; Super Mario War | No verified rights for all proprietary/shareware or commercial-IP assets established. An engine's open license or a free download is insufficient. |
| Ports: Tyrian 2000 | OpenTyrian publishes ARM64 archives, but OpenTyrian is not proof of rights or runtime support for spruce's separate OpenTyrian2000 payload/data. No reviewed Tyrian 2000 publisher data release/checksum contract established. No commercial data or PortMaster asset substituted. |
| SNES: Super Sudoku; Sure Instinct | MIT notices exist, but no supported publisher release/archive/checksum contract established; see details below. |
| ZX Spectrum: Automated Cave Explorer; Che-Man; Dreamwalker; Gravibots; Mars Mare-Alienation; Nixy and the Seeds of Doom | Every bundled notice is the same free-hosting/opt-out statement used for C64. It does not grant distribution rights for code and assets. No complete publisher rights/release evidence established. |

### Original-publisher release checks

- [Apotris downloads](https://akouzoukos.com/apotris/downloads): inspected the full download page, including the versioned [v4.2.0 GBA ZIP](https://apotrisstorage.blob.core.windows.net/binaries/Apotris-v4.2.0GBA.zip). ARM64 desktop builds also exist; do not claim otherwise. No checksum is published on that page. [GitHub releases](https://api.github.com/repos/akouzoukos/apotris/releases) returned an empty list.
- [Catacombs of the Damned 1.0](https://github.com/jhhoward/Arduboy3D/releases/tag/1.0): one `.arduboy` asset; no separate publisher checksum or checksum in the release text. An `.arduboy` may itself be ZIP data; the filename alone is not a compatibility rejection.
- [ccleste v1.4.0](https://github.com/lemon-sherbet/ccleste/releases/tag/v1.4.0), [Digger v0.5](https://github.com/sobomax/digger/releases/tag/v0.5), [Hydra Castle Labyrinth v0.1.0](https://github.com/ptitSeb/hydracastlelabyrinth/releases/tag/v0.1.0), [L'Abbaye des Morts releases](https://github.com/nevat/abbayedesmorts-gpl/releases), [MrBoom 5.5](https://github.com/Javanaise/mrboom-libretro/releases/tag/5.5), and [OpenTyrian v2.1.20260913](https://github.com/opentyrian/opentyrian/releases/tag/v2.1.20260913): inspected release text and asset inventories, rather than assuming that the spruce binary is the publisher's supported package.
- [Dinothawr publisher](https://github.com/Themaister/Dinothawr): no GitHub releases; README advertises historical HTTP Windows/Android bundles and describes the separately needed data/core. A mutable libretro content download is not a version-pinned publisher release.
- [KNULLI system matrix](https://github.com/knulli-cfw/distribution/blob/ab3a6809286b3cff1648fb8c96a0200bc869beb6/package/batocera/emulationstation/batocera-es-system/es_systems.yml): confirms Arduous, GBA, GB/GBC, NES, SNES, PrBoom, EasyRPG and MrBoom configuration. No GameTank entry. This is source review, not proof that any particular device build includes every optional core.

The three Apps entries are outside the game import: Grout and zitch are apps, and AestheticSpruce is a spruce theme tool, not a game. All fourteen eBooks are outside the game scope. PortMaster remains external-only.

## Earlier detailed exclusions

These detailed findings remain useful alongside the complete source inventory above. They are not claims that unaccepted games are illegal.

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

## Verification limits

New tests cover licensed game acceptance, unlicensed/blank-license game rejection (including upstream-managed and candidate cases), and the PrBoom prerequisite. A small offline archive fixture covers the new manifest's install/repair/removal behavior, non-executable WADs, retained local/shared saves, unrelated firmware and PortMaster data, and menu ownership.

The actual publisher archive also passed install, repair, health/hash and removal checks in a disposable fixture root, including existing menu ownership and a menu failure that leaves no installed WADs. This validates the real pinned download/extraction path and installed bytes; it does not execute a downloaded installer or launch a game on KNULLI.

An x86_64 build of KNULLI's pinned PrBoom source loaded each original IWAD with an empty system directory (no external `prboom.wad`). A temporary libretro harness ran 400 cycles per game with simulated Start/A input: each produced 403 video callbacks, with 109 distinct frame hashes for Phase 1 and 322 for Phase 2, and created its per-WAD save/configuration directory. The harness initially omitted the logging interface required by this core; adding that callback resolved its null-function crash. This is a host smoke check, not a KNULLI frontend or device result, and does not prove completed levels or in-game save/reload.

The [owner's later report](real-device-tests.md#2026-11-02-freedoom-phase-1-and-phase-2-on-tsp) says both phases work well on TSP with experimental Store v2026.11.02-f193885. It supplies no firmware, architecture, observed resolution or separate menu/controller/audio/performance/lifecycle/save results. Those details still need testing. Neither phase earns an exact-matrix tested badge. The publisher checksum signature was not authenticated against an independently trusted signing key.

JSON Schema checks passed for the new manifest, 16 license/status/upstream-managed game cases, the existing utility license exception, and unknown-type rejection. A catalog-wide schema probe also found an existing unrelated mismatch: `io.github.squirbie.handheld-link-cores` has an empty `compatibility.minimum_version`, while the schema requires a nonempty value. That entry was not changed; the Go catalog validator accepts it.
