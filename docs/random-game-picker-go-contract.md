# Random Game Picker 0.4.0 contract

The catalogue pins the public stable Go release, not development-build bytes.
The final tag resolves to
[9e3a91c](https://github.com/jellydn/knulli-random-game-picker/commit/9e3a91cb6f6e99e3c150ff0d0f0b925e2c1d055f).

- Asset: `random-game-picker-0.4.0-knulli-aarch64.zip`, from
  <https://github.com/jellydn/knulli-random-game-picker/releases/download/v0.4.0/random-game-picker-0.4.0-knulli-aarch64.zip>.
- ZIP size: 3,279,887 bytes; regular-file total: 7,877,205 bytes across 21 files.
- SHA-256: `8a91b69cc89f2d3acd8300f3294384db78c1ef47757c671e2eba2a968cbe392b`.
- Publisher `artifact.json`, `SHA256SUMS.txt`, `abi.txt` and
  `archive-inventory.txt` accompany the ZIP, with `build-info.txt` for build
  provenance. All six anonymous downloads match GitHub sizes and digests;
  the ZIP agrees with the publisher checksum, inventory and manifest.
- GitHub reports `immutable=false`. The version-specific URL and enforced
  size/SHA pin protect the reviewed content; no server-enforced immutability
  or repository setting change is claimed.
- Flat layout: `RandomGamePicker.sh`, `Play Random Game.sh`, `random-game-picker`,
  `assets/DejaVuSans.ttf`, `licenses/`, `LICENSE`, `README.md`, `THIRD_PARTY.md`.
- Keep destination `/userdata/roms/ports/RandomGamePicker`, strip components 0,
  all three executable names, and primary menu `./RandomGamePicker/RandomGamePicker.sh`.
  Add `./RandomGamePicker/Play Random Game.sh` as an independently owned
  `install.additional_menus` entry. Both use the existing allowed Ports gamelist.
  Ship the new Store binary and signed catalogue together: old Store binaries
  reject this new optional v1 field.
- Preserve the entire `state/` directory. The settings `system` string and
  `favorites` boolean remain compatible. An absent `yolo` boolean defaults off.
  Atomic saves use `settings.json.tmp`; `last-run.log` is replaced per run and
  `ui.log` remains under state. Old settings need no migration.
- KNULLI Linux AArch64, glibc >= 2.34, firmware SDL2/image/ttf/gfx and libc.
  No libstdc++ dependency or replacement firmware libraries. The only network
  calls are bounded loopback ES API requests on port 1234.

Version 0.4.0 adds a separate **Play Random Game** shortcut. It invokes
`--play-random`, requires saved settings, selects once with the saved system
and favorites filters, revalidates the exact game and submits one ES launch
without initializing SDL or opening the picker UI. It works with YOLO on or
off and leaves that saved value unchanged. Missing/malformed settings,
unavailable saved systems, empty pools, stale/ambiguous games, save failures
and ES errors stop safely, without changing systems or retrying the launch.
Configure and save filters through the original **Random Game Picker** entry.

The original app retains optional persistent YOLO mode. Toggle it in Settings with
A or left/right; clean exit or launch saves it to `state/settings.json`.
When enabled, A on System picks once and starts fresh launch revalidation
without a preview, cancel or reroll step. Saved-on runs open Settings so it
can be disabled. With it off, normal startup, preview, A Play, X reroll and
B back remain unchanged. Empty pools, stale games and failed settings saves
still block launch. Demo mode never scans, saves or launches.

The confirmed `tools` and `emulators` system exclusions, boundary-aware
application-root matching, Ports policy and duplicate-path checks remain.
Fresh discovery, successful save and SDL teardown before one POST remain
required. No launch retry or emulator command is added.

The package stays experimental and **UNTESTED on every hardware matrix**.
The owner's Picker 0.2.2 and 0.2.0 reports remain historical exact-version
evidence; they do not grant a badge to 0.4.0. Host/race checks include 44
real-binary headless cases. SDL fixtures cover both target sizes:
UI-save/direct-launch/UI, YOLO on/off byte persistence, toggle/save/reopen, one-pick launch, stale/save-failure
blocks, disabling back to the normal preview/reroll/cancel/Play flow, demo
isolation, controller mappings and 100 rerolls per size. They do not prove
physical controls, audio, gameplay/hotkey return, external pads, real
overrides/second-system behavior, MagicX or handheld performance.

The Store verified published-byte fresh Install, legacy 0.3.0→0.4.0 Update,
damaged-file/deleted-menu Repair and Uninstall in disposable roots for both 1280×720 TSP
and 640×480 MagicX fixture matrices. All 21 installed files matched the ZIP,
all three executables kept their modes, both owned menu entries were added then removed,
and nested `state/` data including `yolo: true` survived. These checks did not
execute the ARM64 app. See the
[Picker release workflow](https://github.com/jellydn/knulli-random-game-picker/actions/runs/43131032710)
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

Picker 0.4.0 release contract and remaining hardware limits:
<https://github.com/jellydn/knulli-random-game-picker/releases/tag/v0.4.0>.
