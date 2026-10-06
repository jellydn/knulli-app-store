# Coding Conventions

**Analysis Date:** 2026-09-21

## Naming Patterns

**Files:**
- One concern per file, named for that concern: `guard.go`, `transaction.go`, `commit.go`, `stage.go`, `verdict.go`, `toast.go`, `footer.go`.
- The package's primary type/orchestrator typically lives in a file named after the package (`installer.go`, `catalog.go`, `platform.go`, `manifest.go`, `archive.go`), with siblings split by sub-concern.
- Tests are `<file>_test.go` in the *same* package (internal tests), so unexported behavior is testable. Shared helpers go in a `test_helpers_test.go`.

**Functions:**
- Exported: full words, no abbreviations. `NewGuard`, `Resolve`, `PendingForPath`, `PreExisting`, `RefreshGameList`'s public `outcome`, `AssessResolutions`, `DisplayHeader`.
- Design-reason suffixes exist for a purpose: a method returning a copy of a value type is named `With<Thing>` (`WithPlatform`, `WithStatusCache`, `WithCandidates`).
- Method receivers are short and consistent per type: `m` for `Manager`, `s` for `Service`, `g` for `Guard`, `tx` for `Transaction`, `tab`/`tabs` for tabs, `store` for `Store`.
- Predicates read as assertions: `allows`, `bindable`, `optional`, `coveredBy`, `under`, `containsAction`, `comparableVersions`, `retryable`.

**Variables:**
- SHOUTING_CASE is reserved for exported constants (`SchemaV1`, `OpInstall`, `StateExternal`, `VerbNavigate`, `managerPath` is exported-as-lowercase since it is package-private).
- Schema strings are declared as a constant next to the type they version, e.g. `const lifecycleSchemaV1 = "org.knulli.app-store/lifecycle-state/v1"`.
- Loop indices over byte/slice arithmetic are `index`/`position`, not `i`.
- Boolean locals read as predicates: `managed`, `preserved`, `found`, `comparable`, `skip`, `due`.

**Types:**
- Exported domain types are singular nouns: `Package`, `Release`, `Compatibility`, `Install`, `Menu`, `Verdict`, `Item`, `Status`, `Guard`, `Transaction`, `Info`, `Source`, `Mapping`, `Session`, `Hint`.
- Enumerations are a named string or int type plus constants: `type Op string`, `type Action string`, `type State string`, `type ReasonKind string`, `type Focus int`, `type Mode string`, `type StepKind string`.
- Structured error types are `<Thing>Error` with a pointer receiver and an `Error()` returning an actionable message: `AdoptionConflictError`, `corruptFileError`.
- Interfaces are minimal and named for the capability: `appstore.Backend` (4 methods).

## Code Style

**Formatting:**
- `gofmt` is the only formatter; there is no formatter config file. CI fails on `test -z "$(gofmt -l .)"`, and `prek.toml` runs `gofmt -w` on staged Go files.
- Tabs for indentation; struct fields aligned by gofmt.
- `.gitignore` and `prek.toml` enforce `mixed-line-ending --fix=lf` and `end-of-file-fixer`.

**Linting:**
- `go vet ./...` **and** `go vet -tags sdl ./...` — both tag sets, always. `make vet` runs exactly that pair.
- `staticcheck@v0.8.1` via `go run` (never installed globally), on both tag sets, in `make check`'s CI counterpart and on pre-push.
- Unused layout constants in `internal/sdlui/layout.go` are suppressed with a reason comment rather than deleted:
  ```go
  //lint:ignore U1000 used by the sdl-tagged draw code in this package
  canvasWidth = 640
  ```
  This is the only suppression idiom in the repo (13 occurrences, all in `layout.go`).

## Import Organization

**Order:**
1. Standard library (alphabetical within the group)
2. Third-party (`golang.org/x/image/draw`)
3. This module (`github.com/jellydn/knulli-app-store/internal/...`)

gofmt does not reorder groups, so the separation is maintained by hand and matches the rest of the codebase.

**Aliases:**
- Aliases are used only to disambiguate a package name from a local concept:
  - `storearchive "github.com/jellydn/knulli-app-store/internal/archive"` in `internal/installer/commit.go` and `stage.go`
  - `storeinput`, `storeui`, `xdraw` in `internal/sdlui/run.go` and `walk.go`
- No path-alias mechanism exists beyond the module path.

## Error Handling

**Patterns:**
- `fmt.Errorf` with `%w` for wrapping so `errors.As`/`errors.Is` keep working; `errors.As` is used for typed failures (`appstore.recoverableAdoptionFailure`).
- Sentinel-free: failures are descriptive strings, not exported error variables.
- Validation collects *all* problems, sorts them, and joins with `"; "`:
  ```go
  if len(problems) > 0 {
      sort.Strings(problems)
      return errors.New(strings.Join(problems, "; "))
  }
  ```
  A manifest review must not hide a second problem behind the first.
