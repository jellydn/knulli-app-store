# Testing Patterns

**Analysis Date:** 2026-09-21

## Test Framework

**Runner:**
- Go 1.27 `testing` (stdlib). No third-party test framework, no runner config file.
- Config: none needed. Package-level `go test` with no build tags for the CLI/core, plus `-tags sdl` for the GUI.

**Assertion Library:**
- None. Tests use `t.Fatal` / `t.Fatalf` / `t.Errorf` with hand-written comparisons.
- Repo-local assertion helpers exist per package where the shape repeats, e.g. `assertRootFile`, `assertMissing`, `assertContains`, `assertNotContains` in `internal/installer/installer_test.go`.

**Run Commands:**
```bash
go test ./...                                   # all CLI/core tests
go test -race ./...                             # what CI and pre-push run (make test / make check)
go test -tags sdl ./...                         # GUI compile + layout tests; needs libsdl2-dev
make check                                      # fmt + vet (both tag sets) + go test -race ./...
make catalogue-check                            # determinism gate: two index builds must match byte for byte
make cover                                      # coverage gate (untested check + floor)
make cover-untested                             # only: every prod package must have a test file
make gui                                        # interactive desktop GUI, no device
make walkthrough                                # scripted GUI evidence; WALK_FLAGS=--install adds network flows
```

Measured on 2026-09-21: `go test ./...` is green and total statement coverage is **84.2%** (gate floor is 70%).

## Test File Organization

**Location:**
- Strictly co-located: `internal/<pkg>/<file>_test.go` sitting beside the code it covers.
- All tests are **internal tests** (same package, not `_test` package), so unexported functions, unexported fields, and unexported sentinels are all reachable. `cmd/knulli-app/main_test.go` drives the unexported `run` dispatch and `runApply`/`runUninstall` directly, and tests `writeNewFile`.

**Naming:**
- `Test<Behavior>` describing the rule, not the function: `TestManageExistingRepairUpdateAndUninstallEndToEnd`, `TestGuardRejectsPathsOutsidePolicyAndSymlinkParents`, `TestRecoverRestoresOpenJournalAfterCrash`, `TestExtractZIPRejectsTraversalAndLinks`, `TestHealthCheckHashesContentWhenSizeAndTimeMatch`.
- Shared helpers go in `<pkg>_test_helpers_test.go` when several test files need them — only `internal/appstore/test_helpers_test.go` exists today.

**Structure:**
```
internal/appstore/
├── service.go
├── service_test.go
├── test_helpers_test.go     # catalogForTest, installablePackage, writeIndexForTest
└── verdict.go               # tested through service_test.go
```

## Test Structure

**Suite Organization:**
- Plain `func TestX(t *testing.T)` with no setup/teardown framework. Setup is inline, cleanup is `t.TempDir()` (147 uses across the suite) plus `defer`.
- Subtests via `t.Run` for table-driven cases, typically over a map when the cases are independent:
  ```go
  for name, mode := range map[string]os.FileMode{
      "../outside":       0644,
      "folder/../../bad": 0644,
      "folder\\..\\bad":  0644,
      "link":             os.ModeSymlink | 0777,
  } {
      t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) { ... })
  }
  ```
  (from `internal/archive/archive_test.go`)

**Patterns:**
- **Temp-root integration over a fake device.** The dominant pattern: `t.TempDir()` becomes `Manager.Root` or `platform.Resolve`'s root, then the test writes the files a real device would have (`writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", ...)`) and asserts on the resulting tree. This is required by `CONTRIBUTING.md`: "New filesystem behavior needs a temporary-root integration test."
- **One long end-to-end test plus focused unit tests.** `TestManageExistingRepairUpdateAndUninstallEndToEnd` in `internal/installer/installer_test.go` walks adopt → repair → damage detection → repair → update → uninstall in a single flow, asserting preserved files, restored originals, menu entries, and the emitted log events. Focused tests cover individual rules.
- **Observable-side-effect assertions, not internal-state assertions.** Tests read files back (`os.ReadFile`), stat modes, and grep the log for `event=` lines rather than inspecting structs.
- **Assert failure messages carry the actual value:** `t.Fatalf("private key mode = %o, want 600", ...)`, `t.Fatalf("changed external files should need repair after management: %#v, %v", status, err)`.

## Mocking

**Framework:** None. Fakes and injectable seams replace mocking libraries.

**Patterns:**
1. **Real HTTP server for downloads.** `httptest.NewTLSServer` (6 uses) serves fake release bytes, with a `rewriteClient` helper that repoints the manager's client at the test server. The installer's `Manager.Client` field is the seam.
   ```go
   server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
       data, found := assets[r.URL.Path]
       if !found { http.NotFound(w, r); return }
       w.Header().Set("Content-Length", strconv.Itoa(len(data)))
       w.Write(data)
   }))
   manager := Manager{Root: root, Client: rewriteClient(t, server), Diagnostics: diagnosticLog}.WithPlatform(testPlatform())
   ```
