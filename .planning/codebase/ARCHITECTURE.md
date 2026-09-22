# Architecture

**Analysis Date:** 2026-09-21

## Pattern Overview

**Overall:** Layered domain modules behind a single policy-owning service, with two thin adapters (CLI, SDL GUI) and a pure-Go UI state machine.

**Key Characteristics:**
- One component owns side effects: `internal/installer`. Front ends never extract archives, edit XML, or shell out.
- Dependencies point inward and downward. `manifest` is the contract, `main` is the outer edge, nothing in between imports presentation.
- Data flows through typed, persisted records (installed state, lifecycle record, transaction journal) rather than through in-memory handoffs.
- The GUI is behind the `sdl` build tag, so the CLI stays static and CGO-free by construction.
- Determinism is a stated invariant: the same inputs must produce byte-identical catalogue output.

## Layers

**Contract / policy (`internal/manifest`):**
- Purpose: The v1 JSON contract plus every semantic rule JSON Schema cannot express.
- Location: `internal/manifest/manifest.go`, `internal/manifest/load.go`
- Contains: `Package`, `Release`, `Compatibility`, `Install`, `Menu`, `BinaryPatch`, `Evidence`; `Validate()` and `validateInstallable()`; path helpers `safeAbsolute`, `safeRelative`, `under`, `coveredBy`
- Depends on: stdlib only
- Used by: everything else

**Deterministic index (`internal/catalog`):**
- Purpose: Build, validate, write, and sign the catalogue index.
- Location: `internal/catalog/catalog.go`, `internal/catalog/sign.go`
- Contains: `Build`, `Load`, `Write` (atomic temp+rename), `Entry`, `Index`, ed25519 `Sign`/`Verify`/`SignFile`/`ParsePublicKey`
- Depends on: `manifest`
- Used by: `internal/appstore`, all three `cmd/` binaries, `internal/updatecheck` (via `cmd/check-updates`)

**Detection and compatibility (`internal/platform`):**
- Purpose: Read Knulli-owned files into a normalized `Info` with recorded provenance, and gate packages against it.
- Location: `internal/platform/platform.go`
- Contains: `Resolve` with functional options, `Info`/`Source`/`evidence`, resolution candidate assessment, ABI/glibc/dependency detection, `Check`
- Depends on: `manifest`, `internal/version`
- Used by: `internal/appstore`, `internal/installer`, both `cmd/knulli-app*`

**Verification (`internal/version`):**
- Purpose: Every version comparison in the project, lenient and strict variants separated on purpose.
- Location: `internal/version/version.go`
- Contains: `Normalize`, `Compare` (always returns an order), `Parse`, `CompareNumeric` (refuses to guess)
- Used by: `internal/platform`, `internal/updatecheck`

**Filesystem policy (`internal/safefs`):**
- Purpose: Allowed-path resolution, symlink rejection, atomic file writes, free-space checks, rollback snapshots.
- Location: `internal/safefs/guard.go`, `internal/safefs/transaction.go`, `internal/safefs/files.go`
- Contains: `Guard` (`NewGuard`, `Resolve`, `Virtual`), `rejectSymlinkParents`, `Transaction` (`Begin`, `Write`, `Copy`, `Chmod`, `Remove`, `Commit`, `Rollback`), `Recover`, `PendingForPath`, `AtomicWrite`, `Copy`, `SHA256`
- Depends on: stdlib only
- Used by: `internal/installer`, `internal/gamelist`

**Bounded extraction (`internal/archive`):**
- Purpose: Extract ZIP and tar.gz into staging with every unsafe shape rejected.
- Location: `internal/archive/archive.go`
- Contains: `Extract`, `extractZIP`, `extractTarGZ`, `safeName` (backslash normalization, absolute/traversal rejection, `strip_components`), `interruptible` (in-transfer cancellation), per-entry and total size caps
- Rejects: symlinks, non-regular entries, devices, duplicate paths, entries expanding beyond the installed-size budget
- Cancellation: `Extract` takes a `context.Context` and checks it at the top, before each entry, and inside each entry's transfer, so an interrupted extraction returns `context.Canceled` instead of finishing the archive
- Used by: `internal/installer` only

**Menu ownership (`internal/gamelist`):**
- Purpose: Derive and apply EmulationStation `gamelist.xml` edits without ever touching an entry it did not create.
- Location: `internal/gamelist/menu.go`, `internal/gamelist/xml.go`
- Contains: `Derive(oldOwned, oldMenu, menu) Plan`, `Apply(tx, guard, plan)`, `Step`/`StepKind` (add/replace/remove), XML read/write
- Depends on: `manifest`, `safefs`
- Used by: `internal/installer` (`commit.go`, `uninstall.go`)

