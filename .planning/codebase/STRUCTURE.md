# Codebase Structure

**Analysis Date:** 2026-09-21

## Directory Layout

```
knulli-app-store/
├── .agents/setup              # Cloud-agent bootstrap: pinned Go 1.27.1 (SHA-256 checked) + libsdl2-dev
├── .github/workflows/         # check.yml, release.yml, catalogue-updates.yml
├── catalogue/
│   ├── packages/              # One reviewed manifest per package (5 files)
│   └── providers/             # External provider declarations (portmaster.json)
├── cmd/
│   ├── check-updates/         # Read-only upstream release metadata reporter
│   ├── knulli-app/            # Static CGO-free CLI
│   └── knulli-app-ui/         # SDL2 GUI entry point (sdl build tag)
├── docs/
│   ├── adr/                   # 8 accepted architecture decision records
│   └── *.md                   # Device guides, security model, review notes
├── internal/
│   ├── appstore/              # Backend interface + typed catalogue verdict
│   ├── archive/               # Bounded ZIP / tar.gz extraction
│   ├── catalog/               # Deterministic index, ed25519 signing
│   ├── diagnostics/           # Redacted, bounded, rotating event log
│   ├── gamelist/              # EmulationStation menu ownership
│   ├── input/                 # Semantic actions, mappings, session, footer
│   ├── installer/             # The only owner of side effects
│   ├── manifest/              # v1 contract + semantic policy
│   ├── platform/              # Detection, evidence, compatibility checks
│   ├── safefs/                # Path guard, transaction/journal, atomic writes
│   ├── sdlui/                 # SDL2 event/render adapter (mostly sdl-tagged)
│   ├── ui/                    # Pure-Go catalogue state machine
│   └── version/               # Every version comparison
├── packaging/
│   ├── magicx-zero-28/        # "Knulli App Store.sh" launcher + README.txt
│   └── trimui-smart-pro/      # "Knulli App Store.sh" launcher + README.txt
├── schema/package-manifest-v1.schema.json
├── scripts/                   # device-build, package-device, desktop-fixture, desktop-walk
├── AGENTS.md                  # Agent-facing constraints and commands
├── CONTEXT.md                 # Normative domain glossary
├── CONTRIBUTING.md
├── Makefile
├── prek.toml
├── go.mod / go.sum
└── renovate.json
```

## Directory Purposes

**`catalogue/packages/`:**
- Purpose: The reviewed package manifests — the product of the project.
- Contains: One JSON file per package, named `<reverse-domain-id>.json`. Currently `app.romm.grout.json`, `io.github.jellydn.retsend.json`, `io.github.misantronic.raofflineproxy.json`, `io.github.tomtombombadil.pocketcurator.json`, `io.github.unitreign.playtime.json`.
- Key files: `io.github.unitreign.playtime.json` is the richest example (release, compatibility, install, review evidence).

**`catalogue/providers/`:**
- Purpose: Declare catalogues this project features but does not own, mirror, or install from.
- Contains: `portmaster.json` (ADR-0005).

**`internal/`:**
- Purpose: All production logic. Nothing here is `main`; nothing here imports a `cmd/` package.
- Split rule: one package per concern, and within a large package one file per concern (see `internal/installer/`).

**`docs/adr/`:**
- Purpose: Accepted decisions with Context / Decision / Consequences, indexed by `docs/adr/README.md`.

**`packaging/`:**
- Purpose: Per-device launcher scripts and end-user README text that ship inside the release zip.
- Contains: `Knulli App Store.sh` (POSIX sh launcher) and `README.txt` per device.

**`scripts/`:**
- Purpose: The shell side of the build and verification story.
- Key files: `device-build.sh` (the single home of the signed cross-compile and ABI gate), `package-device.sh`, `desktop-fixture.sh` (writes Knulli fixture files into a scratch root), `desktop-walk.sh` (scripted keyboard walkthrough with evidence).

