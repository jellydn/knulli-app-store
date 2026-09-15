package installer

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

func TestInstallRepairUpdateAndUninstallEndToEnd(t *testing.T) {
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

	manager := Manager{Root: root, Platform: testPlatform(), Client: rewriteClient(t, server), Diagnostics: diagnosticLog}
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", v1, "1.0.0")
	if err := manager.Install(context.Background(), pkg); err != nil {
		t.Fatal(err)
	}
	status, err := manager.Status(pkg.ID)
	if err != nil || !status.Installed || !status.Healthy {
		t.Fatalf("unexpected installed status: %#v, %v", status, err)
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
	if err := manager.Repair(context.Background(), pkg); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "version one")

	updated := testPackage("https://github.com/example/demo/releases/download/v2/demo.zip", v2, "2.0.0")
	if err := manager.Update(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "version two")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
	assertRootFile(t, root, "userdata/roms/tools/demo/added.txt", "new file")
	assertMissing(t, root, "userdata/roms/tools/demo/obsolete.txt")

	if err := manager.Uninstall(pkg.ID); err != nil {
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
	manager := Manager{Root: root, Platform: testPlatform(), Client: rewriteClient(t, server)}
	if err := manager.Install(context.Background(), pkg); err == nil || !strings.Contains(err.Error(), "parse gamelist") {
		t.Fatalf("expected gamelist failure, got %v", err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "original")
	assertMissing(t, root, "userdata/system/knulli-app-store/installed/org.example.demo.json")
	assertMissing(t, root, strings.TrimPrefix(originalPath(pkg.ID, "/userdata/roms/tools/demo/launch.sh"), "/"))
}

func TestChecksumFailureWritesNoPackageFiles(t *testing.T) {
	root := t.TempDir()
	asset := zipBytes(t, map[string]string{"launch.sh": "replacement"})
	server := serveAsset(t, asset)
	defer server.Close()
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", asset, "1.0.0")
	pkg.Release.SHA256 = strings.Repeat("0", 64)
	manager := Manager{Root: root, Platform: testPlatform(), Client: rewriteClient(t, server)}
	if err := manager.Install(context.Background(), pkg); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
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
	manager := Manager{Root: root, Platform: testPlatform(), Client: rewriteClient(t, server)}
	if err := manager.Install(context.Background(), pkg); err != nil {
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
	if err := manager.Repair(context.Background(), pkg); err != nil {
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
		{Firmware: "knulli", Version: "scarab 2026/08/19 16:06", Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1280x720"},
		{Firmware: "knulli", Version: "scarab 2026/08/19 16:06", Arch: "aarch64", Device: "magicx-zero-28", Resolution: "640x480"},
	} {
		t.Run(current.Device, func(t *testing.T) {
			root := t.TempDir()
			logger, err := diagnostics.Open(root)
			if err != nil {
				t.Fatal(err)
			}
			pkg := playTimeTestPackage(t, asset)
			manager := Manager{Root: root, Platform: current, Client: rewriteClient(t, server), Diagnostics: logger}

			if err := manager.Install(context.Background(), pkg); err != nil {
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
			if err := manager.Repair(context.Background(), pkg); err != nil {
				t.Fatal(err)
			}
			if status, err = manager.Status(pkg.ID); err != nil || !status.Healthy {
				t.Fatalf("repair did not restore health: %#v, %v", status, err)
			}
			if err := manager.Uninstall(pkg.ID); err != nil {
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
	manager := Manager{Root: root, Platform: platform.Info{Firmware: "knulli", Version: "scarab", Arch: "aarch64", Device: "magicx-zero-28", Resolution: "640x480"}, Client: rewriteClient(t, server)}
	if existing, err := manager.PreExisting(pkg); err != nil || !existing {
		t.Fatalf("MagicX PlayTime copy was not offered for adoption: existing=%v err=%v", existing, err)
	}
	if err := manager.Install(context.Background(), pkg); err != nil {
		t.Fatal(err)
	}
	if status, err := manager.Status(pkg.ID); err != nil || !status.Healthy {
		t.Fatalf("adopted PlayTime was not healthy: %#v, %v", status, err)
	}
	if err := manager.Uninstall(pkg.ID); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/PlayTime/local-note.txt", "existing note")
	assertRootFile(t, root, "userdata/system/configs/playtime/playtime.db", "existing statistics")
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
	manager := Manager{Root: root, Platform: testPlatform(), Client: rewriteClient(t, server)}
	preExisting, err := manager.PreExisting(pkg)
	if err != nil || !preExisting {
		t.Fatalf("existing package was not detected: %v, %v", preExisting, err)
	}
	if err := manager.Install(context.Background(), pkg); err != nil {
		t.Fatal(err)
	}
	if err := manager.Install(context.Background(), pkg); err == nil || !strings.Contains(err.Error(), "already installed") {
		t.Fatalf("repeated install was not rejected: %v", err)
	}
	baseGuard, err := safefs.NewGuard(root, []string{managerPath})
	if err != nil {
		t.Fatal(err)
	}
	state, err := loadState(baseGuard, pkg.ID)
	if err != nil || len(state.Originals) != 5 {
		t.Fatalf("pre-existing inventory was not saved: %#v, %v", state, err)
	}
	for _, backup := range state.Originals {
		if host, err := baseGuard.Resolve(backup); err != nil {
			t.Fatal(err)
		} else if info, err := os.Stat(host); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("missing adoption backup %s: %v", backup, err)
		}
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "reviewed launcher")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "old credentials")
	writeRootFile(t, root, "userdata/roms/tools/demo/config.ini", "updated credentials")
	writeRootFile(t, root, "userdata/roms/tools/demo/local-config.ini", "updated local credentials")
	writeRootFile(t, root, "userdata/roms/tools/demo/unknown/cache.db", "changed cache")
	writeRootFile(t, root, "userdata/roms/tools/demo/logs/session.log", "new log")
	if err := manager.Uninstall(pkg.ID); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "old launcher")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "updated credentials")
	assertRootFile(t, root, "userdata/roms/tools/demo/local-config.ini", "updated local credentials")
	assertRootFile(t, root, "userdata/roms/tools/demo/unknown/cache.db", "old cache")
	assertRootFile(t, root, "userdata/roms/tools/demo/logs/session.log", "new log")
	assertRootFile(t, root, "userdata/system/configs/playtime/playtime.db", "statistics")
	assertMissing(t, root, "userdata/roms/tools/demo/tool")
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
	if err := (Manager{Root: root, Platform: testPlatform()}).Install(context.Background(), pkg); err == nil || !strings.Contains(err.Error(), "non-regular") {
		t.Fatalf("expected unsafe adoption rejection, got %v", err)
	}
}

func TestCandidateCannotBeInstalled(t *testing.T) {
	pkg := testPackage("https://example.com/releases/download/v1/demo.zip", []byte("x"), "1.0.0")
	pkg.Review.Status = "candidate"
	pkg.Release = nil
	pkg.Compatibility = nil
	pkg.Install = nil
	if err := (Manager{}).Install(context.Background(), pkg); err == nil || !strings.Contains(err.Error(), "candidate") {
		t.Fatalf("expected candidate rejection, got %v", err)
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
		Compatibility: &manifest.Compatibility{Firmware: "knulli", MinimumVersion: "2025.1", Architectures: []string{"aarch64"}, Devices: []string{"h700"}, Resolutions: []string{"640x480"}},
		Install: &manifest.Install{
			Destination: "/userdata/roms/tools/demo", Launcher: "launch.sh", Preserve: []string{"config.ini"},
			AllowedWritePaths: []string{"/userdata/roms/tools/demo", "/userdata/roms/tools/gamelist.xml"},
			Menu:              &manifest.Menu{Gamelist: "/userdata/roms/tools/gamelist.xml", Path: "./demo/launch.sh", Name: "Demo", Description: "Fixture utility."},
		},
	}
}

func testPlatform() platform.Info {
	return platform.Info{Firmware: "knulli", Version: "2025.2", Arch: "aarch64", Device: "h700", Resolution: "640x480"}
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
