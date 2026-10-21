# Testing Patterns

**Analysis Date:** 2026-10-21

## Test Framework

**Runner:** Go standard-library `testing`, no third-party assertion or mocking framework (`go.mod`). Default packages compile without SDL; tagged GUI tests require SDL2 headers.

**Assertion Library:** Direct `t.Fatal`, `t.Fatalf`, and package-local helpers, such as `assertRootFile` in `internal/installer/installer_test.go`.

**Run Commands:** From the repository root (`Makefile`):

```sh
go test ./...                  # Core and portable UI/input logic
go test -race ./...            # CI/pre-push race tests
go test -tags sdl ./...        # Includes SDL draw tests and GUI compilation
make check                    # Rewriting fmt + vet both tags + race tests
make cover                    # Untested-package check and >= 70% total coverage
make catalogue-check          # Two generated catalogue builds must match
SDL_VIDEODRIVER=dummy make walkthrough  # Headless real-binary offline flows
```

`make check` is not the whole CI gate: Staticcheck, coverage, tagged tests, cross-builds, and walkthroughs are additional checks (`.github/workflows/check.yml`).

## Test File Organization

**Location:** Co-located `*_test.go`, using the production package name so tests can also reach private helpers. See `internal/installer/installer_test.go`, `internal/ui/model_test.go`, and `cmd/knulli-app/main_test.go`.

**Naming:** `Test<Behavior>`, with named table subtests for rejection and boundary cases. `internal/appstore/test_helpers_test.go` shares service factories.

**Structure:**

```text
internal/installer/
├── installer.go / installer_test.go
├── selfupdate.go / selfupdate_test.go
├── cache.go / cache_test.go
└── testdata/
    ├── gamelist.xml
    └── retsend-v1.0.0.sh
```

## Test Structure

**Suite Organization:** Direct tests and table-driven `t.Run` cases; `t.TempDir`, `t.Cleanup`, `t.Setenv`, and `defer` isolate files, processes, clients, and layout policies (`internal/installer/selfupdate_test.go`, `internal/input/input_test.go`).

**Patterns:**
- Build real fixture-root trees, invoke the operation, then inspect file contents, ownership, modes, menus, and preserved data (`internal/installer/installer_test.go`).
- Test adversarial archives where accepting the wrong path/type would have an observable result (`internal/archive/archive_test.go`).
- Test cancellation after an observable extraction event, not a sleep or arbitrary read count (`internal/archive/archive_test.go`, `internal/installer/installer_test.go`).

## Mocking

**Framework:** None; use explicit seams and fakes.

**Patterns:**
- `httptest.NewTLSServer` plus a rewrite client serves controlled release bytes while exercising the real downloader (`internal/installer/selfupdate_test.go`).
- An in-memory `appstore.Backend` isolates navigation and asynchronous completion (`internal/ui/model_test.go`).
- `Manager.Now`, `Manager.AvailableBytes`, refresh clients/URLs, and checker sleep/base URL fields make failure scenarios deterministic (`internal/installer/installer.go`, `internal/updatecheck/check.go`).
- A running copied `sleep` executable checks that Store replacement preserves a live process (`internal/installer/selfupdate_test.go`).

**What to Mock:** Remote HTTP, timing, capacity, and the presentation/backend boundary. Use fixture roots for device detection (`internal/platform/platform_test.go`).

**What NOT to Mock:** Real filesystem safety, transaction recovery, archive validation, and manifest/platform policy. Temp-root integration tests are required for new filesystem behavior (`AGENTS.md`, `internal/safefs/guard_test.go`).

## Fixtures and Factories

**Test Data:** Tests build ZIP/tar bytes and calculate matching release digests; unsafe archive paths remain visible in test source (`internal/archive/archive_test.go`, `internal/installer/installer_test.go`).

**Location:** Local factories in test files; shared catalogue factories in `internal/appstore/test_helpers_test.go`; XML and upstream RetSend launcher fixtures in `internal/installer/testdata/`. `scripts/desktop-fixture.sh` generates desktop Knulli roots rather than committing runtime state.

## Coverage

**Requirements:** `make cover` first rejects an untagged production package with no test file, then enforces total statement coverage >= `COVER_MIN` (default 70). It uses atomic coverage mode and writes ignored `build/cover.out` (`Makefile`).

**View Coverage:**

```sh
make cover
go tool cover -func=build/cover.out
```

Do not treat old percentages as current results. Default coverage includes portable `internal/sdlui` logic but excludes tagged `draw.go`, `run.go`, and `cmd/knulli-app-ui/main.go`; `go test -tags sdl ./...` is separate. The GUI entry point has no dedicated test file (`Makefile`, `cmd/knulli-app-ui/main.go`).

Measured during this refresh on 2026-10-21 with Go 1.27.1: `make cover` passed at **83.9% total statement coverage**, above the 70% floor. This measures the refreshed local tree, not device behavior.

## Test Types

**Unit Tests:** Version ordering, manifest rules, compatibility, layout, binding policy, repeat handling, and UI state transitions (`internal/version/version_test.go`, `internal/manifest/manifest_test.go`, `internal/sdlui/layout_test.go`, `internal/input/input_test.go`).

**Integration Tests:**
- Full adopt/repair/update/uninstall and force-reinstall recovery; package-specific Grout, PlayTime, RetSend, and Random Game Picker preservation/menu cases (`internal/installer/installer_test.go`).
- Store official asset selection, seven-file replacement, running-binary replacement, unsafe inventory/space/hash refusal, interrupted update recovery, and lock-safe abandoned staging cleanup (`internal/installer/selfupdate_test.go`).
- Journal rejection, rollback-failure snapshot retention, path policy, signatures, log redaction/rotation, and service gating (`internal/safefs/guard_test.go`, `internal/catalog/sign_test.go`, `internal/diagnostics/log_test.go`, `internal/appstore/service_test.go`).
- Automatic controls under Knulli, Xbox, and desktop layouts; ignored old mapping files and hotplug; unknown/untested devices still permitted when runtime requirements pass (`internal/input/input_test.go`, `internal/appstore/service_test.go`).

**E2E Tests:** Desktop walkthroughs exercise the real GUI binary, not a real device. `scripts/desktop-walk.sh` records per-step PNGs and `walk.tsv`, checking first-run cards, confirmation/More, filters, read-only actions, quit chord, diagnostics, and blocked-controller export. `--install` adds network-backed lifecycle flows. CI runs offline flows with the dummy SDL video driver (`.github/workflows/check.yml`).

Device GameController behavior, real SD-card semantics, and target SDL loading still require hardware evidence (`docs/real-device-tests.md`, `CONTEXT.md`). Screenshot rendering is available through `KNULLI_UI_SCREENSHOT_DIR` in `internal/sdlui/draw_test.go`.

## Common Patterns

**Async Testing:** UI tests drive a fake backend and poll completion, rather than depending on a real network. Race checks run separately (`internal/ui/model_test.go`, `Makefile`).

**Error Testing:** Rejections assert errors and unchanged files; typed conflicts use `errors.As`, context cancellation uses `errors.Is`. Recovery tests simulate process death by leaving a transaction open, then invoke recovery and inspect restored data (`internal/installer/installer_test.go`, `internal/installer/selfupdate_test.go`).

---

*Testing analysis: 2026-10-21*
