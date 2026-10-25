package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

func TestCheckStoreUpdatePinsOfficialTargetAsset(t *testing.T) {
	for _, test := range []struct{ name, target, current, url, digest, want string }{
		{"new prerelease", "magicx-zero-28", "old", storeReleaseBase + "v2/knulli-app-store-magicx-zero-28-experimental-new.zip", "sha256:" + strings.Repeat("a", 64), "v2"},
		{"current build", "magicx-zero-28", "new", storeReleaseBase + "v2/knulli-app-store-magicx-zero-28-experimental-new.zip", "sha256:" + strings.Repeat("a", 64), "current"},
		{"current tag", "magicx-zero-28", "v2", storeReleaseBase + "v2/knulli-app-store-magicx-zero-28-experimental-new.zip", "sha256:" + strings.Repeat("a", 64), "current"},
		{"older numbered release", "magicx-zero-28", "v3", storeReleaseBase + "v2/knulli-app-store-magicx-zero-28-experimental-new.zip", "sha256:" + strings.Repeat("a", 64), "current"},
		{"newer numbered release", "magicx-zero-28", "v1", storeReleaseBase + "v2/knulli-app-store-magicx-zero-28-experimental-new.zip", "sha256:" + strings.Repeat("a", 64), "v2"},
		{"equivalent numbered release", "magicx-zero-28", "2.0", storeReleaseBase + "v2/knulli-app-store-magicx-zero-28-experimental-new.zip", "sha256:" + strings.Repeat("a", 64), "current"},
		{"foreign asset", "magicx-zero-28", "old", "https://example.org/new.zip", "sha256:" + strings.Repeat("a", 64), "error"},
		{"missing digest", "magicx-zero-28", "old", storeReleaseBase + "v2/knulli-app-store-magicx-zero-28-experimental-new.zip", "", "error"},
		{"no target asset", "trimui-smart-pro", "old", storeReleaseBase + "v2/knulli-app-store-magicx-zero-28-experimental-new.zip", "sha256:" + strings.Repeat("a", 64), "error"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, `[{"tag_name":"v2","prerelease":true,"assets":[{"name":"knulli-app-store-magicx-zero-28-experimental-new.zip","browser_download_url":%q,"digest":%q,"size":123}]}]`, test.url, test.digest)
			}))
			defer server.Close()
			update, err := (Manager{Client: rewriteClient(t, server)}).CheckStoreUpdate(context.Background(), test.target, test.current)
			switch test.want {
			case "error":
				if err == nil {
					t.Fatal("unsafe or missing asset accepted")
				}
			case "current":
				if err != nil || update != nil {
					t.Fatalf("current build offered update: %#v, %v", update, err)
				}
			default:
				if err != nil || update == nil || update.Version != test.want || update.Release.URL != test.url || update.Release.Size != 123 || update.Release.SHA256 != strings.Repeat("a", 64) {
					t.Fatalf("incorrect pinned release: %#v, %v", update, err)
				}
			}
		})
	}
}

