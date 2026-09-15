# Architecture

**Analysis Date:** 2026-09-15

## Pattern Overview

**Overall:** Layered installer core with a thin CLI and an optional SDL2 adapter

**Key Characteristics:**
- Installer owns policy and side effects; UI never extracts archives or edits XML
- Manifest-driven catalogue with strict JSON decoding and review-status gates
- Transactional filesystem writes behind an allowed-path guard
- SDL2 GUI compiled only with `-tags sdl`; CLI stays static and CGO-free

## Layers

**Manifest and catalogue:**
- Purpose: v1 data contract, semantic validation, deterministic signed index
- Location: `internal/manifest`, `internal/catalog`, `schema/package-manifest-v1.schema.json`, `catalogue/`
- Contains: package structs, review status, SHA-256 of canonical JSON
- Depends on: encoding/json with `DisallowUnknownFields`
- Used by: CLI, installer, appstore service, update checker

**Platform:**
- Purpose: detect firmware, version, arch, device, and resolution; enforce compatibility
- Location: `internal/platform`
- Contains: `/etc/os-release` `OS_NAME`, `/usr/share/knulli/knulli.version`, board files, resolution candidates
- Depends on: `internal/manifest` compatibility fields
- Used by: CLI flags, GUI, installer `Check`

**Safe filesystem and archives:**
- Purpose: bound extraction and destination mutation
- Location: `internal/archive`, `internal/safefs`
- Contains: ZIP/`tar.gz` extractors, symlink rejection, atomic writes, journaled transactions
- Depends on: stdlib archive packages
- Used by: `internal/installer`

**Installer service:**
- Purpose: download, adopt, install, update, repair, uninstall, health, menu XML
- Location: `internal/installer`
- Contains: HTTPS download, ownership state, gamelist rewrite, loopback refresh
- Depends on: manifest, platform, archive, safefs, diagnostics
- Used by: `cmd/knulli-app`, `internal/appstore`

**Application facade:**
- Purpose: expose only actions the current package state allows
- Location: `internal/appstore`
- Contains: `Backend` interface, `Service`, `Item`, `Action`
- Depends on: catalog, installer, platform
- Used by: `internal/ui`, `cmd/knulli-app-ui`

**Presentation:**
- Purpose: controller-native catalogue browser and first-run mapping setup
- Location: `internal/ui`, `internal/input`, `internal/sdlui`
- Contains: pure-Go model, semantic mappings, SDL event/render adapter
- Depends on: `appstore.Backend`; SDL2 only in `sdlui`
- Used by: `cmd/knulli-app-ui`

## Data Flow

**Install / update / repair:**
1. Load and validate manifest; reject candidates
2. `platform.Check` against detected or flagged device matrix
3. HTTPS download with size and SHA-256 (`internal/installer/download.go`)
4. Stage ZIP or `tar.gz` (`internal/archive`)
5. Transaction writes package files, optional `gamelist.xml` entry, backups, installed state (`internal/safefs`)
6. On failure, restore snapshots in reverse order; on success, optionally GET `/reloadgames`

**GUI action:**
1. `appstore.Service.Items` joins catalogue index with installer status
2. `internal/ui.Model` offers only returned actions
3. `sdlui` maps SDL GameController buttons to semantic actions through `internal/input`
4. `Service.Execute` calls `installer.Manager`; UI shows progress, health, restart-required

**State Management:**
- Installed ownership and observed modes live in manager state under `/userdata/system/knulli-app-store`
- Controller mappings are versioned JSON keyed by device and controller identity
- GUI model is in-memory; operations run asynchronously through an event channel

## Key Abstractions

**`manifest.Package`:**
- Purpose: reviewed package contract
- Examples: `internal/manifest/manifest.go`, `catalogue/packages/*.json`
- Pattern: status-gated struct with `Validate()` and `Installable()`

**`safefs.Guard` / `safefs.Transaction`:**
- Purpose: allowed-path resolution and reversible writes
- Examples: `internal/safefs/guard.go`, `internal/safefs/transaction.go`
- Pattern: virtual `/userdata` paths mapped into `-root`

**`appstore.Backend`:**
- Purpose: UI/CLI-neutral catalogue actions
- Examples: `internal/appstore/service.go`
- Pattern: interface with in-memory fakes in UI tests

**`input.Action` / `input.Mapping`:**
- Purpose: semantic controller actions instead of physical A/B
- Examples: `internal/input/mapping.go`, `internal/input/store.go`
- Pattern: per-device/controller atomic JSON store

## Entry Points

**`cmd/knulli-app`:**
- Location: `cmd/knulli-app/main.go`
- Triggers: `validate`, `catalogue`, `install`, `adopt`, `update`, `repair`, `uninstall`
- Responsibilities: flag parsing, diagnostics open, installer calls

**`cmd/knulli-app-ui`:**
- Location: `cmd/knulli-app-ui/main.go` (`sdl` tag)
- Triggers: Ports launcher `packaging/*/Knulli App Store.sh`
- Responsibilities: detect platform, open catalogue index beside the binary, run SDL loop

**`cmd/check-updates`:**
- Location: `cmd/check-updates/main.go`
- Triggers: weekly workflow
- Responsibilities: GitHub release metadata report, no asset download

## Error Handling

**Strategy:** return `error` values; CLI/GUI print and exit or show `Model.Error`

**Patterns:**
- Wrap with `fmt.Errorf("...: %w", err)` at process boundaries
- Compatibility failures include raw, normalized, source, and matrix evidence (`internal/platform`)
- Health issues are structured path/check/expected/actual values (`internal/installer/status.go`)
- Failed transactions roll back; checksum failure writes no package files

## Cross-Cutting Concerns

**Logging:** `internal/diagnostics` UTC logger, secret redaction, 512 KiB cap, one rotated copy

**Validation:** JSON Schema plus Go `Validate()`; unknown JSON fields rejected

**Authentication:** none; trust is catalogue review plus pinned SHA-256

---

*Architecture analysis: 2026-09-15*
