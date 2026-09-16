# Desktop GUI verification

The SDL GUI can be walked end to end on a development machine, with no handheld
and no controller attached. The keyboard carries the eight semantic actions and a
scratch fixture root supplies the Knulli files that platform detection reads, so
a desktop run reports the same device matrix a handheld would.

Use this to check wording, layout, focus order, and every flow before touching a
device. It does not replace a device run: only a device exercises a real SDL
GameController, and only a device writes to real `/userdata`.

## Prerequisites

SDL2 development libraries, so the tagged build can link:

```sh
brew install sdl2 sdl2_image sdl2_ttf      # macOS
sudo apt install libsdl2-dev libsdl2-image-dev libsdl2-ttf-dev   # Debian/Ubuntu
```

## One command

```sh
make gui
```

That builds the GUI, builds the catalogue index, writes the scratch root, and
opens a windowed run driven by the keyboard. To verify the other device:

```sh
make gui GUI_DEVICE=magicx-zero-28 GUI_RESOLUTION=640x480
```

`GUI_ROOT` selects the scratch root (default `build/scratch-root`). Delete it to
return to first-run setup, since it also holds the saved controller mapping:

```sh
rm -rf build/scratch-root && make gui
```

## The same run by hand

```sh
make build-ui catalogue
scripts/desktop-fixture.sh trimui-smart-pro "$PWD/build/scratch-root"
./build/knulli-app-ui \
  -input keyboard \
  -windowed \
  -root "$PWD/build/scratch-root" \
  -resolution 1280x720 \
  -catalog build/catalog-index.json
```

Drop `-resolution` to use the resolution the fixture root reports, the way a
device does. Add `-screenshot out.png` to render one frame and exit instead of
opening a window, which is how a screen is captured for a pull request.

## The walkthrough

```sh
make walkthrough
```

That walks every flow below with the keyboard, captures one frame per step under
`build/walkthrough/<flow>/`, and renders every screen state under
`build/walkthrough/screens/`. It needs no display: `SDL_VIDEODRIVER=dummy` runs
the same code path on a runner with no window server.

A flow is a key sequence:

```sh
./build/knulli-app-ui -input keyboard -root "$PWD/build/scratch-root" \
  -catalog build/catalog-index.json \
  -keys "down,down,enter" \
  -shot-dir /tmp/walk/03-customize \
  -walk-timeout 60s
```

- `-keys` accepts `up`, `down`, `left`, `right`, `enter`, `esc`, `y`, `q`, in
  the order to press them. Each key goes through the same handler as a real
  keystroke, so a walkthrough reaches the same screens a user does.
- `-shot-dir` keeps one frame per step and a `walk.tsv` record, and needs
  `-keys`: without a sequence there is no step to capture. A key waits for any
  operation it started, so an install finishes before the next key.
- `-walk-timeout` fails a stuck flow instead of hanging a runner.
- The run ends when the sequence is spent, or when a key exits the app.

| Flow | Covers |
| --- | --- |
| `first-run-use-detected` | Setup, saving the detected mapping |
| `first-run-test-detected` | Setup, preview, testing all eight actions, save |
| `first-run-customize` | Calibration, assignment review, preview, save |
| `first-run-safe-exit` | Leaving setup without a mapping |
| `catalogue-walk` | Catalogue, details, action list, confirmation, back out |
| `catalogue-read-only` | A candidate package that offers no action |
| `settings-export-diagnostics` | Settings and a diagnostics export |
| `blocked-export-diagnostics` | The blocked screen with no controller at all |
| `install-package` | Download, verify, apply, and the progress screen (`--install`) |
| `install-package-health` | A tampered file, the health check, and repair (`--install`) |
| `installed-uninstall` | The installed action list and uninstall (`--install`) |

The `--install` flows download a real package, so they need network access and
are left out of CI. Everything else runs offline.

### Reading the evidence

`build/walkthrough/summary.tsv` lists every flow with the screens it reached and
whether it reached them all. Beside each flow, `walk.tsv` records one row per
frame:

```text
file	state	key	item	focus	action	busy	mode	session_message	message	error
```

The script fails a flow that never reaches a screen it claims, or one whose run
exits non-zero, so a broken flow fails the run rather than passing quietly.

## Continuous integration

The `gui-walkthrough` job in `.github/workflows/check.yml` runs
`make walkthrough` on every pull request and uploads the frames and records as
the `gui-walkthrough` artifact. Reviewers read the screens a change affects
without a handheld, and the same artifact is useful evidence when a device run
is not available.

## Keyboard bindings

| Action | Key |
| --- | --- |
| Up, Down, Left, Right | Arrow keys |
| Confirm | Enter |
| Back | Escape |
| Diagnostics | Y |
| Exit | Q |

These are a binding set like any other, so the footer and the mapping summary
name them: `SELECT (KEY ENTER)  BACK (KEY ESC)  SETTINGS (KEY Y)`. Keyboard codes
sit above every SDL GameController button, so a key is never mistaken for a pad
button.

## The fixture root

`scripts/desktop-fixture.sh` writes only the files the detector reads:

| File | Supplies |
| --- | --- |
| `/etc/os-release` | Knulli firmware identity (`OS_NAME="knulli"`) |
| `/usr/share/knulli/knulli.version` | Release name, for example `scarab` |
| `/boot/boot/knulli.board` | Device ID, for example `trimui-smart-pro` |
| `/lib/ld-linux-aarch64.so.1`, `/lib/libc.so.6` | Architecture ABI and glibc version |
| `/usr/lib/libSDL2*.so.0`, `/lib/libresolv.so.2`, `/lib/libpthread.so.0` | Declared runtime dependencies |
| `/sys/class/graphics/fb0/mode` | Display resolution |

Everything else the run writes — installs, logs, saved mappings, recovery
backups — lands below `ROOT`, which is exactly how the device path is exercised
without a device.

**The root is fabricated evidence and must stay a throwaway.** The script
refuses `/`, requires an absolute path, and refuses to overwrite a tree that
looks like a real device root unless its own marker file is present. Never point
it at a mounted SD card or a live `/userdata`.

## Verify against a package

The fixture root reports a device, so a manifest that matches it becomes
actionable and the install, repair, and uninstall flows can be walked. Install
writes below the scratch root and can be undone by deleting `ROOT`; nothing
outside it changes. The catalogue index must be built first, since the GUI reads
it beside the binary or from `-catalog`.

## What this covers, and what it does not

Covers: every screen, the footer on each one, focus order, wording, layout
bounds, setup, calibration, preview, settings, catalogue, details, health,
confirmations, recovery, blocked, and error states.

Does not cover: a real SDL GameController, real kernel SDL library loading, real
device filesystem behaviour, or a real `/userdata` write. Record those runs in
`docs/real-device-tests.md`, and see the device guides for
[TrimUI Smart Pro](trimui-smart-pro.md) and
[MagicX Zero 28](magicx-zero-28.md).
