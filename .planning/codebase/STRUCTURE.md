# Codebase Structure

**Analysis Date:** 2026-09-15

## Directory Layout

```
knulli-app-store/
├── cmd/                      # Process entry points
│   ├── knulli-app/           # CGO-free installer CLI
│   ├── knulli-app-ui/        # SDL2 GUI (sdl build tag)
│   └── check-updates/        # Weekly GitHub release metadata checker
├── internal/                 # Importable only inside this module
│   ├── appstore/             # Catalogue + installer facade for UI
│   ├── archive/              # ZIP and tar.gz staging
│   ├── catalog/              # Deterministic index build/load
│   ├── diagnostics/          # Redacted bounded logs and export
│   ├── input/                # Semantic controller mappings
│   ├── installer/            # Download, lifecycle, gamelist, state
│   ├── manifest/             # Package contract and validation
│   ├── platform/             # Device/firmware/resolution detection
│   ├── safefs/               # Allowed paths, atomic writes, transactions
│   ├── sdlui/                # SDL adapter and raster UI
│   ├── ui/                   # Pure-Go catalogue state machine
│   └── updatecheck/          # Read-only GitHub release reports
├── catalogue/
│   ├── packages/             # One JSON manifest per package
│   └── providers/            # External providers (PortMaster)
├── schema/                   # package-manifest-v1 JSON Schema
├── packaging/                # Per-device Ports launcher and README
├── scripts/                  # Device ZIP packaging
├── docs/                     # Operator, review, security, ADR docs
├── .github/workflows/        # check and catalogue-updates
├── .planning/codebase/       # Generated codebase map
├── Makefile                  # fmt, vet, test, build, catalogue
└── go.mod                    # module github.com/jellydn/knulli-app-store
```

## Directory Purposes

**cmd:**
- Purpose: thin main packages only
- Contains: flag parsing and wiring
- Key files: `cmd/knulli-app/main.go`, `cmd/knulli-app-ui/main.go`, `cmd/check-updates/main.go`

**internal:**
- Purpose: all product logic, kept out of other modules
- Contains: Go packages with co-located `_test.go` files
- Key files: `internal/installer/installer.go`, `internal/appstore/service.go`, `internal/sdlui/run.go`

**catalogue:**
- Purpose: reviewed data, not code
- Contains: package manifests and provider records
- Key files: `catalogue/packages/*.json`, `catalogue/providers/portmaster.json`

**docs:**
- Purpose: operator and contributor documentation
- Contains: architecture, security, device guides, ADRs
- Key files: `docs/architecture.md`, `docs/security-model.md`, `docs/adr/`

## Key File Locations

**Entry Points:**
- `cmd/knulli-app/main.go`: installer CLI
- `cmd/knulli-app-ui/main.go`: device GUI
- `cmd/check-updates/main.go`: weekly metadata report
- `packaging/*/Knulli App Store.sh`: Ports launchers

**Configuration:**
- `go.mod`, `Makefile`, `renovate.json`, `.gitignore`
- `schema/package-manifest-v1.schema.json`
- `.github/workflows/check.yml`, `.github/workflows/catalogue-updates.yml`

**Core Logic:**
- `internal/manifest/manifest.go`
- `internal/installer/installer.go`
- `internal/safefs/guard.go`, `internal/safefs/transaction.go`
- `internal/platform/platform.go`
- `internal/appstore/service.go`

**Testing:**
- Co-located `*_test.go` under each `internal/` package
- `internal/installer/testdata/gamelist.xml`
- SDL layout tests in `internal/sdlui/draw_test.go`

## Naming Conventions

**Files:**
- Package directory matches Go package name: `internal/safefs`
- Tests: `*_test.go`; helpers: `test_helpers_test.go`
- Manifests: reverse-domain id + `.json`, e.g. `io.github.unitreign.playtime.json`
- ADRs: `NNNN-kebab-title.md`

**Directories:**
- lowercase product nouns (`installer`, `catalog`, `sdlui`)
- Device packaging folders use device ids: `trimui-smart-pro`, `magicx-zero-28`

## Where to Add New Code

**New Feature:**
- Primary code: new or existing package under `internal/`
- Tests: co-located `*_test.go` with a temporary-root integration test if files change

**New Component/Module:**
- Implementation: `internal/<name>/`; wire a thin command only if a new process is required

**Utilities:**
- Shared helpers stay in the owning package; no global `utils` package

**New package in the store:**
- One file in `catalogue/packages/` plus evidence in the pull request (`docs/how-to-add-a-package.md`)

## Special Directories

**.planning/codebase:**
- Purpose: generated stack/architecture/quality map
- Generated: Yes (by /codemap)
- Committed: Yes

**build/ and dist/:**
- Purpose: local binaries, catalogue index, device ZIPs
- Generated: Yes
- Committed: No (`.gitignore`)

**catalogue/:**
- Purpose: reviewed source of truth for packages and providers
- Generated: No
- Committed: Yes

---

*Structure analysis: 2026-09-15*