**`schema/`:**
- Purpose: The JSON Schema that describes the manifest shape. Semantic rules live in `internal/manifest/manifest.go`, not here.

**`build/` and `dist/`:**
- Purpose: Generated output. **Gitignored** — never committed.

## Key File Locations

**Entry Points:**
- `cmd/knulli-app/main.go`: CLI (`validate`, `catalogue`, `install`, `adopt`, `update`, `repair`, `uninstall`); `run` dispatches, `runApply`/`runUninstall` take an `applyDeps` seam so a test can supply a loopback release server and refresh address
- `cmd/knulli-app-ui/main.go`: GUI (`//go:build sdl`); flags for catalogue, root, platform overrides, screenshots, walkthroughs
- `cmd/check-updates/main.go`: weekly read-only release metadata report
- `internal/sdlui/run.go`: the SDL main loop (`Run`)
- `internal/installer/installer.go`: `Manager.Apply` — the lifecycle entry point

**Configuration:**
- `Makefile`: every developer/CI task
- `prek.toml`: local git hooks mirroring CI
- `.github/workflows/check.yml`: the CI gate (`test`, `gui-walkthrough`, `device-artifact`)
- `.github/workflows/release.yml`: pre-release and tag release publishing
- `go.mod`: module path and the single direct dependency
- `renovate.json`: dependency update policy

**Core Logic:**
- `internal/manifest/manifest.go`: contract + policy (414 lines, the largest single policy file)
- `internal/installer/stage.go`: acquire, lock, recover, download, extract
- `internal/installer/commit.go`: the transaction that commits a release
- `internal/installer/status.go`: health checks and pre-existence detection
- `internal/safefs/guard.go`: path policy; `internal/safefs/transaction.go`: journal + rollback
- `internal/archive/archive.go`: adversarial-path-resistant extraction
- `internal/platform/platform.go`: detection, evidence, `Check`
- `internal/appstore/verdict.go`: the single home of the catalogue verdict
- `internal/ui/model.go`: catalogue state machine
- `internal/input/mapping.go` / `session.go`: semantic actions and the setup mode machine
- `internal/sdlui/draw.go` (730 lines) and `layout.go`: rendering and all geometry tokens

**Testing:**
- Co-located `*_test.go` beside every production file's package. 35 test files.
- `internal/installer/testdata/gamelist.xml`: the only committed fixture directory
- `internal/installer/installer_test.go` (1,421 lines): the largest test file, integration over temp roots
- `internal/sdlui/draw_test.go` (788 lines): layout/render assertions
- `internal/appstore/test_helpers_test.go`: shared fake-backend/manager helpers

## Naming Conventions

**Files:**
- Production: short lowercase noun for the concern — `guard.go`, `transaction.go`, `commit.go`, `verdict.go`, `toast.go`. A package's orchestrator often carries the package name (`installer.go`, `catalog.go`, `platform.go`) while siblings name the concern.
- Tests: `<file>_test.go` in the same package (internal tests, not `_test` packages), except `internal/appstore/test_helpers_test.go` for shared helpers.
- Shell: lowercase-hyphenated verbs — `device-build.sh`, `package-device.sh`, `desktop-walk.sh`.
- Docs: lowercase-hyphenated with a date when the content is a snapshot — `catalogue-review-2026-09-15.md`, `catalogue-review-2026-09-17.md`.
- ADRs: `NNNN-title-in-kebab-case.md`.

**Directories:**
- Go packages: single lowercase word, never pluralized, never abbreviated — `internal/safefs`, `internal/gamelist`, `internal/diagnostics`.
- Manifests: reverse-domain id as the filename, so the file and the `id` field cannot disagree.
- Exceptions: `cmd/knulli-app` and `cmd/knulli-app-ui` use hyphens because they are binary names.