func TestStoreUpdatePreservesAppsAndRollsBackUnsafeReleases(t *testing.T) {
	for _, scenario := range []string{"success", "running UI", "checksum", "inventory", "symlink", "write failure", "space"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			success := scenario == "success" || scenario == "running UI"
			files := map[string]string{
				"Knulli App Store.sh":                     "new launcher",
				"images/Knulli App Store.png":             "new image",
				"knulli-app-store/knulli-app":             "new CLI",
				"knulli-app-store/knulli-app-ui":          "new UI",
				"knulli-app-store/catalog-index.json":     "new catalogue",
				"knulli-app-store/catalog-index.json.sig": "new signature",
				"knulli-app-store/README.txt":             "new readme",
			}
			for name := range files {
				writeRootFile(t, root, "userdata/roms/ports/"+name, "old "+name)
			}
			if scenario == "running UI" {
				sleep, err := exec.LookPath("sleep")
				if err != nil {
					t.Fatal(err)
				}
				ui := filepath.Join(root, "userdata/roms/ports/knulli-app-store/knulli-app-ui")
				if err := safefs.Copy(sleep, ui, 0755); err != nil {
					t.Fatal(err)
				}
				process := exec.Command(ui, "60")
				if err := process.Start(); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					process.Process.Kill()
					process.Wait()
				})
				defer func() {
					if err := process.Process.Signal(syscall.Signal(0)); err != nil {
						t.Errorf("running UI did not survive replacement: %v", err)
					}
				}()
			}
			writeRootFile(t, root, "userdata/system/configs/demo/settings", "keep settings")
			writeRootFile(t, root, "userdata/roms/ports/demo/app", "keep app")
			if scenario == "inventory" {
				files["knulli-app-store/unexpected"] = "extra"
			}
			asset := zipBytes(t, files)
			hash := sha256.Sum256(asset)
			update := StoreUpdate{Version: "v2", Release: manifest.Release{URL: storeReleaseBase + "v2/store.zip", Size: int64(len(asset)), SHA256: hex.EncodeToString(hash[:])}}
			if scenario == "checksum" {
				update.Release.SHA256 = strings.Repeat("0", 64)
			}
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(asset) }))
			defer server.Close()
			manager := Manager{Root: root, Client: rewriteClient(t, server)}
			if scenario == "space" {
				manager.AvailableBytes = func(string) (uint64, error) { return 0, nil }
			}
			blocked := filepath.Join(root, "userdata/roms/ports/knulli-app-store/README.txt")
			if scenario == "symlink" || scenario == "write failure" {
				if err := os.Remove(blocked); err != nil {
					t.Fatal(err)
				}
				if scenario == "symlink" {
					if err := os.Symlink(filepath.Join(root, "userdata/system/configs/demo/settings"), blocked); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Mkdir(blocked, 0755); err != nil {
					t.Fatal(err)
				}
			}
			err := manager.UpdateStore(context.Background(), update)
			if (err == nil) != success {
				t.Fatalf("update result: %v", err)
			}
			for name, content := range files {
				if name == "knulli-app-store/unexpected" {
					assertMissing(t, root, "userdata/roms/ports/"+name)
					continue
				}
				if name == "knulli-app-store/README.txt" && (scenario == "symlink" || scenario == "write failure") {
					continue
				}
				want := "old " + name
				if success {
					want = content
				}
				assertRootFile(t, root, "userdata/roms/ports/"+name, want)
			}
			assertRootFile(t, root, "userdata/system/configs/demo/settings", "keep settings")
			assertRootFile(t, root, "userdata/roms/ports/demo/app", "keep app")
			if success {
				info, err := os.Stat(filepath.Join(root, "userdata/roms/ports/knulli-app-store/knulli-app-ui"))
				if err != nil || info.Mode().Perm() != 0755 {
					t.Fatalf("UI execute mode: %v, %v", info, err)
				}
			}
		})
	}
}

func TestRecoveryDiscardsOnlyAbandonedStaging(t *testing.T) {
	root := t.TempDir()
	manager := Manager{Root: root}
	base := strings.TrimPrefix(manifest.ManagerStatePath, "/")
	for _, name := range []string{"work-interrupted", "store-update-interrupted", "originals", "recovery-backups"} {
		writeRootFile(t, root, base+"/"+name+"/release", "keep until recovery")
	}
	// A similarly named symlink is not installer staging and must not be followed.
	link := filepath.Join(root, base, "work-link")
	if err := os.Symlink(filepath.Join(root, base, "originals"), link); err != nil {
		t.Fatal(err)
	}
	lock, err := acquireLock(filepath.Join(root, base, "lock"))
	if err != nil {
		t.Fatal(err)
	}
	err = manager.RecoverTransactions()
	releaseLock(lock)
	if err == nil {
		t.Fatal("recovery ran while an operation held the manager lock")
	}
	assertRootFile(t, root, base+"/work-interrupted/release", "keep until recovery")
	assertRootFile(t, root, base+"/store-update-interrupted/release", "keep until recovery")
	for range 2 {
		if err := manager.RecoverTransactions(); err != nil {
			t.Fatal(err)
		}
		assertMissing(t, root, base+"/work-interrupted")
		assertMissing(t, root, base+"/store-update-interrupted")
		assertRootFile(t, root, base+"/originals/release", "keep until recovery")
		assertRootFile(t, root, base+"/recovery-backups/release", "keep until recovery")
		if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("recovery changed a non-staging symlink: %v, %v", info, err)
		}
	}
}

func TestRecoverInterruptedStoreUpdate(t *testing.T) {
	root := t.TempDir()
	manager := Manager{Root: root}
	if err := manager.RecoverTransactions(); err != nil {
		t.Fatal(err)
	}
	virtual := StoreDirectory + "/catalog-index.json"
	writeRootFile(t, root, strings.TrimPrefix(virtual, "/"), "old signed catalogue")
	guard, err := safefs.NewGuard(root, []string{StoreDirectory})
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := safefs.Begin(guard, filepath.Join(root, manifest.ManagerStatePath))
	if err != nil {
		t.Fatal(err)
	}
	if err := transaction.Write(virtual, []byte("partial new catalogue"), 0644); err != nil {
		t.Fatal(err)
	}
	// No commit or rollback: emulate process death before the next startup.
	if err := manager.RecoverTransactions(); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, strings.TrimPrefix(virtual, "/"), "old signed catalogue")
}
