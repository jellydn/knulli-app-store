# Architecture

**Analysis Date:** 2026-10-25

## Pattern Overview

**Overall:** A transactional package manager with a static CLI, an application service, and an optional gabagool-backed SDL frontend.

**Key Characteristics:**
- The installer owns package policy and package writes. Frontends call the manager or service; they do not extract archives or edit menu XML (`internal/installer/installer.go`, `internal/appstore/service.go`).
- The GUI uses the owner-selected Library (Option B): a compact list with an expanded selected row, visible filters, a local Store mark, and contextual button glyphs. Gabagool owns window/renderer lifetime, not semantic controls or model transitions (`internal/sdlui/draw.go`, `internal/sdlui/run.go`, `docs/adr/0010-gabagool-knulli-frontend.md`).
- The `sdl` build tag keeps the graphical adapter out of the static CLI. The model, input mapper, and portable event handler do not need SDL (`cmd/knulli-app-ui/main.go`, `internal/ui/model.go`, `internal/input/mapping.go`, `internal/sdlui/events.go`).
- Firmware, architecture, ABI, and runtime requirements gate installation. Device and display scope are advisory. A tested badge needs exact device evidence (`internal/platform/platform.go`, `internal/manifest/manifest.go`).
- Only KNULLI is implemented. The two packaged targets are TrimUI Smart Pro and MagicX Zero 28 (`scripts/package-device.sh`, `docs/adr/0010-gabagool-knulli-frontend.md`).

## Layers

**Manifest and catalogue:**
- Purpose: Define reviewed packages and build a deterministic catalogue.
- Location: `internal/manifest/`, `internal/catalog/`.
- Contains: Strict decoding, semantic validation, canonical package bytes, sorted index entries, and Ed25519 signatures.
- Depends on: The Go standard library; catalogue code also uses manifest code.
- Used by: CLI validation and builds, the application service, the installer, and the maintenance checker (`cmd/knulli-app/main.go`, `internal/appstore/service.go`, `cmd/check-updates/main.go`).

**Platform and version:**
- Purpose: Detect platform values with source evidence and check runtime compatibility.
- Location: `internal/platform/platform.go`, `internal/version/version.go`.
- Contains: Firmware, architecture, device, and resolution detection; explicit overrides; compatibility checks; version comparison.
- Depends on: Manifest and version code.
- Used by: CLI setup, GUI setup, verdicts, and installer preconditions (`cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`, `internal/appstore/verdict.go`).

**Archive, safe filesystem, and menu:**
- Purpose: Supply bounded staging, guarded paths, reversible writes, and exact menu-entry ownership.
- Location: `internal/archive/archive.go`, `internal/safefs/`, `internal/gamelist/`.
- Contains: ZIP/tar.gz extraction, embedded base64 tar.gz decoding, path and symlink checks, atomic replacement, durable journals, XML merge rules, and reverse rollback.
- Depends on: Standard-library facilities and shared manifest contracts where needed.
- Used by: Package lifecycle operations and Store self-update (`internal/installer/stage.go`, `internal/installer/commit.go`, `internal/installer/selfupdate.go`).

**Installer:**
- Purpose: Own package installation, removal, health, recovery, and Store replacement.
- Location: `internal/installer/`.
- Contains: Manager sessions and locks, download verification, staging, adoption, file ownership, preserved data, installed records, lifecycle retry records, and status caching.
- Depends on: Manifest, platform, archive, safe filesystem, menu, and diagnostics packages.
- Used by: The CLI directly and the GUI through the application service (`cmd/knulli-app/main.go`, `internal/appstore/service.go`).

**Application service:**
- Purpose: Present typed package states and permitted actions without rendering code.
- Location: `internal/appstore/service.go`, `internal/appstore/verdict.go`, `internal/appstore/selfupdate.go`.
- Contains: `Backend`, `Service`, `Item`, `Verdict`, action validation, diagnostic export, and the synthetic Store card.
- Depends on: Catalogue, installer, manifest, and platform packages.
- Used by: The UI model and GUI startup (`internal/ui/model.go`, `cmd/knulli-app-ui/main.go`).

**Model and semantic input:**
- Purpose: Own navigation, confirmations, asynchronous progress, and button meaning.
- Location: `internal/ui/`, `internal/input/`.
- Contains: Browse/Health/Actions/Confirm/ForceConfirm states, tab selection memory, notices, logical controller bindings, keyboard bindings, connection state, and the quit chord.
- Depends on: The application backend and portable data types; not on gabagool.
- Used by: The SDL adapter and fake-backend tests (`internal/sdlui/events.go`, `internal/ui/model_test.go`).

