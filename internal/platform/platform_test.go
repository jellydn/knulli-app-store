package platform

import (
	"os"
	"path/filepath"
	"strings"
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
	write("etc/os-release", "NAME=Buildroot\nID=buildroot\nVERSION_ID=2025.02\nOS_NAME=\"knulli\"\nOS_VERSION=scarab\nOS_DATE=20260510\n")
	write("usr/share/knulli/knulli.version", "scarab 2026/05/10 14:23\n")
	write("boot/boot/knulli.board", "trimui-smart-pro\n")
	write("etc/knulli-device", "legacy-device\n")
	write("sys/class/graphics/fb0/virtual_size", "1280,720\n")
	got := Detect(root)
	if got.Firmware != "knulli" || got.FirmwareRaw != "knulli" || got.FirmwareSource != "/etc/os-release:OS_NAME" || got.Version != "scarab" || got.VersionRaw != "scarab 2026/05/10 14:23" || got.VersionSource != "/usr/share/knulli/knulli.version" || got.Device != "trimui-smart-pro" || got.Resolution != "1280x720" {
		t.Fatalf("unexpected detection: %#v", got)
	}
}

func TestDetectUsesDeterministicCurrentThenLegacyPriority(t *testing.T) {
	root := t.TempDir()
	writePlatformFile(t, root, "etc/os-release", "  OS_NAME = ' KNULLI '  \nOS_VERSION=\nOS_DATE=20260510\n")
	writePlatformFile(t, root, "etc/knulli-release", "ID=knulli\nVERSION_ID=legacy\n")
	got := Detect(root)
	if got.Firmware != "knulli" || got.FirmwareSource != "/etc/os-release:OS_NAME" || got.Version != "20260510" || got.VersionSource != "/etc/os-release:OS_DATE" {
		t.Fatalf("unexpected priority result: %#v", got)
	}
}

func TestDetectDoesNotTreatBuildrootOrMalformedVersionAsKnulli(t *testing.T) {
	root := t.TempDir()
	writePlatformFile(t, root, "etc/os-release", "ID=buildroot\nVERSION_ID=2025.02\nOS_NAME=other\nOS_VERSION=   \n")
	got := Detect(root)
	if got.Firmware != "" || got.FirmwareRaw != "other" || got.FirmwareSource != "/etc/os-release:OS_NAME" || got.Version != "" {
		t.Fatalf("unknown firmware became compatible: %#v", got)
	}
}

func TestDetectUsesNarrowLegacyFallback(t *testing.T) {
	root := t.TempDir()
	writePlatformFile(t, root, "etc/os-release", "ID=buildroot\n")
	writePlatformFile(t, root, "etc/knulli-release", "ID=KNULLI\nVERSION_ID=2025.2-dev-a1b2c3\n")
	got := Detect(root)
	if got.Firmware != "knulli" || got.FirmwareSource != "/etc/knulli-release:ID" || got.Version != "2025.2-dev-a1b2c3" {
		t.Fatalf("legacy fallback failed: %#v", got)
	}
}

func TestNormalizeVersionKeepsReleaseSuffixAndRejectsWhitespace(t *testing.T) {
	if got := normalizeVersion("  scarab-dev-a1b2c3 2026/05/10 14:23\r\n"); got != "scarab-dev-a1b2c3" {
		t.Fatalf("unexpected suffixed version: %q", got)
	}
	if got := normalizeVersion(" \r\n\t "); got != "" {
		t.Fatalf("malformed version became %q", got)
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

func TestCheckRejectsUnknownCodenameOrderingForMinimumVersion(t *testing.T) {
	pkg := manifest.Package{Compatibility: &manifest.Compatibility{
		Firmware: "knulli", MinimumVersion: "2025.10", Architectures: []string{"aarch64"}, Devices: []string{"trimui-smart-pro"}, Resolutions: []string{"1280x720"},
	}}
	current := Info{Firmware: "knulli", FirmwareRaw: "knulli", FirmwareSource: "/etc/os-release:OS_NAME", Version: "scarab", VersionRaw: "scarab 2026/05/10 14:23", VersionSource: "/usr/share/knulli/knulli.version", Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1280x720"}
	err := Check(pkg, current)
	for _, wanted := range []string{"field=firmware_version", `version_raw="scarab 2026/05/10 14:23"`, `version="scarab"`, `version_source="/usr/share/knulli/knulli.version"`, "ordering is unknown"} {
		if err == nil || !strings.Contains(err.Error(), wanted) {
			t.Fatalf("version diagnostic %q missing from %v", wanted, err)
		}
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

func TestCheckFailureIncludesDetectedEvidenceAndConstraint(t *testing.T) {
	pkg := manifest.Package{Compatibility: &manifest.Compatibility{Firmware: "knulli", Architectures: []string{"aarch64"}, Devices: []string{"trimui-smart-pro"}, Resolutions: []string{"1280x720"}}}
	current := Info{FirmwareRaw: "buildroot", FirmwareSource: "/etc/os-release:ID", Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1280x720"}
	err := Check(pkg, current)
	for _, wanted := range []string{"field=firmware", `device="trimui-smart-pro"`, `architecture="aarch64"`, `resolution="1280x720"`, `firmware_raw="buildroot"`, `firmware=""`, `firmware_source="/etc/os-release:ID"`, `requires firmware="knulli"`} {
		if err == nil || !strings.Contains(err.Error(), wanted) {
			t.Fatalf("diagnostic %q missing from %v", wanted, err)
		}
	}
}

func writePlatformFile(t *testing.T, root, name, value string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0644); err != nil {
		t.Fatal(err)
	}
}
