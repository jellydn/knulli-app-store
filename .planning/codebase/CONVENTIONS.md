# Coding Conventions

**Analysis Date:** 2026-10-25

## Naming Patterns

**Files:**
- Use lowercase Go filenames. Split related behavior into files in the same package, such as `stage.go`, `commit.go`, and `uninstall.go` in `internal/installer/`.
- Put tests beside the source in `*_test.go` files. Use `testdata/` for stored fixtures (`internal/installer/installer_test.go`, `internal/installer/testdata/gamelist.xml`).

**Functions:**
- Use MixedCaps for exported functions and mixedCaps for private functions (`Manager.Apply` and `checkPreconditions` in `internal/installer/installer.go`).
- Name copy builders `With...`. `WithPlatform` clones platform data; `WithStatusCache` attaches a shared cache (`internal/installer/installer.go`).

**Variables:**
- Use Go names for constants, not uppercase underscore names: `SchemaV1`, `MaxPackageBytes`, and `OpInstall` (`internal/manifest/manifest.go`, `internal/installer/installer.go`).
- Use short local names where the type and scope are clear, such as `ctx`, `pkg`, `tx`, and `err` (`internal/installer/commit.go`).

**Types:**
- Use domain nouns for structs and named string types for actions. Examples are `Manager`, `OperationOutcome`, `Op`, and `appstore.Action` (`internal/installer/installer.go`, `internal/installer/refresh.go`, `internal/appstore/service.go`).
- Use the terms in `CONTEXT.md`. Show adopt as **Manage existing**. An external installation means existing files, not empty directories. An accepted menu refresh is not a completed refresh.

## Code Style

**Formatting:**
- Use `gofmt`. `make fmt` changes files; CI rejects any output from `gofmt -l .` (`Makefile`, `.github/workflows/check.yml`).
- Hooks check trailing whitespace, final newlines, JSON, YAML, large files, conflict markers, and LF line endings (`prek.toml`).

**Linting:**
- Run `go vet` for the default build and the `sdl` build. `make check` runs formatting, both vet checks, and default race tests (`Makefile`).
- Pre-push hooks and CI use Staticcheck v0.8.1 for both build tag sets (`prek.toml`, `.github/workflows/check.yml`).
- Setup and main CI pin exact Go 1.27.1. Setup replaces a different detected version, including newer patches. CI's comment states that Staticcheck v0.8.1 cannot read Go 1.27.2 export data; the module's Go 1.27 minimum is not the analyzer-compatible setup contract (`.agents/setup`, `.github/workflows/check.yml`, `go.mod`).

## Import Organization

**Order:**
1. Standard library imports.
2. A separate group for module and external imports, where used.
3. Names sorted by `gofmt` within each group. External and module imports can share a group; there is no enforced three-group rule (`internal/sdlui/run.go`, `internal/installer/commit.go`).

**Path Aliases:**
- Use aliases when names would be unclear: `storearchive`, `storeinput`, `storeui`, and `gaba` (`internal/installer/commit.go`, `internal/sdlui/run.go`).
- Use the full module import path for internal packages (`internal/appstore/service.go`, `go.mod`).

## Error Handling

**Patterns:**
- Return errors to the caller. Add context with `fmt.Errorf` and `%w` when the cause must remain available (`internal/appstore/service.go`, `internal/installer/refresh.go`).
- Use typed errors for actionable conflicts. `AdoptionConflictError` carries the path that blocks adoption (`internal/installer/installer.go`).
- Use `(value, bool)` for an expected lookup miss, not an error (`internal/input/mapping.go`).
- Collect manifest validation problems and sort them for stable output. Validate package policy before installer filesystem work (`internal/manifest/manifest.go`, `internal/installer/installer.go`).
- Defer rollback against the named result. Keep rollback failures in the reported error rather than losing the first failure (`internal/installer/commit.go`, `internal/installer/commit_test.go`).
- Keep commit results separate from menu refresh results. A refused refresh sets `RestartRequired`; it does not undo a committed operation (`internal/installer/refresh.go`).
- Reject failed firmware, architecture, ABI, or dependency checks. Device identity and reviewed display scope do not gate installation. Report raw values, normalized values, sources, and the detected matrix (`CONTEXT.md`, `internal/platform/platform.go`).

