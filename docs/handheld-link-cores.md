# Handheld Link Cores — manual mGBA Dual profile

Reviewed on 2026-10-29 (Asia/Singapore). Package version **2026.10.29.1**, status **experimental**. This is a limited integration of [Squirbie/handheld-link-cores](https://github.com/Squirbie/handheld-link-cores), not an installation of every upstream component. The owner reconfirmed this current version working and tested on TSP on 2026-10-30; see the [current owner report](real-device-tests.md#2026-10-30-current-owner-confirmation). Separate multiplayer and lifecycle results were not supplied.

## Scope and safety boundary

The Store downloads only `gba-mgba-dual-linux-aarch64.zip` and extracts four verified regular files into `/userdata/system/handheld-link-cores/mgba-dual`. The `.so` is a libretro entry point loaded by RetroArch, not a process launcher. The existing manifest `launcher` field checks that this entry exists in the archive; there is no `menu`, `menu_command` or executable-mode declaration. No new installer API or runtime code is needed.

No Tools entry, shared game list edit, stock-core replacement, `/usr` link, mount, overlay save, configgen patch, service or game hook is installed. The source [installer](https://github.com/Squirbie/handheld-link-cores/blob/ceed2f8d15c35893a0b9c57d096c0d4d5115eef7/install.sh#L490-L503) and [launcher patch](https://github.com/Squirbie/handheld-link-cores/blob/ceed2f8d15c35893a0b9c57d096c0d4d5115eef7/knulli-scripts/launcher/emulatorlauncher.patch) require writes outside `/userdata`; the Store never downloads or executes that installer or applies that patch.

The full bundle is not installed: it includes non-commercial arcade/Game Gear components, C++ runtimes, DSperate requiring glibc 2.38 and helper integration. mGBA Dual has a supplied `.info`, an MPL-2.0 licence and only libc/libm dependencies. This smaller profile avoids those additional requirements. Automatic ES core discovery, Join/Solo, per-player save exchange, paired editions, and guest single-pak setup are **not provided**. Normal same-ROM netplay requires manual configuration on both devices. A working multiplayer session remains a device-test requirement, not an established result.

## Release evidence

- [Pinned release](https://github.com/Squirbie/handheld-link-cores/releases/tag/2026.10.29.1), source commit `ceed2f8d15c35893a0b9c57d096c0d4d5115eef7`.
- [Asset 688030452](https://api.github.com/repos/Squirbie/handheld-link-cores/releases/assets/688030452): **387,153 bytes**, SHA-256 **`864a9ddd26d2f56520d7273e1e3d446e3ce35f29149cdfc6723e83cb5446aaee`**. Independently downloaded bytes match both the API digest and separate publisher [`SHA256SUMS.txt`](https://github.com/Squirbie/handheld-link-cores/releases/download/2026.10.29.1/SHA256SUMS.txt). The release is not GitHub-immutable; the Store pins bytes and rejects a replacement that changes them.
- Four regular entries below `gba-mgba-dual/`, **1,267,929 extracted bytes**. Complete inventory below. No duplicate names, absolute paths, traversal, links or special entries. Strip one path component.
- [`LICENSES/mgba_LICENSE`](https://github.com/Squirbie/handheld-link-cores/blob/ceed2f8d15c35893a0b9c57d096c0d4d5115eef7/LICENSES/mgba_LICENSE) and the packaged licence are MPL-2.0. This licence does not describe the other upstream assets, which are excluded. The Store links to the publisher; it does not mirror this binary.

| Archive entry | Bytes |
| --- | ---: |
| `gba-mgba-dual/LICENSE` | 16,726 |
| `gba-mgba-dual/README.txt` | 703 |
| `gba-mgba-dual/mgba_dual_libretro.info` | 1,804 |
| `gba-mgba-dual/mgba_dual_libretro.so` | 1,248,696 |

`sha256sum` on the extracted library gives `747af348dc59ecf2fcbd91d7cee8a5718ea6776938c545a01c07838c92e0c54e`, matching the packaged README. `readelf -h -d --version-info` reports ELF64 little-endian AArch64, `libc.so.6`, `libm.so.6` and a highest required symbol version of **GLIBC_2.35**. The manifest gates Knulli, aarch64, glibc >=2.35, libc, libm and an existing `/usr/bin/retroarch`. No firmware minimum, device list or display dimensions are guessed. Upstream RG35XX Pro claims are not promoted to Store-tested badges.

Source patches describe RetroArch save/state/BIOS paths and `/tmp/hb_dual_screen`, `/tmp/hb_dual_p2rom`, `/tmp/hb_ownsave`, plus reads of `/proc` and `/dev/urandom`. Existing upstream helpers can influence those temporary files; do not test alongside an active upstream setup without reviewing them. Runtime core code is not sandboxed. Source review does not prove that the binary was reproducibly built or that save handling is safe on every firmware. Use copies of saves. This asset contains no ROM, BIOS, firmware, installer, network downloader or updater; networking is through RetroArch netplay.

## Device test

1. Install **Handheld Link Cores (mGBA Dual)** in the Store. Read the warning. No Tools launcher will appear.
2. Back up saves and use a disposable copy of a legally supplied GBA ROM. Close any running game before testing. Do not use upstream's installer to activate this Store package.
3. Open RetroArch, enable Advanced Settings if necessary, then use **Load Core** to browse to `/userdata/system/handheld-link-cores/mgba-dual/mgba_dual_libretro.so`. Navigate above the default system core folder if needed. Use **Load Content** to select the copied GBA ROM. A bare `.so` is not a shell command. Some firmware hides Load Core; in that case use the SSH alternative below, not a root-overlay change.
4. Use the same core build and ROM on the other device. In core options, enable `mgba_dual_link`, set `mgba_dual_screen` to `1P` on the host and `2P` on the guest. Use RetroArch's own netplay host/connect controls, not the ES lobby or Join/Solo. Check controls, sound, link behavior and disposable saves. Do not expect independent save exchange without the excluded helpers.
5. Close RetroArch before Store Update, Repair or removal. These operations leave outside saves, BIOS and configuration untouched. User-created `config/`, `saves/` and `states/` inside the private destination are preserved too. No shared hook cleanup is needed for this profile.
6. Report package and Store versions, firmware, architecture, device, resolution, install/load/netplay/save/repair/removal results. Failed or incomplete gameplay does not become a tested badge.

SSH alternative, **on the device**, after closing the current game: replace the example ROM path with your disposable GBA ROM. This invokes existing RetroArch directly and does not register the core or modify `/usr`:

```sh copy Copy Device Test Command
retroarch --config /userdata/system/configs/retroarch/retroarchcustom.cfg \
  -L /userdata/system/handheld-link-cores/mgba-dual/mgba_dual_libretro.so \
  '/userdata/roms/gba/Your Test Game.gba'
```

The command uses the existing RetroArch controller/video configuration, which must already exist. ROMs and any required BIOS are user supplied. Direct core loading and the exact firmware controls still need confirmation on a physical device.

## Validation

The permanent offline lifecycle regression uses the real manifest with synthetic library bytes. It checks Install, Update, damage/Repair and removal without executing a library, creating a menu entry, changing a stock core or changing existing configuration/saves. The platform regression checks that absent RetroArch/libm are not reported present, and that installed prerequisites are detected. Real-asset fixture and full-check results are recorded in the package PR; filesystem fixtures are not physical-device execution evidence.

Temporary real-asset validation passed with the race detector on both TrimUI Smart Pro 1280×720 and MagicX Zero 28 640×480 filesystem fixtures: Install, simulated version Update using the same pinned asset, damage/Repair, rejected checksum-mismatch Update, and removal. Managed health, unchanged stock core/configuration/menu/outside saves, retained private saves and an unmanaged file were checked. The temporary test was removed; the permanent regression has no network/download dependency. No upstream binary, installer, launcher or ROM was executed.

Local gates passed: `make check`, `go test -race -tags sdl ./...`, `make cover` (84.0%), `make catalogue-check`, `make build`, Staticcheck v0.8.2 with both tag sets, and the offline `make walkthrough` with SDL's dummy video/software renderer. The new package card and install warning were also rendered at both device sizes. These are Store checks and rendered desktop fixtures, not proof of core loading or multiplayer on hardware.
