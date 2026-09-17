# Desktop GUI verification

The SDL GUI can be walked end to end on a development machine, with no handheld
and no controller attached. The keyboard carries the eight semantic actions and a
scratch fixture root supplies the Knulli files that platform detection reads, so
a desktop run can simulate the selected device matrix. The target architecture
and resolution are explicit desktop overrides; they are not measurements from a
real handheld.

Use this to check wording, layout, focus order, offline flows, and optional
network-backed lifecycle flows before touching a device. It does not replace a
device run: only a device exercises a real SDL
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

`GUI_ARCH` defaults to `aarch64`. The override is required on an x86 desktop so
platform detection can evaluate the fixture's AArch64 loader and ABI evidence.

## The same run by hand

```sh
make build-ui catalogue
scripts/desktop-fixture.sh trimui-smart-pro "$PWD/build/scratch-root"
./build/knulli-app-ui \
  -input keyboard \
  -windowed \
  -root "$PWD/build/scratch-root" \
  -arch aarch64 \
  -resolution 1280x720 \
  -catalog build/catalog-index.json
```

Keep `-resolution` to pin the target size. Without it, SDL renderer and window
sizes can take precedence over the fixture's framebuffer value. Add
`-screenshot out.png` to show the SDL window, render and save one frame, and
exit without waiting for interaction.

## The walkthrough

```sh
make walkthrough
```

That walks the offline flows below with the keyboard and writes their opening,
post-key, and operation-completion frames under `build/walkthrough/<flow>/`. It
also renders every screen state under `build/walkthrough/screens/`. For a host
with no window server, run:

```sh
SDL_VIDEODRIVER=dummy make walkthrough
```

Include the network-backed package lifecycle flows with:

```sh
make walkthrough WALK_FLAGS=--install
```

A flow is a key sequence:

```sh
./build/knulli-app-ui -input keyboard -root "$PWD/build/scratch-root" \
  -arch aarch64 -resolution 1280x720 \
  -catalog build/catalog-index.json \
  -keys "down,down,enter" \
  -shot-dir /tmp/walk/03-customize \
  -walk-timeout 60s
```

- `-keys` accepts `up`, `down`, `left`, `right`, `enter`, `esc`, `y`, `tab`, in
  the order to press them. Each key goes through the same handler as a real
  keystroke, so a walkthrough reaches the same screens a user does. `tab` holds
  the quit chord's anchor, so a flow that quits ends with `tab,y`, the desktop
  spelling of holding Select and pressing Y.
- `-shot-dir` keeps the opening frame, each post-key frame, any
  operation-completion frames, and a `walk.tsv` record. It needs `-keys`:
  without a sequence there is no walkthrough to capture. A key waits for any
  operation it started, so an install finishes before the next key.
- `-walk-timeout` fails a stuck flow instead of hanging a runner.
- The run ends when the sequence is spent. A quit must be the final key; an
  early quit fails and records its final frame.

| Flow | Covers |
| --- | --- |
| `first-run-use-detected` | Setup, saving the detected mapping |
| `first-run-test-detected` | Setup, preview, testing all seven actions, save |
| `first-run-customize` | Calibration, assignment review, preview, save |
| `first-run-safe-exit` | Leaving setup without a mapping |
| `catalogue-walk` | Catalogue, details, action list, confirmation, back out |
| `catalogue-read-only` | A candidate package that offers no action |
| `quit-chord` | Escape at the catalogue does not leave; a held Tab plus Y does |
| `settings-export-diagnostics` | Settings and a diagnostics export |
| `blocked-export-diagnostics` | The blocked screen with no controller at all |
| `install-package` | Download, verify, apply, and the progress screen (`--install`) |
| `install-package-health` | A tampered file, the health check, and repair (`--install`) |
| `installed-uninstall` | The installed action list and uninstall (`--install`) |

The `--install` flows download a real package, so they need network access and
are left out of the default run and CI. Everything else runs offline.

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
the `gui-walkthrough` artifact, including partial evidence when a flow fails.
Reviewers read the screens a change affects without a handheld, and the same
artifact is useful evidence when a device run is not available.

## Keyboard bindings

| Action | Key |
| --- | --- |
| Up, Down, Left, Right | Arrow keys |
| Confirm | Enter |
| Back | Escape |
| Diagnostics | Y |
| Quit | Tab (or F5) held, then Y |

The seven actions above Quit are a binding set like any other, so the footer and
the mapping summary name them: `Confirm (ENTER)  Settings (Y)  Quit (TAB + Y)`.
Quitting is deliberately not one of them: it is the Select chord, the one thing
no single key carries. Keyboard codes
sit above every SDL GameController button, so a key is never mistaken for a pad
button.

## The fixture root

`scripts/desktop-fixture.sh` writes the files the detector reads:

| File | Supplies |
| --- | --- |
| `/etc/os-release` | Knulli firmware identity (`OS_NAME="knulli"`) |
| `/usr/share/knulli/knulli.version` | Release name, for example `scarab` |
| `/boot/boot/knulli.board` | Device ID, for example `trimui-smart-pro` |
| `/lib/ld-linux-aarch64.so.1`, `/lib/libc.so.6` | Architecture ABI and glibc version |
| `/usr/lib/libSDL2*.so.0`, `/lib/libresolv.so.2`, `/lib/libpthread.so.0` | Declared runtime dependencies |
| `/sys/class/graphics/fb0/mode` | Display resolution |

It also writes `.desktop-fixture`, a safety marker that permits the script to
refresh only a fixture directory it created.

Everything else the run writes — installs, logs, saved mappings, recovery
backups — lands below `ROOT`, which is exactly how the device path is exercised
without a device.

**The root is fabricated evidence and must stay a throwaway.** The script
refuses `/`, symlinks, relative paths, and non-empty directories without its own
marker file. Never point it at a mounted SD card or a live `/userdata`.

## Verify against a package

The fixture root reports a device, so a manifest that matches it becomes
actionable and the install, repair, and uninstall flows can be walked. Install
writes below the scratch root and can be undone by deleting `ROOT`; nothing
outside it changes. The catalogue index must be built first, since the GUI reads
it beside the binary or from `-catalog`.

## What this covers, and what it does not

Covers: every rendered screen state, the footer on each one, focus order,
wording, layout bounds, setup, calibration, preview, settings, catalogue,
details, health, confirmations, recovery, blocked, and error states. The
default scripted flows are offline; package lifecycle flows require
`WALK_FLAGS=--install`.

Does not cover: a real SDL GameController, real kernel SDL library loading, real
device filesystem behaviour, or a real `/userdata` write. Record those runs in
`docs/real-device-tests.md`, and see the device guides for
[TrimUI Smart Pro](trimui-smart-pro.md) and
[MagicX Zero 28](magicx-zero-28.md).
