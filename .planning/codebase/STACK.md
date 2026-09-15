# Technology Stack

**Analysis Date:** 2026-09-15

## Languages

**Primary:**
- Go 1.19 - CLI, installer, catalogue, GUI state machine, and tests (`go.mod`)

**Secondary:**
- C (cgo, SDL 2.0 only) - thin wrappers in `internal/sdlui/run.go` behind the `sdl` build tag
- POSIX sh - device packaging in `scripts/package-device.sh` and Ports launchers in `packaging/`
- JSON - package manifests, catalogue index, controller mappings, JSON Schema

## Runtime

**Environment:**
- Go 1.19 toolchain (`go 1.19` in `go.mod`, CI `go-version: "1.19.x"`)
- Device target: linux/arm64 (`aarch64`) on Knulli
- Host tests: local Go with `-race`; GUI tests need SDL2 headers

**Package Manager:**
- Go modules
- Lockfile: `go.sum` present

## Frameworks

**Core:**
- Go standard library for HTTP, archives, JSON, filesystem, and CLI flags
- SDL 2.0 via pkg-config (`#cgo pkg-config: sdl2`) for the optional GUI

**Testing:**
- `testing` plus `net/http/httptest` - no third-party test framework

**Build/Dev:**
- `Makefile` - `fmt`, `vet`, `test`, `build`, `build-ui`, `catalogue`
- `gofmt` - formatting gate
- `go vet` and `staticcheck v0.3.3` in CI (`.github/workflows/check.yml`)
- Renovate `config:recommended` (`renovate.json`)

## Key Dependencies

**Critical:**
- `golang.org/x/image v0.7.0` - bitmap scaling and `basicfont` text in `internal/sdlui`
- SDL2 shared library `libSDL2-2.0.so.0` - GUI only; CLI is CGO-free

**Infrastructure:**
- GitHub Actions `actions/checkout@v4`, `actions/setup-go@v5`, `actions/upload-artifact@v4`
- Debian Bookworm `golang:1.19-bookworm` container for aarch64 device artifacts
- `gcc-aarch64-linux-gnu` and `libsdl2-dev:arm64` for GUI cross-builds

## Configuration

**Environment:**
- No required `.env` file
- `GITHUB_TOKEN` optional for `cmd/check-updates` rate limits; never written into reports
- CLI/GUI flags: `-root`, `-firmware`, `-firmware-version`, `-arch`, `-device`, `-resolution`, `-catalog`

**Build:**
- `go.mod`, `go.sum`, `Makefile`
- `sdl` build tag for GUI packages (`cmd/knulli-app-ui`, `internal/sdlui`)
- Dual build constraints `//go:build sdl` and `// +build sdl` for Go 1.19

## Platform Requirements

**Development:**
- Go 1.19+
- `libsdl2-dev` for GUI compile and `go test -tags sdl`
- `.agents/setup` installs `golang-go` and `libsdl2-dev` on Debian-like hosts

**Production:**
- Knulli on reviewed `aarch64` devices
- TrimUI Smart Pro 1280×720 and MagicX Zero 28 640×480 experimental GUI artifacts
- GUI binary needs `libSDL2-2.0.so.0` and glibc symbols through 2.34 (CI `readelf` checks)
- Writes only under `/userdata`; Ports launcher lives at `/userdata/roms/ports`

---

*Stack analysis: 2026-09-15*
