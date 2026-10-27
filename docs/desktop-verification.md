# Desktop GUI verification

The real SDL binary can be walked offline without a handheld. Keyboard input
opens the catalogue immediately: no setup, calibration, or saved assignments.
The scratch fixture supplies platform files; architecture and resolution are
explicit overrides, not measurements from a real device.

Release target: v2026.10.27. Hardware reports remain scoped to the artifacts
actually tested. The earlier audit baseline was v2026.10.25 at `14490119`.
Picker 0.2.2 is published but UNTESTED on every hardware matrix. Its Tools and
Emulators exclusions add no hardware gameplay evidence. Launcher, settings schema, `state/`
preservation, firmware SDL requirements and the local ES API contract remain.
The historical Picker 0.2.0 Smart Pro report does not verify 0.2.2. The other
six current package versions retain Smart Pro-only evidence; all seven remain
experimental. See [device evidence](real-device-tests.md).

For v2026.10.27, published Picker 0.2.2 bytes passed all six asset digest/size
checks, ZIP inventory/modes and AArch64/GLIBC 2.34 inspection. The real CLI
passed fresh Install, 0.2.1→0.2.2 Update, damaged-binary Repair and Uninstall
in both device fixture roots; nested state survived. No ARM64 code ran.

For the v2026.10.25.1 work, the lead reported passing race suites, 84.0% coverage,
both Staticcheck tag sets and all ten network-backed GUI flows. Separately,
published Picker 0.2.1 assets/digests, its 20 regular archive files and exact ABI
were independently checked. Fixture-root Install, 0.2.0→0.2.1 Update, Repair and
Uninstall passed with nested `state/` retained. No ARM64 runtime was executed;
these package and desktop checks do not grant a hardware-tested badge.

## Prerequisites and interactive run

Install SDL2 development libraries (`brew install sdl2 sdl2_image sdl2_ttf sdl2_gfx` or
`sudo apt install libsdl2-dev libsdl2-image-dev libsdl2-ttf-dev libsdl2-gfx-dev`), then run:

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
one real SDL frame and exit. The gabagool-backed frontend reads the renderer
backbuffer before Present, so these captures include texture upload and
letterboxing rather than only the source canvas. It retains a software renderer
fallback when acceleration is unavailable. `INPUT_CAPTURE` is refused: the
toolkit's developer mapping wizard is not part of the Store.

## Automatic controls

| Action | Desktop key | KNULLI ES button |
| --- | --- | --- |
| Browse Library rows / action buttons | Arrow keys | D-pad |
| Filter All / Ready / Installed | Page Up / Page Down | L1 / R1 |
| Confirm | Enter | A |
| Back | Escape | B |
| More; export diagnostics inside More | Y | Y |
| Quit | Hold Tab (or F5), then Y | Hold Select, then Y |

The footer shows contextual actions and navigation/filter hints. The Library
keeps nearby rows visible and expands the selected row. It opens confirmation
directly for the first allowed action. More keeps
the full action list and details; Y there exports diagnostics. Empty catalogues
and blocked-controller screens still export directly. Confirmation panels show
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
SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy make walkthrough
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
| `automatic-first-run` | Immediate Library, navigation and filtering |
| `catalogue-walk` | Direct confirmation, cancel, More, secondary confirmation and quit |
| `catalogue-tabs` | All, Ready, Installed and stepping out of an empty tab |
| `upstream-managed-confirmation` | Full warnings for an actionable upstream-managed package |
| `quit-chord` | Escape does not quit; held Tab plus Y does |
| `export-diagnostics` | Y for More, Y there to export |
| `blocked-export-diagnostics` | No-controller diagnostics and quit |
| `install-package` | Download, verify, apply (`--install`) |
| `install-package-health` | Tamper, health check, repair (`--install`) |
| `installed-uninstall` | Uninstall (`--install`) |

For a custom sequence:

```sh
./build/knulli-app-ui -input keyboard -root "$PWD/build/scratch-root" \
  -arch aarch64 -resolution 1280x720 -catalog build/catalog-index.json \
  -keys "right,left,enter,esc,y,y,esc,tab,y" \
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

The Library starts on All; up/down browse rows, and left/right also work.
L1/R1 (Page Up/Down) step All, Ready and Installed. Filters remember selection.
Operation notices clear after three seconds; failures remain until answered.
Normal runs poll at 30 Hz but redraw only after input, window/controller events,
held navigation repeats, operation events or notice expiry. Screenshot and
scripted walkthrough runs still render complete frames on each step. They do
not measure quiet interactive CPU use.

Renderer and router tests cover automatic controls in both letter modes,
ignored legacy mappings, disconnect/hotplug, footers, lifecycle confirmations,
health, recovery and errors. Desktop evidence does **not** prove a physical SDL
GameController, kernel library loading, device filesystem behaviour, or a real
`/userdata` write. Record device runs in `docs/real-device-tests.md` and follow
the [TrimUI](trimui-smart-pro.md) and [MagicX](magicx-zero-28.md) checklists.
