package installer

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	storearchive "github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

func TestManageExistingRepairUpdateAndUninstallEndToEnd(t *testing.T) {
	root := t.TempDir()
	diagnosticLog, err := diagnostics.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original launcher")
	writeRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
	gamelist, err := os.ReadFile("testdata/gamelist.xml")
	if err != nil {
		t.Fatal(err)
	}
	writeRootFile(t, root, "userdata/roms/tools/gamelist.xml", string(gamelist))

	v1 := zipBytes(t, map[string]string{
		"launch.sh":    "version one",
		"config.ini":   "default configuration",
		"obsolete.txt": "remove on update",
	})
	v2 := zipBytes(t, map[string]string{
		"launch.sh":  "version two",
		"config.ini": "new default configuration",
		"added.txt":  "new file",
	})
	assets := map[string][]byte{"/example/demo/releases/download/v1/demo.zip": v1, "/example/demo/releases/download/v2/demo.zip": v2}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		data, found := assets[request.URL.Path]
		if !found {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
		writer.Write(data)
	}))
	defer server.Close()

	manager := Manager{Root: root, Client: rewriteClient(t, server), Diagnostics: diagnosticLog}.WithPlatform(testPlatform())
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", v1, "1.0.0")
	if _, err := manager.Apply(context.Background(), OpAdopt, pkg); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status(pkg.ID)
	if err != nil || !status.Installed || status.Healthy {
		t.Fatalf("changed external files should need repair after management: %#v, %v", status, err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original launcher")
	if _, err := manager.Apply(context.Background(), OpRepair, pkg); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "version one")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
	assertRootFile(t, root, "userdata/roms/tools/demo/obsolete.txt", "remove on update")
	assertContains(t, root, "userdata/roms/tools/gamelist.xml", "<image>./images/existing.png</image>")
	assertContains(t, root, "userdata/roms/tools/gamelist.xml", "<path>./demo/launch.sh</path>")

	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "damaged")
	status, err = manager.Status(pkg.ID)
	if err != nil || status.Healthy {
		t.Fatalf("damaged file was not detected: %#v, %v", status, err)
	}
	if _, err := manager.Apply(context.Background(), OpRepair, pkg); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "version one")

	updated := testPackage("https://github.com/example/demo/releases/download/v2/demo.zip", v2, "2.0.0")
	if _, err := manager.Apply(context.Background(), OpUpdate, updated); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "version two")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
	assertRootFile(t, root, "userdata/roms/tools/demo/added.txt", "new file")
	assertMissing(t, root, "userdata/roms/tools/demo/obsolete.txt")

	if _, err := manager.Uninstall(context.Background(), pkg.ID); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original launcher")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
	assertMissing(t, root, "userdata/roms/tools/demo/added.txt")
	assertMissing(t, root, "userdata/system/knulli-app-store/installed/org.example.demo.json")
	assertNotContains(t, root, "userdata/roms/tools/gamelist.xml", "./demo/launch.sh")
	assertContains(t, root, "userdata/roms/tools/gamelist.xml", "<name>Existing Tool</name>")
	logData, err := os.ReadFile(filepath.Join(root, "userdata/system/logs/knulli-app-store.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []string{"event=operation_start", "event=compatibility_allowed", "event=download_start", "event=download_verified", "event=extraction_complete", "event=transaction_begin", "event=backup_complete", "event=operation_complete"} {
		if !strings.Contains(string(logData), event) {
			t.Fatalf("operation log lacks %s: %s", event, logData)
		}
	}
}

