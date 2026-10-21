# Coding Conventions

**Analysis Date:** 2026-10-21

## Naming Patterns

**Files:** One concern per lowercase filename; orchestration often uses the package name. Examples: `internal/installer/installer.go`, `internal/installer/stage.go`, `internal/installer/selfupdate.go`. Tests are co-located `*_test.go`.

**Functions:** Exported Go identifiers use MixedCaps (`NewGuard`, `RecoverTransactions`); private helpers use mixedCaps (`checkPreconditions`, `acquireManager`). `WithPlatform` and `WithStatusCache` return manager copies (`internal/installer/installer.go`).

**Variables:** Constants also use Go MixedCaps, not SHOUTING_CASE: `SchemaV1`, `MaxPackageBytes`, `OpInstall`. Lowercase schema constants remain private; build-stamped `StoreVersion`/`StoreTarget` are package variables (`internal/manifest/manifest.go`, `internal/installer/lifecycle.go`, `internal/appstore/selfupdate.go`).

**Types:** Domain nouns (`Package`, `Verdict`, `StoreUpdate`, `Guard`) and typed enumerations (`Action`, `State`, `Focus`, `Mode`) make policy explicit (`internal/manifest/manifest.go`, `internal/appstore/verdict.go`, `internal/ui/model.go`, `internal/input/session.go`).

## Code Style

**Formatting:** `gofmt`; tabs and aligned structs. `make fmt` rewrites Go files, while CI rejects a nonempty `gofmt -l .` result. `prek.toml` checks whitespace, LF endings, JSON/YAML, and conflict markers (`Makefile`, `.github/workflows/check.yml`).

**Linting:**
- `make vet` checks both default and `sdl` tag sets; `make check` adds formatting and race tests (`Makefile`). SDL headers are therefore needed for `make check`.
- Pre-push uses `go run honnef.co/go/tools/cmd/staticcheck@v0.8.1`; CI installs that version, then checks both tag sets (`prek.toml`, `.github/workflows/check.yml`).
- Layout constants used only by tagged rendering have reasoned `//lint:ignore U1000` comments (`internal/sdlui/layout.go`).

## Import Organization

**Order:** Standard library, external dependencies, then module-local imports, separated into groups where needed (`internal/sdlui/run.go`). `gofmt` sorts within groups; no import-alias configuration is present in `go.mod`.

**Path Aliases:** Local aliases clarify names: `storearchive` in `internal/installer/stage.go`, `storeinput` and `storeui` in `internal/sdlui/events.go`. Internal imports use the full module path.

## Error Handling

**Patterns:**
- Return `(T, error)` for failures and `(T, bool)` for legitimate misses; wrap with `%w` for callers that use `errors.Is`/`errors.As` (`internal/installer/stage.go`, `internal/input/mapping.go`, `internal/appstore/service.go`).
- Manifest validation accumulates, sorts, and joins problems; validation runs before package filesystem work (`internal/manifest/manifest.go`, `internal/installer/installer.go`).
- Defer rollback against the named result, retaining rollback failures rather than hiding the original error (`internal/installer/commit.go`, `internal/installer/selfupdate.go`).
- Separate mutation failure from post-commit outcome: reload refusal sets `RestartRequired`, not an operation error (`internal/installer/refresh.go`).
- Fail closed on firmware/architecture/runtime requirements, but not on unknown device or display scope (`internal/platform/platform.go`, `CONTEXT.md`).

## Logging

**Framework:** `diagnostics.Log`, not a third-party logger (`internal/diagnostics/log.go`).

**Patterns:** `Event(name, "key", value, ...)`, lowercase event names, sanitized field names, and redaction at write/export. Nil log receivers are supported. Log decisions such as `compatibility_decision`, not only process steps (`internal/appstore/service.go`).

## Comments

**When to Comment:** Explain reasons and rejected alternatives, not the statement below the comment (`AGENTS.md`). Examples explain why patches precede file mappings (`internal/installer/stage.go`), why copying replaces the inode (`internal/safefs/files.go`), and why old mapping files remain untouched (`internal/input/session.go`).

**JSDoc/TSDoc:** Not applicable; this is Go. Exported declarations commonly carry Go doc comments, including `Manager.Apply`, `Transaction`, and `Extract` (`internal/installer/installer.go`, `internal/safefs/transaction.go`, `internal/archive/archive.go`).

## Function Design

**Size:** Coherent helpers within the owning package, with lifecycle orchestration separate from staging and commit. Avoid pushing installer side effects into presentation (`internal/installer/installer.go`, `AGENTS.md`).

**Parameters:** Contexts carry cancellation; manager fields inject HTTP clients, clocks, and space checks. Platform detection uses functional options (`internal/installer/installer.go`, `internal/platform/platform.go`).

**Return Values:** Explicit outcome structs distinguish accepted reload from restart. Value-copy builders bind manager configuration; pointer receivers mutate transactions, sessions, and UI state (`internal/installer/refresh.go`, `internal/safefs/transaction.go`, `internal/ui/model.go`).

## Module Design

**Exports:** All application packages are under `internal/`. `appstore.Backend` is the four-method presentation seam; UI tests use an in-memory implementation. Installer code does not depend on presentation packages (`internal/appstore/service.go`, `internal/ui/model_test.go`, `AGENTS.md`).

**Barrel Files:** No re-export layer; each source file owns behavior. Shared constants belong to the contract (`MaxPackageBytes`, `ManagerStatePath` in `internal/manifest/manifest.go`).

**Policy source of truth:** Bindings and labels use the same selected controller layout, without swapping SDL events again (`internal/input/mapping.go`, `internal/sdlui/events.go`). Device-test evidence controls the badge, not permission to install (`internal/manifest/manifest.go`, `internal/appstore/verdict.go`).

---

*Convention analysis: 2026-10-21*