**Lifecycle (`internal/installer`):**
- Purpose: The single owner of every side effect a package operation performs.
- Location: `internal/installer/` split by concern — `installer.go` (orchestrator + `Manager`), `stage.go` (acquire/lock/recover/download/extract), `commit.go` (transaction, file application, adoption), `uninstall.go`, `adoption.go` (destination inventory), `recovery.go` (force-reinstall backups), `state.go` (installed record), `status.go` (health + pre-existence), `cache.go`, `lifecycle.go`, `refresh.go`, `download.go`, `files.go`
- Depends on: `manifest`, `platform`, `archive`, `safefs`, `gamelist`, `diagnostics`
- Used by: `internal/appstore` only

**Application service (`internal/appstore`):**
- Purpose: Join review status, compatibility, install state, and health into one typed catalogue verdict, and expose the operations the UI may run.
- Location: `internal/appstore/service.go`, `internal/appstore/verdict.go`
- Contains: `Backend` interface, `Service` (`Items`, `Execute`, `ExportDiagnostics`, `SetPlatform`), `Item`, `Action`, `assess`, `Verdict`, `State`, `Reason`
- Depends on: `catalog`, `installer`, `manifest`, `platform`
- Used by: `internal/ui`, `internal/sdlui`, the GUI adapter

**Catalogue state machine (`internal/ui`):**
- Purpose: Pure-Go navigation, tabs, toasts, and operation progress; no CGO, no SDL, no filesystem.
- Location: `internal/ui/model.go`, `internal/ui/tabs.go`, `internal/ui/toast.go`
- Contains: `Model` (`Load`, `Move`, `Page`, `Horizontal`, `Select`, `Back`, `Poll`, `Window`), `Focus`, `Tab`/`Tabs`, `Toasts`
- Depends on: `appstore` (the `Backend` interface)
- Used by: `internal/sdlui`

**Input semantics (`internal/input`):**
- Purpose: Semantic actions, mappings, calibration, assignment review, controller identity, atomic persistence, footer hints.
- Location: `internal/input/mapping.go`, `session.go`, `calibration.go`, `store.go`, `keyboard.go`, `footer.go`
- Contains: `Action` (nine bindables), `Mapping`, `Plan`/`Required`/`Optional`, `Session` mode machine, `Store`, `Identity`, `Hint`/`Footer`
- Depends on: stdlib only
- Used by: `internal/sdlui`

**SDL adapter (`internal/sdlui`):**
- Purpose: Translate SDL events to semantic actions, render the fixed 640×360 canvas, drive scripted walkthroughs.
- Location: `internal/sdlui/run.go`, `events.go`, `draw.go`, `layout.go`, `hints.go`, `repeat.go`, `walk.go`, `input_mode.go`
- Depends on: `appstore`, `input`, `ui`, `platform`, `diagnostics`, SDL2 via cgo, `golang.org/x/image/draw`
- Used by: `cmd/knulli-app-ui` only

## Data Flow

**Install (the canonical path):**
1. `cmd/knulli-app` or GUI → `appstore.Service.Execute` → `installer.Manager.Apply(ctx, OpInstall, pkg)`
2. **Preconditions** (`installer.go` → `checkPreconditions`): `pkg.Validate()`, `Installable()`, `platform.Check`, `validateDownloadURL`
3. **Acquire** (`stage.go` → `acquireManager`): new `safefs.Guard` rooted at `Manager.Root`, `os.MkdirAll` manager state, `syscall.Flock` exclusive non-blocking lock, `safefs.Recover` rolls back any open journal
4. **Reconcile**: `loadState`, reject install-if-installed / repair-of-different-release, inventory the destination, free-space check (`release + 3×installed + 2×existing`)
5. **Download and verify** (`download.go`): stream to a temp file while hashing, enforce `Content-Length`, byte count, and SHA-256; redirects must stay HTTPS
6. **Extract** (`archive.Extract`) into `work/staging` with the installed-size cap, honouring the operation context so a cancelled run stops mid-archive; then `applyExecutableModes` and `applyBinaryPatches`; the launcher must exist
7. **Commit** (`commit.go`): `safefs.Begin` creates a journal directory; each mutation calls `tx.prepare`, which snapshots the previous file and persists the open journal *before* the destination changes
8. Files are copied (preserved files are recorded but not overwritten), stale owned files are removed or restored from originals, `gamelist.Apply` writes the menu effect, then `tx.Write(statePath(id), ...)`
9. `tx.Commit()` marks the journal committed and removes the directory. **Installed state is the last write.**
10. `m.outcome(ctx, gameListChanged)` performs the loopback reload request; a failure sets `RestartRequired` while the operation stays committed.
11. Any failure between 7 and 9 runs `m.rollback`, restoring snapshots in reverse order and folding a rollback failure into the returned error.

