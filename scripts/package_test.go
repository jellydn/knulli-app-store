package scripts_test

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSingleProfilePackageReleaseAndLegacyBridge(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"cli", "ui", "catalog.json", "catalog.json.sig"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0755); err != nil {
			t.Fatal(err)
		}
	}
	run := func(script string, args ...string) {
		t.Helper()
		command := exec.Command("sh", append([]string{script}, args...)...)
		command.Dir = ".."
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", script, err, output)
		}
	}
	packaged := filepath.Join(root, "packaged")
	run("scripts/package-device.sh", filepath.Join(root, "cli"), filepath.Join(root, "ui"), filepath.Join(root, "catalog.json"), packaged, "v9.1")
	name := "knulli-app-store-knulli-aarch64-glibc-sdl2-experimental-v9.1.zip"
	archive, err := zip.OpenReader(filepath.Join(packaged, name))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	wanted := map[string]bool{
		"Knulli App Store.sh": false, "images/Knulli App Store.png": false,
		"knulli-app-store/knulli-app": false, "knulli-app-store/knulli-app-ui": false,
		"knulli-app-store/catalog-index.json": false, "knulli-app-store/catalog-index.json.sig": false,
		"knulli-app-store/README.txt": false,
	}
	for _, file := range archive.File {
		if _, ok := wanted[file.Name]; !ok || wanted[file.Name] {
			t.Fatalf("unexpected or duplicate archive path %q", file.Name)
		}
		wanted[file.Name] = true
		if strings.HasSuffix(file.Name, ".sh") || strings.HasSuffix(file.Name, "/knulli-app") || strings.HasSuffix(file.Name, "/knulli-app-ui") {
			if file.Mode().Perm() != 0755 {
				t.Fatalf("lost executable mode: %s", file.Name)
			}
		}
	}
	for path, found := range wanted {
		if !found {
			t.Fatalf("missing %s", path)
		}
	}
	release := filepath.Join(root, "release")
	run("scripts/collect-release.sh", packaged, release)
	entries, err := os.ReadDir(release)
	if err != nil || len(entries) != 2 {
		t.Fatalf("release must contain one ZIP and checksums: %v, %v", entries, err)
	}
	command := exec.Command("sha256sum", "-c", "SHA256SUMS.txt")
	command.Dir = release
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("release checksums: %v\n%s", err, output)
	}
	bridge := filepath.Join(root, "legacy-bridge")
	run("scripts/prepare-legacy-bridge.sh", filepath.Join(release, name), bridge)
	original, err := os.ReadFile(filepath.Join(release, name))
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"trimui-smart-pro", "magicx-zero-28"} {
		alias, err := os.ReadFile(filepath.Join(bridge, "knulli-app-store-"+target+"-experimental-v9.1.zip"))
		if err != nil || !bytes.Equal(original, alias) {
			t.Fatalf("bridge must retain identical signed bytes: %s: %v", target, err)
		}
	}
	for _, source := range []string{bridge, root} {
		command := exec.Command("sh", "scripts/collect-release.sh", source, filepath.Join(root, "invalid-release"))
		command.Dir = ".."
		if output, err := command.CombinedOutput(); err == nil {
			t.Fatalf("legacy/duplicate ZIPs entered a new release: %s", output)
		}
	}
	for _, name := range []string{"", "knulli-app-store-trimui-smart-pro-experimental-v9.1.zip", "knulli-app-store-knulli-armhf-musl-sdl2-experimental-v9.1.zip"} {
		source := t.TempDir()
		if name != "" {
			if err := os.WriteFile(filepath.Join(source, name), original, 0644); err != nil {
				t.Fatal(err)
			}
		}
		command := exec.Command("sh", "scripts/collect-release.sh", source, filepath.Join(source, "release"))
		command.Dir = ".."
		if output, err := command.CombinedOutput(); err == nil {
			t.Fatalf("empty/legacy/wrong-profile input accepted: %q: %s", name, output)
		}
	}
}