**SDL presentation:**
- Purpose: Own the graphical loop, draw the Library and focused review screens, translate SDL events, and capture evidence.
- Location: `internal/sdlui/run.go`, `internal/sdlui/draw.go`, `internal/sdlui/events.go`, `internal/sdlui/walk.go`.
- Contains: Gabagool advanced API calls, SDL GameController handling, a responsive software-rasterized canvas, letterboxing, navigation repeat, and scripted walkthroughs. The Random Game Picker 0.2.0 layout uses logical height 480 and width max(640, output width × 480 / output height); embedded DejaVu Sans supplies 18/26 px body/title faces, not a fixed bitmap font (`internal/sdlui/layout.go`, `internal/sdlui/assets/DejaVuSans.ttf`).
- Depends on: The model, semantic input, platform, diagnostics, gabagool/v2, go-sdl2, and image code.
- Used by: The SDL-tagged GUI command (`cmd/knulli-app-ui/main.go`).

**Diagnostics and maintenance:**
- Purpose: Record bounded redacted logs and produce upstream review reports.
- Location: `internal/diagnostics/log.go`, `internal/updatecheck/check.go`, `internal/updatecheck/report.go`.
- Contains: Rotating logs, user-requested exports, GitHub release metadata comparison, and JSON/Markdown reports.
- Depends on: Standard-library code; the update checker also uses manifest and version code.
- Used by: Runtime operations and the separate maintenance command (`internal/installer/installer.go`, `cmd/check-updates/main.go`).

## Data Flow

**Package install, Manage existing, update, repair, or force reinstall:**
1. The CLI calls `Manager.Apply`. The GUI calls `Service.Execute`, which rechecks permitted actions and records lifecycle/retry context (`cmd/knulli-app/main.go`, `internal/appstore/service.go`).
2. Preconditions validate the manifest, actionable status, runtime compatibility, and release URL before package filesystem work (`internal/installer/installer.go`).
3. A manager session takes a nonblocking exclusive file lock, recovers interrupted journals, and removes abandoned staging directories (`internal/installer/stage.go`).
4. Staging reconciles installed and external files, checks space, downloads exact release bytes, and verifies SHA-256 before extraction (`internal/installer/stage.go`, `internal/installer/download.go`).
5. Extraction accepts regular files within path and byte limits. An embedded archive is decoded only after whole-release verification. The Store never executes its script prefix (`internal/archive/archive.go`, `internal/installer/stage.go`).
6. Staging applies reviewed binary patches, file mappings, executable modes, and launcher checks (`internal/installer/stage.go`, `internal/installer/files.go`).
7. A journaled transaction applies managed/preserved/original-file rules and menu changes. Force reinstall first backs up the complete existing destination. The installed record is the last data write before transaction commit (`internal/installer/commit.go`, `internal/installer/recovery.go`).
8. After commit, a menu change requests a loopback game list refresh. Acceptance means queued, not completed; refusal reports restart required without undoing the operation (`internal/installer/refresh.go`, `CONTEXT.md`).

**Package removal:**
1. The manager takes the same lock and recovers journals, then reads installed state (`internal/installer/uninstall.go`, `internal/installer/stage.go`).
2. Read-only uninstall checks can block removal for shared hooks or active-service state before package mutation. The Store does not run upstream cleanup scripts (`internal/installer/uninstall.go`).
3. A transaction restores originals or removes owned files. Preserved and unmanaged files stay. It removes only an unchanged menu entry the Store created, then removes backup/state records and commits (`internal/installer/uninstall.go`, `internal/gamelist/menu.go`).

**Catalogue browsing and GUI execution:**
1. `Service.Open` recovers transactions before it loads the catalogue. A self-update can replace both the key-bearing executable and catalogue, so this order is required (`internal/appstore/service.go`, `internal/installer/selfupdate.go`).
2. `Items` reads package health, compatibility, pre-existing files, and retry context. It derives typed verdicts/actions, adds the Store card when enabled, and sorts by name (`internal/appstore/service.go`, `internal/appstore/verdict.go`).
3. `Model` filters All/Ready/Installed and remembers selection per tab. Confirm opens an action or health; Y opens More (`internal/ui/model.go`, `internal/ui/tabs.go`, `internal/sdlui/events.go`).
4. Operations run asynchronously. A buffered event channel carries progress and completion back to `Model.Poll`; rendering does not perform installation (`internal/ui/model.go`).
5. Notices expire after three seconds. Failures stay visible until answered (`internal/ui/toast.go`, `internal/ui/model.go`).