func TestFailedGamelistUpdateRollsBackFiles(t *testing.T) {
	root := t.TempDir()
	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original")
	writeRootFile(t, root, "userdata/roms/tools/gamelist.xml", "<gameList><broken></gameList>")
	asset := zipBytes(t, map[string]string{"launch.sh": "replacement"})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	if _, err := manager.Apply(context.Background(), OpAdopt, pkg); err == nil || !strings.Contains(err.Error(), "parse gamelist") {
		t.Fatalf("expected gamelist failure, got %v", err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original")
	assertMissing(t, root, "userdata/system/knulli-app-store/installed/org.example.demo.json")
	assertMissing(t, root, strings.TrimPrefix(originalPath(pkg.ID, "/userdata/roms/tools/demo/launch.sh"), "/"))
}

func TestUninstallRemovesOwnedMenuEntryAndRefreshesGameList(t *testing.T) {
	root := t.TempDir()
	asset := zipBytes(t, map[string]string{"launch.sh": "managed"})
	assetServer := serveAsset(t, asset)
	defer assetServer.Close()
	refreshCalls := 0
	refreshServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		refreshCalls++
		if request.Method != http.MethodGet || request.URL.Path != "/reloadgames" {
			t.Fatalf("unexpected refresh request: %s %s", request.Method, request.URL.Path)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer refreshServer.Close()
	var outcomes []OperationOutcome
	manager := Manager{
		Root: root, Client: rewriteClient(t, assetServer),
		RefreshClient: refreshServer.Client(), RefreshURL: refreshServer.URL + "/reloadgames",
	}.WithPlatform(testPlatform())
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	outcome, err := manager.Apply(context.Background(), OpInstall, pkg)
	if err != nil {
		t.Fatal(err)
	}
	outcomes = append(outcomes, outcome)
	outcome, err = manager.Uninstall(context.Background(), pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	outcomes = append(outcomes, outcome)
	if refreshCalls != 2 || len(outcomes) != 2 || !outcomes[0].GameListRefreshAccepted || !outcomes[1].GameListRefreshAccepted {
		t.Fatalf("game-list refresh outcomes are wrong: calls=%d outcomes=%#v", refreshCalls, outcomes)
	}
	assertNotContains(t, root, "userdata/roms/tools/gamelist.xml", "./demo/launch.sh")
}

func TestSharedPreExistingMenuEntryIsNotOwnedOrRemoved(t *testing.T) {
	root := t.TempDir()
	writeRootFile(t, root, "userdata/roms/tools/gamelist.xml", `<gameList><game><path>./demo/launch.sh</path><name>Manual Demo</name><custom>keep</custom></game></gameList>`)
	asset := zipBytes(t, map[string]string{"launch.sh": "managed"})
	server := serveAsset(t, asset)
	defer server.Close()
	var outcomes []OperationOutcome
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	outcome, err := manager.Apply(context.Background(), OpInstall, pkg)
	if err != nil {
		t.Fatal(err)
	}
	outcomes = append(outcomes, outcome)
	baseGuard, err := safefs.NewGuard(root, []string{managerPath})
	if err != nil {
		t.Fatal(err)
	}
	state, err := loadState(baseGuard, pkg.ID)
	if err != nil || state.MenuOwned {
		t.Fatalf("manual menu entry became manager-owned: %#v, %v", state, err)
	}
	outcome, err = manager.Uninstall(context.Background(), pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	outcomes = append(outcomes, outcome)
	assertContains(t, root, "userdata/roms/tools/gamelist.xml", "<name>Manual Demo</name>")
	assertContains(t, root, "userdata/roms/tools/gamelist.xml", "<custom>keep</custom>")
	if len(outcomes) != 2 || outcomes[0].GameListChanged || outcomes[1].GameListChanged {
		t.Fatalf("unchanged shared menu requested refresh: %#v", outcomes)
	}
}

func TestUninstallGamelistFailureRollsBackManagedFilesAndState(t *testing.T) {
	root := t.TempDir()
	asset := zipBytes(t, map[string]string{"launch.sh": "managed"})
	server := serveAsset(t, asset)
	defer server.Close()
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	if _, err := manager.Apply(context.Background(), OpInstall, pkg); err != nil {
		t.Fatal(err)
	}
	writeRootFile(t, root, "userdata/roms/tools/gamelist.xml", "<gameList><broken></gameList>")
	if _, err := manager.Uninstall(context.Background(), pkg.ID); err == nil || !strings.Contains(err.Error(), "parse gamelist") {
		t.Fatalf("expected uninstall gamelist failure, got %v", err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "managed")
	if status, err := manager.Status(pkg.ID); err != nil || !status.Installed {
		t.Fatalf("uninstall rollback lost state: %#v, %v", status, err)
	}
}

func TestRefreshFailureRequiresRestartWithoutFailingOperation(t *testing.T) {
	root := t.TempDir()
	asset := zipBytes(t, map[string]string{"launch.sh": "managed"})
	assetServer := serveAsset(t, asset)
	defer assetServer.Close()
	refreshServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "unavailable", http.StatusServiceUnavailable)
	}))
	defer refreshServer.Close()
	manager := Manager{
		Root: root, Client: rewriteClient(t, assetServer),
		RefreshClient: refreshServer.Client(), RefreshURL: refreshServer.URL,
	}.WithPlatform(testPlatform())
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	outcome, err := manager.Apply(context.Background(), OpInstall, pkg)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.GameListChanged || !outcome.RestartRequired || outcome.GameListRefreshAccepted {
		t.Fatalf("failed refresh did not require restart: %#v", outcome)
	}
	if status, err := manager.Status(pkg.ID); err != nil || !status.Installed {
		t.Fatalf("refresh failure rolled back committed install: %#v, %v", status, err)
	}
}

func TestChecksumFailureWritesNoPackageFiles(t *testing.T) {
	root := t.TempDir()
	asset := zipBytes(t, map[string]string{"launch.sh": "replacement"})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	pkg.Release.SHA256 = strings.Repeat("0", 64)
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	if _, err := manager.Apply(context.Background(), OpInstall, pkg); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("expected checksum failure, got %v", err)
	}
	assertMissing(t, root, "userdata/roms/tools/demo/launch.sh")
}

func TestDeclarativeExecutablesAreRestoredAndRepaired(t *testing.T) {
	root := t.TempDir()
	asset := zipBytesWithModes(t, map[string]zipFixture{
		"launch.sh": {body: "launcher", mode: 0644},
		"tool":      {body: "binary", mode: 0644},
	})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	pkg.Install.Executables = []string{"launch.sh", "tool"}
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	if _, err := manager.Apply(context.Background(), OpInstall, pkg); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"launch.sh", "tool"} {
		path := filepath.Join(root, "userdata/roms/tools/demo", name)
		if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0755 {
			t.Fatalf("%s mode was not restored: %v, %v", name, info, err)
		}
	}
	tool := filepath.Join(root, "userdata/roms/tools/demo/tool")
	if err := os.Chmod(tool, 0644); err != nil {
		t.Fatal(err)
	}
	if status, err := manager.Status(pkg.ID); err != nil || status.Healthy {
		t.Fatalf("changed executable mode was not detected: %#v, %v", status, err)
	}
	if _, err := manager.Apply(context.Background(), OpRepair, pkg); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(tool); err != nil || info.Mode().Perm() != 0755 {
		t.Fatalf("repair did not restore executable mode: %v, %v", info, err)
	}
}

