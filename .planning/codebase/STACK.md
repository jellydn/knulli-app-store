# Technology Stack

**Analysis Date:** 2026-10-21

## Languages

**Primary:** Go 1.27, declared in `go.mod`. Production code lives in `cmd/` and `internal/`.

**Secondary:**
- C through cgo in `internal/sdlui/run.go`, for SDL2 window, controller, event, and texture calls.
- POSIX shell in `scripts/`; Bash in the device launchers, including `packaging/magicx-zero-28/Knulli App Store.sh`.
- JSON for manifests and persisted records; YAML for `.github/workflows/`.
- No JavaScript web frontend or server framework is present in `go.mod` or `cmd/`.

## Runtime

**Environment:**
- Linux aarch64 on device; the CLI also builds without cgo. `scripts/device-build.sh` cross-compiles for `GOOS=linux GOARCH=arm64`.
- `.agents/setup` installs checksum-verified Go 1.27.1 if an existing Go >= 1.27 is not available. It also installs SDL2 development headers.
- `.github/workflows/check.yml` pins test and walkthrough jobs to Go 1.27.1 because Staticcheck v0.8.1 cannot read Go 1.27.2 export data. Device builds use `golang:1.27-bookworm`; the weekly checker uses 1.27.x.

**Package Manager:** Go modules with `go.mod` and `go.sum`. SDL2 is a system dependency found through `pkg-config` in `internal/sdlui/run.go`.

## Frameworks

**Core:** Standard-library Go modules with explicit ownership boundaries, not an application framework. See `internal/installer/installer.go` and `internal/appstore/service.go`.

**Testing:** Standard-library `testing` and `net/http/httptest`. The refreshed tree has 31 Go test files and 290 top-level `Test` functions across `cmd/` and `internal/`, including the new staging-recovery regression case; these are source counts, not executed results.

**Build/Dev:**
- `Makefile`: build, catalogue, formatting, vet, race tests, coverage, SDL tests, desktop GUI, and walkthrough tasks.
- `prek.toml`: local commit/push hooks; Staticcheck is pinned to v0.8.1.
- `.github/workflows/check.yml`, `.github/workflows/release.yml`, and `.github/workflows/catalogue-updates.yml`: verification, release publication, and read-only package update reports.
- `renovate.json`: dependency update configuration.

## Key Dependencies

**Critical:** `golang.org/x/image v0.47.0` is the only direct Go module requirement (`go.mod`). It supplies image scaling and text drawing in `internal/sdlui/run.go` and `internal/sdlui/draw.go`.

**Infrastructure:**
- SDL2 shared library and glibc for the GUI; `scripts/device-build.sh` requires an SDL2 dependency, exactly two ELF NEEDED entries, and a highest numeric required GLIBC version no newer than 2.34.
- POSIX `syscall.Flock` and `syscall.Statfs` in `internal/installer/stage.go` and `internal/safefs/files.go`; native Windows support is not implemented.
- Standard-library ed25519/SHA-256 for catalogue and release integrity (`internal/catalog/sign.go`, `internal/installer/download.go`).

## Configuration

**Environment:**
- Runtime flags and detected Knulli files, not `.env` files: `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`, `internal/platform/platform.go`.
- `STORE_CONTROLLER_LAYOUT=knulli|xbox` selects logical face-button bindings and labels in `internal/input/mapping.go`. Device launchers choose it from `/var/run/ports_xbox_layout.flag`.
- `SDL_GAMECONTROLLERCONFIG` supplies Knulli's controller database; `SDL_VIDEODRIVER=dummy` supports headless walkthroughs (`internal/sdlui/run.go`, `.github/workflows/check.yml`).
- Optional `GITHUB_TOKEN` is used by the maintenance checker, not Store self-update (`cmd/check-updates/main.go`, `internal/installer/selfupdate.go`).

**Build:**
- CLI: `make build` uses `CGO_ENABLED=0` and `-trimpath` (`Makefile`).
- GUI: `make build-ui` uses `-tags sdl`; only `internal/sdlui/run.go`, `internal/sdlui/draw.go`, SDL draw tests, and the GUI entry point require the SDL build path.
- `make catalogue-check` compares two generated index files byte for byte; a diff against ignored `build/` output is not a valid gate (`Makefile`).
- `scripts/device-build.sh TARGET VERSION OUTPUT_DIR` signs the catalogue, stamps its public key plus Store version/target into both binaries, and calls `scripts/package-device.sh`.
- Device ZIPs contain seven files, including the launcher, icon, binaries, catalogue/signature pair, and README (`scripts/package-device.sh`).

## Platform Requirements

**Development:** Go >= 1.27; SDL2 development headers and a C compiler for `make vet`, `make check`, `make test-ui`, and GUI work. `make build` and untagged `go test ./...` do not need SDL (`Makefile`).

**Production:**
- Two packaged build targets: TrimUI Smart Pro and MagicX Zero 28 (`scripts/device-build.sh`). This is not an installer device allowlist.
- Firmware/version, architecture, ABI/glibc, and dependencies gate package installation. Device and display scope are advisory; exact matching evidence controls the tested badge (`internal/platform/platform.go`, `internal/manifest/manifest.go`).
- Package mutations stay below `/userdata`, under declared path policy and journaled transactions (`internal/safefs/guard.go`, `internal/installer/commit.go`).

---

*Stack analysis: 2026-10-21*
