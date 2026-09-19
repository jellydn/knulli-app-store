# Architecture

**Analysis Date:** 2026-09-19

## Pattern Overview

**Overall:** Layered installer core with thin adapters and one optional SDL2 presentation layer

**Key Characteristics:**
- The installer owns every side effect. No front end extracts an archive, writes XML, or runs a shell command (`docs/architecture.md`)
- Manifest-driven catalogue with strict JSON decoding (`DisallowUnknownFields` in `internal/manifest/load.go`) and separate review/approval gates
- One typed verdict decides what a package may offer; policy stays in the package that owns it (`internal/appstore/verdict.go`)
- Transactional, journaled writes behind an allowed-path guard (`internal/safefs`)
- GUI compiled only with the `sdl` tag, so the CLI stays static and CGO-free
- The same GUI runs headless on a desktop through scripted key sequences (`internal/sdlui/walk.go`)

## Layers

**Manifest and catalogue:**
- Purpose: the v1 data contract, semantic validation, deterministic index build, ed25519 signing
- Location: `internal/manifest`, `internal/catalog`, `schema/package-manifest-v1.schema.json`, `catalogue/`
- Contains: `Package`, `Review`, `Release`, `Compatibility`, `Install`, `Menu`, `BinaryPatch`; `catalog.Index`/`Entry`; `Sign`, `Verify`, `GenerateKey`
- Depends on: `encoding/json` only
- Used by: CLI, installer, appstore service, update checker, GUI startup

**Platform:**
- Purpose: detect firmware, version, architecture, ABI, device, and resolution; enforce compatibility
- Location: `internal/platform/platform.go`
- Contains: `Info` with per-field `Source` evidence, `Resolve`/`WithX` options, `Check`, `AssessResolutions`, `DisplayName`, `Summary`
- Depends on: `internal/manifest` compatibility fields
- Used by: CLI, GUI, `installer.Manager.checkPreconditions`, `appstore.verdict.assess`

**Safe filesystem and archives:**
- Purpose: bound extraction and every destination mutation
- Location: `internal/archive`, `internal/safefs`
- Contains: ZIP and `tar.gz` extraction with traversal/link/duplicate rejection; `Guard.Resolve`; `AtomicWrite`, `Copy`, `SHA256`, `AvailableBytes`; `Transaction` with snapshots and a journal
- Depends on: `archive/zip`, `archive/tar`, `compress/gzip`, `crypto/sha256`
- Used by: `internal/installer`, `internal/gamelist`

**Menu integration:**
- Purpose: own exactly one `gamelist.xml` entry and nothing else
- Location: `internal/gamelist/menu.go`, `internal/gamelist/xml.go`
- Contains: `Plan`/`Step` derivation (`Derive`), `Apply`, and XML add/replace/remove that preserves unknown elements
- Depends on: `internal/manifest` (`Menu`), `internal/safefs` (`Transaction`, `Guard`)
- Used by: `internal/installer` during commit and uninstall
- Note: this package was split out of `internal/installer`; there is no `internal/installer/gamelist.go` any more

**Installer service:**
- Purpose: download, install, adopt, update, repair, force-reinstall, uninstall, health, lifecycle/retry, refresh
- Location: `internal/installer`
- Contains: `Manager` with a three-stage `apply` (`checkPreconditions` → `stage` → `commit`), `state.go`, `lifecycle.go`, `status.go`, `refresh.go`, `download.go`
- Depends on: manifest, platform, archive, safefs, gamelist, diagnostics
- Used by: `cmd/knulli-app` and `internal/appstore`

**Application facade:**
- Purpose: expose only the actions the current package state allows
- Location: `internal/appstore`
- Contains: `Backend` interface, `Service`, `Item`, `Action`, and `verdict.go` with `State`, `Reason`, `ReasonKind`, `assess`
- Depends on: catalog, installer, platform, manifest
- Used by: `internal/ui` and `cmd/knulli-app-ui`

**Presentation:**
- Purpose: a controller-native catalogue browser, first-run mapping setup, and a raster UI
- Location: `internal/ui` (pure Go model), `internal/input` (semantic mapping and setup session), `internal/sdlui` (SDL adapter, layout, draw, hints, walkthrough)
- Contains: `Model` with `Focus`, `Tabs`, `Toasts`; `Session` setup state machine, `Calibration`, `Store`, `Footer`; `Run`, `Handle`, `draw`, `layout`, `WalkRunner`
- Depends on: `appstore.Backend` only — SDL appears solely in `internal/sdlui`
- Used by: `cmd/knulli-app-ui`

**Thin adapters:**
- Purpose: flag parsing, logging setup, wiring, and exit codes
- Location: `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`, `cmd/check-updates/main.go`

## Data Flow

**Install / adopt / update / repair (force-reinstall shares the path):**
1. `manifest.Load` decodes strictly and `Validate` rejects candidates and policy violations
2. `platform.Resolve` reads Knulli-owned files, then `platform.Check` compares the detected matrix
3. `validateDownloadURL` requires HTTPS and a version-pinned, immutable URL
4. `acquireManager` opens manager state under its own guard, takes an exclusive lock, and runs `safefs.Recover` on any interrupted journal
5. `stage` reconciles the requested operation with installed state, inventories an existing destination, checks free space, downloads with exact size + SHA-256, and extracts into a temp tree
6. `commit` performs package files, the owned menu entry, backups, and installed state inside one `safefs.Transaction`; the transaction commits, then state is written last
7. On success `outcome` asks loopback `/reloadgames` and reports accepted or `RestartRequired`