**Catalogue read (every GUI frame / list load):**
1. `appstore.Service.Items` iterates index entries → `s.item(ctx, entry)`
2. `manager.Status(id)` loads the installed state, checks the `StatusCache` by state digest, and otherwise hashes every immutable managed file and compares owner permission bits
3. `manager.PreExisting(pkg)` walks the destination and stops at the first entry (a file, or an unreadable entry) — an empty skeleton counts as absent
4. `manager.LifecycleState(id)` and `RecoveryStatus(pkg)` supply retry context
5. `assess(pkg, platform, status, preExisting)` produces a typed `Verdict` (state, reasons, actions); `actions(item)` folds in force-reinstall and retry ordering
6. Items are sorted by name and returned

**Controller setup (first run per device/controller identity):**
1. `sdlui.Run` opens an SDL GameController or falls back to a keyboard session (`input.NewDesktopSession`)
2. `input.Session.connect` loads a saved mapping for the identity or opens setup
3. Setup asks the paging question (`PagingItems`) before any mapping is built, then saves / previews / calibrates
4. Calibration walks `input.Plan(paging)`, reviews each assignment (`ReviewItems`), optionally previews every bound button, and saves atomically to `controller-mappings.json`
5. Quitting is always the Select+Y chord (`chordAnchorButton = 4`), never a single mapped button — `Exit` is deliberately not in `input.Actions`

**State Management:**
- No global state, no singletons. `Manager` is a value with copies returned by `WithPlatform`/`WithStatusCache`.
- UI state lives in `ui.Model` and `input.Session`; the SDL loop owns both and polls `model.Poll()` each frame.
- Operations run on a goroutine and communicate back through a buffered `chan operationEvent` (capacity 8) drained by `Model.Poll`.
- Persisted state is always written atomically (`safefs.AtomicWrite`: temp file, `Sync`, `Rename`).

## Key Abstractions

**`appstore.Backend` interface:**
- Purpose: The seam that keeps the GUI free of installer types, and lets GUI tests run against an in-memory fake.
- Definition: `internal/appstore/service.go` — `Items`, `Execute`, `ExportDiagnostics`, `SetPlatform`
- Consumers: `internal/ui.Model`, `internal/sdlui.Run`, `cmd/knulli-app-ui/main.go`, `internal/ui/model_test.go` fakes

**`safefs.Guard`:**
- Purpose: Convert a virtual `/userdata/...` path into a host path under a root, while enforcing allowed prefixes and rejecting symlinked components.
- Examples: `internal/safefs/guard.go`, constructed in `internal/installer/stage.go`, `internal/gamelist`, `internal/installer/status.go`
- Pattern: Capability object — nothing can write without one, and every guard carries its own allowed-path list

**`safefs.Transaction`:**
- Purpose: A reversible group of filesystem writes with a durable journal.
- Examples: `internal/safefs/transaction.go`; used by `internal/installer/commit.go` and `uninstall.go`
- Pattern: Write-ahead journal + reverse-order snapshot restore. `prepare` persists `status=open` before each mutation; `Commit` writes `status=committed` then deletes; `Recover` rolls back anything left open at the next locked start.

**`manifest.Package.Validate()`:**
- Purpose: The one gate that decides whether a manifest is a candidate, experimental/installable/verified, and safe to act on.
- Examples: `internal/manifest/manifest.go`; called by `Load`/`Decode`, `catalog.Build`/`Load`, and `installer.checkPreconditions`
- Pattern: Accumulate-all-problems then sort and join, so a review sees every issue at once

**`input.Mapping` + `input.Session`:**
- Purpose: Name actions independently of physical buttons, and store bindings per device/controller identity.
- Examples: `internal/input/mapping.go`, `internal/input/session.go`
- Pattern: A semantic-action map validated for required coverage and button conflicts, plus a mode machine (`Normal`, `Setup`, `Paging`, `Blocked`, `Settings`, `Calibrating`, `Review`, `Preview`)

**`installer.StatusCache`:**
- Purpose: Make a whole-catalogue read affordable without weakening content verification.
- Examples: `internal/installer/cache.go`
- Pattern: Cache key is the SHA-256 of the canonical installed-state encoding, so a changed record invalidates itself even if a caller forgets.

