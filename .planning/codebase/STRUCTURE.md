# Codebase Structure

**Analysis Date:** 2026-10-25

## Directory Layout

```text
knulli-app-store/
├── .agents/setup             # Orb development setup
├── .github/workflows/        # Checks, releases, catalogue update reports
├── .planning/codebase/       # Seven codebase map documents
├── catalogue/
│   ├── packages/             # Reviewed package manifests
│   └── providers/            # External provider metadata
├── cmd/
│   ├── check-updates/        # Read-only upstream release reporter
│   ├── knulli-app/           # Static CLI and recovery entry point
│   └── knulli-app-ui/        # SDL-tagged graphical entry point
├── docs/
│   ├── adr/                  # Architecture decisions and index
│   └── assets/               # Cover and icon sources/images
├── internal/
│   ├── appstore/             # Service, verdicts, synthetic Store card
│   ├── archive/              # Bounded ZIP/tar.gz extraction
│   ├── catalog/              # Deterministic index and signatures
│   ├── diagnostics/          # Redacted logs and exports
│   ├── gamelist/             # Menu ownership and XML updates
│   ├── input/                # Semantic controls and connection state
│   ├── installer/            # Lifecycle, health, recovery, Store updates
│   ├── manifest/             # Package contract and policy
│   ├── platform/             # Detection evidence and compatibility
│   ├── safefs/               # Path guards, atomic writes, journals
│   ├── sdlui/                # Gabagool/SDL adapter and Library drawing
│   ├── ui/                   # Portable model, tabs, notices
│   ├── updatecheck/          # Upstream metadata comparison/reports
│   └── version/              # Version comparison helpers
├── packaging/               # Per-device launcher and README
├── schema/                  # Manifest JSON Schema
├── scripts/                 # Device builds, packaging, desktop evidence
├── AGENTS.md                # Engineering rules and check commands
├── CONTEXT.md               # Normative domain vocabulary
├── CONTRIBUTING.md          # Contribution guide
├── README.md                # Project entry document
├── Makefile                 # Build, validation, coverage, GUI commands
├── go.mod / go.sum           # Go module and pinned dependencies
├── prek.toml                # Local commit/push checks
└── renovate.json            # Dependency update configuration
```

## Directory Purposes

