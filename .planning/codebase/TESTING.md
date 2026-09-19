# Testing Patterns

**Analysis Date:** 2026-09-19

## Test Framework

**Runner:**
- Go `testing` on toolchain 1.27
- No configuration file; the `Makefile` and `.github/workflows/check.yml` are the only harness

**Assertion Library:**
- Standard library only: `t.Fatal`, `t.Fatalf`, `t.Errorf`, plus hand-written helpers such as `assertRootFile`, `assertMissing`, `assertContains`
- No testify, no gomock, no third-party matcher

**Run Commands:**
```bash
make test        # go test -race ./...
make test-ui     # go test -tags sdl ./...
make vet         # go vet ./... && go vet -tags sdl ./...
make check       # fmt + vet + test
make walkthrough # offline GUI flows + render every screen (needs build-ui, catalogue)
make walkthrough WALK_FLAGS=--install   # adds the network-backed lifecycle flows
go test ./internal/updatecheck          # the weekly checker package, as CI runs it
```

Counts today: 248 `TestXxx` functions across 29 test files.

## Test File Organization

**Location:**
- Co-located with the code under `internal/<pkg>`, plus `cmd/knulli-app/main_test.go`
- Only one on-disk fixture directory: `internal/installer/testdata`

**Naming:**
- `*_test.go`; shared fixtures go in `test_helpers_test.go` (for example `internal/appstore/test_helpers_test.go`)
- Audit tests are named for the rule they enforce: `internal/input/physical_button_audit_test.go`

**Structure:**
```
cmd/knulli-app/main_test.go                     # 1 test, CLI command surface
internal/installer/installer_test.go            # 34 tests, the lifecycle suite
internal/platform/platform_test.go              # 21 tests, detection fixtures
internal/sdlui/draw_test.go                     # 22 tests, renders every screen (sdl tag)
internal/input/input_test.go                    # 17 tests, mappings and session
internal/ui/tabs_test.go                        # 10 tests, tab filtering and memory
internal/gamelist/menu_test.go                  # 11 tests, XML ownership
internal/installer/testdata/gamelist.xml        # XML fixture
```

## Test Structure

**Suite Organization:**
```go
func TestChecksumFailureWritesNoPackageFiles(t *testing.T) {
	root := t.TempDir()
	manager := Manager{Root: root, Platform: testPlatform(), Client: rewriteClient(t, server)}
	if _, err := manager.Apply(context.Background(), OpInstall, pkg); err == nil {
		t.Fatal("expected checksum failure")
	}
	assertMissing(t, root, "userdata/roms/tools/demo/launch.sh")
}
```

**Patterns:**
- Setup: `t.TempDir()` supplies `-root`; `diagnostics.Open(root)`; `httptest.NewTLSServer` for release downloads
- Teardown: `defer server.Close()`; temp roots are removed by `testing`
- Assertions compare file bytes, XML fragments, `Status`/`HealthIssue` structs, and `Verdict` values rather than rendered strings
- Tests name the invariant in the failure message (`t.Fatal("candidate must not install")`)

## Mocking

**Framework:** none

**Patterns:**
```go
server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write(assets[r.URL.Path])
}))
manager := Manager{Root: root, Client: rewriteClient(t, server), RefreshClient: refreshClient, RefreshURL: refreshURL}
```

**What to Mock:**
- HTTPS release downloads: `httptest` TLS server plus `rewriteClient` (`internal/installer/installer_test.go`)
- EmulationStation `/reloadgames`: `Manager.RefreshURL` pointed at a local test server so accepted and refused outcomes are both exercised
- `appstore.Backend`: an in-memory fake in `internal/ui` tests
- Time and free space: `Manager.Now` and `Manager.AvailableBytes` seams
- SDL: the GUI is driven headlessly through `input.Session` and the walkthrough harness rather than by mocking SDL

**What NOT to Mock:**
- `safefs`, `archive` extraction, `gamelist` XML rewriting, catalogue hashing and signing, and platform file parsing are all exercised for real
- Catalogue data is real: `catalogForTest()` calls `catalog.Build(filepath.Join("..", "..", "catalogue", "packages"))`, so a manifest change can break the appstore suite

