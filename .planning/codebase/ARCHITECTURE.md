# Architecture

**Analysis Date:** 2026-10-21

## Pattern Overview

**Overall:** A policy-owning transactional installer with CLI and SDL2 adapters, an application service, and a pure-Go UI state machine.

**Key Characteristics:**
- `internal/installer/installer.go` owns package lifecycle side effects; frontends do not extract archives or edit menus themselves.
- `internal/appstore/service.go` combines catalogue, compatibility, health, and recovery into typed actions. The CLI calls the manager directly; the GUI uses `appstore.Backend`.
- `internal/sdlui/run.go` and `internal/sdlui/draw.go` are behind `sdl`; event routing and state logic remain testable without cgo.
- Runtime compatibility gates are separate from advisory device/display scope (`internal/platform/platform.go`, `docs/adr/0009-advisory-device-compatibility.md`).

## Layers

| Layer | Ownership and dependencies | Main locations |
| --- | --- | --- |
| Manifest | JSON contract, semantic policy, shared size/state-path constants; stdlib only | `internal/manifest/manifest.go`, `internal/manifest/load.go` |
| Catalogue | Sorted deterministic index, manifest digests, optional embedded-key signature validation; depends on manifest | `internal/catalog/catalog.go`, `internal/catalog/sign.go` |
| Platform/version | Detection with provenance, runtime checks, version comparison; uses manifest and version helpers | `internal/platform/platform.go`, `internal/version/version.go` |
| Safe filesystem | Allowed paths, symlink rejection, atomic replacement, durable journals and recovery | `internal/safefs/guard.go`, `internal/safefs/files.go`, `internal/safefs/transaction.go` |
| Archive | Bounded ZIP/tar.gz extraction with context cancellation; no live destination writes | `internal/archive/archive.go` |
| Menu | Derive and apply owned-entry changes; fill blank metadata on unowned exact paths without claiming them | `internal/gamelist/menu.go`, `internal/gamelist/xml.go` |
| Installer | Lock, recovery, download/stage, transactional lifecycle, installed state, health/cache, Store replacement | `internal/installer/installer.go`, `internal/installer/stage.go`, `internal/installer/commit.go`, `internal/installer/selfupdate.go` |
| Application | Backend interface, typed verdicts, permitted actions, diagnostics, synthetic Store card | `internal/appstore/service.go`, `internal/appstore/verdict.go`, `internal/appstore/selfupdate.go` |
| UI/input | Card selection, filters, confirmation/health/progress, toasts; automatic logical button bindings and quit chord | `internal/ui/model.go`, `internal/ui/tabs.go`, `internal/input/session.go`, `internal/input/mapping.go` |
| SDL adapter | SDL lifecycle, portable event dispatch, rendering, repeat, scripted walkthroughs | `internal/sdlui/run.go`, `internal/sdlui/events.go`, `internal/sdlui/draw.go`, `internal/sdlui/walk.go` |
| Maintenance | Read-only upstream release comparison and review reports | `internal/updatecheck/check.go`, `internal/updatecheck/report.go` |

## Data Flow

**Package installation:**
1. CLI calls `Manager.Apply`; GUI calls `Service.Execute`, revalidates the action, and records lifecycle/retry context (`cmd/knulli-app/main.go`, `internal/appstore/service.go`).
2. Preconditions validate the manifest, actionable review status, runtime platform, and release URL (`internal/installer/installer.go`). Candidates remain read-only; experimental is actionable but not a tested claim.
3. Acquire a guarded manager directory and exclusive nonblocking flock, recover open transaction journals, then reclaim abandoned package/Store staging directories (`internal/installer/stage.go`).
4. Reconcile installed/external state and check space. Download exact release bytes and verify SHA-256 before bounded extraction (`internal/installer/stage.go`, `internal/installer/download.go`).
5. Apply reviewed binary patches to archive paths, then declarative file mappings with collision/missing-source checks, then executable modes and launcher checks (`internal/installer/stage.go`, `internal/installer/files.go`).
6. Begin a journaled transaction. Apply managed/preserved/original-file rules and menu effects, then write installed state last and commit (`internal/installer/commit.go`). Force reinstall first backs up the complete existing destination; uninstall restores originals and keeps preserved/unmanaged files (`internal/installer/recovery.go`, `internal/installer/uninstall.go`).
7. Request a loopback game list refresh after a menu change. A refusal reports restart required but leaves the operation committed (`internal/installer/refresh.go`).

**Catalogue and card browsing:**
1. `Service.Open` recovers interrupted transactions before loading the signed catalogue. This avoids reading a partially replaced binary/catalogue pair (`internal/appstore/service.go`).
2. `Items` derives each package's verdict from platform, health, pre-existing files, and retry context, optionally adds the Store item, and sorts by name (`internal/appstore/service.go`, `internal/appstore/verdict.go`).
3. `Model` filters All/Ready/Installed and remembers selection per tab. Arrows browse one app card; shoulders/Page Up/Page Down filter; confirm opens the first action or health; Y opens More (`internal/ui/model.go`, `internal/ui/tabs.go`, `internal/sdlui/events.go`).
4. Operations run asynchronously and send progress/completion through a buffered channel drained by `Model.Poll`. Notices expire after three seconds; errors remain until answered (`internal/ui/model.go`, `internal/ui/toast.go`). Catalogue loads, not every rendered frame, perform status reads.