**Gabagool rendering and input:**
1. `Run` rejects `INPUT_CAPTURE`, locks the OS thread, and probes rendering to retain the software fallback (`internal/sdlui/run.go`).
2. Gabagool initializes the window/renderer with keyboard, controller, and joystick input processors disabled. The Store supplies an empty toolkit mapping (`internal/sdlui/run.go`).
3. Runtime display candidates update platform evidence and the backend. The Store renders its own canvas to an SDL texture and handles letterboxing itself (`internal/sdlui/run.go`, `internal/sdlui/layout.go`).
4. SDL events pass through the existing semantic mapper. Automatic GameController input has no setup or saved custom assignments; disconnect enters Blocked. Desktop keyboard mode uses fixed bindings (`internal/sdlui/events.go`, `internal/input/session.go`, `internal/input/keyboard.go`).
5. The launcher selects Knulli or Xbox letter policy. Quit remains Select+Y; keyboard quit is held Tab or F5 plus Y (`packaging/trimui-smart-pro/Knulli App Store.sh`, `internal/input/mapping.go`, `internal/sdlui/events.go`).
6. The current working-tree loop polls input and model at 30 Hz. Normal mode redraws initially and after SDL events (including window/controller events), held-navigation repeat, `Model.Poll` changes, or a change in live-toast count as notices expire. Idle ticks skip canvas allocation, texture upload, and presentation. Screenshot and walkthrough modes still render full frames; captures read the actual renderer backbuffer before `Present`, not only the source canvas (`internal/sdlui/run.go`, `internal/sdlui/repeat.go`, `internal/ui/model.go`, `internal/sdlui/walk.go`).

**Store self-update:**
1. GUI startup enables the synthetic Store card only at the canonical packaged executable path with a stamped target (`cmd/knulli-app-ui/main.go`, `internal/appstore/selfupdate.go`).
2. Check reads official GitHub releases, including prereleases, and pins the target ZIP URL, size, and GitHub SHA-256 digest. Comparable equal/older numeric tags are not offered; hash builds and ambiguous prereleases still use API order. Update is a separate confirmed action; a failed recheck clears the old offer (`internal/installer/selfupdate.go`, `internal/appstore/selfupdate.go`, `internal/version/version.go`).
3. `UpdateStore` takes the manager lock, recovers journals, downloads/verifies/extracts, checks space, and requires the exact seven-file package inventory (`internal/installer/selfupdate.go`, `scripts/package-device.sh`).
4. One transaction replaces only Store package files. Atomic rename keeps the running Linux executable usable. Installed apps and settings are outside that inventory; the user must exit and reopen (`internal/installer/selfupdate.go`, `internal/safefs/files.go`).
5. The device launcher runs CLI recovery before loading the GUI. Menu registration follows and is nonfatal (`packaging/magicx-zero-28/Knulli App Store.sh`).

**Maintenance update report:**
1. The checker builds the local reviewed catalogue and reads upstream GitHub release metadata (`cmd/check-updates/main.go`, `internal/updatecheck/check.go`).
2. It writes JSON and Markdown for human review. It does not download assets, change manifests, or install updates (`cmd/check-updates/main.go`, `internal/updatecheck/report.go`).

**State Management:**
- Installed records, originals, recovery backups, and journals use the manager state area below `/userdata`. Root redirection supports disposable fixtures (`internal/installer/state.go`, `internal/manifest/manifest.go`, `internal/safefs/transaction.go`).
- Lifecycle records retain failed-operation retry context separately from installed state (`internal/installer/lifecycle.go`).
- The GUI enables `StatusCache`. Its key hashes the installed record, not live file contents. External file edits require invalidation before another full health check; a manager without a cache checks files on each status read (`internal/installer/cache.go`, `internal/installer/status.go`).
- Tab memory, input bindings, pending Store offers, and the Store restart flag are process-local (`internal/ui/tabs.go`, `internal/input/session.go`, `internal/appstore/selfupdate.go`).

## Key Abstractions

**`installer.Manager`:**
- Purpose: Run lifecycle policy and side effects with a selected root and platform.
- Examples: `internal/installer/installer.go`, `internal/installer/uninstall.go`.
- Pattern: Orchestrator split into preconditions, staging, and commit.

**`appstore.Backend` and `Verdict`:**
- Purpose: Separate presentation from service policy through four backend methods and typed states, reasons, and actions.
- Examples: `internal/appstore/service.go`, `internal/appstore/verdict.go`, `internal/ui/model_test.go`.
- Pattern: Interface boundary with an in-memory fake for UI tests; no parsing of rendered labels for policy.

