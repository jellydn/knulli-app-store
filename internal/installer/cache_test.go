package installer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

// installCachedDemo installs the demo fixture into a temporary root with a
// status cache attached, then returns the manager, the package, and the path of
// the installed launcher the tamper tests edit.
func installCachedDemo(t *testing.T) (Manager, manifest.Package, string) {
	t.Helper()
	root := t.TempDir()
	asset := zipBytes(t, map[string]string{"launch.sh": "reviewed launcher"})
	server := serveAsset(t, asset)
	t.Cleanup(server.Close)
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform()).WithStatusCache(NewStatusCache())
	if _, err := manager.Apply(context.Background(), OpInstall, pkg); err != nil {
		t.Fatal(err)
	}
	return manager, pkg, filepath.Join(root, "userdata/roms/tools/demo/launch.sh")
}

func tamperLauncher(t *testing.T, launcher string) {
	t.Helper()
	if err := os.WriteFile(launcher, []byte("tampered launcher"), 0755); err != nil {
		t.Fatal(err)
	}
}

// The cache is the whole point of the change, and this is its cost: while an
// entry is live, an edit the installer did not make is not re-hashed. The test
// pins that window and the Invalidate call that closes it, so nobody has to
// guess where the health check stops being fresh.
func TestStatusCacheServesAStaleHealthCheckUntilInvalidated(t *testing.T) {
	manager, pkg, launcher := installCachedDemo(t)
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy || len(status.Issues) != 0 {
		t.Fatalf("initial status = %#v, %v", status, err)
	}
	tamperLauncher(t, launcher)
	status, err := manager.Status(pkg.ID)
	if err != nil || !status.Healthy {
		t.Fatalf("the cached status should have hidden the edit, got %#v, %v", status, err)
	}
	manager.statusCache.Invalidate(pkg.ID)
	status, err = manager.Status(pkg.ID)
	if err != nil || status.Healthy || len(status.Issues) != 1 || status.Issues[0].Check != "content changed" {
		t.Fatalf("an invalidated entry did not re-hash the file: %#v, %v", status, err)
	}
}

// A record rewritten without going through the manager must still not be
// answered from the cache, so the key covers the installed state rather than
// just the package id.
func TestStatusCacheIsKeyedOnTheRecordedInstalledState(t *testing.T) {
	manager, pkg, _ := installCachedDemo(t)
	if status, err := manager.Status(pkg.ID); err != nil || status.Version != "1.0.0" {
		t.Fatalf("initial status = %#v, %v", status, err)
	}
	stateFile := filepath.Join(manager.Root, strings.TrimPrefix(statePath(pkg.ID), "/"))
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	var state Installed
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	state.Manifest.Version = "9.9.9"
	data, err = encodeState(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stateFile, data, 0600); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status(pkg.ID)
	if err != nil || status.Version != "9.9.9" {
		t.Fatalf("a changed installed state reused the cached status: %#v, %v", status, err)
	}
}

// A failed operation can leave a destination half-written before the rollback,
// so it invalidates too, not only a successful one.
func TestFailedOperationInvalidatesTheCachedHealthCheck(t *testing.T) {
	manager, pkg, launcher := installCachedDemo(t)
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("initial status = %#v, %v", status, err)
	}
	tamperLauncher(t, launcher)
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("the cached status should have hidden the edit, got %#v, %v", status, err)
	}
	mismatched := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", zipBytes(t, map[string]string{"launch.sh": "other launcher"}), "1.0.0")
	mismatched.Release.SHA256 = strings.Repeat("b", 64)
	if _, err := manager.Apply(context.Background(), OpInstall, mismatched); err == nil {
		t.Fatal("expected the checksum mismatch to fail the operation")
	}
	status, err := manager.Status(pkg.ID)
	if err != nil || status.Healthy || len(status.Issues) != 1 || status.Issues[0].Check != "content changed" {
		t.Fatalf("a failed operation left the cached status in place: %#v, %v", status, err)
	}
}

func TestInvalidateAllDropsEveryCachedHealthCheck(t *testing.T) {
	manager, pkg, launcher := installCachedDemo(t)
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("initial status = %#v, %v", status, err)
	}
	tamperLauncher(t, launcher)
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("the cached status should have hidden the edit, got %#v, %v", status, err)
	}
	manager.statusCache.InvalidateAll()
	status, err := manager.Status(pkg.ID)
	if err != nil || status.Healthy {
		t.Fatalf("InvalidateAll left a cached status in place: %#v, %v", status, err)
	}
}

// A caller reading a cached status must not be able to reach into the copy the
// cache holds.
func TestStatusCacheHandsBackACopyOfItsIssues(t *testing.T) {
	manager, pkg, launcher := installCachedDemo(t)
	tamperLauncher(t, launcher)
	first, err := manager.Status(pkg.ID)
	if err != nil || len(first.Issues) != 1 {
		t.Fatalf("first status = %#v, %v", first, err)
	}
	first.Issues[0].Check = "mutated by the caller"
	second, err := manager.Status(pkg.ID)
	if err != nil || len(second.Issues) != 1 || second.Issues[0].Check != "content changed" {
		t.Fatalf("a caller mutated the cached status: %#v, %v", second, err)
	}
}

// A manager built without a cache keeps verifying on every call.
func TestManagerWithoutACacheVerifiesEveryCall(t *testing.T) {
	root := t.TempDir()
	asset := zipBytes(t, map[string]string{"launch.sh": "reviewed launcher"})
	server := serveAsset(t, asset)
	t.Cleanup(server.Close)
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	if _, err := manager.Apply(context.Background(), OpInstall, pkg); err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(root, "userdata/roms/tools/demo/launch.sh")
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("initial status = %#v, %v", status, err)
	}
	tamperLauncher(t, launcher)
	status, err := manager.Status(pkg.ID)
	if err != nil || status.Healthy || len(status.Issues) != 1 || status.Issues[0].Check != "content changed" {
		t.Fatalf("a cacheless manager skipped the health check: %#v, %v", status, err)
	}
}