func TestPlayTimeFirstLaunchHealthRepairAndUninstallOnExperimentalDevices(t *testing.T) {
	asset := zipBytesWithModes(t, map[string]zipFixture{
		"PlayTime/icons/playtime.png": {body: "icon", mode: 0644},
		"PlayTime/playtime":           {body: "reviewed arm64 binary", mode: 0644},
		"PlayTime/playtime.sh":        {body: "reviewed launcher", mode: 0644},
	})
	server := serveAsset(t, asset)
	defer server.Close()

	for _, current := range []platform.Info{
		{Firmware: "knulli", Version: "scarab 2026/08/19 16:06", Arch: "aarch64", ABI: "linux-aarch64-glibc", GLIBCVersion: "2.40", Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}, Device: "trimui-smart-pro", Resolution: "1280x720"},
		{Firmware: "knulli", Version: "scarab 2026/08/19 16:06", Arch: "aarch64", ABI: "linux-aarch64-glibc", GLIBCVersion: "2.40", Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}, Device: "magicx-zero-28", Resolution: "640x480"},
	} {
		t.Run(current.Device, func(t *testing.T) {
			root := t.TempDir()
			logger, err := diagnostics.Open(root)
			if err != nil {
				t.Fatal(err)
			}
			pkg := playTimeTestPackage(t, asset)
			manager := Manager{Root: root, Client: rewriteClient(t, server), Diagnostics: logger}.WithPlatform(current)

			if _, err := manager.Apply(context.Background(), OpInstall, pkg); err != nil {
				t.Fatal(err)
			}
			baseGuard, err := safefs.NewGuard(root, []string{managerPath})
			if err != nil {
				t.Fatal(err)
			}
			state, err := loadState(baseGuard, pkg.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, file := range state.Files {
				info, err := os.Stat(filepath.Join(root, strings.TrimPrefix(file.Path, "/")))
				if err != nil || file.Mode != uint32(info.Mode().Perm()) {
					t.Fatalf("state did not record destination mode for %s: state=%04o info=%v err=%v", file.Path, file.Mode, info, err)
				}
			}

			// These files are created or changed by PlayTime at runtime and are not immutable release files.
			writeRootFile(t, root, "userdata/system/configs/playtime/playtime.db", "play statistics")
			writeRootFile(t, root, "userdata/system/scripts/playtime-hook.sh", "runtime hook")
			writeRootFile(t, root, "userdata/roms/tools/gamelist.xml", "<gameList></gameList>")
			status, err := manager.Status(pkg.ID)
			if err != nil || !status.Healthy || len(status.Issues) != 0 {
				t.Fatalf("normal first-launch files made PlayTime unhealthy: %#v, %v", status, err)
			}

			binary := filepath.Join(root, "userdata/roms/tools/PlayTime/playtime")
			if err := os.WriteFile(binary, []byte("corrupt"), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(binary, 0644); err != nil {
				t.Fatal(err)
			}
			status, err = manager.Status(pkg.ID)
			if err != nil || status.Healthy || len(status.Issues) < 2 || status.Issues[0].Check != "content changed" || status.Issues[1].Check != "mode changed" {
				t.Fatalf("immutable corruption lacks useful health reasons: %#v, %v", status, err)
			}
			if _, err := manager.Apply(context.Background(), OpRepair, pkg); err != nil {
				t.Fatal(err)
			}
			if status, err = manager.Status(pkg.ID); err != nil || !status.Healthy {
				t.Fatalf("repair did not restore health: %#v, %v", status, err)
			}
			if _, err := manager.Uninstall(context.Background(), pkg.ID); err != nil {
				t.Fatal(err)
			}
			assertRootFile(t, root, "userdata/system/configs/playtime/playtime.db", "play statistics")
			assertRootFile(t, root, "userdata/system/scripts/playtime-hook.sh", "runtime hook")
			logData, err := os.ReadFile(filepath.Join(root, "userdata/system/logs/knulli-app-store.log"))
			if err != nil {
				t.Fatal(err)
			}
			for _, wanted := range []string{"event=package_health_issue", `check="content changed"`, `check="mode changed"`, `path="/userdata/roms/tools/PlayTime/playtime"`} {
				if !strings.Contains(string(logData), wanted) {
					t.Fatalf("health log lacks %q: %s", wanted, logData)
				}
			}
		})
	}
}

func TestPlayTimeMagicXAdoptionAndUninstallPreserveExistingData(t *testing.T) {
	root := t.TempDir()
	writeRootFile(t, root, "userdata/roms/tools/PlayTime/local-note.txt", "existing note")
	writeRootFile(t, root, "userdata/system/configs/playtime/playtime.db", "existing statistics")
	asset := zipBytesWithModes(t, map[string]zipFixture{
		"PlayTime/icons/playtime.png": {body: "icon", mode: 0644},
		"PlayTime/playtime":           {body: "reviewed arm64 binary", mode: 0644},
		"PlayTime/playtime.sh":        {body: "reviewed launcher", mode: 0644},
	})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := playTimeTestPackage(t, asset)
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(platform.Info{Firmware: "knulli", Version: "scarab", Arch: "aarch64", ABI: "linux-aarch64-glibc", GLIBCVersion: "2.40", Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc"}, Device: "magicx-zero-28", Resolution: "640x480"})
	if existing, err := manager.PreExisting(pkg); err != nil || !existing {
		t.Fatalf("MagicX PlayTime copy was not offered for adoption: existing=%v err=%v", existing, err)
	}
	if _, err := manager.Apply(context.Background(), OpAdopt, pkg); err != nil {
		t.Fatal(err)
	}
	if status, err := manager.Status(pkg.ID); err != nil || status.Healthy {
		t.Fatalf("incomplete external PlayTime copy did not request repair: %#v, %v", status, err)
	}
	assertMissing(t, root, "userdata/roms/tools/PlayTime/playtime")
	if _, err := manager.Apply(context.Background(), OpRepair, pkg); err != nil {
		t.Fatal(err)
	}
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("repaired PlayTime was not healthy: %#v, %v", status, err)
	}
	if _, err := manager.Uninstall(context.Background(), pkg.ID); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/PlayTime/local-note.txt", "existing note")
	assertRootFile(t, root, "userdata/system/configs/playtime/playtime.db", "existing statistics")
}

func TestGroutPreviousVersionUpdateRepairRollbackAndUninstallPreserveState(t *testing.T) {
	root := t.TempDir()
	v51 := zipBytes(t, map[string]string{"Grout.sh": "launcher 5.1", "grout": "binary 5.1"})
	v52 := zipBytes(t, map[string]string{"Grout.sh": "launcher 5.2", "grout": "binary 5.2"})
	assets := map[string][]byte{"/example/demo/releases/download/v5.1/grout.zip": v51, "/example/demo/releases/download/v5.2/grout.zip": v52}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		data, ok := assets[request.URL.Path]
		if !ok {
			http.NotFound(writer, request)
			return
		}
		writer.Header().Set("Content-Length", strconv.Itoa(len(data)))
		writer.Write(data)
	}))
	defer server.Close()
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	old := groutTestPackage(v51, "https://github.com/example/demo/releases/download/v5.1/grout.zip", "5.1.0.0")
	if _, err := manager.Apply(context.Background(), OpInstall, old); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"config.json": "credentials", "save_slots.json": "slots", ".cache/grout.db": "cache", "logs/grout.log": "log"} {
		writeRootFile(t, root, "userdata/roms/tools/Grout/"+name, body)
	}
	current := groutTestPackage(v52, "https://github.com/example/demo/releases/download/v5.2/grout.zip", "5.2.0.0")
	writeRootFile(t, root, "userdata/roms/tools/gamelist.xml", "<gameList><broken></gameList>")
	if _, err := manager.Apply(context.Background(), OpUpdate, current); err == nil || !strings.Contains(err.Error(), "parse gamelist") {
		t.Fatalf("expected update rollback trigger, got %v", err)
	}
	assertRootFile(t, root, "userdata/roms/tools/Grout/grout", "binary 5.1")
	for name, body := range map[string]string{"config.json": "credentials", "save_slots.json": "slots", ".cache/grout.db": "cache", "logs/grout.log": "log"} {
		assertRootFile(t, root, "userdata/roms/tools/Grout/"+name, body)
	}
	writeRootFile(t, root, "userdata/roms/tools/gamelist.xml", "<gameList></gameList>")
	if _, err := manager.Apply(context.Background(), OpUpdate, current); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/Grout/grout", "binary 5.2")
	writeRootFile(t, root, "userdata/roms/tools/Grout/grout", "damaged")
	if _, err := manager.Apply(context.Background(), OpRepair, current); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/Grout/grout", "binary 5.2")
	if _, err := manager.Uninstall(context.Background(), current.ID); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"config.json": "credentials", "save_slots.json": "slots", ".cache/grout.db": "cache", "logs/grout.log": "log"} {
		assertRootFile(t, root, "userdata/roms/tools/Grout/"+name, body)
	}
}

