# retroshelf 0.6.3 — experimental release review

Review date: 2026-10-29. Compared with the previously pinned 0.5.6. This remains a direct-download, upstream-managed experimental package with an unverified license. No upstream executable or script was run.

## Release integrity and ABI

The downloaded [Knulli ZIP](https://github.com/retroshelf-org/linux/releases/download/v0.6.3/retroshelf-knulli-v0.6.3.zip) matches the publisher's [checksums.txt](https://github.com/retroshelf-org/linux/releases/download/v0.6.3/checksums.txt):

- Compressed size: 19,739,401 bytes; regular-file size: 42,693,255 bytes.
- SHA-256: `b0f5d67c8851552b7a74b59cf67e53cb66865ea75b9c3fcaa4c6171cb1398234`.
- Eight regular files and four directories, under `retroshelf/`; no links or special entries.
- UI SHA-256: `f38ac69d6eecdc5ed77a3e055375cff2456e54c8a4b0ecd69d9a505ab994c825`.
- Sync engine SHA-256: `1dc1b764deb9dbffddd1d7732e5175a07b472c04791eed22a0bcd0e1f8de060b`.

ELF inspection establishes AArch64 for both executables. The UI requires the existing firmware SDL2/image/ttf, libc, libresolv and libpthread libraries, with GLIBC_2.17 symbols. The sync engine is statically linked. Both bundled SDL2_gfx copies require **GLIBC_2.29**, so the package minimum is corrected to **2.29**. Main-executable inspection alone would miss this requirement. Static ABI inspection is not device execution proof.

## Launcher, hooks, updater and data

The 2,500-byte launcher and 195-byte uninstall wrapper match 0.5.6 byte for byte. The launcher relocates through `/tmp`, applies `.update` contents, may run a staged sync engine, repairs game-session hooks and then starts the UI. The Store never executes these scripts. Upstream runtime updates are not pinned or sandboxed by the Store.

The Store destination remains `/userdata/roms/tools/retroshelf`, with exact launcher entry `./retroshelf/retroshelf.sh` in the shared Tools gamelist. The publisher's Knulli guide still describes account linking, session hooks under `/userdata/system/scripts`, and account/session data outside the package. The Store removes only its owned files, leaving unmanaged runtime data. Users must close the app and run `bin/retroshelf-uninstall.sh --keep-data` themselves before Store removal. The read-only shared-hook reference check remains mandatory; the Store does not remove hooks or run cleanup scripts.

The linked source repository still returned HTTP 404 during review. There is no verified license to add and no source-level proof of binary hook ownership, runtime write containment, updater safety or database migration behavior. These limits remain visible in the experimental warning. This review does not authorize redistribution or promotion to installable/verified status.

## Validation scope

Temporary real-archive validation uses the actual old and new ZIPs through a local TLS server, with their pinned sizes and hashes unchanged. It drives `installer.Manager` against disposable filesystem fixtures, not devices. It covers old-version installation, failed-update rollback, update, exact executable hashes, damage detection, Repair, blocked removal while a hook reference exists, and removal after simulated manual hook cleanup. Synthetic account/history/local files and an unrelated menu entry must survive. A glibc 2.28 fixture must be rejected.

`go test -race ./internal/installer -run TestScratchRetroshelfPublishedLifecycle -v` passed on both TrimUI Smart Pro 1280×720 and MagicX Zero 28 640×480 filesystem fixtures (50.198 seconds). All listed assertions passed, including glibc rejection. The temporary test was removed after validation; permanent offline installer regressions remain.

These fixtures do not launch upstream code, link an account, migrate a real database, exercise hooks or prove physical controls. Separately, the owner reconfirmed all current apps working and tested on TSP on 2026-10-30, including 0.6.3 on the previously reported Knulli AArch64 1280×720 matrix. See the [current owner report](real-device-tests.md#2026-10-30-current-owner-confirmation). MagicX remains untested.
