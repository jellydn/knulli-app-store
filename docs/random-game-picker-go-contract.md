# Random Game Picker 0.2.2 contract

The catalogue pins the public stable Go release, not development-build bytes.
The final tag resolves to
[956f9b4](https://github.com/jellydn/knulli-random-game-picker/commit/956f9b4d8e1c0cce91ad4a5c45a6ff0df8850cc7).

- Asset: `random-game-picker-0.2.2-knulli-aarch64.zip`, from
  <https://github.com/jellydn/knulli-random-game-picker/releases/download/v0.2.2/random-game-picker-0.2.2-knulli-aarch64.zip>.
- ZIP size: 3,275,031 bytes; regular-file total: 7,873,304 bytes across 20 files.
- SHA-256: `9a1a086a649ef62dd3203328a2da400517c6e9301a273f2bef8dbe013735f897`.
- Publisher `artifact.json`, `SHA256SUMS.txt`, `abi.txt` and
  `archive-inventory.txt` accompany the ZIP, with `build-info.txt` for build
  provenance. All six anonymous downloads match GitHub sizes and digests;
  the ZIP agrees with the publisher checksum, inventory and manifest.
- GitHub reports `immutable=false`. The version-specific URL and enforced
  size/SHA pin protect the reviewed content; no server-enforced immutability
  or repository setting change is claimed.
- Flat layout: `RandomGamePicker.sh`, `random-game-picker`,
  `assets/DejaVuSans.ttf`, `licenses/`, `LICENSE`, `README.md`, `THIRD_PARTY.md`.
- Keep destination `/userdata/roms/ports/RandomGamePicker`, strip components 0,
  both existing executable names, and menu `./RandomGamePicker/RandomGamePicker.sh`.
- Preserve the entire `state/` directory. The settings `system` string and
  `favorites` boolean remain compatible. Atomic saves use `settings.json.tmp`;
  `last-run.log` is replaced per run and `ui.log` is new. No schema migration.
- KNULLI Linux AArch64, glibc >= 2.34, firmware SDL2/image/ttf/gfx and libc.
  No libstdc++ dependency or replacement firmware libraries. The only network
  calls are bounded loopback ES API requests on port 1234.

Version 0.2.2 excludes the confirmed `tools` and `emulators` systems before
gamelist parsing, from in-memory selection and during Play revalidation.
Path-only candidates use boundary-aware matching of the two confirmed
application roots; similarly named sibling paths remain eligible. No title,
platform, core, hardware or extension blacklist is added. The existing Ports
policy and duplicate-path safety checks remain unchanged. Fresh discovery
before Play, successful save and SDL teardown before one POST remain required.

The package stays experimental and **UNTESTED on every hardware matrix**.
The owner's Picker 0.2.0 report remains historical exact-version evidence;
it does not grant a badge to 0.2.2. Host/race and SDL fixtures cover both target
sizes, not physical controls, audio, gameplay/hotkey return, external pads,
real overrides/second-system behavior, MagicX or handheld performance.

The Store verified published-byte fresh Install, 0.2.1→0.2.2 Update, Repair
and Uninstall in disposable roots. All 20 installed files matched the ZIP,
both executables kept their modes and nested `state/` data survived. These
checks did not execute the ARM64 app. See the
[Picker release workflow](https://github.com/jellydn/knulli-random-game-picker/actions/runs/40435138065)
and [device evidence](real-device-tests.md).

## UI reference

The public `internal/ui/canvas.go` and `run.go` use Go with gabagool/v2 and SDL:
centered art/title/metadata, semantic footer actions, typed selector/settings/exit
states, remembered settings focus and asynchronous scan/verification. The Store
uses its responsive coordinate rule, font scale and palette in the owner-selected
Library layout. Its local catalogue mark avoids unreviewed artwork. It retains
its own typed router, asynchronous backend,
confirmation guards and SDL lifetime boundary. No picker code or binary is used
as the Store frontend.

Coordination and device evidence:
<https://ampcode.com/threads/T-01a16e60-ed52-70d8-ac24-d7b17380f4e5>.

Picker 0.2.2 release contract and remaining hardware limits:
<https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.2.2>.