func groutTestPackage(asset []byte, url, version string) manifest.Package {
	pkg := testPackage(url, asset, version)
	pkg.ID = "app.romm.grout"
	pkg.Name = "Grout"
	pkg.Install.Destination = "/userdata/roms/tools/Grout"
	pkg.Install.Launcher = "Grout.sh"
	pkg.Install.Executables = []string{"Grout.sh", "grout"}
	pkg.Install.Preserve = []string{"config.json", "save_slots.json", ".cache", "logs"}
	pkg.Install.AllowedWritePaths = []string{"/userdata/roms/tools/Grout", "/userdata/roms/tools/gamelist.xml"}
	pkg.Install.Menu = &manifest.Menu{Gamelist: "/userdata/roms/tools/gamelist.xml", Path: "./Grout/Grout.sh", Name: "Grout"}
	return pkg
}

func playTimeTestPackage(t *testing.T, asset []byte) manifest.Package {
	t.Helper()
	pkg, err := manifest.Load("../../catalogue/packages/io.github.unitreign.playtime.json")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(asset)
	pkg.Release.URL = "https://github.com/unitreign/playtime/releases/download/v1.0.0/PlayTime.zip"
	pkg.Release.SHA256 = hex.EncodeToString(digest[:])
	pkg.Release.Size = int64(len(asset))
	pkg.Release.InstalledSize = int64(len("icon") + len("reviewed arm64 binary") + len("reviewed launcher"))
	return pkg
}

func TestAdoptBacksUpExistingFilesAndUninstallRestoresThem(t *testing.T) {
	root := t.TempDir()
	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "old launcher")
	writeRootFile(t, root, "userdata/roms/tools/demo/config.ini", "old credentials")
	writeRootFile(t, root, "userdata/roms/tools/demo/local-config.ini", "local credentials")
	writeRootFile(t, root, "userdata/roms/tools/demo/tool", "reviewed binary")
	writeRootFile(t, root, "userdata/roms/tools/demo/unknown/cache.db", "old cache")
	writeRootFile(t, root, "userdata/roms/tools/demo/logs/session.log", "old log")
	writeRootFile(t, root, "userdata/system/configs/playtime/playtime.db", "statistics")
	asset := zipBytes(t, map[string]string{
		"launch.sh":  "reviewed launcher",
		"config.ini": "default configuration",
		"tool":       "reviewed binary",
	})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	pkg.Install.Preserve = append(pkg.Install.Preserve, "local-config.ini", "logs")
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	preExisting, err := manager.PreExisting(pkg)
	if err != nil || !preExisting {
		t.Fatalf("existing package was not detected: %v, %v", preExisting, err)
	}
	if _, err := manager.Apply(context.Background(), OpAdopt, pkg); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Apply(context.Background(), OpAdopt, pkg); err == nil || !strings.Contains(err.Error(), "already installed") {
		t.Fatalf("repeated install was not rejected: %v", err)
	}
	baseGuard, err := safefs.NewGuard(root, []string{managerPath})
	if err != nil {
		t.Fatal(err)
	}
	state, err := loadState(baseGuard, pkg.ID)
	if err != nil || len(state.Originals) != 6 {
		t.Fatalf("pre-existing inventory was not saved: %#v, %v", state, err)
	}
	for _, backup := range state.Originals {
		if host, err := baseGuard.Resolve(backup); err != nil {
			t.Fatal(err)
		} else if info, err := os.Stat(host); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("missing adoption backup %s: %v", backup, err)
		}
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "old launcher")
	if _, err := manager.Apply(context.Background(), OpRepair, pkg); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "reviewed launcher")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "old credentials")
	writeRootFile(t, root, "userdata/roms/tools/demo/config.ini", "updated credentials")
	writeRootFile(t, root, "userdata/roms/tools/demo/local-config.ini", "updated local credentials")
	writeRootFile(t, root, "userdata/roms/tools/demo/unknown/cache.db", "changed cache")
	writeRootFile(t, root, "userdata/roms/tools/demo/logs/session.log", "new log")
	if _, err := manager.Uninstall(context.Background(), pkg.ID); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "old launcher")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "updated credentials")
	assertRootFile(t, root, "userdata/roms/tools/demo/local-config.ini", "updated local credentials")
	assertRootFile(t, root, "userdata/roms/tools/demo/unknown/cache.db", "changed cache")
	assertRootFile(t, root, "userdata/roms/tools/demo/logs/session.log", "new log")
	assertRootFile(t, root, "userdata/system/configs/playtime/playtime.db", "statistics")
	assertRootFile(t, root, "userdata/roms/tools/demo/tool", "reviewed binary")
}

