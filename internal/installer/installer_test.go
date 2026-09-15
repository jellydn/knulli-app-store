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

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

func TestInstallRepairUpdateAndUninstallEndToEnd(t *testing.T) {
	root := t.TempDir()
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

	manager := Manager{Root: root, Platform: testPlatform(), Client: rewriteClient(t, server)}
	pkg := testPackage("https://github.com/example/demo/releases/download/v1/demo.zip", v1, "1.0.0")
	if err := manager.Install(context.Background(), pkg); err != nil {
		t.Fatal(err)
	}
	assertRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "version one")
	assertRootFile(t, root, "userdata/roms/tools/demo/config.ini", "user configuration")
	assertRootFile(t, root, "userdata/roms/tools/demo/obsolete.txt", "remove on update")
	assertContains(t, root, "userdata/roms/tools/gamelist.xml", "<image>./images/existing.png</image>")
	assertContains(t, root, "userdata/roms/tools/gamelist.xml", "<path>./demo/launch.sh</path>")

	writeRootFile(t, root, "userdata/roms/tools/demo/launch.sh", "damaged")
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
