# Codebase Structure

**Analysis Date:** 2026-09-19

## Directory Layout

```
knulli-app-store/
├── cmd/                      # Process entry points (thin adapters)
│   ├── knulli-app/           # CGO-free installer CLI
│   ├── knulli-app-ui/        # SDL2 GUI (sdl build tag)
│   └── check-updates/        # Weekly GitHub release metadata checker
├── internal/                 # Importable only inside this module
│   ├── appstore/             # Catalogue facade: Service, Item, verdict
│   ├── archive/              # ZIP and tar.gz staging
│   ├── catalog/              # Deterministic index build/load + ed25519 signing
│   ├── diagnostics/          # Redacted, bounded logs and export
│   ├── gamelist/             # Owned gamelist.xml entry planning and XML edits
│   ├── input/                # Semantic mappings, setup session, keyboard
│   ├── installer/            # Download, lifecycle, state, health, refresh
│   ├── manifest/             # Package contract and validation
│   ├── platform/             # Device/firmware/resolution detection
│   ├── safefs/               # Allowed paths, atomic writes, transactions
│   ├── sdlui/                # SDL adapter, raster UI, walkthrough harness
│   ├── ui/                   # Pure-Go catalogue model, tabs, notices
│   └── updatecheck/          # Read-only GitHub release reports
├── catalogue/
│   ├── packages/             # One JSON manifest per package (five)
│   └── providers/            # External providers (PortMaster)
├── schema/                   # package-manifest-v1 JSON Schema
├── packaging/                # Per-device Ports launcher and README
├── scripts/                  # Device packaging and desktop verification
├── docs/                     # Operator, review, security, ADR docs
├── .agents/setup             # Installs Go 1.27.1 + libsdl2-dev when missing
├── .github/workflows/        # check, release, catalogue-updates
├── .planning/codebase/       # Generated codebase map
├── CONTEXT.md                # Shared domain glossary
├── Makefile                  # fmt, vet, test, build, build-ui, catalogue, gui, walkthrough
└── go.mod                    # module github.com/jellydn/knulli-app-store
```

## Directory Purposes

**cmd:**
- Purpose: flag parsing and wiring only; no product policy
- Contains: `main.go` per command plus `cmd/knulli-app/main_test.go`
- Key files: `cmd/knulli-app/main.go` (223 lines), `cmd/knulli-app-ui/main.go` (97), `cmd/check-updates/main.go`

**internal:**
- Purpose: all product logic, kept out of other modules
- Contains: 13 packages with co-located `_test.go` files
- Key files: `internal/installer/installer.go` (819), `internal/sdlui/draw.go` (730), `internal/platform/platform.go` (531), `internal/input/session.go` (520), `internal/ui/model.go` (417)

**catalogue:**
- Purpose: reviewed data, not code
- Contains: five package manifests and one provider record
- Key files: `catalogue/packages/io.github.unitreign.playtime.json`, `app.romm.grout.json`, `io.github.jellydn.retsend.json`, `io.github.misantronic.raofflineproxy.json`, `io.github.tomtombombadil.pocketcurator.json`, `catalogue/providers/portmaster.json`

**schema:**
- Purpose: the machine-readable half of the manifest contract
- Contains: `schema/package-manifest-v1.schema.json`, Draft 2020-12 with `additionalProperties: false` and conditional `allOf` rules per review status

**packaging:**
- Purpose: per-device Ports launcher and operator README copied into the release archive
- Contains: `packaging/trimui-smart-pro/`, `packaging/magicx-zero-28/`

**scripts:**
- Purpose: shell entry points for packaging and device-free verification
- Contains: `package-device.sh` (build the device ZIP), `desktop-fixture.sh` (write a scratch Knulli root), `desktop-walk.sh` (walk GUI flows and record evidence)

**docs:**
- Purpose: operator, review, and design documentation
- Contains: `docs/architecture.md`, `docs/security-model.md`, `docs/desktop-verification.md`, `docs/real-device-tests.md`, `docs/how-to-add-a-package.md`, device guides, dated catalogue reviews, and `docs/adr/`

## Key File Locations

**Entry Points:**
- `cmd/knulli-app/main.go`: installer CLI (`validate`, `catalogue`, `install`, `adopt`, `update`, `repair`, `uninstall`)
- `cmd/knulli-app-ui/main.go`: device GUI, `//go:build sdl`
- `cmd/check-updates/main.go`: weekly metadata report
- `packaging/*/Knulli App Store.sh`: Ports launchers

**Configuration:**
- `go.mod`, `go.sum`, `Makefile`, `renovate.json`, `.gitignore`
- `schema/package-manifest-v1.schema.json`
- `.github/workflows/check.yml`, `.github/workflows/release.yml`, `.github/workflows/catalogue-updates.yml`
- `.agents/setup`