func TestAdoptionRejectsNonRegularExistingPaths(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "userdata/roms/tools/demo")
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("missing", filepath.Join(destination, "unsafe-link")); err != nil {
		t.Fatal(err)
	}
	asset := zipBytes(t, map[string]string{"launch.sh": "reviewed"})
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	if _, err := (Manager{Root: root}).WithPlatform(testPlatform()).Apply(context.Background(), OpInstall, pkg); err == nil || !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("expected unsafe adoption rejection, got %v", err)
	}
}

func TestInstallRejectsAnExternalCopyAndNamesTheOnScreenAction(t *testing.T) {
	root := t.TempDir()
	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "external copy")
	asset := zipBytes(t, map[string]string{"launch.sh": "reviewed launcher"})
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	_, err := (Manager{Root: root}).WithPlatform(testPlatform()).Apply(context.Background(), OpInstall, pkg)
	want := "an external installation exists; use Manage existing"
	if err == nil || err.Error() != want {
		t.Fatalf("external copy error = %v, want %q", err, want)
	}
}

func TestForceReinstallBacksUpEverythingPreservesDataAndCanRepeat(t *testing.T) {
	root := t.TempDir()
	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "damaged launcher")
	writeRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
	writeRootFile(t, root, "userdata/roms/tools/demo/unknown/cache.db", "unknown data")
	asset := zipBytes(t, map[string]string{"launch.sh": "reviewed launcher", "config.ini": "default configuration"})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	now := time.Date(2026, 9, 15, 23, 1, 2, 3, time.UTC)
	refreshes := 0
	refreshServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		refreshes++
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer refreshServer.Close()
	manager := Manager{Root: root, Client: rewriteClient(t, server), Now: func() time.Time { return now }, RefreshClient: refreshServer.Client(), RefreshURL: refreshServer.URL}.WithPlatform(testPlatform())
	if _, err := manager.Apply(context.Background(), OpForceReinstall, pkg); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "reviewed launcher")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
	assertRootFile(t, root, "userdata/roms/tools/demo/unknown/cache.db", "unknown data")
	backupDirectory := filepath.Join(root, "userdata/system/knulli-app-store/recovery-backups/org.example.demo/20260915T230102.000000003Z")
	data, err := os.ReadFile(filepath.Join(backupDirectory, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var backup recoveryBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		t.Fatal(err)
	}
	if backup.PackageID != pkg.ID || len(backup.Files) != 3 {
		t.Fatalf("recovery inventory is incomplete: %#v", backup)
	}
	for _, file := range backup.Files {
		if file.OriginalPath == "" || file.SHA256 == "" || file.BackupPath == "" {
			t.Fatalf("recovery entry lacks path/hash evidence: %#v", file)
		}
		assertRootFile(t, root, strings.TrimPrefix(file.BackupPath, "/"), map[string]string{
			"/userdata/roms/tools/demo/launch.sh":        "damaged launcher",
			"/userdata/roms/tools/demo/config.ini":       "user configuration",
			"/userdata/roms/tools/demo/unknown/cache.db": "unknown data",
		}[file.OriginalPath])
	}
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("force reinstall did not create clean state: %#v, %v", status, err)
	}
	now = now.Add(time.Second)
	if _, err := manager.Apply(context.Background(), OpForceReinstall, pkg); err != nil {
		t.Fatal(err)
	}
	backups, err := filepath.Glob(filepath.Join(root, "userdata/system/knulli-app-store/recovery-backups/org.example.demo/*/manifest.json"))
	if err != nil || len(backups) != 2 {
		t.Fatalf("repeated force reinstall did not retain separate backups: %v, %v", backups, err)
	}
	if refreshes != 2 {
		t.Fatalf("game list refresh count = %d, want 2 for two proven owned-entry writes", refreshes)
	}
}

func TestForceReinstallMandatoryChecksCannotBeBypassed(t *testing.T) {
	asset := zipBytes(t, map[string]string{"launch.sh": "reviewed launcher"})
	tests := []struct {
		name   string
		change func(*manifest.Package, *Manager, string)
		want   string
	}{
		{name: "checksum", change: func(pkg *manifest.Package, _ *Manager, _ string) { pkg.Release.SHA256 = strings.Repeat("0", 64) }, want: "SHA-256 mismatch"},
		{name: "compatibility", change: func(_ *manifest.Package, manager *Manager, _ string) { manager.platform.Arch = "x86_64" }, want: "field=architecture"},
		{name: "insufficient space", change: func(_ *manifest.Package, manager *Manager, _ string) {
			manager.AvailableBytes = func(string) (uint64, error) { return 1, nil }
		}, want: "not enough free space"},
		{name: "patch source", change: func(pkg *manifest.Package, _ *Manager, _ string) {
			pkg.Install.BinaryPatches = []manifest.BinaryPatch{{Path: "launch.sh", Offset: 0, BeforeHex: "00", AfterHex: "01", SHA256: strings.Repeat("a", 64)}}
		}, want: "binary patch source mismatch"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original")
			server := serveAsset(t, asset)
			defer server.Close()
			pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
			manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
			test.change(&pkg, &manager, root)
			_, err := manager.Apply(context.Background(), OpForceReinstall, pkg)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("force reinstall bypassed %s: %v", test.name, err)
			}
			assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original")
			backups, _ := filepath.Glob(filepath.Join(root, "userdata/system/knulli-app-store/recovery-backups/org.example.demo/*"))
			if len(backups) != 0 {
				t.Fatalf("failed pre-write check retained a backup: %v", backups)
			}
		})
	}
}

