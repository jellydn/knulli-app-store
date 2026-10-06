# Technology Stack

**Analysis Date:** 2026-09-21

## Languages

**Primary:**
- Go 1.27 - Entire codebase: 9,595 production lines across `internal/`, `cmd/`, `scripts/` helpers are POSIX `sh`

**Secondary:**
- C - A single inline cgo block in `internal/sdlui/run.go` wraps the SDL2 functions the GUI needs
- POSIX `sh` - `scripts/device-build.sh`, `scripts/package-device.sh`, `scripts/desktop-fixture.sh`, `scripts/desktop-walk.sh`, `packaging/*/Knulli App Store.sh`
- JSON - Manifest contract (`catalogue/packages/*.json`), JSON Schema (`schema/package-manifest-v1.schema.json`), catalogue index, installed/lifecycle/recovery state
- YAML - GitHub Actions workflows in `.github/workflows/`

**Deliberately absent:**
- TypeScript / JavaScript - there is no web front end; the GUI is SDL2 rendered from Go
- Python - not used anywhere in the build or test path

## Runtime

**Environment:**
- Go 1.27 (`go.mod` declares `go 1.27`; `.github/workflows/check.yml` pins `go-version: "1.27.x"`)
- Device target: Linux `aarch64`, glibc ABI `linux-aarch64-glibc`, minimum glibc 2.34
- Pinned Go toolchain for cloud agents: `go1.27.1` in `.agents/setup` with SHA-256 verification

**Build Modes:**
- CLI (`cmd/knulli-app`): `CGO_ENABLED=0`, `-trimpath`, fully static — no libc or SDL dependency
- GUI (`cmd/knulli-app-ui`): `CGO_ENABLED=1` + `//go:build sdl`, links exactly two shared libraries (libSDL2-2.0.so.0 and glibc), enforced by `readelf` in `scripts/device-build.sh`
- Desktop GUI: same `sdl` tag, built on macOS/Linux with `libsdl2-dev` / `brew install sdl2`

**Package Manager:**
- Go modules (`go.mod` + `go.sum`)
- System libs via `pkg-config: sdl2`; device builds install `libsdl2-dev:arm64` and `aarch64-linux-gnu-gcc` in `scripts/device-build.sh`

## Frameworks

**Core:** None. The project uses the Go standard library only, plus one image helper.

**Testing:**
- `testing` (stdlib) — 34 test files, 7,735 test lines, 191 `func Test` functions
- `net/http/httptest` — 6 uses for fake release servers and loopback refresh servers
- No third-party assertion, mocking, or snapshot library

**Build/Dev:**
- `make` — the canonical task runner (`Makefile`)
- `gofmt` / `go vet` / `staticcheck`
- `prek` (`prek.toml`) — git hooks mirroring CI, with `staticcheck@v0.8.1` pinned via `go run`
- GitHub Actions — `check.yml`, `release.yml`, `catalogue-updates.yml`
- Renovate (`renovate.json`, `config:recommended` only)

## Key Dependencies

**Critical:**
- `golang.org/x/image v0.46.0` — the only direct module dependency. Used in `internal/sdlui/run.go` for `xdraw.NearestNeighbor` canvas-to-display scaling and in `internal/sdlui/draw.go` for font/raster drawing.

**Infrastructure (system, not Go modules):**
- SDL2 (`libSDL2-2.0.so.0`) — game controller enumeration, events, window/renderer/texture. Behind the `sdl` tag.
- `syscall.Flock` — exclusive manager lock in `internal/installer/stage.go` (Linux/POSIX only)
- `crypto/ed25519` (stdlib) — catalogue index signing in `internal/catalog/sign.go`

**Notable stdlib reliance:**
- `archive/zip`, `archive/tar`, `compress/gzip` — bounded extraction in `internal/archive/archive.go`
- `flag` — CLI parsing in `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`, `cmd/check-updates/main.go`
- `encoding/json` — every persisted schema; decode paths use `DisallowUnknownFields()` and reject trailing JSON

## Configuration

**Environment:**
- No `.env` files, no config files read at runtime. All runtime configuration is flags or detected files.
- `GITHUB_TOKEN` — optional, read only by `cmd/check-updates/main.go` for API rate limits; never written to a report
- `SDL_GAMECONTROLLERCONFIG` — read by the GUI to prefer Knulli's generated mapping (`internal/sdlui/run.go`)
- `SDL_VIDEODRIVER=dummy` — set by CI for headless walkthrough runs
- Makefile knobs: `GUI_DEVICE`, `GUI_ARCH`, `GUI_RESOLUTION`, `GUI_ROOT`, `WALK_ROOT`, `WALK_OUT`, `WALK_FLAGS`, `COVER_MIN` (default 70)
- Walkthrough knobs in `scripts/desktop-walk.sh`: `WALK_BIN`, `WALK_INDEX`, `WALK_DEVICE`, `WALK_ARCH`, `WALK_RESOLUTION`, `WALK_TIMEOUT`, `WALK_PACKAGE_PATH`

**Build:**
- `Makefile` — `check`, `build`, `build-ui`, `catalogue`, `catalogue-check`, `fmt`, `vet`, `test`, `test-ui`, `cover`, `cover-untested`, `gui`, `walkthrough`, `clean`
- `make catalogue-check` — the determinism gate: builds the index twice and requires the two builds to be byte-identical. `build/` is gitignored, so a `git diff` on the generated index cannot verify it; the `prek.toml` hook calls this target so the documented command and the enforced check stay one implementation.
- `scripts/device-build.sh TARGET VERSION OUTPUT_DIR` — the single home of the signed cross-compile, ABI gate, and packaging; used by both CI and tag releases
- `scripts/device-fixture.sh` / `scripts/desktop-fixture.sh` — write the Knulli files platform detection reads into a scratch root
- `-ldflags "-X github.com/jellydn/knulli-app-store/internal/catalog.embeddedPublicKeyHex=..."` bakes the catalogue public key into device binaries

## Platform Requirements

**Development:**
- Go >= 1.27
- `libsdl2-dev` (Linux) or `sdl2` via Homebrew (macOS) for `-tags sdl` work
- No SDL and no CGO needed for `make check`, `make build`, or the CLI tests
- GCC/PKG_CONFIG not required unless the `sdl` tag is used

**Production:**
- Knulli running on `aarch64` with glibc >= 2.34
- Explicitly supported device identities read from a Knulli board file: `trimui-smart-pro` (1280×720) and `magicx-zero-28` (640×480)
- Writes confined to `/userdata`; `/userdata` is on the SD card and may normalize permission bits

---

*Stack analysis: 2026-09-21*
