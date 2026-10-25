# Testing Patterns

**Analysis Date:** 2026-10-25

**Release scope:** Release target: Store v2026.10.25.1. Hardware reports remain scoped to the artifacts actually tested. The audit baseline was published v2026.10.25 at `14490119`.

**2026-10-25 release-target validation:** Default and SDL race suites, vet, 84.0% coverage, both Staticcheck tag sets, both hook stages and all ten network-backed GUI flows passed. The Picker 0.2.1 catalogue is deterministic at 61,574 bytes. Both local signed ARM64 packages passed the exact shared-library and GLIBC_2.34 gates, checksum/CRC, inventory/modes, current guide/font notice, catalogue equality, embedded-key signature verification and tamper rejection. Picker 0.2.1's six published assets/digests, 20 regular archive files and exact ABI were independently verified. Published-byte Install, 0.2.0→0.2.1 Update, Repair and Uninstall passed in fixture roots with nested `state/` retained. No ARM64 runtime was executed. These checks are not hardware evidence; remote CI and release results are recorded separately on GitHub.

## Test Framework

**Runner:**
- Use the Go standard-library `testing` package. The module requires Go 1.27; setup and main CI require exact Go 1.27.1 for analyzer compatibility, not any newer patch (`go.mod`, `.agents/setup`, `.github/workflows/check.yml`).
- Commands and coverage rules are in `Makefile`. Hooks are in `prek.toml`. There is no separate test framework dependency in `go.mod`.
- SDL tests need SDL2, SDL2_image, SDL2_ttf, and SDL2_gfx development files. CI and `.agents/setup` install these libraries (`.github/workflows/check.yml`, `.agents/setup`).

**Assertion Library:**
- Use `t.Fatal`, `t.Fatalf`, and local helpers. No third-party assertion or mock framework is declared (`internal/installer/installer_test.go`, `go.mod`).

**Run Commands:**
```sh
go test ./...                         # Default build tests
go test -race ./...                   # Default tests with race detection
go test -race -tags sdl ./...         # Include SDL tests, as CI does
make check                           # Reformat, vet both builds, default race tests
make test-ui                         # SDL tests without the race flag
make cover                           # Package test-file check and coverage floor
make catalogue-check                 # Compare two catalogue builds byte for byte
SDL_VIDEODRIVER=dummy SDL_AUDIODRIVER=dummy make walkthrough
make walkthrough WALK_FLAGS=--install # Also run network-backed lifecycle flows
```

Commands are from `Makefile` and `.github/workflows/check.yml`. There is no watch target in `Makefile`. `make check` needs SDL headers because it vets both build tag sets. It does not run all CI checks.

CI also runs Staticcheck v0.8.1 for both tag sets, default coverage, a static arm64 CLI build, and catalogue determinism. Separate jobs run offline GUI walkthroughs and build device artifacts for both supported targets (`.github/workflows/check.yml`).

## Test File Organization

**Location:**
- Tests are beside production files and use the same package name. They can test private helpers (`internal/installer/commit_test.go`, `internal/ui/model_test.go`, `cmd/knulli-app/main_test.go`).
- SDL-only tests carry `//go:build sdl`; portable layout and input logic also have untagged tests (`internal/sdlui/draw_test.go`, `internal/sdlui/run_test.go`, `internal/sdlui/layout_test.go`).

**Naming:**
- Use `Test<Behavior>` and descriptive subtest names. Share package-local fixtures in test helper files (`internal/archive/archive_test.go`, `internal/appstore/test_helpers_test.go`).

**Structure:**
```text
internal/installer/
  installer.go
  installer_test.go
  commit.go
  commit_test.go
  selfupdate.go
  selfupdate_test.go
  testdata/
    gamelist.xml
    retsend-v1.0.0.sh
```

These paths show the local source, test, and fixture layout (`internal/installer/`).

## Test Structure

**Suite Organization:** Use direct tests and table-driven subtests. For example, `internal/archive/archive_test.go` tests unsafe ZIP entries with this pattern:

```go
for name, mode := range map[string]os.FileMode{
    "../outside": 0644,
    "link": os.ModeSymlink | 0777,
} {
    t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
        archivePath := makeZIP(t, name, mode, "unsafe")
        _, err := Extract(context.Background(), archivePath, "zip", t.TempDir(), 0, 1024)
        if err == nil {
            t.Fatalf("expected %q to be rejected", name)
        }
    })
}
```

**Patterns:**
- Use `t.TempDir` for filesystem roots, `t.Setenv` for environment policy, and `t.Cleanup` or `defer` for resources (`internal/installer/selfupdate_test.go`, `internal/sdlui/run_test.go`).
- Assert file contents, preserved data, modes, menu entries, and installed state after operations, not only the returned error (`internal/installer/installer_test.go`).
- For archive cancellation, observe extracted file size. Check cancellation during a transfer and between entries rather than relying on an arbitrary sleep (`internal/archive/archive_test.go`).
- New filesystem behavior requires a temp-root integration test. New archive behavior requires an adversarial fixture that exposes unsafe behavior (`AGENTS.md`).

## Mocking

**Framework:** Use explicit Go fakes and injected dependencies, not a mock library (`go.mod`, `internal/ui/model_test.go`).

**Patterns:** `internal/ui/model_test.go` implements `appstore.Backend` in memory. Its execution method records the action and sends progress:

```go
func (fake *fakeBackend) Execute(_ context.Context, _ string, action appstore.Action, progress func(string)) error {
    fake.action = action
    progress("Working")
    if fake.completion != "" {
        progress(fake.completion)
    }
    return fake.err
}
```

**What to Mock:**
- Serve controlled release data with `httptest.NewTLSServer` and a rewrite client. Exercise the real downloader against those bytes (`internal/installer/selfupdate_test.go`).
- Inject free-space results, time, refresh endpoints, and HTTP clients through `Manager` fields (`internal/installer/installer.go`, `internal/installer/selfupdate_test.go`).
- Use an in-memory backend for UI transitions and a fixture root for platform detection (`internal/ui/model_test.go`, `internal/platform/platform_test.go`).

**What NOT to Mock:**
- Keep real filesystem writes, archive validation, transaction recovery, and policy checks in safety tests (`internal/safefs/guard_test.go`, `internal/archive/archive_test.go`).
- The running-UI update case copies and starts a real `sleep` executable. It verifies that the process survives replacement and cleans it up (`internal/installer/selfupdate_test.go`).

## Fixtures and Factories

**Test Data:** Build archives in test helpers and calculate matching release metadata. For example, the Store update fixture uses this pattern (`internal/installer/selfupdate_test.go`):

```go
asset := zipBytes(t, files)
hash := sha256.Sum256(asset)
```

The test pins the byte length and hex digest in `manifest.Release`. Other cases deliberately change the digest, inventory, path type, or available space (`internal/installer/selfupdate_test.go`).

**Location:**
- Shared package factories and catalogue loading helpers are in `internal/appstore/test_helpers_test.go`.
- XML and the RetSend launcher fixture are in `internal/installer/testdata/`. Tests generate malicious ZIP, tar, and embedded base64 payloads in temporary directories (`internal/archive/archive_test.go`).
- `scripts/desktop-fixture.sh` creates throwaway Knulli roots. `scripts/desktop-walk.sh` creates a separate root and evidence directory per flow.

## Coverage

**Requirements:** `make cover` first checks that each package returned by default `go list ./...` has a `*_test.go` file. It then measures atomic statement coverage and requires total coverage of at least `COVER_MIN`, which defaults to 70 (`Makefile`).

**View Coverage:**
```sh
make cover
go tool cover -func=build/cover.out
go tool cover -html=build/cover.out
```

The profile is written to ignored `build/cover.out`. This is a default-build coverage gate, not a per-package percentage gate. It does not measure SDL-tagged renderer files or the tagged GUI entry point (`Makefile`, `internal/sdlui/run.go`, `cmd/knulli-app-ui/main.go`, `.gitignore`).

