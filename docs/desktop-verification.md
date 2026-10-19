# Desktop GUI verification

The real SDL binary can be walked offline without a handheld. Keyboard input
opens the catalogue immediately: no setup, calibration, or saved assignments.
The scratch fixture supplies platform files; architecture and resolution are
explicit overrides, not measurements from a real device.

## Prerequisites and interactive run

Install SDL2 development libraries (`brew install sdl2` or
`sudo apt install libsdl2-dev`), then run:

```sh
make gui
make gui GUI_DEVICE=magicx-zero-28 GUI_RESOLUTION=640x480
```

`GUI_ROOT` defaults to `build/scratch-root`; `GUI_ARCH` defaults to `aarch64`.
Never use a mounted SD card or live `/userdata` as a fixture root.

The equivalent manual run is:

```sh
make build-ui catalogue
scripts/desktop-fixture.sh trimui-smart-pro "$PWD/build/scratch-root"
./build/knulli-app-ui -input keyboard -windowed \
  -root "$PWD/build/scratch-root" -arch aarch64 -resolution 1280x720 \
  -catalog build/catalog-index.json
```

Keep `-resolution` to pin the target size. Add `-screenshot out.png` to render
one real SDL frame and exit.

## Automatic controls

| Action | Desktop key | KNULLI ES button |
| --- | --- | --- |
| Navigate rows / tabs | Arrow keys | D-pad |
| Page a long list | Page Up / Page Down | L1 / R1 |
| Confirm | Enter | A |
| Back | Escape | B |
| Export diagnostics directly | Y | Y |
| Quit | Hold Tab (or F5), then Y | Hold Select, then Y |

The desktop footer reads `Confirm (ENTER)  Diagnostics (Y)  Quit (TAB + Y)`.
Paging appears only when the list outgrows its window. Confirmation panels show
Confirm and Back; a busy operation advertises no unavailable actions. Keyboard
repeat navigates only, so holding Enter cannot accept multiple confirmations.

The device launchers select `STORE_CONTROLLER_LAYOUT=knulli` normally, or
`xbox` when `/var/run/ports_xbox_layout.flag` exists. Knulli's default mapping
translates ES a/b/x/y to SDL b/a/y/x; Xbox mode uses direct letters. Bindings
compensate exactly once, leaving SDL events unchanged. Desktop SDL defaults to
normal logical letters. No raw device indices or system mapping writes are
used. Update the binary and launcher together. Legacy
`controller-mappings.json` files are ignored and left untouched.

## Offline walkthrough

```sh
make walkthrough
# Without a window server:
SDL_VIDEODRIVER=dummy make walkthrough
# Other device:
WALK_DEVICE=magicx-zero-28 WALK_RESOLUTION=640x480 make walkthrough
# Optional network-backed lifecycle flows:
make walkthrough WALK_FLAGS=--install
```

Frames and records go under `build/walkthrough/<flow>/`; renderer tests also
capture representative states at both target sizes under `screens/`. CI
publishes this generated evidence as `gui-walkthrough`, including partial
evidence when checks fail. Screenshots are not checked in.

| Flow | Covers |
| --- | --- |
| `automatic-first-run` | Immediate catalogue, navigation and paging |
| `catalogue-walk` | Details, actions, confirmation, cancel and quit |
| `catalogue-tabs` | All, Ready, Installed and stepping out of an empty tab |
| `catalogue-read-only` | A package with no safe action |
| `quit-chord` | Escape does not quit; held Tab plus Y does |
| `export-diagnostics` | Direct Y export without a settings detour |
| `blocked-export-diagnostics` | No-controller diagnostics and quit |
| `install-package` | Download, verify, apply (`--install`) |
| `install-package-health` | Tamper, health check, repair (`--install`) |
| `installed-uninstall` | Uninstall (`--install`) |

For a custom sequence:

```sh
./build/knulli-app-ui -input keyboard -root "$PWD/build/scratch-root" \
  -arch aarch64 -resolution 1280x720 -catalog build/catalog-index.json \
  -keys "down,up,enter,enter,esc,esc,y,tab,y" \
  -shot-dir /tmp/store-walk -walk-timeout 60s
```

`-keys` accepts `up,down,left,right,pgup,pgdown,enter,esc,y,tab`. Keys use the
real event handler and wait for work to finish. `tab` holds the anchor; a quit
must be the last key. The run otherwise ends when the sequence settles.
`-shot-dir` requires a sequence and captures the opening frame, post-key frames,
and operation completions. A timeout or missing expected screen fails the flow.

`summary.tsv` lists each flow's screens and result. Beside frames, `walk.tsv`
records file, state, key, item, focus, action, busy, mode, tab, session message,
message, toast, and error. The toast column carries operation completion.

## Fixture and limits

`scripts/desktop-fixture.sh` creates OS/release, board, architecture ABI, glibc,
runtime-library, and framebuffer evidence below a throwaway root. Its
`.desktop-fixture` marker permits safe refresh; it rejects `/`, symlinks,
relative paths, and unmarked non-empty directories. Installs, logs, recovery
backups and diagnostics remain below the fixture root.

The catalogue starts on All; left/right step All, Ready and Installed while
up/down move rows. Tabs remember selection. Operation notices clear after three
seconds; failures remain until answered.

Renderer and router tests cover automatic controls in both letter modes,
ignored legacy mappings, disconnect/hotplug, footers, lifecycle confirmations,
health, recovery and errors. Desktop evidence does **not** prove a physical SDL
GameController, kernel library loading, device filesystem behaviour, or a real
`/userdata` write. Record device runs in `docs/real-device-tests.md` and follow
the [TrimUI](trimui-smart-pro.md) and [MagicX](magicx-zero-28.md) checklists.