**Automatic input:**
1. The launcher sets `STORE_CONTROLLER_LAYOUT` from the Ports Xbox-layout flag (`packaging/trimui-smart-pro/Knulli App Store.sh`).
2. An SDL GameController connection uses automatic logical bindings immediately; no setup, calibration, or mapping persistence occurs (`internal/input/session.go`, `internal/input/mapping.go`).
3. Knulli layout changes bindings/labels, not incoming events. Disconnect enters Blocked; keyboard mode has fixed bindings. Select+Y (Tab/F5+Y on keyboard) quits (`internal/sdlui/events.go`, `internal/input/keyboard.go`).

**Store self-update:**
1. Only a canonical packaged GUI with a stamped build target enables the synthetic Store card (`cmd/knulli-app-ui/main.go`).
2. Check queries official releases including prereleases, pins a target asset using GitHub size/digest, and offers a separate confirmed Update action (`internal/appstore/selfupdate.go`, `internal/installer/selfupdate.go`).
3. `UpdateStore` acquires the manager lock and recovers journals, downloads/verifies/extracts, checks the seven-file inventory and free space, and transactionally replaces only the packaged files. Atomic rename keeps the running executable alive (`internal/installer/selfupdate.go`, `internal/safefs/files.go`).
4. Installed apps/settings remain outside that inventory. Exit/reopen is required. The device launcher runs the CLI `recover` command before loading the GUI, then attempts menu registration without making metadata failure fatal (`packaging/magicx-zero-28/Knulli App Store.sh`).

**State Management:** Installed state, lifecycle records, original/recovery backups, and transaction journals live below manager state. Input bindings are process-local. `StatusCache` hashes the installed record for its key, not live file contents; unchanged records can reuse health until invalidation (`internal/installer/state.go`, `internal/installer/lifecycle.go`, `internal/installer/cache.go`, `internal/input/session.go`).

## Key Abstractions

- **`appstore.Backend`:** Four methods (`Items`, `Execute`, `ExportDiagnostics`, `SetPlatform`) separate presentation from the real service; UI tests use a fake (`internal/appstore/service.go`, `internal/ui/model_test.go`).
- **`Verdict`:** Typed state/reasons/actions, not parsed display strings. Untested devices still get actions when runtime checks pass (`internal/appstore/verdict.go`).
- **`Guard` and `Transaction`:** Virtual-path policy plus write-ahead snapshots; reverse rollback and startup recovery preserve mutations as one unit (`internal/safefs/guard.go`, `internal/safefs/transaction.go`).
- **`Mapping` and `Session`:** Logical SDL/keyboard actions plus Normal/Blocked connection state. Old mapping files are ignored and untouched (`internal/input/mapping.go`, `internal/input/session.go`).
- **`StoreUpdate`:** A pinned official release handled separately from reviewed package manifests and installed-package records (`internal/installer/selfupdate.go`).

## Entry Points

- `cmd/knulli-app/main.go`: validate, catalogue build/sign, install/adopt/update/repair, uninstall, recover, register-store-menu. Calls `installer.Manager` directly.
- `cmd/knulli-app-ui/main.go`: SDL-tagged flag wiring, platform/log/service initialization, canonical-location self-update gate, SDL run.
- `cmd/check-updates/main.go`: read-only JSON/Markdown review report, invoked manually or weekly.
- `scripts/device-build.sh`: signed cross-build and ABI checks; `scripts/package-device.sh`: seven-file device ZIP.

## Error Handling

**Strategy:** Refuse unsafe inputs, roll back failed transactions, and distinguish committed outcomes from follow-up failures.

**Patterns:**
- Manifest validation accumulates, sorts, and joins all problems (`internal/manifest/manifest.go`).
- Wrapped errors retain `%w`; archive extraction honors context between entries and during transfers (`internal/archive/archive.go`, `internal/installer/stage.go`).
- Deferred rollback retains rollback errors; failed recovery retains evidence for a later attempt (`internal/installer/commit.go`, `internal/safefs/transaction.go`).
- Compatibility errors include detected values and provenance; device/resolution absence is not itself a runtime refusal (`internal/platform/platform.go`).
- Game list acceptance is not completion; Store completion instead requires exit/reopen (`internal/installer/refresh.go`, `internal/appstore/selfupdate.go`).

## Cross-Cutting Concerns

**Logging:** Bounded rotating, redacted event logs and user-triggered exports (`internal/diagnostics/log.go`).

**Validation:** JSON Schema documents manifest shape; Go decoding and `Validate` enforce runtime policy, followed by platform checks. The runtime does not invoke a JSON Schema engine (`schema/package-manifest-v1.schema.json`, `internal/manifest/load.go`).

**Authentication:** No user identity service. Catalogue signatures are per build; asset size/hash checks establish release integrity, and Store update metadata comes from the official GitHub HTTPS API (`internal/catalog/sign.go`, `internal/installer/selfupdate.go`).

---

*Architecture analysis: 2026-10-21*