Before the Picker 0.2.1 update, the 2026-10-25 concerns audit ran the default and SDL race suites, vet, both Staticcheck tag sets, coverage, catalogue determinism, GUI walkthroughs, hooks and both signed ARM64 device builds. Default-build coverage was 84.0%. That earlier index was 61,404 bytes; the current release-target results are above. These are local checks, not hardware tests. Gate definitions remain in `Makefile` and `.github/workflows/check.yml`.

## Test Types

**Unit Tests:**
- Test version rules, manifest validation, compatibility evidence, input mappings, layout, tab filtering, and notices (`internal/version/version_test.go`, `internal/manifest/manifest_test.go`, `internal/platform/platform_test.go`, `internal/input/input_test.go`, `internal/sdlui/layout_test.go`, `internal/ui/tabs_test.go`, `internal/ui/toast_test.go`).
- Tagged tests send raw SDL events to check confirmation, controller identity, ignored releases and repeats, layout policy, and the quit chord (`internal/sdlui/run_test.go`, `internal/sdlui/events_test.go`).
- Responsive layout and drawing cases live in `internal/sdlui/layout_test.go` and `internal/sdlui/draw_test.go`; they cover the shared 480-high canvas rather than an obsolete fixed one-card/bitmap-font screen. Held-navigation repeat has portable tests in `internal/sdlui/repeat_test.go`.

**Integration Tests:**
- Test Manage existing, repair, update, uninstall, forced recovery, rollback, preserved files, and menu ownership through the installer (`internal/installer/installer_test.go`).
- Cover upstream-managed packages, read-only uninstall checks, embedded archives that must not execute their script prefix, and package-specific launcher behavior (`internal/installer/installer_test.go`).
- Test exact Store update inventory, checksum and space refusal, live executable replacement, interrupted update recovery, and abandoned staging cleanup (`internal/installer/selfupdate_test.go`).
- Test journal validation, retained snapshots after failed rollback, path policy, catalogue signatures, and diagnostics redaction (`internal/safefs/guard_test.go`, `internal/catalog/sign_test.go`, `internal/diagnostics/log_test.go`).
- Test filling blank menu metadata without claiming existing entries, and preserving edited artwork (`internal/gamelist/menu_test.go`).
- Test update-report assessment, malformed release metadata, bounded rate-limit retries, and shared repository requests (`internal/updatecheck/check_test.go`).