**Core Logic:**
- `internal/manifest/manifest.go` (390) and `internal/manifest/load.go`
- `internal/catalog/catalog.go` (148) and `internal/catalog/sign.go` (166)
- `internal/platform/platform.go` (531)
- `internal/safefs/guard.go` (97), `internal/safefs/transaction.go` (330), `internal/safefs/files.go` (110)
- `internal/archive/archive.go` (183)
- `internal/installer/installer.go`, `lifecycle.go`, `state.go`, `status.go`, `refresh.go`, `download.go`
- `internal/gamelist/menu.go`, `internal/gamelist/xml.go`
- `internal/appstore/service.go` (352), `internal/appstore/verdict.go` (163)
- `internal/ui/model.go`, `tabs.go`, `toast.go`
- `internal/input/mapping.go`, `session.go`, `store.go`, `calibration.go`, `footer.go`, `keyboard.go`
- `internal/sdlui/run.go`, `draw.go`, `events.go`, `layout.go`, `hints.go`, `walk.go`, `repeat.go`, `input_mode.go`
- `internal/updatecheck/check.go` (305), `internal/updatecheck/report.go`

**Testing:**
- Co-located `*_test.go` under every `internal/` package and `cmd/knulli-app/`
- `internal/installer/installer_test.go` (1,421) is the lifecycle suite; `internal/installer/testdata/gamelist.xml` is its XML fixture
- `internal/appstore/test_helpers_test.go` holds shared catalogue fixtures
- `internal/input/physical_button_audit_test.go` is the source audit
- `internal/sdlui/draw_test.go` (788) renders and optionally screenshots every screen state

## Naming Conventions

**Files:**
- Package directory matches the Go package name: `internal/safefs`, `internal/gamelist`
- One concept per file: `guard.go`, `download.go`, `refresh.go`, `tabs.go`, `toast.go`, `repeat.go`
- Tests: `*_test.go`; shared fixtures in `test_helpers_test.go`; audits named for their subject
- Manifests: reverse-domain id + `.json`, for example `io.github.unitreign.playtime.json`
- ADRs: `NNNN-kebab-title.md` under `docs/adr/`
- Dated evidence: `docs/catalogue-review-YYYY-MM-DD.md`

**Directories:**
- lowercase product nouns (`installer`, `catalog`, `safefs`, `sdlui`)
- Device packaging folders use device ids: `trimui-smart-pro`, `magicx-zero-28`
- `internal/` for everything not meant to be imported elsewhere

**Go identifiers:**
- Exported verbs for the pipeline: `Load`, `Validate`, `Build`, `Verify`, `Extract`, `Resolve`, `Check`, `Apply`, `Stage`, `Commit`
- Unexported helpers stay local and short: `assess`, `plan`, `wrap`, `blend`, `shorten`
- String-typed enums: `manifest.Package.Review.Status`, `appstore.Action`, `appstore.State`, `installer.Op`, `input.Action`, `ui.Tab`, `sdlui.Key`

## Where to Add New Code

**New lifecycle behavior:**
- Implementation: `internal/installer/installer.go` plus the smallest helper file that owns it
- Required: a temporary-root integration test in `internal/installer/installer_test.go` (`CONTRIBUTING.md`)

**New archive or filesystem behavior:**
- Implementation: `internal/archive` or `internal/safefs`
- Required: an adversarial fixture where an unsafe implementation would produce a different result

**New catalogue policy:**
- Implementation: `internal/manifest/manifest.go` plus the matching rule in `schema/package-manifest-v1.schema.json`; the two must agree
- Visible effect: `internal/appstore/verdict.go` `assess` is the single place a new state or action is surfaced

**New GUI screen or control:**
- Model behavior: `internal/ui` (pure Go, testable with a fake backend)
- Input semantics: `internal/input` (actions, mappings, footer verbs)
- Rendering: `internal/sdlui/draw.go` plus geometry tokens in `internal/sdlui/layout.go`
- Evidence: add a flow to `scripts/desktop-walk.sh` and a case in `internal/sdlui/draw_test.go`

**New package in the store:**
- One file in `catalogue/packages/` plus review evidence in the pull request (`docs/how-to-add-a-package.md`, `CONTRIBUTING.md`)
- Then regenerate the index and keep `git diff --exit-code -- build/catalog-index.json` clean

**Utilities:**
- Shared helpers stay in the owning package. There is no global `utils` package.

## Special Directories

**.planning/codebase:**
- Purpose: generated stack/architecture/quality map
- Generated: Yes (by the `codemap` skill)
- Committed: Yes (`.gitignore` lists only `/build/`, `/dist/`, `/.amp/`)

**build/ and dist/:**
- Purpose: local binaries, the catalogue index, walkthrough evidence, and device ZIPs
- Generated: Yes
- Committed: No

**catalogue/:**
- Purpose: reviewed source of truth for packages and providers
- Generated: No
- Committed: Yes

**internal/installer/testdata/:**
- Purpose: XML fixture for menu ownership tests
- Generated: No
- Committed: Yes

**docs/adr/:**
- Purpose: accepted architecture decisions in Context / Decision / Consequences form
- Generated: No
- Committed: Yes; the index lives in `docs/adr/README.md`

---

*Structure analysis: 2026-09-19*
