# Random Game Picker 0.2.0 contract

The catalogue pins the public stable Go release, not development-build bytes.
The final tag resolves to `da79d67d74185dda23d6804a30d38cb2c66cd57c`.

- Asset: `random-game-picker-0.2.0-knulli-aarch64.zip`, from
  <https://github.com/jellydn/knulli-random-game-picker/releases/download/v0.2.0/random-game-picker-0.2.0-knulli-aarch64.zip>.
- ZIP size: 3,273,845 bytes; regular-file total: 7,872,532 bytes across 20 files.
- SHA-256: `3f8a225cffcb2a89ffee166ff4f06dbf87923142309d97203e738a8200737df4`.
- Publisher `artifact.json`, `SHA256SUMS.txt`, `abi.txt` and
  `archive-inventory.txt` accompany the ZIP. Independently downloaded bytes
  agree with the checksum and manifest.
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

The package stays experimental. Development-build injected controls and ES
handoff evidence do not prove stable-version physical presses, audible sound,
normal gameplay/hotkey return, or MagicX hardware operation.

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
