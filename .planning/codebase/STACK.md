# Technology Stack

**Analysis Date:** 2026-09-19

## Languages

**Primary:**
- Go 1.27 - installer CLI, catalogue, GUI state machine, and tests (`go.mod` declares `go 1.27`; CI pins `go-version: "1.27.x"`). 16,056 lines across `cmd/` and `internal/`.

**Secondary:**
- C (cgo, SDL 2.0 only) - hand-written wrappers in the cgo preamble of `internal/sdlui/run.go`, compiled only under the `sdl` tag
- POSIX sh - `scripts/package-device.sh`, `scripts/desktop-fixture.sh`, `scripts/desktop-walk.sh`, and `.agents/setup`
- Bash - the Ports launchers `packaging/trimui-smart-pro/Knulli App Store.sh` and `packaging/magicx-zero-28/Knulli App Store.sh`
- JSON - package manifests (`catalogue/packages/*.json`), the catalogue index, controller mappings, provider record, JSON Schema (`schema/package-manifest-v1.schema.json`)
- XML - EmulationStation `gamelist.xml` parsed and rewritten in `internal/gamelist/xml.go`
- TSV/Markdown - walkthrough records (`internal/sdlui/walk.go`) and catalogue update reports (`internal/updatecheck/report.go`)

## Runtime

**Environment:**
- Go 1.27 toolchain; device target is linux/arm64 (`aarch64`) on Knulli
- Host tests run with `-race`; the GUI packages require SDL2 headers
- Device GUI binaries are checked for exactly two dynamic dependencies (`libSDL2-2.0.so.0` and glibc) with symbols no newer than `GLIBC_2.34` (`.github/workflows/check.yml`)

**Package Manager:**
- Go modules with `go.sum` lockfile
- Exactly one direct dependency; everything else is the standard library

## Frameworks

**Core:**
- Go standard library: `net/http`, `archive/zip`, `archive/tar`, `compress/gzip`, `encoding/json`, `encoding/xml`, `crypto/ed25519`, `crypto/sha256`, `flag`, `log`, `image`, `testing`
- SDL 2.0 through pkg-config (`#cgo pkg-config: sdl2` in `internal/sdlui/run.go`)

**Testing:**
- `testing` plus `net/http/httptest` - no third-party test framework or assertion library
- 248 `TestXxx` functions across 29 `_test.go` files

**Build/Dev:**
- `Makefile` - `all`, `build`, `build-ui`, `catalogue`, `check`, `clean`, `fmt`, `gui`, `test`, `test-ui`, `vet`, `walkthrough`
- `gofmt` - CI fails when `gofmt -l .` is non-empty
- `go vet ./...` and `go vet -tags sdl ./...`
- `staticcheck` v0.8.1, run with and without the `sdl` tag (`.github/workflows/check.yml`)
- Renovate `config:recommended` (`renovate.json`)

## Key Dependencies

**Critical:**
- `golang.org/x/image v0.46.0` - the only module dependency; bitmap scaling in `internal/sdlui/run.go` and text rendering in `internal/sdlui/draw.go`
- `libSDL2-2.0.so.0` - GUI only; the CLI is built with `CGO_ENABLED=0` and stays static

**Infrastructure:**
- GitHub Actions `actions/checkout@v7`, `actions/setup-go@v7`, `actions/upload-artifact@v7`, `actions/download-artifact@v8`
- `golang:1.27-bookworm` container plus `gcc-aarch64-linux-gnu` and `libsdl2-dev:arm64` for device cross-builds
- `zip`, `sha256sum`, and `file`/`readelf` (binutils) in the packaging and ABI-verification steps

## Configuration

**Environment:**
- No `.env` file and no required environment variables for the CLI or GUI
- `GITHUB_TOKEN` is optional and only raises the GitHub API rate limit for `cmd/check-updates`; `.github/workflows/catalogue-updates.yml` supplies `github.token`
- Verification-only variables: `KNULLI_UI_SCREENSHOT_DIR` (`internal/sdlui/draw_test.go`), `WALK_*` overrides and `SDL_VIDEODRIVER=dummy` (`scripts/desktop-walk.sh`)

**CLI flags** (`cmd/knulli-app/main.go`):
- `-root`, `-firmware`, `-firmware-version`, `-arch`, `-device`, `-resolution`
- `catalogue`: `-dir`, `-output`, `-signing-key`, `-generate-signing-key`

**GUI flags** (`cmd/knulli-app-ui/main.go`):
- `-catalog`, `-root`, `-firmware`, `-firmware-version`, `-arch`, `-device`, `-resolution`
- `-windowed`, `-screenshot`, `-input`, `-keys`, `-shot-dir`, `-walk-timeout`

**Build:**
- `go.mod`, `go.sum`, `Makefile`, `renovate.json`
- The `sdl` build tag (`//go:build sdl`) gates `cmd/knulli-app-ui`, `internal/sdlui`, and their tests
- The catalogue public key is injected at link time: `-X github.com/jellydn/knulli-app-store/internal/catalog.embeddedPublicKeyHex` (`internal/catalog/sign.go`)

## Platform Requirements

**Development:**
- Go 1.27 or newer and `libsdl2-dev` for `go test -tags sdl ./...`
- `.agents/setup` installs a checksum-verified Go 1.27.1 tarball when the local toolchain is older than 1.27, then installs `libsdl2-dev` on Debian-like hosts
- `make gui` needs no handheld and no controller: the keyboard and a fixture root stand in (`scripts/desktop-fixture.sh`)

**Production:**
- Knulli on reviewed `aarch64` devices; writes stay below `/userdata`
- TrimUI Smart Pro 1280×720 and MagicX Zero 28 640×480 are the two packaged GUI targets
- The GUI binary needs `libSDL2-2.0.so.0` and glibc symbols through 2.34
- The Ports launcher lands at `/userdata/roms/ports`; the app tree sits at `/userdata/roms/ports/knulli-app-store`

---

*Stack analysis: 2026-09-19*