2. **In-memory fake backend for the UI.** `internal/ui/model_test.go` implements `appstore.Backend` (`Items`, `Execute`, `ExportDiagnostics`, `SetPlatform`) in memory, so the whole state machine is testable with no filesystem and no SDL. This is the seam `AGENTS.md` calls out: "GUI tests use an in-memory fake backend."
3. **Injectable clocks, buffers, and byte counters.**
   - `Manager.Now func() time.Time` — deterministic recovery-backup timestamps.
   - `Manager.AvailableBytes func(string) (uint64, error)` — force the free-space failure.
   - `Checker.Sleep func(time.Duration)` and `Checker.BaseURL` — exercise retry/backoff without waiting and point the API client at a local server.
   - `Manager.RefreshURL` / `Manager.RefreshClient` — make the loopback reload succeed, fail, or return a non-2xx.
4. **Archive builders instead of fixture archives.** `makeZIP`, `makeTarGZ`, `makeArchive`, `zipBytes` construct adversarial archives in-process, so the unsafe shape is explicit in the test rather than hidden in a binary file. This satisfies "New archive behavior needs an adversarial fixture where an unsafe implementation would produce a different result." `makeArchive` takes an ordered slice, so a cancellation test always meets the same member first and can assert that the member after the interrupted one was never started.
5. **Contexts that cancel on an observed fact, not a timer or a count.** Cancellation tests use a context whose `Err()` consults the extraction's own output — an entry that has grown past a threshold, or a staging directory that now holds a file — so the interruption lands at the same point on every run. No sleeps, and no dependence on how many reads a transfer takes. The installer-level test also records that the trigger was reached, so a cancellation that fired earlier cannot make it pass for the wrong reason.
6. **On-disk fixture only where the XML shape matters:** `internal/installer/testdata/gamelist.xml` is the single committed fixture directory.

**What to Mock:**
- Network: always fake. No test touches a real GitHub URL.
- Time and space: inject the clock, the free-bytes function, and the retry sleep.
- The presentation boundary: fake `appstore.Backend` for UI tests.
- The SDL boundary: `internal/sdlui` splits almost all logic into non-tagged files (`layout.go`, `hints.go`, `walk.go`, `repeat.go`, `input_mode.go`) that test without cgo; only `run.go` and `draw.go` carry the tag.

**What NOT to Mock:**
- The filesystem. Every filesystem behavior test uses a real `t.TempDir()` tree, precisely so path policy, symlink rejection, and rollback are exercised against the kernel rather than a stub.
- `safefs`, `archive`, `manifest.Validate`, and `platform.Check` — these are covered directly and then used for real inside the integration tests.
- The catalogue's own manifests. `appstore.catalogForTest()` builds the real index from `../../catalogue/packages` rather than a hand-written one, so a broken manifest fails the appstore suite too.

## Fixtures and Factories

**Test Data:**
```go
// internal/appstore/test_helpers_test.go
func installablePackage() manifest.Package {
    return manifest.Package{
        Schema: manifest.SchemaV1, ID: "org.example.test", Name: "Test", Version: "1.0.0",
        Repository: "https://github.com/example/test", License: "MIT",
        Review: manifest.Review{Status: "installable"},
        Release: &manifest.Release{URL: "https://github.com/example/test/releases/download/v1/test.zip",
            SHA256: strings.Repeat("a", 64), Size: 1, InstalledSize: 1, Format: "zip", Immutable: true},
        Compatibility: &manifest.Compatibility{Firmware: "knulli", MinimumVersion: "2026.05",
            Architectures: []string{"aarch64"}, ABIs: []string{"linux-aarch64-glibc"},
            Dependencies: []string{"sdl2"}, Devices: []string{"trimui-smart-pro"}, Resolutions: []string{"1280x720"}},
        Install: &manifest.Install{Destination: "/userdata/roms/ports/test", Launcher: "run.sh",
            AllowedWritePaths: []string{"/userdata/roms/ports/test"}},
    }
}
```

- `internal/installer/installer_test.go` has `testPlatform()` and `testPackage(url, archiveBytes, version)` factories that derive the release size and SHA-256 from the generated bytes, so a fixture can never disagree with its own declared hash.
- `internal/appstore/test_helpers_test.go` has `catalogForTest()` and `writeIndexForTest()`.

**Location:**
- Package-local, in the `_test.go` files themselves. Only `internal/installer/testdata/` holds on-disk fixtures.
- The device fixture that desktop/CI runs use is generated at runtime by `scripts/desktop-fixture.sh`, not committed.

## Coverage

**Requirements:** Enforced by `make cover`, run on every pull request in `.github/workflows/check.yml`.

The gate is two checks, deliberately, as documented in the `Makefile`:
1. `cover-untested` — every package that holds production Go files must have at least one `*_test.go` file. A new package arriving with no tests fails here even if the total stays high.
2. Total statement coverage must be >= `COVER_MIN` (default **70**), measured with `-covermode=atomic`.

Current per-package coverage (2026-09-21), highest to lowest:

| Package | Coverage |
| --- | --- |
| `internal/version` | 100.0% |
| `internal/manifest` | 98.4% |
| `internal/sdlui` | 91.8% |
| `internal/platform` | 91.5% |
| `internal/gamelist` | 88.7% |
| `internal/input` | 85.5% |
| `internal/safefs` | 84.3% |
| `internal/ui` | 84.2% |
| `internal/updatecheck` | 84.2% |
| `internal/archive` | 84.0% |
| `cmd/knulli-app` | 83.8% |
| `internal/appstore` | 79.6% |
| `internal/installer` | 79.4% |
| `internal/diagnostics` | 73.6% |
| `cmd/check-updates` | 70.4% |
| `internal/catalog` | 67.5% |
| **Total** | **84.2%** |

The three lowest packages were raised in one pass: `internal/manifest` (65.9% -> 98.4%), `internal/safefs` (64.1% -> 84.3%), and `cmd/knulli-app` (4.7% -> 83.8%). The first two gained table-driven cases for every validation rule and for the transaction and journal APIs; the CLI gained end-to-end tests that drive `run` against a temp root with a loopback release server. What remains uncovered in all three is I/O error returns that need an injected syscall fault, plus `main` itself.

**View Coverage:**
```bash
make cover                # gate + per-function table tail
go tool cover -func=build/cover.out
```

The `sdl`-tagged packages are excluded from the `make cover` run because they need `libsdl2-dev`; they are measured separately by `make test-ui` / `go test -tags sdl ./...`.

## Test Types

**Unit Tests:**
- Scope: pure logic with no I/O — `internal/version` (ordering rules, prerelease handling), `internal/manifest` (validation rules), `internal/platform` (resolution candidate assessment, version comparability), `internal/sdlui/layout.go` (geometry tokens), `internal/input/mapping.go` (plan/validation/conflict rules).
- Approach: table-driven or direct assertions, no temp root.

**Integration Tests:**
- Scope: everything that touches the filesystem or the network. `internal/installer` (47 test funcs, 1,477 lines), `internal/safefs` (guard policy, transaction rollback, journal recovery), `internal/archive` (adversarial archives, including cancellation at both interruption points), `internal/gamelist` (menu ownership), `internal/appstore` (action gating against the real catalogue), `internal/diagnostics` (redaction, rotation, export).
- Approach: `t.TempDir()` as a device root, `httptest` as a release host, then assertions on the resulting file tree and the emitted log events.

**GUI/Walkthrough Tests (not E2E against a device):**
- `go test -tags sdl ./internal/sdlui` compiles the SDL adapter and runs the layout, draw, hints, repeat, walk, and input-mode tests. It also renders every screen state to PNG when `KNULLI_UI_SCREENSHOT_DIR` is set.
- `make walkthrough` runs the **real binary** through scripted keyboard sequences against a scratch root, capturing one frame per step plus a `walk.tsv` record naming the screen each key reached. `scripts/desktop-walk.sh` asserts each flow reached its expected screens and that skipping paging saved no paging binding. CI runs the offline flows headlessly and uploads the evidence as a 14-day artifact.
- Explicit limitation, stated in `CONTEXT.md`: a walkthrough "never exercises a real GameController, a real `/userdata` write, or the kernel's SDL loading." Device-only checks are listed in `docs/desktop-verification.md` and `docs/real-device-tests.md`.

**E2E Tests:**
- No automated device E2E. Real-device results are recorded as manifest `review.evidence` entries with `kind: "real-device-test"` and the tester, date, package version, firmware, architecture, device, resolution, and a `passed` result — required for `verified` status and validated by `manifest.hasVerifiedMatrixEvidence`.

## Common Patterns

**Async Testing:**
- No goroutine-racing tests. Operations are called synchronously in tests (`manager.Apply(ctx, op, pkg)`), and the UI's async path is exercised by draining the fake backend through `Model.Load`/`Model.Poll` deterministically.
- `go test -race` is the standard runner, so any accidental shared state is caught rather than asserted around.

**Error Testing:**
```go
if _, err := guard.Resolve("/userdata/roms/ports/outside"); err == nil {
    t.Fatal("expected path outside policy to fail")
}
```
- The house style is: call it, assert `err != nil`, and keep the message human-readable. Typed errors are checked with `errors.As` (e.g. `installer.AdoptionConflictError`).
- Negative assertions are the bulk of the security suite — a rejected archive entry, a rejected symlink parent, a rejected non-HTTPS redirect, a rejected candidate manifest carrying release metadata.

**State/Log Assertion:**
```go
for _, event := range []string{
    "event=operation_start", "event=compatibility_allowed", "event=download_start",
    "event=download_verified", "event=extraction_complete", "event=transaction_begin",
    "event=backup_complete", "event=operation_complete",
} {
    if !strings.Contains(string(logData), event) {
        t.Fatalf("operation log lacks %s: %s", event, logData)
    }
}
```
The diagnostics log doubles as an ordered event trace that tests assert on, which is how transaction ordering is pinned without reading internal state.

---

*Testing analysis: 2026-09-21*
