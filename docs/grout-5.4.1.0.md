# Grout 5.4.1.0 — experimental release review

Review date: 2026-10-29. Source: [v5.4.1.0](https://github.com/rommapp/grout/tree/1484116313cee19d12b92590c2057fa02bc3cba2), compared with the previously pinned 5.2.0.0. The package remains experimental. No upstream binary or launcher was executed during this review.

## Release integrity and inventory

- Versioned [Knulli ZIP](https://github.com/rommapp/grout/releases/download/v5.4.1.0/Grout-Knulli.zip), GitHub asset [686708108](https://api.github.com/repos/rommapp/grout/releases/assets/686708108).
- Compressed size: 24,862,889 bytes. Installed regular-file size: 48,758,713 bytes.
- ZIP SHA-256: `7850d50388f4c13d4d0ac7d4ba353fba80008cdcce70216ae5aeb3903c93088f`.
- The independently downloaded ZIP matches both GitHub's asset digest and the publisher's `stable.assets.Grout-Knulli.zip` entry in [versions.json at 48b229c](https://github.com/rommapp/grout/blob/48b229cc04002681335ba66547a267d680c152e3/docs/versions.json). The release tag predates that checksum publication; the current website agrees. Installation uses only the pinned manifest values, never a fresh expected checksum.
- GitHub does not mark the release immutable. Exact pinned size and digest reject replacement bytes.

All eight archive entries were inspected: six regular files and two directories. Names are unique relative paths without traversal, links or special files. `strip_components: 1` removes the `Grout/` prefix.

| Installed relative path | Bytes | Mode | Unmodified SHA-256 |
| --- | ---: | --- | --- |
| `Grout.sh` | 444 | 0755 | `fcfa2a2e16463437111f5273d00e4743c2fcf7d9e79cba3ba16c8a09fac0f0b6` |
| `grout` | 48,647,736 | 0755 | `9d5c6a51683462aa81ce83baef65940c65105603e6a7b3ef588ef815b6ed6958` |
| `LICENSE` | 1,096 | 0644 | `f8535bb4329d754f9e08053a74845edd4b84f7810da3a58a33c9c02321a3ccae` |
| `lib/libSDL2_gfx-1.0.so.0` | 65,344 | 0644 | `8f124e2666901f68913fcce997816c167ab28a5723c709803eed4d91ad1f24ad` |
| `logo.png` | 39,378 | 0644 | `eb4f3db1bc56e7e95dc3704651e8dfb3ca9d2ad2be3fc88721508923a9164db5` |
| `README.md` | 4,715 | 0644 | `f18e769881c7a903a017c596c449c1746e4a2a2fbe27371e318bd1e2d8fffb27` |

The archive retains the source's MIT license. The Store does not mirror the publisher's asset.

## Updater controls

There is no supported updater-off setting. Background and manual checks both call `CheckForUpdate`, which must fetch [the single versions URL](https://github.com/rommapp/grout/blob/1484116313cee19d12b92590c2057fa02bc3cba2/update/versions.go) before obtaining an update asset. No fallback or manifest cache was found.

The downloaded binary contains exactly one `https://grout.romm.app/versions.json`, at zero-based offset **9,819,700**. The existing same-length replacement `https://self-update.disabled.invalid` remains valid. Patched binary SHA-256: `02a2475559f7daedeb6186f9cb21ba60361150238f66e8245d4c2a548c257a10`.

The unchanged upstream launcher would still copy an old `../.update` into the shared Tools directory and remove it before starting Grout. A second same-length patch replaces that complete 103-byte block at offset **96** with a comment. Patched launcher SHA-256: `061df095b8eefdcaf12e617ea4fb19bde5e06000c0559488a17523a8cd6b2b16`. It leaves any existing staged update untouched rather than applying or deleting it. Remaining launch and ES-restart behavior is unchanged.

Both patches use the existing installer mechanism: exact source bytes, fixed length, bounded offset and final whole-file hash are verified during staging before mutation. Independent byte checks verified both declared patches and hashes; `bash -n` accepted the patched launcher. No upstream script was run.

## Runtime, paths and preservation

`readelf -l -d --version-info` on the downloaded files establishes ELF64 AArch64 and interpreter `/lib/ld-linux-aarch64.so.1`. The main binary requires SDL2, SDL2_image, SDL2_ttf, libresolv, SDL2_gfx, libpthread and libc; its symbol floor is GLIBC_2.17. The bundled SDL2_gfx also requires libm and **GLIBC_2.29**, so the package minimum is corrected to **2.29**. SDL2_gfx is bundled; the system SDL and libc requirements remain. This is static ABI evidence, not device execution proof.

Store destination and menu ownership remain `/userdata/roms/tools/Grout` and its exact `./Grout/Grout.sh` entry in `/userdata/roms/tools/gamelist.xml`. Preserve `config.json`, `save_slots.json`, `.cache`, `logs`, and now explicitly `input_mapping.json` for saved controls. No write scope is expanded.

Upstream supports [RomM 5.4.1 or newer](https://github.com/rommapp/grout/blob/1484116313cee19d12b92590c2057fa02bc3cba2/docs/usage/faq.md). Grout reads and writes ROMs, artwork and gamelists under `/userdata/roms`, BIOS under `/userdata/bios`, and saves/backups under `/userdata/saves`. Save sync can replace save directories and prune backups. Back up saves first.

These are expected runtime roots, not a sandbox. Environment and configured mappings can redirect paths, temporary files can use `/tmp`, and upstream runtime ZIP extraction does not enforce the Store's traversal/link policy. Use only trusted RomM servers and content you have the right to use. Per-host settings can disable TLS verification. The Store validates its release archive, not later server-provided ROM/BIOS/save content. Credentials remain in the preserved configuration; history remains in the cache. No upstream code or user data was sent to a server in this review.

## Lifecycle validation and device evidence

A temporary test, `TestScratchGroutPublishedReleaseLifecycle`, uses the actual downloaded 5.2.0.0 and 5.4.1.0 ZIPs through a local TLS server with their original manifest hashes and sizes unchanged. It exercises the real `installer.Manager` on temp-root TrimUI Smart Pro 1280×720 and MagicX Zero 28 640×480 filesystem fixtures:

1. Install the old published release and write distinct saved configuration, slots, controls, cache and logs.
2. Fail update with malformed shared XML and verify rollback retains a healthy 5.2.0.0 installation.
3. Update to 5.4.1.0, verify the transformed binary hash and complete managed-file health.
4. Damage the binary, verify health detects it, then Repair and verify the transformed hash again.
5. Uninstall, verify the executable is removed, all five saved data paths remain, and the unrelated menu entry survives.

It also rejects glibc 2.28 and accepts 2.29. The scratch test is removed after validation; existing permanent offline lifecycle/patch/security regressions remain. Neither the ARM binary nor the upstream launcher is executed. These fixtures do not verify cache migrations, account login, sync, physical controls or display behavior.

Local validation passed: `make check`, `go test -race -tags sdl ./...`, `make cover` (84.1%), `make catalogue-check` (two identical 65,858-byte builds), Staticcheck v0.8.1 with both tag sets, and the CGO-free Linux ARM64 CLI build. The real-archive lifecycle test also passed with the race detector on both filesystem fixtures.

The owner reconfirmed all current apps working and tested on TSP on 2026-10-30, including Grout 5.4.1.0 on the previously reported Knulli AArch64 1280×720 matrix. See the [current owner report](real-device-tests.md#2026-10-30-current-owner-confirmation). Older reports remain historical. No MagicX test or separate complete device lifecycle is claimed.
