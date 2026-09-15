# Coding Conventions

**Analysis Date:** 2026-09-15

## Naming Patterns

**Files:**
- Go files named for the concept they own: `guard.go`, `download.go`, `gamelist.go`
- Tests named for the invariant: `TestChecksumFailureWritesNoPackageFiles`
- Device docs and packaging folders use Knulli board ids

**Functions:**
- Exported verbs: `Load`, `Validate`, `Install`, `Extract`, `Detect`
- Unexported helpers stay short: `override`, `zipBytes`, `writeRootFile`

**Variables:**
- Short local names (`pkg`, `m`, `err`)
- Platform fields use device terms: `Firmware`, `Resolution`, `Arch`

**Types:**
- Structs for records: `manifest.Package`, `installer.Manager`, `safefs.Transaction`
- String enums for actions and review status: `appstore.Action`, `review.status`

## Code Style

**Formatting:**
- `gofmt`; CI fails if `gofmt -l .` is non-empty (`Makefile` `fmt` / `check`)
- Tabs, no extra blank-line ceremony
- Import groups: stdlib, then module packages, then rare third-party (`golang.org/x/image`)

**Linting:**
- `go vet ./...` and `go vet -tags sdl ./...`
- CI `staticcheck` v0.3.3 with and without `sdl`

## Import Organization

**Order:**
1. Standard library
2. `github.com/jellydn/knulli-app-store/internal/...`
3. `golang.org/x/image/...` in SDL packages

**Path Aliases:**
- Collision aliases: `storearchive`, `storeinput`, `storeui`, `xdraw` / `imagedraw`
- No Go module path aliases

## Error Handling

**Patterns:**
- Return `error`; do not panic in product code
- Wrap at boundaries: `fmt.Errorf("open diagnostics log: %w", err)`
- User-facing CLI prefix: `fmt.Fprintln(os.Stderr, "error:", err)`
- Compatibility and health errors carry structured evidence, not only a generic string

## Logging

**Framework:** `log` through `internal/diagnostics.Log`

**Patterns:**
- Event names as first argument: `operation_start`, `compatibility_rejected`
- Key/value pairs after the event name
- Redact credentials, query strings, fragments, and common secret fields
- Cap at 512 KiB with `.1` rotation

## Comments

**When to Comment:**
- `CONTRIBUTING.md` requires comments only for a design reason the code cannot make clear
- Build-tag files include both `//go:build sdl` and `// +build sdl`
- cgo preamble in `internal/sdlui/run.go` documents the SDL functions in use

**JSDoc/TSDoc:**
- Not used. Go doc comments are sparse; exported types mostly stand alone

## Function Design

**Size:**
- CLI `run` / `runApply` stay as command switches
- Installer `apply` is the large lifecycle function; helpers split download, XML, status, refresh

**Parameters:**
- `context.Context` first on operations that download or refresh
- `Manager` is a value struct with optional clients and callbacks

**Return Values:**
- `(T, error)` or `error`
- Tests use `t.Fatal` on setup failure and `t.Fatalf` with actual values

## Module Design

**Exports:**
- Small public surface per package; `internal/` prevents external importers
- UI depends on `appstore.Backend`, not on installer internals

**Barrel Files:**
- None. No `doc.go` barrels

**Prose:**
- User and contributor docs follow What / Why / How and ASD-STE100-style short sentences (`README.md` product sections, `docs/`, `CONTRIBUTING.md`)

---

*Convention analysis: 2026-09-15*