**`catalogue/`:**
- Purpose: Store reviewed package and external provider metadata.
- Contains: Seven package JSON files. All seven currently have experimental status. The synthetic Store card is not a manifest (`catalogue/packages/`, `internal/appstore/selfupdate.go`).
- The [owner's 2026-10-25 confirmation](https://ampcode.com/threads/T-01a17039-648c-7024-abe9-8785444a2b5b) (Asia/Singapore) supplies passed package evidence for the six unchanged current versions and historical Picker 0.2.0 on TrimUI Smart Pro only, Knulli AArch64 at 1280×720. Current Random Game Picker is 0.2.1 and UNTESTED on all matrices; its published maintenance release adds no hardware evidence. Review statuses remain experimental; MagicX and other matrices remain untested. Older exact-version reports do not establish separate lifecycle or frontend results (`catalogue/packages/io.github.jellydn.random-game-picker.json`, `CONTEXT.md`, `docs/real-device-tests.md`).
- Key files: `catalogue/packages/org.retroshelf.knulli.json`, `catalogue/packages/app.romm.grout.json`, `catalogue/providers/portmaster.json`.
- PortMaster is an external provider, not a Store-owned install source (`catalogue/providers/portmaster.json`, `CONTEXT.md`).

**`cmd/`:**
- Purpose: Supply three executable entry points with little policy of their own.
- Contains: CLI argument handling, graphical startup, and maintenance report setup.
- Key files: `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`, `cmd/check-updates/main.go`.

**`internal/`:**
- Purpose: Keep application packages private to this module.
- Contains: Fourteen Go package directories. Backend packages do not import the graphical toolkit; the optional SDL adapter does (`internal/installer/installer.go`, `internal/sdlui/run.go`).
- Key files: `internal/appstore/service.go`, `internal/installer/installer.go`, `internal/ui/model.go`.

**`docs/`:**
- Purpose: Record architecture, security, review evidence, device use, and verification steps.
- Contains: Ten numbered ADRs, their index, dated catalogue reviews, package-review guidance, device guides, and image assets (`docs/adr/`, `docs/assets/`).
- Key files: `docs/architecture.md`, `docs/security-model.md`, `docs/desktop-verification.md`, `docs/adr/0010-gabagool-knulli-frontend.md`.

**`packaging/`:**
- Purpose: Supply target-specific launchers and package instructions.
- Contains: TrimUI Smart Pro and MagicX Zero 28 directories. Each has a Bash launcher and `README.txt` (`packaging/trimui-smart-pro/`, `packaging/magicx-zero-28/`).
- Key files: `packaging/trimui-smart-pro/Knulli App Store.sh`, `packaging/magicx-zero-28/Knulli App Store.sh`.
- Launchers run CLI recovery before GUI startup, attempt menu registration, and select the controller letter policy from the Ports Xbox-layout flag.

**`scripts/`:**
- Purpose: Build target artifacts and run disposable desktop fixtures.
- Contains: Device cross-build/ABI checks, seven-file ZIP packaging, fixture setup, and walkthrough execution.
- Key files: `scripts/device-build.sh`, `scripts/package-device.sh`, `scripts/desktop-fixture.sh`, `scripts/desktop-walk.sh`.

## Key File Locations

**Entry Points:**
- `cmd/knulli-app/main.go`: Validation, catalogue build/sign, lifecycle commands, recovery, and menu registration.
- `cmd/knulli-app-ui/main.go`: SDL-only flags, diagnostics, platform/service setup, and canonical-path self-update gate.
- `cmd/check-updates/main.go`: Upstream metadata reports without manifest or asset changes.
- `internal/sdlui/run.go`: Gabagool window/renderer lifetime, SDL event loop, display evidence, texture upload, and backbuffer capture.

**Configuration:**
- `go.mod`, `go.sum`: Go 1.27 module and pinned dependencies, including gabagool/v2 and go-sdl2.
- `Makefile`: Local checks, static CLI, catalogue determinism, coverage, GUI, and walkthrough targets.
- `prek.toml`: Local pre-commit and pre-push checks.
- `.agents/setup`: Orb setup commands.
- `.github/workflows/check.yml`: CI checks and verification.
- `.github/workflows/release.yml`: Device release builds.
- `.github/workflows/catalogue-updates.yml`: Scheduled upstream metadata report.
- `schema/package-manifest-v1.schema.json`: Manifest shape; runtime policy is also enforced in Go.

**Core Logic:**
- `internal/manifest/manifest.go`, `internal/manifest/load.go`: Package types, semantic policy, strict JSON decoding.
- `internal/catalog/catalog.go`, `internal/catalog/sign.go`: Sorted catalogue index, package digests, and Ed25519 verification.
- `internal/platform/platform.go`: Detection with provenance, resolution candidates, runtime gates, and advisory device scope.
- `internal/appstore/service.go`, `internal/appstore/verdict.go`: Backend interface, item state, and permitted actions.
- `internal/installer/installer.go`, `internal/installer/stage.go`, `internal/installer/commit.go`: Preconditions, lock/recovery, verified staging, and transaction commit.
- `internal/installer/adoption.go`, `internal/installer/recovery.go`: Existing-file inventory and force-reinstall backups.
- `internal/installer/uninstall.go`: Read-only uninstall checks, originals restoration, and owned-file removal.
- `internal/installer/state.go`, `internal/installer/status.go`, `internal/installer/cache.go`: Installed records, full health checks, and record-keyed cache.
- `internal/installer/lifecycle.go`: Failed-operation and retry context.
- `internal/appstore/selfupdate.go`, `internal/installer/selfupdate.go`: Synthetic Store card, official release check, exact-inventory replacement, startup recovery, and Store menu registration.
- `internal/archive/archive.go`: Interruptible bounded extraction, including reviewed embedded base64 tar.gz payloads.
- `internal/safefs/guard.go`, `internal/safefs/files.go`, `internal/safefs/transaction.go`: Allowed paths, symlink refusal, atomic replacement, snapshots, rollback, and startup recovery.
- `internal/gamelist/menu.go`, `internal/gamelist/xml.go`: Exact-path menu merge and ownership.
- `internal/installer/refresh.go`: Loopback game list refresh and committed outcomes.
- `internal/ui/model.go`, `internal/ui/tabs.go`, `internal/ui/toast.go`: Screens, asynchronous operations, filters, and notices.
- `internal/input/mapping.go`, `internal/input/session.go`, `internal/input/keyboard.go`, `internal/input/footer.go`: Semantic bindings, source state, fixed keyboard controls, and shared labels.
- `internal/sdlui/events.go`, `internal/sdlui/repeat.go`: Portable event routing and held-navigation repeat.
- `internal/sdlui/layout.go`, `internal/sdlui/draw.go`, `internal/sdlui/hints.go`: Shared responsive Library geometry, software-rasterized OpenType text, local Store mark, and contextual footer glyphs.
- `internal/sdlui/assets/DejaVuSans.ttf`, `internal/sdlui/assets/DejaVuSans-LICENSE.txt`: Embedded font and its license; drawing uses 18/26 logical px body/title faces.
- `internal/sdlui/walk.go`, `internal/sdlui/input_mode.go`: Scripted verification and input-source selection.
- `internal/diagnostics/log.go`: Bounded redacted logs and exports.
- `internal/updatecheck/check.go`, `internal/updatecheck/report.go`: Release metadata comparison and review output.

**Testing:**
- `internal/installer/installer_test.go`, `internal/installer/commit_test.go`, `internal/installer/selfupdate_test.go`: Temp-root lifecycle, failure, and Store replacement tests.
- `internal/installer/testdata/`: Tracked installer fixtures, including `gamelist.xml` and `retsend-v1.0.0.sh`.
- `internal/archive/archive_test.go`, `internal/safefs/guard_test.go`, `internal/safefs/files_test.go`: Unsafe archive/path and transaction behavior.
- `internal/appstore/test_helpers_test.go`: Shared application test setup.
- `internal/ui/model_test.go`: Fake-backend state and operation tests.
- `internal/sdlui/run_test.go`, `internal/sdlui/draw_test.go`: SDL-tagged runtime and render tests.
- `internal/sdlui/events_test.go`, `internal/sdlui/walk_test.go`: Portable event and walkthrough tests.

## Naming Conventions

**Files:**
- Lowercase concern names identify Go files. Tests add `_test.go`: `internal/installer/selfupdate.go`, `internal/installer/selfupdate_test.go`.
- Command entry files use `main.go`: `cmd/check-updates/main.go`.
- Package JSON filenames use reverse-domain IDs: `catalogue/packages/io.github.jellydn.random-game-picker.json`.
- Shell automation uses hyphenated names: `scripts/device-build.sh`.
- ADR files use four digits and a hyphenated title: `docs/adr/0010-gabagool-knulli-frontend.md`.

**Directories:**
- Go packages use short lowercase names under `internal/`: `internal/safefs/`, `internal/appstore/`.
- Command directories match executable names: `cmd/knulli-app/`, `cmd/knulli-app-ui/`.
- Target directories use reviewed device identifiers: `packaging/trimui-smart-pro/`, `packaging/magicx-zero-28/`.
- Domain terms follow `CONTEXT.md`: Adopt is shown as Manage existing; external installations count files, not directories; a refresh accepted means queued, not completed.

## Where to Add New Code

**New Feature:**
- Primary code: Put lifecycle side effects in `internal/installer/`; expose actions through `internal/appstore/service.go` and `internal/appstore/verdict.go`.
- Tests: Add temp-root integration cases beside the installer code. Use the fake backend in `internal/ui/model_test.go` for presentation changes.
- New package metadata: Add `catalogue/packages/<reverse-domain-id>.json` under the review rules in `docs/how-to-add-a-package.md`. Validate deterministic output with `make catalogue-check` (`Makefile`).
- New manifest field: Change `internal/manifest/manifest.go`, `schema/package-manifest-v1.schema.json`, and the consuming policy code. Check stored-state compatibility in `internal/installer/state.go`.

**New Component/Module:**
- Implementation: Keep model transitions in `internal/ui/`, semantic controls in `internal/input/`, portable event routing in `internal/sdlui/events.go`, and toolkit lifetime in `internal/sdlui/run.go`.
- Drawing: Use `internal/sdlui/draw.go` and `internal/sdlui/layout.go`; keep the Option B Library list with expanded selection, shared Random Game Picker 0.2.0 responsive 480-high canvas, and software fallback (`AGENTS.md`, `docs/adr/0010-gabagool-knulli-frontend.md`).
- Firmware support: Follow `docs/adr/0010-gabagool-knulli-frontend.md`. No generic multi-firmware framework or second firmware implementation exists.

**Utilities:**
- Shared helpers: Put path and atomic-write helpers in `internal/safefs/`; version helpers in `internal/version/version.go`.
- Keep installer policy independent of presentation, as required by `AGENTS.md`.

## Special Directories

**`build/` and `dist/`:**
- Purpose: Generated binaries, catalogue indexes, fixture roots, verification output, reports, and release packages (`Makefile`, `scripts/device-build.sh`, `scripts/desktop-walk.sh`).
- Generated: Yes.
- Committed: No; both paths are ignored (`.gitignore`). An ignored catalogue diff is not a determinism check; use `make catalogue-check` (`AGENTS.md`).

**`.planning/codebase/`:**
- Purpose: Seven source-based maps of architecture, structure, stack, integrations, conventions, testing, and concerns.
- Generated: Written by analysis; not runtime or compiler output.
- Committed: Yes; these are repository documentation files (`.planning/codebase/ARCHITECTURE.md`, `.planning/codebase/STRUCTURE.md`).

**`internal/installer/testdata/`:**
- Purpose: Stable filesystem and release fixtures for installer tests (`internal/installer/installer_test.go`).
- Generated: No; checked-in test inputs.
- Committed: Yes.

**`.amp/`:**
- Purpose: Orb-local tooling and review artifacts, not application source (`.gitignore`, `AGENTS.md`).
- Generated: Local tooling output.
- Committed: No; the directory is ignored.

---

*Structure analysis: 2026-10-25*