**GUI action:**
1. `appstore.Open` loads and signature-verifies the index beside the binary (`-catalog` overrides)
2. `Service.Items` joins each index entry with `Manager.Status` and `PreExisting`, then `assess` returns one `Verdict` per package
3. `ui.Model` offers only the returned actions and filters them into `TabAll`, `TabReady`, or `TabInstalled`
4. `input.Session` gates the catalogue behind first-run setup and turns physical buttons into semantic actions
5. `sdlui.Handle`/`dispatchAction` calls `Service.Execute`; the model shows progress, health, retry, and restart-required outcomes

**Update report:**
1. `cmd/check-updates` builds the index, passes the decoded packages to `updatecheck.Checker.Check`, and writes a JSON plus Markdown report

**State Management:**
- Installed ownership, hashes, modes, and originals live in `/userdata/system/knulli-app-store/<id>.json`
- Requested operation, install type, and safe retry target live in `/userdata/system/knulli-app-store/lifecycle/`
- Controller mappings are versioned JSON keyed by device + controller identity
- The GUI model is in-memory; operations run on a goroutine and report through a channel (`internal/ui/model.go` `operationEvent`, `Poll`)

## Key Abstractions

**`manifest.Package`:**
- Purpose: the reviewed contract for one package
- Examples: `internal/manifest/manifest.go`, `catalogue/packages/io.github.unitreign.playtime.json`
- Pattern: status-gated struct with `Validate`, `Installable`, `Experimental`, `DeviceTested`

**`safefs.Guard` / `safefs.Transaction`:**
- Purpose: resolve a virtual `/userdata` path into `-root` and make every write reversible
- Examples: `internal/safefs/guard.go`, `internal/safefs/transaction.go`
- Pattern: allowed-path list per operation; `Recover`, `PendingForPath`, and a durable journal

**`installer.Manager`:**
- Purpose: the only thing that mutates a package destination
- Examples: `internal/installer/installer.go` (`apply`, `stage`, `commit`)
- Pattern: value struct with injectable `Client`, `RefreshClient`, `Now`, and `AvailableBytes` for tests

**`appstore.Backend`:**
- Purpose: a UI-neutral catalogue surface
- Examples: `internal/appstore/service.go` (`Items`, `Execute`, `ExportDiagnostics`, `SetPlatform`)
- Pattern: interface with an in-memory fake in `internal/ui` tests

**`appstore.Verdict`:**
- Purpose: combine review status, platform compatibility, install state, and health into one decision
- Examples: `internal/appstore/verdict.go` (`assess`, `State`, `Reason`)
- Pattern: policy-free combiner — `manifest`, `platform`, and `installer` keep owning the rules

**`input.Action` / `input.Mapping` / `input.Session`:**
- Purpose: semantic actions instead of physical A/B, and per-identity saved bindings
- Examples: `internal/input/mapping.go`, `internal/input/store.go`, `internal/input/session.go`
- Pattern: 9 actions (7 required + optional paging pair), atomic JSON store, one setup state machine

**`gamelist.Plan`:**
- Purpose: declare menu changes before they happen
- Examples: `internal/gamelist/menu.go` (`Derive`, `Apply`)
- Pattern: pure plan derivation, then `Apply` into the transaction

## Entry Points

**`cmd/knulli-app`:**
- Location: `cmd/knulli-app/main.go`
- Triggers: `validate`, `catalogue`, `install`, `adopt`, `update`, `repair`, `uninstall`
- Responsibilities: flag parsing, `diagnostics.Open`, platform resolution with overrides, manager calls, and printing the game-list outcome

**`cmd/knulli-app-ui`:**
- Location: `cmd/knulli-app-ui/main.go` (`//go:build sdl`)
- Triggers: the Ports launcher `packaging/*/Knulli App Store.sh`, or `make gui`
- Responsibilities: locate the index beside the binary, open diagnostics, resolve platform, build the service, and run the SDL loop (interactive, screenshot, or scripted walkthrough)

**`cmd/check-updates`:**
- Location: `cmd/check-updates/main.go`
- Triggers: the weekly `catalogue updates` workflow or a manual run
- Responsibilities: build the index, query release metadata, and write the review report; it never edits or approves a manifest

## Error Handling

**Strategy:** return `error` values; the CLI prints `error: <err>` to stderr and exits 1, the GUI records a `startup_error`/`final_error` event and reports the message on screen

**Patterns:**
- Wrap at the boundary with `fmt.Errorf("...: %w", err)` (for example `open diagnostics log: %w`)
- Compatibility failures carry the raw value, normalized value, source, and the full detected matrix (`internal/platform` `compatibilityError`, `Summary`)
- Health issues carry path, check, expected, and actual (`internal/installer/status.go` `HealthIssue`)
- Adoption conflicts and stale transactions are typed so the UI can offer recovery: `installer.AdoptionConflictError`, `installer.RecoveryStatus`
- A failed step rolls the transaction back in reverse order; a checksum failure writes no package files
- Corrupt manager state is a typed error (`internal/input/store.go` `corruptFileError`) rather than a silent reset

## Cross-Cutting Concerns

**Logging:** `internal/diagnostics` writes UTC events with the event name first; log capped at 512 KiB with one rotated copy; secrets, URL query strings, and terminal control sequences are redacted before writing

**Validation:** JSON Schema (`schema/package-manifest-v1.schema.json`) plus Go `Validate()`; unknown JSON fields are rejected; manifest tests assert the two agree

**Authentication:** none for users. Trust is catalogue review plus pinned SHA-256, with an ed25519 signature over the device index verified against a key compiled into the binary

**Determinism:** `catalog.Build` sorts and re-encodes canonically, and CI compares two independent builds byte for byte

---

*Architecture analysis: 2026-09-19*