func TestBinaryPatchAppliesReviewedBytesAndFinalHash(t *testing.T) {
	before := []byte("prefix-https://grout.romm.app/versions.json-suffix")
	afterURL := "https://self-update.disabled.invalid"
	expected := []byte("prefix-" + afterURL + "-suffix")
	digest := sha256.Sum256(expected)
	host := filepath.Join(t.TempDir(), "grout")
	if err := os.WriteFile(host, before, 0755); err != nil {
		t.Fatal(err)
	}
	files := []storearchive.File{{Path: host, Relative: "grout", Mode: 0755}}
	patches := []manifest.BinaryPatch{{
		Path: "grout", Offset: int64(len("prefix-")),
		BeforeHex: hex.EncodeToString([]byte("https://grout.romm.app/versions.json")),
		AfterHex:  hex.EncodeToString([]byte(afterURL)), SHA256: hex.EncodeToString(digest[:]),
	}}
	if err := applyBinaryPatches(files, patches); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(host)
	if err != nil || !bytes.Equal(data, expected) || files[0].SHA256 != patches[0].SHA256 {
		t.Fatalf("verified updater patch result = %q, sha=%s, err=%v", data, files[0].SHA256, err)
	}
}

func TestGroutAdoptionUsesTransformedHashAndEffectiveDestinationMode(t *testing.T) {
	root := t.TempDir()
	before := []byte("prefix-https://grout.romm.app/versions.json-suffix")
	after := []byte("prefix-https://self-update.disabled.invalid-suffix")
	asset := zipBytesWithModes(t, map[string]zipFixture{
		"Grout/grout":    {body: string(before), mode: 0644},
		"Grout/Grout.sh": {body: "launcher", mode: 0644},
	})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := groutTestPackage(asset, "https://github.com/example/demo/releases/download/v5.2/grout.zip", "5.2.0.0")
	pkg.Install.StripComponents = 1
	digest := sha256.Sum256(after)
	pkg.Install.BinaryPatches = []manifest.BinaryPatch{{
		Path: "grout", Offset: int64(len("prefix-")),
		BeforeHex: hex.EncodeToString([]byte("https://grout.romm.app/versions.json")),
		AfterHex:  hex.EncodeToString([]byte("https://self-update.disabled.invalid")),
		SHA256:    hex.EncodeToString(digest[:]),
	}}
	writeRootFile(t, root, "userdata/roms/tools/Grout/grout", string(after))
	writeRootFile(t, root, "userdata/roms/tools/Grout/Grout.sh", "launcher")
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	if _, err := manager.Apply(context.Background(), OpAdopt, pkg); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status(pkg.ID)
	if err != nil || !status.Healthy {
		t.Fatalf("reviewed transformed Grout files were not healthy: %#v, %v", status, err)
	}
	baseGuard, err := safefs.NewGuard(root, []string{managerPath})
	if err != nil {
		t.Fatal(err)
	}
	state, err := loadState(baseGuard, pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range state.Files {
		info, statErr := os.Stat(filepath.Join(root, strings.TrimPrefix(file.Path, "/")))
		if statErr != nil || file.Mode != uint32(info.Mode().Perm()) {
			t.Fatalf("adoption mode was not normalized for %s: state=%04o info=%v err=%v", file.Path, file.Mode, info, statErr)
		}
	}
	binary := filepath.Join(root, "userdata/roms/tools/Grout/grout")
	if err := os.WriteFile(binary, []byte("changed transformed binary"), 0755); err != nil {
		t.Fatal(err)
	}
	changedDigest := sha256.Sum256([]byte("changed transformed binary"))
	status, err = manager.Status(pkg.ID)
	if err != nil || status.Healthy || len(status.Issues) == 0 || status.Issues[0].Expected != pkg.Install.BinaryPatches[0].SHA256 || status.Issues[0].Actual != hex.EncodeToString(changedDigest[:]) {
		t.Fatalf("Grout health did not report full transformed hashes: %#v, %v", status, err)
	}
	if _, err := manager.Apply(context.Background(), OpRepair, pkg); err != nil {
		t.Fatal(err)
	}
	if status, err = manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("Grout repair did not restore transformed health: %#v, %v", status, err)
	}
}

func TestLegacyGroutInstalledStateMigratesToCurrentRuntimeMetadata(t *testing.T) {
	root := t.TempDir()
	pkg, err := manifest.Load("../../catalogue/packages/app.romm.grout.json")
	if err != nil {
		t.Fatal(err)
	}
	pkg.Version = "5.1.0.0"
	pkg.Review.Status = "verified"
	pkg.Compatibility.MinimumVersion = "scarab"
	pkg.Compatibility.ABIs = nil
	pkg.Compatibility.Dependencies = nil
	pkg.Compatibility.DeviceScope = ""
	pkg.Compatibility.DisplayBounds = nil
	pkg.Compatibility.Devices = []string{"trimui-smart-pro"}
	pkg.Compatibility.Resolutions = []string{"1280x720"}
	pkg.Install.BinaryPatches = nil
	state := Installed{Schema: "org.knulli.app-store/installed-state/v1", Manifest: pkg, Originals: map[string]string{}}
	data, err := encodeState(state)
	if err != nil {
		t.Fatal(err)
	}
	writeRootFile(t, root, "userdata/system/knulli-app-store/installed/app.romm.grout.json", string(data))
	status, err := (Manager{Root: root}).Status(pkg.ID)
	if err != nil || !status.Installed || status.Version != "5.1.0.0" {
		t.Fatalf("legacy Grout state did not remain updateable: %#v, %v", status, err)
	}
}

