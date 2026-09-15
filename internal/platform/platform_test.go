package platform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

func TestDetectReadsKnulliDeviceAndFramebuffer(t *testing.T) {
	root := t.TempDir()
	write := func(name, value string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("etc/knulli-release", "ID=knulli\nVERSION_ID=2025.2\n")
	write("etc/knulli-device", "h700\n")
	write("sys/class/graphics/fb0/virtual_size", "640,480\n")
	got := Detect(root)
	if got.Firmware != "knulli" || got.Version != "2025.2" || got.Device != "h700" || got.Resolution != "640x480" {
		t.Fatalf("unexpected detection: %#v", got)
	}
}

func TestCheckRejectsOlderVersionAndWrongResolution(t *testing.T) {
	pkg := manifest.Package{Compatibility: &manifest.Compatibility{
		Firmware: "knulli", MinimumVersion: "2025.10", Architectures: []string{"aarch64"}, Devices: []string{"h700"}, Resolutions: []string{"640x480"},
	}}
	current := Info{Firmware: "knulli", Version: "2025.2", Arch: "aarch64", Device: "h700", Resolution: "640x480"}
	if err := Check(pkg, current); err == nil {
		t.Fatal("expected older firmware to fail")
	}
	current.Version = "2025.11"
	current.Resolution = "720x720"
	if err := Check(pkg, current); err == nil {
		t.Fatal("expected unsupported resolution to fail")
	}
}
