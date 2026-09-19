# Coding Conventions

**Analysis Date:** 2026-09-19

## Naming Patterns

**Files:**
- Go files are named for the concept they own: `guard.go`, `download.go`, `refresh.go`, `tabs.go`, `toast.go`, `repeat.go`, `verdict.go`
- Tests are named for the invariant, not the function: `TestChecksumFailureWritesNoPackageFiles`, `TestRuntimeCodeDoesNotBypassSemanticControllerActions`
- One audit file exists for a source-level rule: `internal/input/physical_button_audit_test.go`
- Device docs and packaging folders use Knulli board ids: `trimui-smart-pro`, `magicx-zero-28`

**Functions:**
- Exported verbs describe the pipeline step: `Load`, `Validate`, `Build`, `Verify`, `Extract`, `Resolve`, `Check`, `Apply`, `Sign`/`SignFile`
- Installer stages are named for what they do: `checkPreconditions`, `acquireManager`, `stage`, `commit`, `rollback`
- Unexported helpers stay short and local: `assess`, `plan`, `wrap`, `blend`, `shorten`, `owns`

**Variables:**
- Short local names: `pkg`, `m`, `tx`, `err`, `index`
- Domain terms come from `CONTEXT.md` and are used verbatim in code: `Destination`, `Preserved`, `Managed`, `Originals`, `PreExisting`, `Outcome`, `Verdict`

**Types:**
- Structs for records: `manifest.Package`, `installer.Manager`, `installer.Installed`, `safefs.Transaction`, `appstore.Item`
- String-typed enums for closed sets: `manifest` review status, `appstore.Action`, `appstore.State`, `installer.Op`, `input.Action`, `sdlui.Key`, `sdlui.EventKind`
- Reason/evidence results are explicit structs, not formatted strings: `appstore.Reason`, `platform.Source`, `installer.HealthIssue`

## Code Style

**Formatting:**
- `gofmt` is the only formatter; CI fails when `gofmt -l .` prints anything, and `make fmt` rewrites offenders in place
- Tabs for indentation, no alignment ceremony
- Import groups: standard library, then `github.com/jellydn/knulli-app-store/internal/...`, then the rare third-party module (`golang.org/x/image`)

**Linting:**
- `go vet ./...` and `go vet -tags sdl ./...`
- `staticcheck` v0.8.1 in CI, run both with and without the `sdl` tag
- Both tag variants must stay clean, so a GUI-only change cannot break the CGO-free build

## Import Organization

**Order:**
1. Standard library
2. `github.com/jellydn/knulli-app-store/internal/...`
3. `golang.org/x/image/...` in SDL packages

**Path Aliases:**
- The module aliases its own packages wherever a name would collide: `storearchive`, `storeinput`, `storeui` (see `internal/sdlui/run.go`, `internal/installer/installer.go`, `internal/sdlui/draw.go`)
- Standard library collisions are aliased too: `xdraw "golang.org/x/image/draw"` and `imagedraw "image/draw"` in `internal/sdlui/draw.go`
- No aliases are used just for brevity

## Error Handling

**Patterns:**
- Return `error`; never panic in product code. `t.Fatal` panics are limited to tests.
- Wrap at the boundary that adds context: `fmt.Errorf("open diagnostics log: %w", err)` (47 `%w` wraps in product code)
- Error text is a lowercase clause with no trailing punctuation, except when it begins with a proper noun: `"GitHub release metadata returned HTTP %d"`, `"SHA-256 mismatch: got %s"`, `"EmulationStation did not accept reload: %w"`
- The CLI prints `fmt.Fprintln(os.Stderr, "error:", err)` and exits 1
- Structured evidence, not prose, for user-facing failures: compatibility errors carry raw value, normalized value, source, and the full detected matrix (`internal/platform/platform.go`); health issues carry path, check, expected, actual (`internal/installer/status.go`)
- Typed errors exist where the UI must branch on the cause: `installer.AdoptionConflictError`, `input.corruptFileError`
- Validation collects all problems and reports them together (`manifest.Validate` builds a `problems []string`)

## Logging

**Framework:** the standard `log` package behind `internal/diagnostics.Log`

**Patterns:**
- Event name first, then alternating key/value pairs: `m.event("operation_start", "package", pkg.ID, "action", operation)`
- Event names are `snake_case` verbs describing the transition: `startup`, `platform_detected`, `compatibility_rejected`, `adoption_inventory`, `action_selected`, `operation_error`
- `Log.Event` redacts before writing: URL credentials, query strings, fragments, common secret fields, and terminal control sequences (`sensitiveValue`, `webURL`, `terminalControl` in `internal/diagnostics/log.go`)
- Never log a package's file contents, credentials, or ROM data. Exports are user-triggered and redacted.
- The active log is capped at 512 KiB with exactly one rotated copy (`boundedWriter`)

## Comments

**When to Comment:**
- `CONTRIBUTING.md` is explicit: comments must explain a design reason the code cannot make clear by itself
- That rule is followed closely in the newer packages, where exported types carry multi-line rationale: `internal/ui/tabs.go` explains why `TabAll` leads and why `TabOrder` is a slice rather than a switch; `internal/appstore/verdict.go` explains that `assess` only combines policy owned elsewhere; `internal/sdlui/run.go` documents each `Options` field
- The cgo preamble in `internal/sdlui/run.go` documents which SDL functions the wrappers exist for
- Build-tag files carry `//go:build sdl` and nothing else on that line
- Total comment volume is modest (about 647 lines) and concentrated where a decision is non-obvious

**Doc Comments:**
- Go doc comments on exported types and non-obvious functions; no JSDoc/TSDoc anywhere
- Sparse on obvious exported helpers (`Load`, `Build`), present on policy decisions

## Function Design

**Size:**
- CLI `run` and `runApply` stay as command switches plus flag setup
- `installer.Manager.apply` is the orchestrator and delegates to three named stages; helper files own download, XML, state, status, and refresh
- Long test files are accepted: `internal/installer/installer_test.go` is 1,421 lines and still one package suite

**Parameters:**
- `context.Context` is the first parameter on anything that downloads, refreshes, or can block (`Apply`, `Execute`, `Items`, `Check`, `Uninstall`)
- `Manager` is a value struct with injectable seams: `Client`, `RefreshClient`, `RefreshURL`, `Now`, `AvailableBytes`
- Options are functional: `platform.Resolve(root, platform.WithFirmware(...), ...)`

**Return Values:**
- `(T, error)` or `error`
- A committed operation returns `installer.OperationOutcome` describing refresh acceptance rather than an error
- Tests use `t.Fatal` on setup failure and `t.Fatalf` with the actual value

## Module Design

**Exports:**
- Small public surface per package; `internal/` prevents any external importer
- UI depends on `appstore.Backend`, never on installer internals
- `internal/gamelist` depends on `safefs` and `manifest` but is called only by `installer`
- `internal/input` and `internal/ui` never import `internal/sdlui`, so the model and mapping logic stay testable without SDL

**Barrel Files:**
- None. No `doc.go`, no re-export shims, no `utils` package

**Policy placement:**
- Each rule lives in the package that owns the data: installability and manifest rules in `manifest`, compatibility in `platform`, health and ownership in `installer`, combination only in `appstore/verdict.go`. A new state or action is added once, in `assess`.

**Prose:**
- User and contributor docs follow What / Why / How with short ASD-STE100-style sentences (`README.md`, `docs/`, `CONTRIBUTING.md`)
- `CONTEXT.md` is the shared glossary; a new concept gets a term there, and the term is then used in code, docs, and review

---

*Convention analysis: 2026-09-19*