func TestLifecycleStatePersistsOperationContextAcrossRestart(t *testing.T) {
	root := t.TempDir()
	first := Manager{Root: root}
	want := LifecycleState{PackageID: "org.example.demo", RequestedOperation: "install", DetectedInstallType: "absent", RetryTarget: "install", Failure: "download release: SHA-256 mismatch"}
	if err := first.RecordLifecycle(want); err != nil {
		t.Fatal(err)
	}
	got, err := (Manager{Root: root}).LifecycleState(want.PackageID)
	if err != nil || got == nil || got.RequestedOperation != want.RequestedOperation || got.DetectedInstallType != want.DetectedInstallType || got.RetryTarget != want.RetryTarget || got.Failure != want.Failure {
		t.Fatalf("lifecycle context did not survive restart: %#v, %v", got, err)
	}
	if err := first.ClearLifecycle(want.PackageID); err != nil {
		t.Fatal(err)
	}
	if got, err := first.LifecycleState(want.PackageID); err != nil || got != nil {
		t.Fatalf("completed lifecycle state was not cleared: %#v, %v", got, err)
	}
}

func TestForceReinstallRejectsArchiveTraversalAndLinks(t *testing.T) {
	for name, mode := range map[string]os.FileMode{"../outside": 0644, "unsafe-link": os.ModeSymlink | 0777} {
		t.Run(strings.ReplaceAll(name, "/", "_"), func(t *testing.T) {
			root := t.TempDir()
			writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original")
			asset := zipBytesWithModes(t, map[string]zipFixture{name: {body: "unsafe", mode: mode}})
			server := serveAsset(t, asset)
			defer server.Close()
			pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
			manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
			if _, err := manager.Apply(context.Background(), OpForceReinstall, pkg); err == nil {
				t.Fatal("unsafe archive passed force-reinstall validation")
			}
			assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original")
		})
	}
}

func TestForceReinstallRollbackAndPowerLossRecoveryRestoreExactState(t *testing.T) {
	root := t.TempDir()
	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "before crash")
	writeRootFile(t, root, "userdata/roms/tools/gamelist.xml", "<gameList><broken></gameList>")
	guard, err := safefs.NewGuard(root, []string{managerPath, "/userdata/roms/tools/demo"})
	if err != nil {
		t.Fatal(err)
	}
	managerHost, err := guard.Resolve(managerPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(managerHost, 0700); err != nil {
		t.Fatal(err)
	}
	crashed, err := safefs.Begin(guard, managerHost)
	if err != nil {
		t.Fatal(err)
	}
	if err := crashed.Write("/userdata/roms/tools/demo/launch.sh", []byte("partial adoption"), 0755); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "partial adoption")
	asset := zipBytes(t, map[string]string{"launch.sh": "reviewed launcher"})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	manager := Manager{Root: root, Client: rewriteClient(t, server), Now: func() time.Time { return time.Date(2026, 9, 15, 23, 2, 0, 0, time.UTC) }}.WithPlatform(testPlatform())
	if _, err := manager.Apply(context.Background(), OpForceReinstall, pkg); err == nil || !strings.Contains(err.Error(), "parse gamelist") {
		t.Fatalf("expected recovered transaction followed by transactional failure, got %v", err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "before crash")
	assertRootFile(t, root, "userdata/roms/tools/gamelist.xml", "<gameList><broken></gameList>")
	assertMissing(t, root, "userdata/system/knulli-app-store/installed/org.example.demo.json")
	backups, _ := filepath.Glob(filepath.Join(root, "userdata/system/knulli-app-store/recovery-backups/org.example.demo/*"))
	if len(backups) != 0 {
		t.Fatalf("rolled-back recovery backup survived: %v", backups)
	}
}