## Entry Points

**`cmd/knulli-app`:**
- Location: `cmd/knulli-app/main.go`
- Triggers: manual/scripted device use; the packaged `Knulli App Store.sh` calls it (`catalogue` path); CI determinism check
- Responsibilities: `validate`, `catalogue` (build/write/sign/generate key), `install`/`adopt`/`update`/`repair`, `uninstall`. Platform flags (`-root`, `-firmware`, `-firmware-version`, `-arch`, `-device`, `-resolution`) override detection. Prints `game list refresh accepted` or `restart required to update game list`.

**`cmd/knulli-app-ui`:**
- Location: `cmd/knulli-app-ui/main.go` (whole file is `//go:build sdl`)
- Triggers: the packaged launcher script on device; `make gui` and `make walkthrough` on desktop
- Responsibilities: resolve root/platform, open the diagnostics log, open `appstore.Service` bound to `installer.Manager`, run the SDL loop. Flags: `-catalog`, `-root`, platform overrides, `-windowed`, `-screenshot`, `-input`, `-keys`, `-shot-dir`, `-walk-timeout`.

**`cmd/check-updates`:**
- Location: `cmd/check-updates/main.go`
- Triggers: weekly `catalogue-updates.yml`, or manual `go run ./cmd/check-updates -output build/catalogue-update-report`
- Responsibilities: build the index, compare each package to upstream GitHub releases, write `.json` + `.md`. Never edits a manifest.

**`scripts/device-build.sh`:**
- Location: `scripts/device-build.sh`
- Triggers: `.github/workflows/check.yml` (`device-artifact`) and `release.yml` (`tag-build`)
- Responsibilities: install the cross toolchain if missing, generate and use an ed25519 catalogue key, cross-compile both binaries with the public key in ldflags, assert the GUI needs exactly two shared libraries and no glibc symbol newer than 2.34, then hand off to `scripts/package-device.sh`.

## Error Handling

**Strategy:** Fail closed, report provenance, and keep committed work committed.

**Patterns:**
- Validation accumulates all problems and returns them sorted (`manifest.Validate`), never the first one only.
- Compatibility failures carry raw value, normalized value, source, and the full detected matrix (`platform.compatibilityError`, `platform.Summary`).
- Security refusals are plain errors that abort before any mutation: unsafe archive path, symlinked write path, non-regular destination entry, non-HTTPS redirect, SHA-256 mismatch.
- Cancellation is reported as the context's own error, unwrapped through every layer: `archive.Extract` returns `ctx.Err()` and `installer.stage` wraps it with `%w`, so `errors.Is(err, context.Canceled)` holds at the operator's boundary. A cancelled operation needs no rollback because it stops before the commit phase.
- Transactional rollback: `commit.go`'s `rollback` defer restores snapshots in reverse and appends a rollback failure to the original error rather than masking it.
- Outcome failures are not operation failures: a rejected loopback reload is reported as `RestartRequired` with the operation still committed (`installer/refresh.go`).
- GUI distinction: a failure stays in `Model.Error` until answered; a completion becomes a `Model.Toasts` notice that expires after three seconds (`internal/ui/model.go`, `internal/ui/toast.go`).
- Health issues are structured, not prose: `HealthIssue{Path, Check, Expected, Actual}` with a `String()` renderer (`internal/installer/status.go`).
- Retry context is persisted so an operation is retried as itself, never as a different one (`internal/installer/lifecycle.go`, `appstore.validRetryAction`).

## Cross-Cutting Concerns

**Logging:** `internal/diagnostics.Log.Event(name, fields...)` emits `event=` lines with redacted values into a 512 KiB bounded, rotating log. Every module takes an optional `*diagnostics.Log` and `Manager.event` no-ops on nil, so tests run without I/O.

**Validation:** Three concentric gates — JSON Schema (`schema/package-manifest-v1.schema.json`) for shape, `manifest.Validate()` for semantic policy, `platform.Check()` for the detected machine. The installer re-runs both `Validate` and `checkPreconditions` before every operation even though `catalog.Load` already validated the package.

**Authentication:** Not a user concern. Integrity rests on the ed25519-signed catalogue index plus per-release size and SHA-256. Device identity is explicit per device and never inferred from a SoC family.

**Redaction:** `diagnostics.Redact` is applied at write time and again on export, and on the existing log file at `Open`, so a format change cannot leak a token that an older build wrote.

---

*Architecture analysis: 2026-09-21*