**Identifiers (from `CONTEXT.md`, normative):**
- Domain vocabulary is capitalized consistently: Adopt / Manage existing, external installation, managed / preserved / unmanaged file, installed state, outcome, notice, menu ownership, game list refresh, semantic action, binding, footer hint, fixture root, walkthrough.
- Schema constants are `org.knulli.app-store/<thing>/v1` strings, each declared next to the type it versions (`manifest.SchemaV1`, `catalog.IndexSchemaV1`, `catalog.SignatureSchemaV1`, `installer.lifecycleSchemaV1`, `input.Schema`, `safefs.journalSchemaV1`).
- Operation names are lowercase strings used both as `installer.Op` values and as the `operation` parameter threaded through `stage`/`commit`/`rollback`.

## Where to Add New Code

**New package manifest (the most common contribution):**
- Add one file: `catalogue/packages/<reverse-domain-id>.json`
- Follow `docs/how-to-add-a-package.md`; start at `review.status: "candidate"` with no `release`/`compatibility`/`install` fields (the validator rejects candidate files that carry them)
- Regenerate the index and confirm it is unchanged-clean: `make catalogue && git diff --exit-code -- build/catalog-index.json`

**New lifecycle operation:**
- Primary code: a new `installer.Op` in `internal/installer/installer.go`, a branch in `Apply`, reconciliation in `internal/installer/stage.go`, and the committed effect in `internal/installer/commit.go`
- Expose it in: `internal/appstore/service.go` (`Action`, `Execute` switch), then `internal/appstore/verdict.go` (`assess` + `validRetryAction`) so the UI can offer it
- Tests: `internal/installer/installer_test.go` over a temp root, plus `internal/appstore/service_test.go` for the action gating

**New compatibility field:**
- Contract: `internal/manifest/manifest.go` (`Compatibility` struct + `validateInstallable`)
- Schema: `schema/package-manifest-v1.schema.json`
- Enforcement: `internal/platform/platform.go` (`Check`)
- Expose to UI: the reason text flows through `appstore.assess` automatically

**New GUI screen:**
- State machine: `internal/ui/model.go` (`Focus` + `Select`/`Back`) or a new mode in `internal/input/session.go`
- Layout tokens: `internal/sdlui/layout.go` (never hardcode pixel numbers in `draw.go`)
- Rendering: `internal/sdlui/draw.go`
- Footer: hints in `internal/sdlui/hints.go` using `input.Hint`/`Verb`
- Screen label: `internal/sdlui/walk.go` (`WalkState`) so the walkthrough can assert it
- Tests: `internal/sdlui/layout_test.go`, `draw_test.go`, `walk_test.go`

**New outbound network call:**
- Put the client, timeout, redirect policy, and any size limit next to the caller (`internal/installer/download.go`, `internal/installer/refresh.go`, `internal/updatecheck/check.go`)
- Injectable for tests via a `*http.Client` field on the owning struct (`Manager.Client`, `Manager.RefreshClient`, `Checker.Client`) or a base URL override (`Manager.RefreshURL`, `Checker.BaseURL`)

**New persisted record:**
- Declare its own `<name>SchemaV1` constant, write it atomically through `safefs.AtomicWrite`, and validate the schema and owner id on load — see `internal/installer/lifecycle.go` for the smallest complete example.

**Utilities:**
- Path and file helpers: `internal/safefs/files.go`
- Version ordering: `internal/version/version.go` (do not add a second comparison)
- Do not create a generic `internal/util` package; each helper goes to the package that owns the concern.

## Special Directories

**`build/`:**
- Purpose: compiled binaries, `catalog-index.json`, scratch fixture roots, walkthrough evidence
- Generated: Yes
- Committed: No (`.gitignore`)

**`dist/`:**
- Purpose: device release zips and `SHA256SUMS.txt`
- Generated: Yes
- Committed: No (`.gitignore`)

**`.planning/codebase/`:**
- Purpose: this codebase map
- Generated: Yes
- Committed: Yes (tracked in git)

**`internal/installer/testdata/`:**
- Purpose: the committed `gamelist.xml` fixture
- Generated: No
- Committed: Yes

---

*Structure analysis: 2026-09-21*
