# Codebase Structure

**Analysis Date:** 2026-10-21

## Directory Layout

```text
knulli-app-store/
├── .agents/setup             # Go/SDL development bootstrap
├── .github/workflows/        # Check, release, weekly catalogue report
├── .planning/codebase/       # Seven architecture map documents
├── catalogue/
│   ├── packages/             # Six reviewed manifests
│   └── providers/            # PortMaster external-provider declaration
├── cmd/
│   ├── check-updates/        # Read-only release metadata reporter
│   ├── knulli-app/           # Static CLI, recovery, menu registration
│   └── knulli-app-ui/        # SDL-tagged GUI entry point
├── docs/
│   ├── adr/                  # Nine decision records plus index
│   └── assets/               # Store cover used in device packaging
├── internal/
│   ├── appstore/             # Service, verdicts, synthetic self-update card
│   ├── archive/              # Bounded, interruptible extraction
│   ├── catalog/              # Deterministic index and signatures
│   ├── diagnostics/          # Redacted rotating logs and exports
│   ├── gamelist/             # XML/menu ownership and metadata fill-in
│   ├── input/                # Automatic SDL/keyboard bindings and session
│   ├── installer/            # Lifecycle, Store updates, health, recovery
│   ├── manifest/             # Package contract and semantic policy
│   ├── platform/             # Detection, provenance, runtime compatibility
│   ├── safefs/               # Path guard, atomic writes, journal rollback
│   ├── sdlui/                # SDL rendering plus portable event/walk logic
│   ├── ui/                   # Card browsing, tabs, confirmations, notices
│   ├── updatecheck/          # Maintenance release comparisons/reports
│   └── version/              # Version ordering helpers
├── packaging/               # Per-target Bash launcher and README
├── schema/                  # Manifest v1 JSON Schema
├── scripts/                 # Device build/package and desktop fixture/walk
├── AGENTS.md                # Engineering constraints and commands
├── CONTEXT.md               # Normative domain vocabulary
├── CONTRIBUTING.md
├── Makefile
├── go.mod / go.sum
├── prek.toml
└── renovate.json
```

## Directory Purposes

- **`catalogue/packages/`:** One manifest per reverse-domain ID. Current packages are Grout, Random Game Picker, RetSend, RAOfflineProxy, PocketCurator, and PlayTime. Four are experimental; two are candidates. This count excludes the synthetic Store card (`internal/appstore/selfupdate.go`).
- **`catalogue/providers/`:** External catalogues, not package install sources (`catalogue/providers/portmaster.json`).
- **`internal/`:** Fourteen packages, each with one concern. Large packages split responsibilities across files rather than exporting presentation dependencies (`internal/installer/installer.go`).
- **`docs/`:** Device guides, security model, dated review reports, desktop verification, and ADRs. `docs/adr/0009-advisory-device-compatibility.md` records the advisory device policy.
- **`packaging/`:** Launchers run recovery before loading the GUI, register menu metadata, and select controller letter policy (`packaging/trimui-smart-pro/Knulli App Store.sh`).
- **`scripts/`:** Build/package automation and generated desktop fixtures (`scripts/device-build.sh`, `scripts/package-device.sh`, `scripts/desktop-fixture.sh`, `scripts/desktop-walk.sh`).

## Key File Locations

**Entry Points:** `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`, `cmd/check-updates/main.go`; the SDL loop is `internal/sdlui/run.go`.

**Configuration:** `go.mod`, `Makefile`, `prek.toml`, `.agents/setup`, `.github/workflows/check.yml`, `.github/workflows/release.yml`, `.github/workflows/catalogue-updates.yml`.

**Core Logic:**
- Policy and schema: `internal/manifest/manifest.go`, `schema/package-manifest-v1.schema.json`.
- Compatibility: `internal/platform/platform.go`; typed action gating: `internal/appstore/verdict.go`.
- Package stages and commits: `internal/installer/stage.go`, `internal/installer/commit.go`.
- Store updates: `internal/appstore/selfupdate.go`, `internal/installer/selfupdate.go`.
- Path/recovery safety: `internal/safefs/guard.go`, `internal/safefs/transaction.go`, `internal/safefs/files.go`.
- Menu metadata: `internal/gamelist/menu.go`, `internal/gamelist/xml.go`.
- Cards and input: `internal/ui/model.go`, `internal/input/mapping.go`, `internal/input/session.go`, `internal/sdlui/events.go`.
- Appearance: `internal/sdlui/layout.go`, `internal/sdlui/draw.go`, `internal/sdlui/hints.go`.

**Testing:** Co-located `*_test.go`; real filesystem fixtures under `internal/installer/testdata/` include `gamelist.xml` and `retsend-v1.0.0.sh`. Shared service test factories live in `internal/appstore/test_helpers_test.go`.

## Naming Conventions

**Files:** Lowercase concern names (`selfupdate.go`, `stage.go`); tests add `_test.go`. Shell scripts use hyphenated names. Package manifests use reverse-domain IDs, such as `catalogue/packages/io.github.jellydn.random-game-picker.json`.

**Directories:** Lowercase Go package names under `internal/`; command directory names match binary names. ADR names use a four-digit prefix and hyphenated title (`docs/adr/0009-advisory-device-compatibility.md`).

**Vocabulary:** Follow `CONTEXT.md`: Adopt is shown as Manage existing; game list acceptance is queued, not completed; device scope is advisory; a tested badge needs exact evidence.

## Where to Add New Code

**New package:** Add `catalogue/packages/<id>.json` using `docs/how-to-add-a-package.md`. A candidate must omit actionable metadata. Check with `make catalogue-check`, not a diff on ignored generated output (`Makefile`).

**Lifecycle behavior:** Put side effects beside the owner in `internal/installer/`; expose permitted actions in `internal/appstore/service.go` and `internal/appstore/verdict.go`. Cover changed filesystem behavior with temp-root integration tests.

**Manifest field:** Change `internal/manifest/manifest.go`, `schema/package-manifest-v1.schema.json`, and the consuming policy/staging code together. Preserve legacy installed-state compatibility in `internal/installer/state.go`.

**GUI behavior:** State in `internal/ui/model.go`; portable events in `internal/sdlui/events.go`; layout/rendering in `internal/sdlui/layout.go` and `internal/sdlui/draw.go`; footer hints in `internal/sdlui/hints.go`; walkthrough labels in `internal/sdlui/walk.go`.

**Utilities:** Put helpers with their owner: filesystem in `internal/safefs/files.go`, version comparisons in `internal/version/version.go`. Do not introduce an unrelated generic utility layer.

## Special Directories

- **`build/`, `dist/`:** Generated binaries, catalogue, coverage, fixtures, walkthroughs, and release ZIPs; ignored by `.gitignore`.
- **`.planning/codebase/`:** Tracked architecture map; refresh it against source rather than copying old measurements.
- **`internal/installer/testdata/`:** Tracked test fixtures, not generated build output.

---

*Structure analysis: 2026-10-21*