- Fail closed before mutating: `checkPreconditions` runs `Validate` + `Installable` + `platform.Check` + `validateDownloadURL` before any filesystem work.
- Report provenance in failures: `platform.compatibilityError` embeds the detected raw value, normalized value, source, and the full matrix; `HealthIssue` carries `Path`, `Check`, `Expected`, `Actual`.
- Never lose a rollback failure: `commit.go`'s `rollback` folds it into the original error (`"%w; rollback also failed: %v"`).
- Distinguish "operation failed" from "operation committed, follow-up refused": a failed loopback reload sets `RestartRequired` and returns a nil error.
- Deferred cleanup with a named result: `apply`, `commit`, `uninstall` use `defer` + `result error` so logging and rollback see the final error.

## Logging

**Framework:** `internal/diagnostics.Log` — no third-party logger.

**Patterns:**
- Call as `log.Event(name, "key", value, ...)`, producing `event=name key="value"` lines.
- `Manager.event(name, fields...)` no-ops when `Diagnostics` is nil, so tests run without I/O. Every production module takes an optional `*diagnostics.Log`.
- Event names are lowercase snake_case and name the state transition: `operation_start`, `compatibility_allowed`, `compatibility_rejected`, `transaction_begin`, `backup_complete`, `operation_complete`, `rollback_complete`, `package_health_checked`, `gamelist_refresh_accepted`, `resolution_candidate`, `controller_source`.
- Log *decisions*, not just steps: `compatibility_decision` records `allowed` and the rendered message; `lifecycle_state` records the retry target beside the failure.
- Never log a raw URL, token, or path containing one — `Redact` runs at write time and again on export.

## Comments

**When to Comment:**
This is the strongest convention in the repo. Comments explain *why*, and specifically record a decision the code cannot show by itself. They are long, prose, and reference the failure they prevent.

Examples that define the house style:
```go
// AskPaging opens the paging question. It is one screen for every setup path,
// because the answer belongs to the mapping rather than to the path that builds
// it.
```
```go
// modeIssue reports a permission regression. The destination filesystem owns
// the mode bits and a Knulli SD card can report 0777 for a file the installer
// requested as 0755 or 0644, so wider bits are not an issue.
```
```go
// What this changes is when the check runs, not how.
```

Guidelines observed:
- A comment on a non-obvious field, branch, or constant explains the alternative that was rejected or the bug it prevents (`internal/input/mapping.go`'s `Action(button)` comment explains the zero-value map-read trap).
- File-level package docs exist where the file is the orchestrator: `internal/installer/installer.go` opens with a map of the whole package and why the files are split the way they are; `internal/version/version.go` explains why two comparison functions exist.
- `AGENTS.md` restates the rule: "Comments explain design reasons the code cannot show, not what the code does."
- No `TODO`/`FIXME`/`HACK` comments exist anywhere in the tree.
- Doc comments on exported identifiers are the norm but not universal; they are most consistent on exported types and functions in `manifest`, `platform`, `input`, `ui`, and `safefs`.

**Parameterized test naming carries meaning:**
- Table tests use `name` fields that read as sentences: `"skipped pair saves no paging binding"`, `"repair requires the installed release"`.

## Function Design

**Size:** Small and single-purpose. The largest functions are the ones that must be exhaustive by nature — `manifest.validateInstallable`, `platform.Check`, `run.Run`, `draw.draw`. Helpers are extracted aggressively (e.g. `installer.go` keeps only `Apply`, `apply`, `checkPreconditions`, `event`, `root` and delegates the rest).

**Parameters:**
- Value receivers for immutable-ish config carriers (`func (m Manager)`) so `WithX` returns a copy and no hidden mutation is possible.
- Pointer receivers only where mutation is the point: `*Transaction`, `*Session`, `*Model`, `*Tabs`, `*Status`, error types.
- Functional options for detection: `platform.Resolve(root, platform.WithFirmware(...), ...)` with `Option func(*request)`.
- Injectable seams are struct fields, not globals: `Manager.Client`, `Manager.RefreshClient`, `Manager.RefreshURL`, `Manager.Now`, `Manager.AvailableBytes`, `Checker.Client`, `Checker.BaseURL`, `Checker.Sleep`, `Checker.MaxRetries`.

**Return Values:**
- `(T, error)`; a lookup that can legitimately miss returns `(T, bool)` instead (`tabs.Remembered`, `mapping.Action`, `containsArchiveFile`).
- Outcome structs are filled through a pointer parameter (`commit(ctx, op, pkg, staged, &outcome)`) so the returned error stays the sole failure signal.
- `Copy()`/`Clone()` methods exist on slice-backed values that must not be shared (`Info.Clone`, `Mapping.Clone`, `Status.clone`, `ResolutionCandidates`).

## Module Design

**Exports:**
- Exports are limited to what a consumer needs. Many helpers stay unexported even in heavily used packages (`manifest.safeRelative`, `installer.installFiles`, `input.actionName`).
- No package exposes a mutable global. `catalog.embeddedPublicKeyHex` is an unexported `string` set only via `-ldflags`.
- `internal/` is used for every package except `main`, so nothing is importable outside the module by construction.

**Barrel Files:**
- None. There are no `doc.go` or re-export files; each file holds real code.

**Dependency direction (enforced by review, see `CONTRIBUTING.md`):**
- `internal/installer` must stay independent of presentation code. The GUI depends on `appstore.Backend`, and GUI tests use an in-memory fake.
- `internal/ui` and `internal/input` depend on stdlib only (plus the `appstore` interface for `ui`), which is why they compile and test without CGO or SDL.

---

*Convention analysis: 2026-09-21*
