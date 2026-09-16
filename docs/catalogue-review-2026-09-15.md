# Catalogue review — 2026-09-15

## Grout 5.2.0.0

The reviewed upstream tag is `v5.2.0.0` at commit `fd470ab8e1ff63d643432727724b5b05b9317ced`. The Knulli asset is `Grout-Knulli.zip`, GitHub asset `566533951`:

- URL: `https://github.com/rommapp/grout/releases/download/v5.2.0.0/Grout-Knulli.zip`
- Compressed size: 24,277,451 bytes
- Extracted regular-file size: 47,590,773 bytes
- SHA-256: `bd1d4eea3cc262a50cf6c58369d95a1c3c474d95e7f4745406f2a5bd449e3b9d`
- License: MIT
- Launcher: `/userdata/roms/tools/Grout/Grout.sh`, mode `0755`
- Program: `/userdata/roms/tools/Grout/grout`, mode `0755`
- Other files: `logo.png`, `README.md`, `LICENSE`, and `lib/libSDL2_gfx-1.0.so.0`, mode `0644`
- ABI: Linux AArch64, `/lib/ld-linux-aarch64.so.1`, GLIBC 2.17 symbols
- Runtime libraries: SDL2, SDL2_image, SDL2_ttf, libc, libresolv, and pthread. The asset includes SDL2_gfx.

GitHub reports the release itself as not immutable. The Store therefore treats the versioned URL only as a locator and pins the accepted bytes with the exact size and SHA-256 above. An upstream replacement cannot install because verification occurs before extraction. The exact extracted inventory contains six regular files and two directories. The Store rejects links, special files, duplicate paths, traversal, excessive extracted size, and undeclared execute paths.

Grout has no supported setting to disable its updater. The reviewed binary contains one updater metadata URL at byte offset 9,490,992. During staging, the Store replaces that 36-byte value with `https://self-update.disabled.invalid`. It then requires the transformed binary SHA-256 `39b5ba053913620aea2db051c2fad2fa0bf05b59c2cc88bcb01734dd048882e7`. A source mismatch or result mismatch stops the operation before the transaction starts. This keeps the upstream asset as the pinned provenance input and prevents Grout from obtaining updater metadata outside Store controls.

Grout writes configuration and state under its install directory: `config.json`, `save_slots.json`, `.cache/`, and `logs/`. These paths are preserved. Its normal function can also write selected ROM, artwork, BIOS, and save content under `/userdata/roms`, `/userdata/bios`, and `/userdata/saves`. The Store itself writes only the reviewed package files and a proven tool-menu entry.

Migration from 5.1.0.0 keeps the same launcher, install path, ABI, library set, and configuration format. Version 5.2.0.0 fixes save-sync path resolution and targets RomM 5.2.0. The main migration risks are using it with an older unsupported RomM server and replacing install-relative user state. Store update, repair, rollback, force reinstall, and uninstall tests preserve the declared state paths. Do not replace the complete Grout directory by hand.

Compatibility is broad but experimental. Installation needs detected Knulli, AArch64, the Linux glibc ABI at 2.17 or later, all declared SDL libraries, a non-empty device identity, and a validated display from 640×480 through 1280×720. The existing real-device report is for Grout 5.1.0.0 on TrimUI Smart Pro. It does not give 5.2.0.0 or another device a verified badge.

## RAOfflineProxy 1.13.0-alpha1

RAOfflineProxy remains blocked. The current Knulli asset is GitHub asset `540054898`, 510,188 bytes, SHA-256 `cd3e8b2eb891751b7f80463e9cbcc5b9065668d26a20c14acf2d8f63191fcda3`. It is a mutable self-extracting shell program, not an inert supported archive.

A CI job could decode its embedded base64 tar payload without execution, but release approval still needs all of these items:

- complete path, mode, size, and hash inventory;
- proof that the native `libraproxy_rchash.so` matches reviewed source and AArch64/glibc requirements;
- a reviewed and packaged Python/pygame/SDL runtime;
- complete GPL and third-party source and notice compliance;
- removal of the unchecked shell-asset updater;
- transactional replacements for edits to `custom.sh`, Knulli/Batocera settings, and RetroArch files;
- an exact preservation and uninstall policy for credentials, databases, queued awards, keys, caches, and logs;
- exact device, display, controller, privilege, and network evidence.

The Store does not execute or repackage this asset until these requirements are complete.

## PocketCurator

PocketCurator remains blocked. The latest stable release is mutable `v1.1.2`. Its ZIP is 5,903,925 bytes with SHA-256 `bb1907289bb5e22c6d24c39b9fae6dac0c17b75821a6565d77fcba3cce9af664`. The newer `v1.1.13` is a mutable prerelease at commit `24c99096774748886468f977b73a59979f9518d2`; its ZIP is 5,905,782 bytes with SHA-256 `cbb0fb49787da17a689df1e8cdc1e5e2418cdcd3a871f22421a80c1d65e9279b`.

A safe Store-owned package would require a reviewed deterministic transformation from pinned source. It must remove the remote installer, fail-open updater, dormant `netsync.py` executable downloader, first-run runtime downloads, broad library discovery, detached metadata scripts, and service-control behavior. It must also pin or reproducibly build every native input, keep MIT/LGPL/OFL notices and LGPL source obligations, contain all ROM/media paths below reviewed Knulli roots, define preservation and uninstall behavior, and receive exact Knulli device tests. The core ROM-download feature also needs an explicit catalogue policy decision. These requirements are unresolved, so no PocketCurator bytes are installable.

## Recovery backups

After **Manage existing** fails, the recovery screen shows the exact reason and offers **Retry manage**, **Export diagnostics**, and **Cancel**. **Force reinstall** is added only for a recoverable ownership failure or a validated stale transaction journal. An active operation and compatibility, free-space, signature, checksum, archive, path, and licensing failures remain blocked. Force reinstall needs two confirmations and cannot disable any mandatory check.

Before a write, the Store inventories and hashes the complete package destination. It saves every regular file and a path/hash/mode/size manifest under:

`/userdata/system/knulli-app-store/recovery-backups/<package-id>/<UTC-timestamp>/`

On failure, rollback restores the exact files and modes from before the operation and removes the new recovery backup. On success, the backup stays for manual restore. Copy a required file back to the `original_path` shown in `manifest.json` only while the package is not running. Declared user data remains in place during the operation.