**E2E Tests:**
- `scripts/desktop-walk.sh` runs scripted keyboard flows through the real GUI binary against fixture roots. It checks expected screens and messages in `walk.tsv`, records PNG frames, and writes `summary.tsv`.
- Offline flows cover Library browsing, filters, confirmations, More, diagnostics, blocked-controller export, and the quit chord. `--install` adds network-backed lifecycle flows (`scripts/desktop-walk.sh`).
- `make walkthrough` also runs tagged screen-render tests with `KNULLI_UI_SCREENSHOT_DIR`. CI uses dummy SDL video and audio drivers and uploads `build/walkthrough` even after failure (`Makefile`, `internal/sdlui/draw_test.go`, `.github/workflows/check.yml`).
- Screenshot/walkthrough runs intentionally keep rendering full frames. They exercise capture and scripted settling, but do not by themselves prove normal idle redraw suppression. The current normal loop polls at 30 Hz and redraws for SDL events, held-navigation repeat, `Model.Poll` changes, and live-toast count changes. This refresh separately exercised normal interactive mode under Xvfb/software SDL: navigation, confirmation cancellation, diagnostics, empty filters, restored focus, actual PlayTime installation, completion and notice expiry with no further input. Inspected screenshots and recordings confirmed those updates (`internal/sdlui/run.go`, `internal/ui/model.go`).
- Desktop evidence does not prove real GameController behavior, real SD-card writes, or device SDL loading. Hardware runs remain separate (`CONTEXT.md`, `docs/real-device-tests.md`).
- The [owner's 2026-10-25 confirmation](https://ampcode.com/threads/T-01a17039-648c-7024-abe9-8785444a2b5b) (Asia/Singapore) supplies passed package evidence for the six unchanged current versions and historical Picker 0.2.0 on TrimUI Smart Pro only, Knulli AArch64 at 1280×720. Published Picker 0.2.1 has no new hardware evidence and is UNTESTED on every matrix. Review statuses remain experimental; MagicX and other matrices remain untested. The confirmation supplies no exact firmware build identifier or separate lifecycle, physical button, audio, power-loss, or Store idle-redraw results. Package evidence is not a frontend-build badge (`catalogue/packages/`, `CONTEXT.md`, `docs/real-device-tests.md`).
- Separately verified evidence supplied from source thread `T-01a16e63-3dfe-7782-8ac5-a9a71a55726c`: public v2026.10.25 uses source `14490119b819f524a8a59919b949359092b3b014`; both release ZIP catalogue signatures passed verification against their embedded GUI keys. The initial assertion that both binaries contained the key failed because the CLI's unused key is eliminated. Build flags are not evidence of linked CLI key retention (`scripts/device-build.sh`, `internal/catalog/catalog.go`, `internal/appstore/service.go`).
- That thread also records native final Option B testing on TrimUI Smart Pro at 1280×720, using a different development ZIP from the same source: fresh Picker 0.2.0 install, update from 0.1.1 to 0.2.0, Repair and preserved state passed; 100/100 injected navigation inputs, filters/focus, EmulationStation quit, and Picker startup/settings/exit passed. These are attributed results, not tests rerun in this refresh; see `docs/real-device-tests.md` for the maintained record.
- Limits of that native evidence: exact firmware was not recorded; no physical button presses, audio, stable gameplay/hotkey return, MagicX run, or new-renderer Uninstall/force reinstall was verified. Same-source development ZIP evidence is not byte-identical public-release evidence, an exact-matrix report, or proof of the subsequent dirty-redraw working-tree change (`CONTEXT.md`, `docs/real-device-tests.md`, `internal/sdlui/run.go`).

**Historical concerns-audit validation (before Picker 0.2.1):**
- `make check build build-ui catalogue catalogue-check cover`, `go test -race -tags sdl ./...`, and Staticcheck v0.8.1 with both tag sets passed. The default-build coverage gate reported 84.0%; this is not SDL runtime coverage.
- All ten offline/network-backed walkthrough flows passed, including Install, health/Repair and Uninstall. Both hook stages, Actionlint, shell syntax checks and unchanged `go mod tidy` lock-file hashes passed.
- Both ARM64 device builds passed the exact six-library allowlist and GLIBC_2.34 ceiling. Package checks passed CRC, seven regular files, executable modes, current guide/font notice, deterministic catalogue equality, embedded GUI key, Ed25519 verification and rejection of a changed index.
- The numbered Store downgrade regression failed before the fix and passed afterward; the same public API test checks that a newer numeric release remains offered and equivalent numeric spellings are not offered (`internal/installer/selfupdate_test.go`). Hash-only build ordering is not promised.
- Controlled normal-mode idle runs used three 6-second quiet intervals after 2-second warm-ups per binary with dummy SDL at 1280×720. Median CPU fell from 19.17% to 1.00% of one core. This is orb evidence, not handheld power or battery evidence (`CONCERNS.md`).

## Common Patterns

**Async Testing:** `waitForModel` polls for up to one second and sleeps one millisecond between polls, then fails if work is still busy. This is a bounded wait, not a fully simulated clock (`internal/ui/model_test.go`).

**Error Testing:** Use `errors.Is` for cancellation. Inspect unchanged files on rejection. Simulate a crash by leaving an open journal, then recover and inspect restored state (`internal/archive/archive_test.go`, `internal/installer/installer_test.go`, `internal/safefs/guard_test.go`).

---

*Testing analysis: 2026-10-25*