## Fixtures and Factories

**Test Data:**
```go
pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", v1, "1.0.0")
v1 := zipBytes(t, map[string]string{"launch.sh": "version one"})
zipBytesWithModes(t, []zipFixture{{name: "run.sh", mode: 0755}})
writeRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
assertRootFile(t, root, "userdata/system/knulli-app-store/demo.json", expected)
```

**Location:**
- Installer helpers: `internal/installer/installer_test.go` lines 1265-1421 (`testPackage`, `testPlatform`, `rewriteClient`, `zipBytes`, `zipBytesWithModes`, `writeRootFile`, `assertRootFile`, `assertMissing`, `assertContains`, `assertNotContains`)
- Appstore helpers: `internal/appstore/test_helpers_test.go` (`catalogForTest`, `installablePackage`, `writeIndexForTest`)
- XML fixture: `internal/installer/testdata/gamelist.xml`
- Live manifests: `catalogue/packages/` used directly as the catalogue fixture

## Coverage

**Requirements:** None enforced. There is no coverage gate in CI and no coverage badge.

**View Coverage:**
```bash
go test -race -cover ./...
go test -tags sdl -cover ./internal/sdlui
```

## Test Types

**Unit Tests:**
- Manifest validation and schema agreement, redaction and log rotation, mapping validation and assignment, resolution parsing, version comparison, catalogue digest and signature mismatch

**Integration Tests:**
- Temporary-root lifecycle in `internal/installer/installer_test.go`: install, rejected repeated install, adopt with mixed content, update, repair refusal on a version mismatch, uninstall, rollback, preserved-path behavior, force reinstall with recovery backup, and the RetSend package specifically
- Adversarial archive fixtures in `internal/archive/archive_test.go`: traversal, links, devices, pipes, duplicate names, size overflow
- Guard and transaction tests in `internal/safefs/guard_test.go`: symlinked parents, journal recovery, reverse-order rollback
- Menu ownership in `internal/gamelist/menu_test.go`: add only when absent, remove only an unchanged owned entry, preserve unknown XML elements
- GUI events in `internal/sdlui/events_test.go` and `walk_test.go`: key sequences dispatched through the real handler

**E2E Tests:**
- No real-device automation. Device results are recorded by hand in `docs/real-device-tests.md`.
- Closest automated equivalent: `scripts/desktop-walk.sh` replays scripted key sequences through the real GUI binary against a scratch fixture root, writing opening, post-key, and completion frames plus a `walk.tsv` record per flow under `build/walkthrough/<flow>/`. Flows declare the screens they must reach, and the script fails a run that does not reach them.
- `internal/sdlui/draw_test.go` renders every screen state at both target resolutions and writes PNGs when `KNULLI_UI_SCREENSHOT_DIR` is set (`make walkthrough` sets it to `build/walkthrough/screens`)
- CI runs `make walkthrough` under `SDL_VIDEODRIVER=dummy` and uploads the frames as the `gui-walkthrough` artifact, including partial evidence when a flow fails

## Common Patterns

**Async Testing:**
```go
// ui operations send on Model.events; tests call Load, Select, and Poll,
// then assert on Message, Error, and LiveToasts.
```

**Error Testing:**
```go
if err := manager.Apply(ctx, OpInstall, candidate); err == nil {
	t.Fatal("candidate must not install")
}
```

**Determinism:**
```bash
go run ./cmd/knulli-app catalogue -output /tmp/index-one.json
go run ./cmd/knulli-app catalogue -output /tmp/index-two.json
cmp /tmp/index-one.json /tmp/index-two.json
```

**Adversarial fixtures:**
- `CONTRIBUTING.md` requires every new archive or filesystem behavior to ship a case where an unsafe implementation would produce a different result

**Source audit:**
- `TestRuntimeCodeDoesNotBypassSemanticControllerActions` (`internal/input/physical_button_audit_test.go`) scans runtime code for physical-button assumptions, so a hardcoded A/B cannot creep back in

---

*Testing analysis: 2026-09-19*
