# Desktop GUI verification

The SDL GUI can be walked end to end on a development machine, with no handheld
and no controller attached. The keyboard carries every semantic action — the
seven required ones and the optional paging pair — and a scratch fixture root
supplies the Knulli files that platform detection reads, so a desktop run can
simulate the selected device matrix. The target architecture
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
also renders every screen state under `build/walkthrough/screens/`, at both
target resolutions. Those stills name the screen they show, so the catalogue
view is `<device>-gui-catalogue.png` with its tab bar, `-gui-tabs-ready.png`
for the bar on another view, `-gui-notice.png` for the notice bar over a
completed operation, and one file per remaining state. The screenshots are
generated evidence and are not checked in; the `gui-walkthrough` artifact
carries them. For a host with no window server, run:

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

- `-keys` accepts `up`, `down`, `left`, `right`, `pgup`, `pgdown`, `enter`,
  `esc`, `y`, `tab`, in the order to press them. Each key goes through the same handler as a real
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
| `first-run-use-detected` | Setup, the paging question, saving the detected mapping |
| `first-run-skip-paging` | The paging question answered Skip: the saved file binds the required actions only |
| `first-run-test-detected` | Setup, the paging question, preview, testing every bound action, save |
| `first-run-customize` | Calibration, assignment review, preview, save |
| `first-run-safe-exit` | Leaving setup without a mapping |
| `catalogue-walk` | Catalogue, details, action list, confirmation, back out |
| `catalogue-tabs` | The tab bar: right steps onto Ready, then the installed tab, and left steps back |
| `catalogue-read-only` | A candidate package that offers no action |
| `quit-chord` | Escape at the catalogue does not leave; a held Tab plus Y does |
| `settings-export-diagnostics` | Settings and a diagnostics export |
| `blocked-export-diagnostics` | The blocked screen with no controller at all |
| `install-package` | Download, verify, apply, and the progress screen (`--install`) |
| `install-package-health` | A tampered file, the health check, and repair (`--install`) |
| `installed-uninstall` | The installed action list and uninstall (`--install`) |

The `--install` flows download a real package, so they need network access and
are left out of the default run and CI. Everything else runs offline.

Two flows answer the paging question the other way round, because the answer is
a decision with no screen left to read afterwards: the walk script checks the
mapping file each of them wrote, so `first-run-use-detected` and
`install-package` must have saved the pair and `first-run-skip-paging` must
not. That check is what keeps "skippable" from meaning "quietly dropped".

### Reading the evidence

`build/walkthrough/summary.tsv` lists every flow with the screens it reached and
whether it reached them all. Beside each flow, `walk.tsv` records one row per
frame:

```text
file	state	key	item	focus	action	busy	mode	tab	session_message	message	toast	error
```

`tab` names the catalogue view the frame was showing, and `toast` carries the
notice bar's line, which is where a completed operation reports itself.

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
| Up, Down | Arrow keys, one row at a time |
| Left, Right | Arrow keys, one tab at a time on the catalogue and one button at a time inside a panel |
| Page Up, Page Down | Page Up, Page Down |
| Confirm | Enter |
| Back | Escape |
| Diagnostics | Y |
| Quit | Tab (or F5) held, then Y |

Every action above Quit is a binding set like any other, so the footer and the
mapping summary name them. Seven of them are required — up, down, left,
right, confirm, back, and diagnostics — and the paging pair is optional, so a
pad with no shoulder buttons answers the paging question with **Skip paging**
and finishes setup without them. When the pair is unbound the catalogue never
advertises it: the footer's paired hint names both buttons or neither. A catalogue that shows all of its rows reads
`Confirm (ENTER)  Settings (Y)  Quit (TAB + Y)`; once the list outgrows the
window, the range under the last row names where the window sits and the footer
gains the one hint that carries two buttons, `Page (PGUP/PGDN)`. Quitting is
deliberately not one of the nine: it is the Select chord, the one thing no
single key carries. Keyboard codes
sit above every SDL GameController button, so a key is never mistaken for a pad
button.

## Tabs and notices

The catalogue carries a tab bar above its rows. `ALL` leads, because the app
lands there and so the default view hides nothing; `READY` holds the packages
with an install or an adoption waiting; `INSTALLED` holds the packages the app
manages on the device, failing health checks included. The active tab carries
the same accent wash and ring as a selected row. Sideways steps the bar and
vertically steps the rows, so a direction key never means two things at once,
and an empty tab still steps, which is the way out of it. Each tab remembers
the row it was left on, so coming back to one lands where the reader stopped;
a tab never opened opens on the package the user was already reading, and a
remembered package that has since left the tab gives way to its first row.

A finished operation reports itself in a notice bar over the panel foot instead
of leaving a status banner behind: it is drawn below the action row and above
the footer, and it clears itself after three seconds. A failure is not a notice:
it keeps the status block until the user answers it.

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
tabs, notices, details, health, confirmations, recovery, blocked, and error
states. The
default scripted flows are offline; package lifecycle flows require
`WALK_FLAGS=--install`.

Does not cover: a real SDL GameController, real kernel SDL library loading, real
device filesystem behaviour, or a real `/userdata` write. Record those runs in
`docs/real-device-tests.md`, and see the device guides for
[TrimUI Smart Pro](trimui-smart-pro.md) and
[MagicX Zero 28](magicx-zero-28.md).