**`safefs.Guard` and `Transaction`:**
- Purpose: Limit paths and restore writes after failure or interruption.
- Examples: `internal/safefs/guard.go`, `internal/safefs/transaction.go`.
- Pattern: Path allowlist plus write-ahead snapshots and reverse rollback.

**`input.Session` and `ui.Model`:**
- Purpose: Keep controls, connection state, and UI transitions independent of SDL lifetime.
- Examples: `internal/input/session.go`, `internal/input/mapping.go`, `internal/ui/model.go`.
- Pattern: Portable state machines driven by semantic events.

**`StoreUpdate`:**
- Purpose: Pin an official Store asset separately from reviewed package manifests and installed-package records.
- Examples: `internal/installer/selfupdate.go`, `internal/appstore/selfupdate.go`.
- Pattern: Check/confirm/replace flow with an exact archive inventory.

## Entry Points

**Static CLI:**
- Location: `cmd/knulli-app/main.go`.
- Triggers: Shell use, build automation, or device launcher startup.
- Responsibilities: Validate manifests; build/sign the catalogue; install, adopt, update, repair, and uninstall packages; recover transactions; register the Store menu entry.

**Optional graphical frontend:**
- Location: `cmd/knulli-app-ui/main.go`.
- Triggers: Device launcher or desktop fixture/walkthrough.
- Responsibilities: Parse flags, resolve platform evidence, open diagnostics/service, gate self-update, and start `sdlui.Run`.

**Maintenance reporter:**
- Location: `cmd/check-updates/main.go`.
- Triggers: Manual command or `.github/workflows/catalogue-updates.yml`.
- Responsibilities: Compare upstream release metadata and write review reports.

**Device build and launch:**
- Location: `scripts/device-build.sh`, `scripts/package-device.sh`, `packaging/trimui-smart-pro/Knulli App Store.sh`, `packaging/magicx-zero-28/Knulli App Store.sh`.
- Triggers: Release automation, local device builds, or the KNULLI Ports menu.
- Responsibilities: Build signed target artifacts, enforce ABI constraints, package seven files, recover before GUI launch, and select controller layout.

## Error Handling

**Strategy:** Refuse unsafe inputs before package mutation. Roll back failed transactions. Keep committed outcomes separate from follow-up failures (`internal/installer/installer.go`, `internal/installer/commit.go`, `internal/installer/refresh.go`).

**Patterns:**
- Strict JSON decoding rejects unknown fields and trailing values; manifest validation collects policy errors (`internal/manifest/load.go`, `internal/manifest/manifest.go`).
- Context cancellation interrupts downloads and archive transfers before destination mutation (`internal/installer/download.go`, `internal/archive/archive.go`).
- Deferred rollback retains rollback failures. Startup recovery retains evidence when it cannot complete restoration (`internal/installer/commit.go`, `internal/safefs/transaction.go`).
- Compatibility refusals report raw values, normalized values, sources, and the detected matrix (`internal/platform/platform.go`).
- Service failures retain retry context; errors after lifecycle cleanup can still follow an already committed operation (`internal/appstore/service.go`).

## Cross-Cutting Concerns

**Logging:** Runtime logs are bounded, rotated, and redacted. Diagnostic exports are user-triggered. Gabagool logs go to stderr, not a new file beside the app (`internal/diagnostics/log.go`, `internal/appstore/service.go`, `internal/sdlui/run.go`).

**Validation:** JSON Schema describes the manifest shape. Go decoding and semantic validation enforce runtime rules. Archive extraction and guarded write paths enforce file safety separately (`schema/package-manifest-v1.schema.json`, `internal/manifest/load.go`, `internal/archive/archive.go`, `internal/safefs/guard.go`).

**Authentication:** There is no user account service. Device build linker flags supply the catalogue public key to both builds; the GUI retains and uses it at catalogue open, while the CLI's unused key is eliminated. Catalogue loading verifies signatures when the key is set. Pinned release size/hash checks verify bytes. Store update metadata trusts official GitHub HTTPS responses (`scripts/device-build.sh`, `internal/appstore/service.go`, `internal/catalog/catalog.go`, `internal/catalog/sign.go`, `internal/installer/download.go`, `internal/installer/selfupdate.go`; linked-artifact evidence attributed to source thread `T-01a16e63-3dfe-7782-8ac5-a9a71a55726c`).

---

*Architecture analysis: 2026-10-25*
