# Catalogue review — 2026-09-17

## RetSend 0.9.1

The reviewed artifact is published by the App Store maintainer's fork `jellydn/retsend` at commit `e73257c052f360f842072ef2da065398c9997ea2`, which is the commit the release tagged `prerelease` pointed at on 2026-09-17. The tag is a moving name, so the manifest pins bytes, not the tag, and every evidence link in the manifest pins the commit.

- URL: `https://github.com/jellydn/retsend/releases/download/prerelease/retsend-knulli.zip`
- Compressed size: 4,119,423 bytes
- Extracted regular-file size: 8,000,277 bytes
- SHA-256: `9d50bad60af73aff059aef3c1523c53a900213cb5cfc68bb9374faa594d661b6`
- License: GPL-3.0-only
- Launcher: `/userdata/roms/tools/RetSend/RetSend.sh`, mode `0755`
- Program: `/userdata/roms/tools/RetSend/retsend`, mode `0755`
- Other files: `LICENSE`, `NOTICES`, and `README.md`, mode `0644`
- ABI: Linux AArch64, `/lib/ld-linux-aarch64.so.1`, GLIBC 2.28 symbols
- Runtime libraries: SDL2 and libc, plus the `libm`, `libdl`, and `libpthread` entries the same glibc userland supplies

The local extraction reproduced the release's published inventory and checksum. The archive holds five regular files in one `RetSend/` directory with no links, devices, pipes, traversal, or duplicate names, and it stores execute mode on the launcher and the binary. The manifest restores execute mode for exactly those two paths.

GitHub reports the release and its tag as not immutable. The Store therefore treats the URL only as a locator and pins the accepted bytes with the exact size and SHA-256 above. Verification happens before extraction, so a replacement asset cannot install. This is the same treatment Grout's mutable release received.

The release is titled RetSend v0.9.1, while the built tree still declares `0.9.0` in `Cargo.toml` and `Cargo.lock`. The catalogue records the release title and the evidence records the commit, so the mismatch is on the record rather than hidden. Upstream `mxmgorin/retsend` publishes v0.9.0 with generic ARM64 and PortMaster assets and no Knulli package, so this fork is the provenance of the reviewed build, not a mirror of an upstream release. That is why the recorded approval provenance is the maintainer rather than the community.

RetSend keeps configuration, transfer history, and its TLS identity in `/userdata/system/configs/retsend`, on the persistent partition outside the package tree, so install, repair, and update cannot destroy or silently regenerate them. Uninstall intentionally leaves that directory, and the receive directory `/userdata/roms/retsend-inbox`, behind; the manifest and the install warning both disclose it. The launcher roots the in-app browser at `/userdata/roms` and the inbox only, points `HOME` at the package directory so SDL's preference path cannot fall back into the read-only rootfs, starts the reviewed binary with `exec`, fails closed when it is missing or not executable, and sources no PortMaster helper. The app never installs remote code; the Store downloads only the reviewed release asset.

Network access is required for the package's only function: LAN multicast discovery on UDP 53317 and HTTPS transfers with self-signed peer certificates between peers on the same network. The Store owns one `gamelist.xml` entry for the reviewed launcher and removes only an unchanged entry it created.

Compatibility is broad but experimental. Installation needs detected Knulli, AArch64, the Linux glibc ABI at 2.28 or later, a detected system SDL2, libc, a non-empty device identity, and a validated display from 640×480 through 1280×720. Detection can only confirm that a system SDL2 exists: RetSend additionally needs SDL 2.0.18 or later for `SDL_RenderGeometry`, which detection cannot verify. No device has run this build, and controller-only navigation and the gamepad label layout are unreviewed, so the entry stays `experimental`.

## Open blockers

The artifact's own tracker, `goals/knulli-build/deferred-blockers.md` at the reviewed commit, lists eight admission blockers and all remain open: TLS certificate pinning; a restrictive mode on the TLS key file; symlink-safe rooted receive writes; a collision-safe overwrite default; transfer quotas; immutable, signed release publication; provenance attestation for CI artifacts; and a per-crate licence inventory or SBOM. The fork's tracker calls the artifact a non-installable candidate until they close. Making the package actionable is an explicit App Store decision recorded here, and each blocker in the fork's tracker stays open until a deliberate edit records how it closed. A `verified` badge additionally needs `real-device-test` evidence for the exact version and every declared device and resolution, which does not exist yet.

The weekly [catalogue update check](https://github.com/jellydn/knulli-app-store/actions/workflows/catalogue-updates.yml) compares the declared repository, so it reports `no-stable-release` for RetSend while the fork publishes only a prerelease. That is the expected report until a versioned release exists, and it never edits or approves a package.

## Catalogue entry and tests

`catalogue/packages/io.github.jellydn.retsend.json` is the reviewed entry. The catalogue tests now expect five packages, and an installer lifecycle test covers this package specifically: install, rejected repeated install, destination modes, the owned menu entry, health with the launcher's runtime files present, corruption, repair, and uninstall that removes managed files and the owned entry while leaving the identity, history, and inbox in place. Both declared Knulli targets are exercised.

The index is regenerated with `go run ./cmd/knulli-app catalogue -output build/catalog-index.json`, and a clean `git diff --exit-code` on that file is part of the package review. Device artifacts sign the index at build time as usual.
