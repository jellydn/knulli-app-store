# Testing Patterns

**Analysis Date:** 2026-09-15

## Test Framework

**Runner:**
- Go `testing` (toolchain 1.19)
- Config: none beyond `Makefile` and CI workflow

**Assertion Library:**
- Standard library only (`t.Fatal`, `t.Fatalf`, helpers like `assertRootFile`)

**Run Commands:**
```bash
make test                 # go test -race ./...
make test-ui              # go test -tags sdl ./...
make check                # fmt + vet + test
go test ./internal/updatecheck   # weekly checker package
```

## Test File Organization

**Location:**
- Co-located next to the code under `internal/<pkg>`

**Naming:**
- `*_test.go`; one audit file `internal/input/physical_button_audit_test.go`

**Structure:**
```
internal/installer/installer_test.go
internal/installer/testdata/gamelist.xml
internal/appstore/service_test.go
internal/appstore/test_helpers_test.go
internal/sdlui/draw_test.go          # requires -tags sdl
```

## Test Structure

**Suite Organization:**
```go
func TestChecksumFailureWritesNoPackageFiles(t *testing.T) {
    root := t.TempDir()
    manager := Manager{Root: root, Platform: testPlatform(), Client: rewriteClient(t, server)}
    if err := manager.Install(context.Background(), pkg); err == nil {
        t.Fatal("expected checksum failure")
    }
    assertMissing(t, root, "userdata/roms/tools/demo/launch.sh")
}
```

**Patterns:**
- Setup: `t.TempDir()` as `-root`, `diagnostics.Open(root)`, `httptest.NewTLSServer`
- Teardown: `defer server.Close()`; temp dirs cleaned by testing
- Assertion: helpers compare file bytes, XML fragments, and status structs

## Mocking

**Framework:** none

**Patterns:**
```go
server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Write(assets[r.URL.Path])
}))
manager := Manager{Client: rewriteClient(t, server), RefreshClient: refreshClient, RefreshURL: refreshURL}
```

**What to Mock:**
- HTTPS release downloads (`httptest` TLS server)
- EmulationStation `/reloadgames` (local test server)
- `appstore.Backend` in `internal/ui` tests (in-memory fake)

**What NOT to Mock:**
- `safefs`, archive extraction, XML rewrite, catalogue hashing, platform file parsing
- Real catalogue fixtures: `catalog.Build("../../catalogue/packages")` in appstore tests

## Fixtures and Factories

**Test Data:**
```go
pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", v1, "1.0.0")
v1 := zipBytes(t, map[string]string{"launch.sh": "version one"})
writeRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
```

**Location:**
- Helpers in the same `_test.go` or `test_helpers_test.go`
- XML fixture: `internal/installer/testdata/gamelist.xml`
- Live manifests: `catalogue/packages/` for catalogue and appstore tests

## Coverage

**Requirements:** None enforced; no coverage gate in CI

**View Coverage:**
```bash
go test -race -cover ./...
```

## Test Types

**Unit Tests:**
- Manifest validation, redaction, mapping validation, resolution parsing, catalogue digest mismatch

**Integration Tests:**
- Temporary-root lifecycle: adopt, repair, update, uninstall, rollback, PlayTime preserve paths (`internal/installer/installer_test.go`)
- Adversarial archive fixtures that fail on traversal, links, size overflow (`internal/archive/archive_test.go`)
- Guard/transaction symlink and rollback tests (`internal/safefs/guard_test.go`)

**E2E Tests:**
- Not used against real devices in CI
- SDL tests compile the GUI and render representative frames at 1280×720 and 640×480 (`internal/sdlui/draw_test.go`)
- Real-device evidence is recorded in `docs/real-device-tests.md`, not automated

## Common Patterns

**Async Testing:**
```go
// UI operations send on Model.events; tests call Load/Select and inspect Message/Error.
```

**Error Testing:**
```go
if err := manager.Install(ctx, candidate); err == nil {
    t.Fatal("candidate must not install")
}
```

**Adversarial fixtures:**
- New archive or filesystem behavior must include a case where an unsafe implementation would produce a different result (`CONTRIBUTING.md`)

**Source audit:**
- `TestRuntimeCodeDoesNotBypassSemanticControllerActions` scans runtime code for physical-button assumptions

---

*Testing analysis: 2026-09-15*
