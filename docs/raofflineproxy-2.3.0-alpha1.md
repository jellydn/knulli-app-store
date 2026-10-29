# RAOfflineProxy 2.3.0-alpha1 — experimental release review

Review date: 2026-10-29. Compared tagged source [2.2.0-alpha1](https://github.com/misantronic/RAOfflineProxy/tree/3958abcecd4762128cdb1d4c24cbf179ea4c0f44) with [2.3.0-alpha1](https://github.com/misantronic/RAOfflineProxy/tree/d86bcb2b66dbeab56792f1f374ee273f4f9063c6). No upstream executable, script, Python module, build or test was run during that review. The package remains experimental and upstream-managed. The owner subsequently reconfirmed 2.3.0-alpha1 working and tested on TSP on 2026-10-30; see the [current owner report](real-device-tests.md#2026-10-30-current-owner-confirmation).

## Integrity and data-only extraction

- Pinned [Knulli shell asset](https://github.com/misantronic/RAOfflineProxy/releases/download/v2.3.0-alpha1/RAOfflineProxy-Knulli-v2.3.0-alpha1-Install.sh), GitHub [asset 679594367](https://api.github.com/repos/misantronic/RAOfflineProxy/releases/assets/679594367).
- Size: **1,466,921 bytes**. Independently calculated SHA-256: `90bc7bf59def667a864a569315010532224968eb4ec475999cb2ffcf6aeb487a`, matching the API digest.
- No separate publisher checksum is published for this Knulli asset. This trusts GitHub and the publisher's GitHub account, not independent publisher authentication. GitHub does not mark the release immutable. Installation keeps its expected size/hash pinned; it never fetches a replacement expected digest.
- The shell prefix is unchanged. The unique `__RAOFFLINEPROXY_PAYLOAD_BELOW__` marker ends at byte **1,173**. Only the following base64 is decoded as data, yielding a **1,085,033-byte tar.gz**.
- All entries inspected: **88 regular files, 10 directories, 2,575,572 regular-file bytes**. No duplicate paths, absolute paths, traversal, links or special entries. One `raofflineproxy-knulli-bundle/` prefix is stripped.

All 67 packaged app Python files match the pinned new source byte for byte. Compared with 2.2.0-alpha1, the package adds `pcsx2_cfg.py` and 21 Windows Python modules; no old file is removed. Windows menu/process/autostart/updater code is platform-gated, not a new Knulli dependency. The pinned source retains GPL-3.0 licensing; the Store does not mirror the asset.

| Installed file | Bytes | SHA-256 |
| --- | ---: | --- |
| `bin/raofflineproxy` | 1,120 | `734b1d165c37201044a08b375d623489f7586ad6e3d3e6700cc72814caea283c` |
| `bin/raofflineproxy-uninstall` | 103 | `bf3a6427922d23d82d0551187fee70b1ede32bae1d58c5611883c0487054aa4b` |
| `lib/libraproxy_rchash.so` | 375,488 | `811b805757a27f1791d8023ef4f15b439ff9d5bb98e497c307a16c66d2793d5a` |
| `app/vendor/sqlite/_sqlite3.cpython-312-aarch64-linux-gnu.so` | 1,197,944 | `95caf4288f3f522d6506411efa9a1ef6a517e642ab0b91fe1aa4ae682c4024cd` |

`readelf -h -d --version-info` establishes ELF64 AArch64, `libc.so.6` and `libpthread.so.0`, with a **GLIBC_2.17** symbol floor for both native libraries. The SQLite extension remains CPython 3.12-specific. Python 3.12 and pygame must already exist; the Store installs no runtime. Existing optional RVZ hashing uses lazily loaded libzstd/libcrypto; Windows DLL lookup additions do not add a mandatory Linux dependency. Static inspection does not prove firmware or device execution.

## Store paths and removal

The destination remains `/userdata/system/raofflineproxy`. Existing file mappings move both `scripts/launcher-*` files into `bin/`. The generated Tools wrapper remains `/userdata/roms/tools/RAOfflineProxy.sh`, invoking the installed launcher with `menu`. Store writes remain confined to the package destination, that wrapper and the shared Tools gamelist.

The Knulli launchers and install/uninstall scripts are unchanged. Store extraction never executes them. The uninstall launcher points at upstream's bundle cleanup script, not a Store-managed data-preserving operation; it must not be used as a substitute for Store removal.

Persistent credentials, cache, queued awards, signing key, logs and patch state remain under `/userdata/system/.config/raofflineproxy`, outside the package destination. Runtime-created files are not adopted as Store-owned. `storage.py` and `state.py` are unchanged; this is not proof of real account/database migration. Back up retained data before testing.

Removal still requires **Stop → disable autostart → `bin/raofflineproxy remove-boot-hook` → Store removal**. Disabling autostart alone does not remove the Knulli boot block. The Store checks boot/game hooks, PID state and the default proxy address in known frontend/emulator configs before mutation; it never stops a service, executes cleanup or edits shared configs. The review adds the previously missing fourth [documented RetroArch candidate](https://github.com/misantronic/RAOfflineProxy/blob/d86bcb2b66dbeab56792f1f374ee273f4f9063c6/linux/raofflineproxy/config.py#L48-L53), `/userdata/system/.config/retroarch/retroarchcustom.cfg`. All nine checks are read-only. Upstream Uninstall still deletes persistent credentials/cache/queued awards and is not safe when those must be retained.

Removal uses the manifest recorded in installed state. The added check applies after installing or updating to this reviewed definition; publishing a new catalogue alone does not add it to an existing 2.2.0-alpha1 installation. Users removing the older installation must inspect this fourth path manually too.

## Runtime changes and limits

1. **Free-space risk:** [single-ROM ZIP hashing](https://github.com/misantronic/RAOfflineProxy/blob/d86bcb2b66dbeab56792f1f374ee273f4f9063c6/linux/raofflineproxy/rom_browser.py#L424-L458) now streams temporary decompressed ROMs to `CONFIG_DIR/rom-hash-*`, not `/tmp`. It has no local decompressed-size/free-space cap. Large or highly compressed input can fill the same filesystem that holds awards and settings; abrupt termination can leave temporary data. Neither the Store's 512 MiB release-archive limit nor the cache budget bounds this runtime operation. Ensure sufficient free space and trusted input.
2. **Credential access:** [authentication](https://github.com/misantronic/RAOfflineProxy/blob/d86bcb2b66dbeab56792f1f374ee273f4f9063c6/linux/raofflineproxy/auth.py#L28-L114) can now read `global.retroachievements.*` credentials from Knulli/Batocera frontend config before RetroArch's config exists. This is an additional read source plus existing login/token caching, not a new frontend credential overwrite. Network use still includes RetroAchievements authentication, media/cache requests, queued awards and upstream update checks.
3. **Opt-in PCSX2 restoration risk:** [custom PCSX2 INI selection](https://github.com/misantronic/RAOfflineProxy/blob/d86bcb2b66dbeab56792f1f374ee273f4f9063c6/linux/raofflineproxy/config.py#L565-L583) works on Linux, not only Windows. It is inactive on Knulli unless explicitly configured. [Restoration](https://github.com/misantronic/RAOfflineProxy/blob/d86bcb2b66dbeab56792f1f374ee273f4f9063c6/linux/raofflineproxy/pcsx2_cfg.py#L46-L69) redetects the current INI path rather than using the saved original path. Changing/removing that setting before Stop can leave the original file redirected or restore its values into a different INI. Keep paths unchanged until Stop; manually verify all custom emulator/proxy paths before removal. No invented Knulli PCSX2 path, runtime dependency or blanket write permission is added.
4. **Upstream update execution remains:** [the unchanged Knulli updater](https://github.com/misantronic/RAOfflineProxy/blob/d86bcb2b66dbeab56792f1f374ee273f4f9063c6/linux/raofflineproxy/update.py#L327-L370) downloads an executable installer without an expected checksum; the menu can execute it. This runtime path is not protected by Store release verification. Close the app before Store operations. Repair restores pinned code, not a sandbox or automatic runtime cleanup.

These are disclosed upstream-managed risks, not claims that runtime writes or downloaded content are safe. No unconditional update-specific blocker was found for the default Knulli configuration in source review. Custom PCSX2 restoration and disk-full behavior remain unverified at runtime and need care before any hardware trial.

## Validation scope

A temporary Store lifecycle test uses the actual old/new shell assets through a local TLS server with original manifest sizes/hashes. It drives `installer.Manager` against disposable TrimUI Smart Pro 1280×720 and MagicX Zero 28 640×480 filesystem fixtures. It covers old installation, failed-update rollback with malformed XML, update, exact mapped-launcher/native-library hashes, managed health, damage/Repair, all declared removal guards, and successful removal after simulated manual cleanup. Credentials/cache/queued-award/key/patch-state placeholders, an unmanaged package file and an unrelated menu entry must survive. Missing Python 3.12 must fail compatibility.

A permanent offline regression uses a synthetic archive and the real catalogue declaration: an emulator redirect in `.config/retroarch/retroarchcustom.cfg` must prevent removal, leave the file and installation healthy, and permit removal only after simulated restoration. No upstream code runs in either test. These fixtures do not exercise account login, cache migrations, ROM hashing, actual service shutdown, physical controls or display behavior. The separate 2026-10-30 owner confirmation supplies current 2.3.0-alpha1 TSP evidence; the older 2.2.0-alpha1 report remains historical.

`go test -race ./internal/installer -run 'TestScratchRAOfflineProxyPublishedLifecycle|TestRAOfflineProxyRemovalRequiresRestoredRetroArchCustomConfig' -v` passed on both filesystem fixtures (8.841 seconds). All nine removal checks prevented mutation of the checked package/library, wrapper, menu, state and cleanup marker. The permanent regression failed without the added guard (`removal accepted an emulator still pointing to the proxy`) and passed with it. The temporary real-asset test was removed after validation; the permanent regression needs no downloads.

Local validation passed: `make check`, `go test -race -tags sdl ./...`, `make cover` (84.0%), `make catalogue-check` (two identical 73,018-byte builds), `make build`, and Staticcheck v0.8.2 with both tag sets. This establishes Store metadata/lifecycle behavior, not upstream runtime safety or physical-device compatibility.
