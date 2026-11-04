# zitch 0.3.0 review

Reviewed 2026-10-26. Technical status: **experimental, upstream-managed**.
No community approval or real-device test is recorded.

## Release and assets

- Upstream: <https://github.com/itchio/zitch>, MIT, copyright 2026 itch corp.
- Source: tag `v0.3.0`, commit `052379ecfa9682217cdb45b0b9445ef818aed0f8`.
- Asset: `zitch-0.3.0-portmaster.zip`, 19,061,242 bytes.
- SHA-256: `7fd61f4cc68bdf6e08d02b1c1fcccd1803c0eb35a1aa03c2d3b39d202293940b`.
- The independent download matches the publisher's release `SHA256SUMS` and GitHub asset digest.
- Inventory: 17 regular files, four directories, 44,174,629 extracted file bytes.
  No duplicate paths, absolute paths, traversal, links or special entries.
- Payload: launcher, PortMaster metadata, zitch frontend, butler, 7z libraries,
  SDL shim, LÖVE and LuaJIT libraries, screenshot and license notices.
- Menu artwork is the original `zitch/screenshot.png` (249,885 bytes).
  It stays in the verified download, not a separate unverified image fetch.
  Existing Store cards use shared Store branding, not per-app covers.
- Keep all bundled license notices. Do not copy `gameinfo.xml` over the user's gamelist.

## Layout and lifecycle

The Store owns only `/userdata/roms/ports/Zitch` and its exact entry in
`/userdata/roms/ports/gamelist.xml`. The menu launcher is
`./Zitch/itch.io.sh`, within Knulli's limited scan depth. The image is
`./Zitch/zitch/screenshot.png`.

Upstream's launcher assumes the shared Ports root. A byte-checked, same-length
patch at offset 687 changes:

```sh
GAMEDIR="/$directory/ports/zitch"
# to the following, with two trailing padding spaces:
GAMEDIR="$(dirname "$0")/zitch"
```

The final launcher hash must be
`358eaeebff64282ec542a2c987b239e49aa4941ac523fe9c61041f8d2aeeff93`.
No installer or launcher runs during Store operations. Only the shell launcher,
frontend, butler and LÖVE executable receive execute permission.

The launcher places HOME and XDG config/cache directories in `zitch/home`.
Account credentials, database, settings, downloaded games, prerequisites and
cover cache stay there. Preserve `zitch/home`, `zitch/args` and `zitch/log.txt`.
Runtime-created files are not Store-owned. Removal keeps these files: it is
**not** sign-out or credential erasure. Close the app and games before Store
operations. No shared boot hook or service installer was found in the launcher.

The PortMaster launcher passes `--no-self-update`. Store Repair restores pinned
frontend files, not downloaded games. PortMaster should not also manage the
Store's private copy.

## Compatibility and runtime limits

Require Knulli, aarch64 glibc, PortMaster and SDL2. Keep upstream's minimum
glibc 2.38 even though the inspected frontend's highest symbol is GLIBC_2.35.
Other maxima: shim 2.34, butler 2.29, liblove 2.27, remaining objects 2.17.

`readelf -d` lists SDL2, libc, libm, libgcc_s and the aarch64 loader for the
frontend; butler also uses libpthread, libresolv and libdl. 7z needs libstdc++.
Bundled LÖVE needs freetype, OpenAL, zlib, modplug, vorbisfile, theoradec, ogg,
mpg123 and the bundled LuaJIT. The Store detects only its existing SDL2, libc,
libresolv, libpthread and PortMaster dependency keys. It does not prove every
transitive runtime library is present. Device launch must check those libraries,
graphics and audio. No board or display claims are added.

The tagged code gates ROM/core launch and the SDL game-shim environment on
`/opt/muos/script/mux/launch.sh`. These are not Knulli integration. Do not promise
ROM/core, native Linux, AppImage or LÖVE game launch on Knulli. Upstream's primary
test firmware is muOS. Knulli login, browsing, rendering, controls and game launch
all remain untested on hardware.

Network use includes itch.io sign-in, library metadata, covers and game downloads
through butler. Games and their installers execute outside Store checks and are
not sandboxed. Use trusted publishers and only content you have the right to use.
The Store supplies no ROM provider.

## Verification

The catalogue regression checks the eighth experimental entry without inventing
approval. The installer regression checks private launcher resolution with spaces
in the fixture path, menu image registration, Repair preservation of account and
game data, and removal that keeps those data. It never executes upstream code.

An additional disposable integration test used the actual downloaded release
with the manifest's unchanged size, hash, patch offset and patched-file hash.
Install, Repair and Uninstall passed against a temporary root, including menu
image registration and account-data retention. No upstream code was executed.
The scratch test was removed after validation; normal tests stay offline.

For a device test, use matching Knulli with glibc 2.38+ and current PortMaster.
Check clean launch/exit, controller navigation, QR sign-in, library access and a
legally owned game separately. Record the exact firmware, architecture, device,
resolution and version before adding device evidence. A successful desktop Store
walkthrough does not prove the ARM frontend runs on Knulli.
