# Technology Stack

**Analysis Date:** 2026-10-25

**Release scope:** Release target: Store v2026.10.25.1. Hardware reports remain scoped to the artifacts actually tested. Published v2026.10.25 at `14490119` was the baseline for the historical artifact checks below. Picker 0.2.1 is published but UNTESTED on all device matrices; 0.2.0 hardware reports and the Store's 0.2.0-derived layout remain separate facts. The other six current package versions retain Smart Pro-only evidence, with all seven review statuses experimental.

## Languages

**Primary:**
- Go 1.27. The module is `github.com/jellydn/knulli-app-store`. Commands are in `cmd/`; implementation packages are in `internal/`. Source: `go.mod`, `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`.

**Secondary:**
- POSIX shell for builds and desktop fixtures; Bash for device launchers. Source: `scripts/device-build.sh`, `scripts/desktop-fixture.sh`, `packaging/trimui-smart-pro/Knulli App Store.sh`.
- JSON for package manifests, the catalogue index, and installed state. YAML defines GitHub Actions workflows. Source: `catalogue/packages/`, `internal/catalog/catalog.go`, `internal/installer/state.go`, `.github/workflows/`.
- The GUI uses C libraries through Go SDL bindings and cgo. There is no JavaScript frontend in the application commands or module dependencies. Source: `internal/sdlui/run.go`, `go.mod`, `scripts/device-build.sh`.

## Runtime

**Environment:**
- Packaged device binaries run on Linux arm64. The CLI uses `CGO_ENABLED=0`; the GUI uses `CGO_ENABLED=1` and the `sdl` build tag. Source: `scripts/device-build.sh`.
- The setup script requires exactly Go 1.27.1 (and `gofmt`); it installs the checksum-verified toolchain when the detected version differs, including newer patch versions. It also installs SDL development packages. Source: `.agents/setup`.
- Main CI test and walkthrough jobs select Go 1.27.1. Their comments record a Staticcheck v0.8.1 export-data limit with Go 1.27.2. Device jobs use `golang:1.27.1-bookworm`; the weekly checker selects `1.27.x`. Source: `.github/workflows/check.yml`, `.github/workflows/catalogue-updates.yml`, `.github/workflows/release.yml`.

**Package Manager:**
- Go modules. `go.mod` declares versions; `go.sum` records dependency checksums. System SDL packages come from apt in setup and device builds. Source: `go.mod`, `go.sum`, `.agents/setup`, `scripts/device-build.sh`.

## Frameworks

**Core:**
- The installer and service use Go standard-library packages, not a server framework. Frontends call the installer through the service or manager. Source: `internal/installer/installer.go`, `internal/appstore/service.go`, `cmd/knulli-app/main.go`.
- Gabagool v2.26.6 owns GUI window and renderer lifetime through its advanced API. Its keyboard, controller, and joystick input processors are disabled. The Store keeps its own event loop and semantic input mapping. Source: `go.mod`, `internal/sdlui/run.go`, `internal/sdlui/events.go`, `internal/input/mapping.go`.

**Testing:**
- Go `testing` and `net/http/httptest` provide unit and integration tests. SDL tests use the `sdl` build tag. Source: `internal/installer/installer_test.go`, `internal/updatecheck/check_test.go`, `internal/sdlui/run_test.go`.
- The offline walkthrough drives the real GUI binary with scripted keys and a fixture root. It records frames and a `walk.tsv` file. This does not prove real-device controller or library behavior. Source: `scripts/desktop-walk.sh`, `internal/sdlui/walk.go`, `internal/sdlui/run.go`, `CONTEXT.md`.

**Build/Dev:**
- Make targets build the CLI and GUI, generate the catalogue, run checks, and record GUI evidence. Source: `Makefile`.
- `prek` hooks run formatting, vet for both tag sets, and the catalogue gate. Pre-push hooks add race tests and Staticcheck v0.8.1 for both tag sets. Source: `prek.toml`.
- GitHub Actions runs verification, device builds, releases, and weekly upstream checks. Renovate uses `config:recommended`. Source: `.github/workflows/check.yml`, `.github/workflows/release.yml`, `.github/workflows/catalogue-updates.yml`, `renovate.json`.

## Key Dependencies

**Critical:**
- `github.com/BrandonKowalski/gabagool/v2 v2.26.6`: GUI initialization, window access, renderer access, presentation, and shutdown. Source: `go.mod`, `internal/sdlui/run.go`.
- `github.com/veandco/go-sdl2 v0.4.40`: SDL events, GameController access, textures, and renderer calls. Source: `go.mod`, `internal/sdlui/run.go`, `internal/sdlui/events.go`.
- `golang.org/x/image v0.47.0`: image scaling and OpenType text rasterization. The responsive Library canvas follows Random Game Picker 0.2.0, using embedded DejaVu Sans at 18/26 logical px rather than fixed bitmap glyphs. Font and license are checked in. Source: `go.mod`, `internal/sdlui/draw.go`, `internal/sdlui/layout.go`, `internal/sdlui/assets/DejaVuSans.ttf`, `internal/sdlui/assets/DejaVuSans-LICENSE.txt`.