func TestLockedOperationRecoversCrashedTransaction(t *testing.T) {
	root := t.TempDir()
	guard, err := safefs.NewGuard(root, []string{managerPath})
	if err != nil {
		t.Fatal(err)
	}
	managerHost, err := guard.Resolve(managerPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(managerHost, 0700); err != nil {
		t.Fatal(err)
	}
	writeRootFile(t, root, "userdata/system/knulli-app-store/existing", "before")
	tx, err := safefs.Begin(guard, managerHost)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Write(managerPath+"/existing", []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.Write(managerPath+"/crashed", []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/system/knulli-app-store/existing", "after")
	assertRootFile(t, root, "userdata/system/knulli-app-store/crashed", "partial")

	asset := zipBytes(t, map[string]string{"launch.sh": "reviewed launcher"})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	manager := Manager{Root: root, Client: rewriteClient(t, server)}.WithPlatform(testPlatform())
	if _, err := manager.Apply(context.Background(), OpInstall, pkg); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/system/knulli-app-store/existing", "before")
	if _, err := os.Stat(filepath.Join(root, "userdata/system/knulli-app-store/crashed")); !os.IsNotExist(err) {
		t.Fatalf("crashed file survived recovery: %v", err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "reviewed launcher")
}

func TestRecoveryStatusDistinguishesInterruptedAndActiveTransactions(t *testing.T) {
	root := t.TempDir()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", zipBytes(t, map[string]string{"launch.sh": "reviewed"}), "1.0.0")
	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "before")
	guard, err := safefs.NewGuard(root, []string{managerPath, pkg.Install.Destination})
	if err != nil {
		t.Fatal(err)
	}
	managerHost, err := guard.Resolve(managerPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(managerHost, 0700); err != nil {
		t.Fatal(err)
	}
	tx, err := safefs.Begin(guard, managerHost)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Write(pkg.Install.Destination+"/launch.sh", []byte("partial"), 0755); err != nil {
		t.Fatal(err)
	}
	manager := Manager{Root: root}
	status, err := manager.RecoveryStatus(pkg)
	if err != nil || !status.ForceAllowed || status.Active || !strings.Contains(status.Reason, "Interrupted") {
		t.Fatalf("stale transaction status = %#v, %v", status, err)
	}
	lock, err := acquireLock(filepath.Join(managerHost, "lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer releaseLock(lock)
	status, err = manager.RecoveryStatus(pkg)
	if err != nil || status.ForceAllowed || !status.Active || status.Reason != "another package operation is active" {
		t.Fatalf("active transaction status = %#v, %v", status, err)
	}
}

func TestCandidateCannotBeInstalled(t *testing.T) {
	pkg := testPackage("https://example.com/releases/download/v1/demo.zip", []byte("x"), "1.0.0")
	pkg.Review.Status = "candidate"
	pkg.Release = nil
	pkg.Compatibility = nil
	pkg.Install = nil
	if _, err := (Manager{}).Apply(context.Background(), OpInstall, pkg); err == nil || !strings.Contains(err.Error(), "candidate") {
		t.Fatalf("expected candidate rejection, got %v", err)
	}
}

func TestApplyRejectsUnsupportedOperation(t *testing.T) {
	pkg := testPackage("https://example.com/releases/download/v1/demo.zip", []byte("x"), "1.0.0")
	_, err := (Manager{}).WithPlatform(testPlatform()).Apply(context.Background(), Op("uninstall"), pkg)
	if err == nil || !strings.Contains(err.Error(), "unsupported lifecycle operation") {
		t.Fatalf("expected unsupported operation rejection, got %v", err)
	}
}

func TestWithPlatformCopiesSlices(t *testing.T) {
	info := testPlatform().WithCandidates([]platform.ResolutionCandidate{{Source: "framebuffer", Width: 640, Height: 480}})
	manager := (Manager{}).WithPlatform(info)
	info.Dependencies[0] = "mutated"
	candidates := info.ResolutionCandidates()
	candidates[0].Source = "mutated"
	bound := manager.Platform()
	if bound.Dependencies[0] == "mutated" || bound.ResolutionCandidates()[0].Source == "mutated" {
		t.Fatalf("bound platform aliases caller slices: %#v", bound)
	}
}

func testPackage(url string, asset []byte, version string) manifest.Package {
	digest := sha256.Sum256(asset)
	installedSize := int64(0)
	reader, err := zip.NewReader(bytes.NewReader(asset), int64(len(asset)))
	if err == nil {
		for _, file := range reader.File {
			installedSize += int64(file.UncompressedSize64)
		}
	}
	if installedSize == 0 {
		installedSize = 1
	}
	return manifest.Package{
		Schema: manifest.SchemaV1, ID: "org.example.demo", Name: "Demo", Version: version,
		Type: "utility", Summary: "Test fixture.", Repository: "https://github.com/example/demo", License: "MIT",
		Review:        manifest.Review{Status: "installable"},
		Release:       &manifest.Release{URL: url, SHA256: hex.EncodeToString(digest[:]), Size: int64(len(asset)), InstalledSize: installedSize, Format: "zip", Immutable: true},
		Compatibility: &manifest.Compatibility{Firmware: "knulli", MinimumVersion: "2025.1", Architectures: []string{"aarch64"}, ABIs: []string{"linux-aarch64-glibc"}, Dependencies: []string{"sdl2"}, Devices: []string{"h700"}, Resolutions: []string{"640x480"}},
		Install: &manifest.Install{
			Destination: "/userdata/roms/tools/demo", Launcher: "launch.sh", Preserve: []string{"config.ini"},
			AllowedWritePaths: []string{"/userdata/roms/tools/demo", "/userdata/roms/tools/gamelist.xml"},
			Menu:              &manifest.Menu{Gamelist: "/userdata/roms/tools/gamelist.xml", Path: "./demo/launch.sh", Name: "Demo", Description: "Fixture utility."},
		},
	}
}

func testPlatform() platform.Info {
	return platform.Info{Firmware: "knulli", Version: "2025.2", Arch: "aarch64", ABI: "linux-aarch64-glibc", GLIBCVersion: "2.40", Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}, Device: "h700", Resolution: "640x480"}
}

func serveAsset(t *testing.T, asset []byte) *httptest.Server {
	t.Helper()
	return httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", strconv.Itoa(len(asset)))
		writer.Write(asset)
	}))
}

type rewriteTransport struct {
	base   http.RoundTripper
	scheme string
	host   string
}

func (transport rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.URL.Scheme = transport.scheme
	clone.URL.Host = transport.host
	clone.Host = transport.host
	return transport.base.RoundTrip(clone)
}

func rewriteClient(t *testing.T, server *httptest.Server) *http.Client {
	t.Helper()
	client := server.Client()
	parts := strings.SplitN(strings.TrimPrefix(server.URL, "https://"), "/", 2)
	client.Transport = rewriteTransport{base: client.Transport, scheme: "https", host: parts[0]}
	return client
}

func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetMode(0755)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

type zipFixture struct {
	body string
	mode os.FileMode
}

func zipBytesWithModes(t *testing.T, files map[string]zipFixture) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, fixture := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetMode(fixture.mode)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, fixture.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func writeRootFile(t *testing.T, root, relative, body string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func assertRootFile(t *testing.T, root, relative, expected string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != expected {
		t.Fatalf("%s = %q, expected %q", relative, data, expected)
	}
}

func assertMissing(t *testing.T, root, relative string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, relative)); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, got %v", relative, err)
	}
}

func assertContains(t *testing.T, root, relative, wanted string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), wanted) {
		t.Fatalf("%s does not contain %q:\n%s", relative, wanted, data)
	}
}

func assertNotContains(t *testing.T, root, relative, unwanted string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), unwanted) {
		t.Fatalf("%s contains %q:\n%s", relative, unwanted, data)
	}
}