## Logging

**Framework:** `diagnostics.Log` wraps the standard `log` package (`internal/diagnostics/log.go`).

**Patterns:**
- Call `Event(name, "key", value, ...)`. Event names use lowercase words with underscores, such as `operation_start` and `catalogue_loaded` (`internal/installer/installer.go`, `internal/appstore/service.go`).
- Sanitize field names and redact sensitive values. Existing logs and exported reports are also redacted. Nil log receivers accept `Event` calls (`internal/diagnostics/log.go`).
- Bound the log to 512 KiB per file and write exports below the fixture or device root (`internal/diagnostics/log.go`).

## Comments

**When to Comment:**
- Explain a design reason that the code does not show. Do not restate the next statement (`AGENTS.md`).
- Explain safety and integration choices. The SDL startup comments explain why toolkit input is disabled and why mapping bytes are empty (`internal/sdlui/run.go`).
- Use Go doc comments for public behavior, such as the copy contract of `WithPlatform` and the commit contract of `Manager.Apply` (`internal/installer/installer.go`).

**JSDoc/TSDoc:**
- Not used for the Go application. Use Go comments and Markdown documentation (`go.mod`, `internal/installer/installer.go`).

## Function Design

**Size:** Keep each helper in the package that owns its behavior. The installer entry point separates preconditions, staging, and commit; no numeric function-size rule is specified (`internal/installer/installer.go`, `AGENTS.md`).

**Parameters:** Pass `context.Context` to work that can be cancelled. Inject HTTP clients, the clock, free-space checks, and refresh endpoints through manager fields (`internal/installer/installer.go`).

**Return Values:** Return explicit result structs where success has more than one outcome. Use value receivers for manager copy builders and pointer state for transactions and UI models (`internal/installer/refresh.go`, `internal/installer/installer.go`, `internal/safefs/transaction.go`, `internal/ui/model.go`).

## Module Design

**Exports:**
- Keep application packages under `internal/`. Frontends call `installer.Manager` or the four-method `appstore.Backend`; they must not extract archives, edit XML, or run shell commands for package operations (`AGENTS.md`, `internal/appstore/service.go`).
- Keep installer policy independent of presentation code. UI tests use an in-memory backend (`AGENTS.md`, `internal/ui/model_test.go`).
- Keep SDL-dependent files behind `//go:build sdl` so the CLI can build without CGO (`internal/sdlui/run.go`, `Makefile`).
- Let Gabagool own the GUI renderer lifetime, but keep its input sources disabled. The Store owns semantic controls and transitions (`internal/sdlui/run.go`, `AGENTS.md`).
- Keep the Option B Library list and expanded selected row on the shared responsive 480-high canvas, with embedded DejaVu Sans text and contextual glyphs. Do not restore the obsolete one-card/bitmap-font design or introduce toolkit blocking screens (`AGENTS.md`, `internal/sdlui/layout.go`, `internal/sdlui/draw.go`, `docs/adr/0010-gabagool-knulli-frontend.md`).

**Barrel Files:**
- No re-export layer is used. Source files define behavior in their owning package. Shared manifest limits and schema names live in `internal/manifest/manifest.go`.
- Keep write policy in the installer and safe filesystem layer. Writes must stay below `/userdata`; reject unsafe paths and links. Use the order download, verify, stage, journal, mutate, then installed state (`AGENTS.md`, `internal/installer/commit.go`, `internal/safefs/guard.go`).
- Keep input bindings and footer labels under the same selected layout. Do not swap SDL events a second time (`CONTEXT.md`, `internal/input/mapping.go`, `internal/sdlui/run.go`).

---

*Convention analysis: 2026-10-25*