**Infrastructure:**
- Indirect modules include `certifiable v1.3.0`, `go-evdev`, `oksvg`, `rasterx`, `go.uber.org/atomic v1.11.0`, and Go `x/net`, `x/sys`, and `x/text`. These are module dependencies, not evidence of enabled Store features. Exact versions are in `go.mod` and `go.sum`.
- GUI development needs SDL2, SDL2_image, SDL2_ttf, SDL2_gfx, a C compiler, and `pkg-config`. Source: `.agents/setup`, `.github/workflows/check.yml`, `scripts/device-build.sh`.
- Device builds link SDL2_gfx statically. The ELF gate permits exactly six shared-library names: `ld-linux-aarch64.so.1`, `libSDL2-2.0.so.0`, `libSDL2_image-2.0.so.0`, `libSDL2_ttf-2.0.so.0`, `libc.so.6`, and `libm.so.6`. The highest required GLIBC version must be no newer than 2.34. Source: `scripts/device-build.sh`.
- Filesystem code uses `syscall.Flock` for the manager lock and `syscall.Statfs` for free-space checks. Source: `internal/installer/stage.go`, `internal/safefs/files.go`.
- Standard-library ed25519 signs the catalogue; SHA-256 checks manifest and release bytes. Source: `internal/catalog/sign.go`, `internal/catalog/catalog.go`, `internal/installer/download.go`.

## Configuration

**Environment:**
- CLI flags select the filesystem root and platform overrides. GUI flags add catalogue path, input mode, window mode, screenshots, and walkthrough controls. Source: `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`.
- Platform detection reads Knulli files. Firmware identity uses `/etc/os-release`; the preferred version source is `/usr/share/knulli/knulli.version`. Device identity and explicit resolution have separate evidence sources. Source: `internal/platform/platform.go`.
- `STORE_CONTROLLER_LAYOUT=knulli` applies the Knulli face-letter mapping. Device launchers select `xbox` when `/var/run/ports_xbox_layout.flag` exists. Other values use direct SDL letters. Source: `internal/input/mapping.go`, `packaging/trimui-smart-pro/Knulli App Store.sh`, `packaging/magicx-zero-28/Knulli App Store.sh`.
- `INPUT_CAPTURE` must be unset: the GUI rejects it to prevent the toolkit mapping wizard from changing the Store input contract. Renderer probing keeps a software fallback. Source: `internal/sdlui/run.go`.
- CI uses `SDL_VIDEODRIVER=dummy` and `SDL_AUDIODRIVER=dummy` for offline GUI evidence. `KNULLI_UI_SCREENSHOT_DIR` selects test screenshot output. Source: `.github/workflows/check.yml`, `Makefile`, `internal/sdlui/draw_test.go`.
- `GITHUB_TOKEN` is optional for the upstream metadata checker. It is not read by Store self-update. Source: `cmd/check-updates/main.go`, `internal/installer/selfupdate.go`.

**Build:**
- `make build` creates a cgo-free CLI. `make build-ui` selects `-tags sdl`. Both use `-trimpath`. Source: `Makefile`.
- `make check` runs formatting, vet for both tag sets, and untagged race tests. `make cover` requires a test file in each production package in the untagged package list and at least 70% total coverage by default. SDL race tests run separately in CI. Source: `Makefile`, `.github/workflows/check.yml`.
- `make catalogue-check` compares two generated index files byte for byte. Generated `build/` files are ignored; a Git diff cannot check their determinism. Source: `Makefile`, `.gitignore`.
- `scripts/device-build.sh TARGET VERSION OUTPUT_DIR` generates a signing key, signs the catalogue, passes public-key and Store version/target linker flags to both builds, cross-compiles, and checks the GUI ABI. The GUI retains the public key used when opening the catalogue; the CLI's unused key is eliminated by linking, so flags alone do not prove key presence in both binaries. Source: `scripts/device-build.sh`, `internal/catalog/catalog.go`, `internal/appstore/service.go`; artifact verification attributed to source thread `T-01a16e63-3dfe-7782-8ac5-a9a71a55726c`.
- Host `zip`, `file`, and `readelf` are checked independently of cross-compiler/arm64 SDL availability; missing tools trigger installation of `zip`, `file`, and `binutils` even when cross libraries already exist. Source: `scripts/device-build.sh`.
- The device ZIP has seven files: launcher, icon, CLI, GUI, index, signature, and README. Source: `scripts/package-device.sh`, `internal/installer/selfupdate.go`.

## Platform Requirements

**Development:**
- The module minimum is Go 1.27; use exact Go 1.27.1 for the setup/CI analyzer-compatible development path. GUI work and the tagged vet step need the SDL development packages and C toolchain. The CLI build disables cgo. Source: `go.mod`, `.agents/setup`, `Makefile`, `.github/workflows/check.yml`.
- Desktop fixtures support keyboard tests without a device. Fixture writes go to a selected scratch root. Source: `scripts/desktop-fixture.sh`, `scripts/desktop-walk.sh`, `Makefile`.

**Production:**
- The build scripts package two targets: TrimUI Smart Pro and MagicX Zero 28. These build targets do not form a package-install device allowlist. Source: `scripts/device-build.sh`, `internal/platform/platform.go`.
- Only Knulli is implemented. Firmware/version, architecture, ABI/glibc, and runtime dependencies gate installation. Device and display scope are advisory. Exact device-test evidence controls the tested badge. Source: `internal/platform/platform.go`, `internal/manifest/manifest.go`, `internal/appstore/service.go`.
- Installer mutations stay below `/userdata` through path guards and transactions. Runtime writes by upstream-managed apps are not sandboxed by the Store. Source: `internal/safefs/guard.go`, `internal/installer/commit.go`, `internal/manifest/manifest.go`.
- This refresh passed both local ARM64 device builds, package/signature/ABI checks, and desktop GUI tests; see `TESTING.md`. No new hardware test ran. Attributed evidence from source thread `T-01a16e63-3dfe-7782-8ac5-a9a71a55726c` records public v2026.10.25 at source `14490119b819f524a8a59919b949359092b3b014`, with both release ZIP catalogue signatures verified against their embedded GUI keys. Native TSP evidence used a different development ZIP from the same source, not those exact release artifacts. See `docs/real-device-tests.md` for the record and limits; earlier native results do not verify the subsequent dirty-redraw change.

---

*Stack analysis: 2026-10-25*
