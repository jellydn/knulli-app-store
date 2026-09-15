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
	write("boot/boot/knulli.board", "trimui-smart-pro\n")
	write("etc/knulli-device", "legacy-device\n")
	write("sys/class/graphics/fb0/virtual_size", "1280,720\n")
	got := Detect(root)
	if got.Firmware != "knulli" || got.Version != "2025.2" || got.Device != "trimui-smart-pro" || got.Resolution != "1280x720" {
		t.Fatalf("unexpected detection: %#v", got)
	}
}

func TestDisplayHeaderUsesKnownDeviceAndRuntimeSize(t *testing.T) {
	info := Info{Device: "trimui-smart-pro", Resolution: "640x480"}
	if got := DisplayHeader(info, 1280, 720); got != "TrimUI Smart Pro / 1280x720" {
		t.Fatalf("unexpected display header: %q", got)
	}
}

func TestDisplayHeaderShowsFallbackStates(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want string
	}{
		{name: "framebuffer size", info: Info{Device: "new-board", Resolution: "1024x600"}, want: "Unknown device (new-board) / 1024x600 fallback"},
		{name: "nothing detected", info: Info{}, want: "Unknown device / size unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DisplayHeader(test.info, 0, 0); got != test.want {
				t.Fatalf("unexpected display header: %q", got)
			}
		})
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

func TestCheckAllowsExperimentalCompatibilityWithoutInventedMinimum(t *testing.T) {
	pkg := manifest.Package{Compatibility: &manifest.Compatibility{
		Firmware: "knulli", Architectures: []string{"aarch64"}, Devices: []string{"trimui-smart-pro"}, Resolutions: []string{"1280x720"},
	}}
	current := Info{Firmware: "knulli", Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1280x720"}
	if err := Check(pkg, current); err != nil {
		t.Fatalf("experimental compatibility should not invent a minimum version: %v", err)
	}
}
