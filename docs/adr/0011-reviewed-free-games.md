# ADR 0011: Admit reviewed free games and homebrew ROMs

- Status: accepted
- Date: 2026-11-02
- Supersedes only ADR 0005's restriction to utilities, themes, and integrations.

## Context

The owner approved native free games and legally redistributable homebrew ROMs after review of the spruceUI Ports-and-Free-Games catalog. Free availability and spruce's `requires_files: false` do not establish distribution rights or KNULLI compatibility. Its mutable Nursery release and spruceOS layouts do not meet this Store's release and installation contracts.

## Decision

Add the `game` package type for native free games, homebrew ROMs, and freely redistributable game data used by a KNULLI-supplied engine. Require a verified license even for game candidates and upstream-managed games. Review rights for code, graphics, audio, levels, and all other bundled content; an engine license does not license commercial game data. Preserve required notices and attribution. Exclude commercial game files, packages that require them, unclear rights, and ROM-download services. Permission granted only to another distributor is not permission for this Store.

Keep PortMaster external-only under ADR 0005. Do not mirror its catalog, use its release archives, or replace its installer. Avoid duplicate Store entries and check PortMaster for conflicting ownership. A directly reviewed original-publisher game must not take ownership of PortMaster-managed paths or modify an existing PortMaster installation.

Use original publishers' version-pinned ZIP or tar.gz release assets with exact size, SHA-256, and the existing checksum evidence rules. Do not import Nursery assets, repack them as Store releases, or assume that matching hardware makes a spruce port compatible. No new archive format or remote installer is introduced.

Reuse the existing manifest, archive, transaction, preservation, and menu contracts. For an emulated game, `install.launcher` names the ROM or game-data file opened by EmulationStation; it is not made executable. Compatibility architecture and ABI describe the supported KNULLI host, not the ROM's instruction set. Require the actual emulator/core as a detected dependency and review the firmware's extension, system, launch, controller, and save configuration. Keep native executables in `install.executables`. No emulator or commercial BIOS is supplied by the Store.

Install under a private package directory within the correct KNULLI system, not a spruceOS root or a shared system directory. Declare only package and menu writes; do not claim shared emulator settings or saves. Review save and configuration paths, ensure repair and removal retain user data, and exercise installation and removal in a fixture root. Additional ROMs in a publisher bundle use the existing system scanner; the manifest's single menu entry describes its primary launch file.

Start new entries as experimental only after rights, release, runtime, and lifecycle review. Keep unresolved entries non-actionable or exclude them. Firmware, host ABI, dependencies, archive safety, and `/userdata` containment still gate operations. Exact physical-device evidence alone grants a tested badge.

Keep schema v1: this adds one type without changing existing field meanings. Old binaries reject the new type, so a game-containing bundled catalog must ship with a compatible Store binary.

## Consequences

- The Store can own and remove reviewed free games without becoming a ROM-download service or PortMaster replacement.
- Original notices and publisher bundles remain intact; a bundle may include more than one playable game but only one explicitly managed menu entry.
- Source review and fixture tests establish an integration contract, not physical-device launch, performance, audio, or controller results. Untested games remain visibly untested.
- Some free games cannot be added because the publisher supplies no supported pinned archive, checksum evidence, transferable permission, or compatible runtime.
