package platform

import (
	"os"
	"path/filepath"
	"runtime"
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
	got := Resolve(root)
	if got.Firmware != "knulli" || got.Evidence(FieldFirmware).Raw != "knulli" || got.Evidence(FieldFirmware).Location != "/etc/os-release:OS_NAME" || got.Version != "scarab" || got.Evidence(FieldVersion).Raw != "scarab 2026/05/10 14:23" || got.Evidence(FieldVersion).Location != "/usr/share/knulli/knulli.version" || got.Device != "trimui-smart-pro" || got.Resolution != "1280x720" {
		t.Fatalf("unexpected detection: %#v", got)
	}
}

func TestDetectUsesDeterministicCurrentThenLegacyPriority(t *testing.T) {
	root := t.TempDir()
	writePlatformFile(t, root, "etc/os-release", "  OS_NAME = ' KNULLI '  \nOS_VERSION=\nOS_DATE=20260510\n")
	writePlatformFile(t, root, "etc/knulli-release", "ID=knulli\nVERSION_ID=legacy\n")
	got := Resolve(root)
	if got.Firmware != "knulli" || got.Evidence(FieldFirmware).Location != "/etc/os-release:OS_NAME" || got.Version != "20260510" || got.Evidence(FieldVersion).Location != "/etc/os-release:OS_DATE" {
		t.Fatalf("unexpected priority result: %#v", got)
	}
}

func TestDetectDoesNotTreatBuildrootOrMalformedVersionAsKnulli(t *testing.T) {
	root := t.TempDir()
	writePlatformFile(t, root, "etc/os-release", "ID=buildroot\nVERSION_ID=2025.02\nOS_NAME=other\nOS_VERSION=   \n")
	got := Resolve(root)
	if got.Firmware != "" || got.Evidence(FieldFirmware).Raw != "other" || got.Evidence(FieldFirmware).Location != "/etc/os-release:OS_NAME" || got.Version != "" {
		t.Fatalf("unknown firmware became compatible: %#v", got)
	}
}

func TestDetectUsesNarrowLegacyFallback(t *testing.T) {
	root := t.TempDir()
	writePlatformFile(t, root, "etc/os-release", "ID=buildroot\n")
	writePlatformFile(t, root, "etc/knulli-release", "ID=KNULLI\nVERSION_ID=2025.2-dev-a1b2c3\n")
	got := Resolve(root)
	if got.Firmware != "knulli" || got.Evidence(FieldFirmware).Location != "/etc/knulli-release:ID" || got.Version != "2025.2-dev-a1b2c3" {
		t.Fatalf("legacy fallback failed: %#v", got)
	}
}

func TestResolveAppliesOverridesWithEvidenceAndResolutionPrecedence(t *testing.T) {
	root := t.TempDir()
	writePlatformFile(t, root, "sys/class/graphics/fb0/virtual_size", "1280,720\n")
	got := Resolve(root,
		WithFirmware("knulli"),
		WithVersion("scarab"),
		WithArch("aarch64"),
		WithDevice("magicx-zero-28"),
		WithResolutionOverride("640x480"),
	)
	if got.Firmware != "knulli" || got.Version != "scarab" || got.Arch != "aarch64" || got.Device != "magicx-zero-28" || got.Resolution != "640x480" {
		t.Fatalf("overrides were not applied: %#v", got)
	}
	if got.Evidence(FieldFirmware) != (Source{Raw: "knulli", Location: "command-line override"}) || got.Evidence(FieldVersion) != (Source{Raw: "scarab", Location: "command-line override"}) || got.Evidence(FieldResolution).Location != "command-line override" {
		t.Fatalf("override evidence is inconsistent: %#v", got)
	}
	candidates := got.ResolutionCandidates()
	if len(candidates) < 2 || candidates[0].Source != "command-line override" || candidates[1].Source != "/sys/class/graphics/fb0/virtual_size" {
		t.Fatalf("resolution precedence is wrong: %#v", candidates)
	}
	got = Resolve(root, WithResolutionOverride("invalid"))
	if got.Resolution != "1280x720" || got.Evidence(FieldResolution).Location != "/sys/class/graphics/fb0/virtual_size" {
		t.Fatalf("invalid override did not fall back: %#v", got)
	}
}

func TestResolveUsesGOARCHWhenArchIsNotOverridden(t *testing.T) {
	want := runtime.GOARCH
	if want == "arm64" {
		want = "aarch64"
	}
	if got := Resolve(t.TempDir()); got.Arch != want {
		t.Fatalf("arch = %q, want GOARCH fallback %q", got.Arch, want)
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

func TestDisplayHeaderNamesMagicXZero28(t *testing.T) {
	info := Info{Device: "magicx-zero-28"}
	if got := DisplayHeader(info, 640, 480); got != "MagicX Zero 28 / 640x480" {
		t.Fatalf("unexpected MagicX header: %q", got)
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

func TestDisplayHeaderRejectsCorruptRuntimeSize(t *testing.T) {
	info := Info{Device: "trimui-smart-pro", Resolution: "1280x720", evidence: evidence{ResolutionSource: "/sys/class/graphics/fb0/mode"}}
	if got := DisplayHeader(info, 1280, 0x3333); got != "TrimUI Smart Pro / 1280x720 fallback" {
		t.Fatalf("header used corrupt runtime size: %q", got)
	}
}

func TestResolutionSelectionRejectsCorruptDisplayMode(t *testing.T) {
	candidates := []ResolutionCandidate{
		{Source: "SDL renderer output", Width: 1280, Height: 720},
		{Source: "SDL current display mode", Width: 1280, Height: 0x3333},
		{Source: "/sys/class/graphics/fb0/virtual_size", Width: 1280, Height: 0x3333},
	}
	got := Info{}.WithCandidates(candidates)
	if got.Resolution != "1280x720" || got.Evidence(FieldResolution).Location != "SDL renderer output" {
		t.Fatalf("selected corrupt resolution: %#v", got)
	}
	assessments := AssessResolutions(candidates)
	for _, index := range []int{1, 2} {
		if assessments[index].Valid || !strings.Contains(assessments[index].Reason, "maximum") {
			t.Fatalf("1280x13107 was not rejected: %#v", assessments[index])
		}
	}
}

func TestResolutionSelectionUsesValidatedFramebufferModeFallback(t *testing.T) {
	root := t.TempDir()
	writePlatformFile(t, root, "sys/class/graphics/fb0/mode", "U:1280x720p-60\n")
	writePlatformFile(t, root, "sys/class/graphics/fb0/virtual_size", "1280,13107\n")
	got := Resolve(root)
	if got.Resolution != "1280x720" || got.Evidence(FieldResolution).Location != "/sys/class/graphics/fb0/mode" {
		t.Fatalf("did not select framebuffer mode fallback: %#v", got)
	}
}

func TestResolutionSelectionBlocksUnknownWhenEveryCandidateIsInvalid(t *testing.T) {
	candidates := []ResolutionCandidate{
		{Source: "uninitialized", Width: 0, Height: 0},
		{Source: "corrupt", Width: 1280, Height: 0x3333},
		{Source: "query", Error: "display query failed"},
	}
	got := Info{Resolution: "1280x720"}.WithCandidates(candidates)
	if got.Resolution != "" || got.Evidence(FieldResolution).Location != "" {
		t.Fatalf("invalid candidates became compatible: %#v", got)
	}
}

func TestResolutionCandidateParserRejectsMalformedValues(t *testing.T) {
	for _, value := range []string{"", "1280", "garbage", "0x0"} {
		candidate := ResolutionCandidateFromString("test", value)
		assessment := AssessResolutions([]ResolutionCandidate{candidate})[0]
		if assessment.Valid {
			t.Fatalf("malformed value %q became valid: %#v", value, assessment)
		}
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
	current := Info{Firmware: "knulli", Version: "scarab", Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1280x720", evidence: evidence{FirmwareRaw: "knulli", FirmwareSource: "/etc/os-release:OS_NAME", VersionRaw: "scarab 2026/05/10 14:23", VersionSource: "/usr/share/knulli/knulli.version"}}
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

func TestBroadExperimentalCompatibilityRequiresEveryRuntimeConstraint(t *testing.T) {
	pkg := manifest.Package{Compatibility: &manifest.Compatibility{
		Firmware: "knulli", Architectures: []string{"aarch64"}, ABIs: []string{"linux-aarch64-glibc"}, MinimumGLIBC: "2.34",
		Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf"}, DeviceScope: "any",
		DisplayBounds: &manifest.DisplayBounds{MinimumWidth: 640, MinimumHeight: 480, MaximumWidth: 1280, MaximumHeight: 720},
	}}
	valid := Info{Firmware: "knulli", Arch: "aarch64", ABI: "linux-aarch64-glibc", GLIBCVersion: "2.40", Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf"}, Device: "new-knulli-board", Resolution: "1024x600"}
	if err := Check(pkg, valid); err != nil {
		t.Fatalf("matching unknown device was not allowed as experimental: %v", err)
	}
	tests := []struct {
		name string
		edit func(*Info)
		want string
	}{
		{name: "unknown architecture", edit: func(info *Info) { info.Arch = "" }, want: "field=architecture"},
		{name: "incompatible architecture", edit: func(info *Info) { info.Arch = "armv7" }, want: "field=architecture"},
		{name: "unknown ABI", edit: func(info *Info) { info.ABI = "" }, want: "field=abi"},
		{name: "old glibc", edit: func(info *Info) { info.GLIBCVersion = "2.17" }, want: `glibc>="2.34"`},
		{name: "missing dependency", edit: func(info *Info) { info.Dependencies = []string{"sdl2", "sdl2-ttf"} }, want: `runtime dependency="sdl2-image"`},
		{name: "unknown device", edit: func(info *Info) { info.Device = "" }, want: "device identity is empty"},
		{name: "unknown display", edit: func(info *Info) { info.Resolution = "" }, want: "no validated display"},
		{name: "display outside bounds", edit: func(info *Info) { info.Resolution = "320x240" }, want: "display bounds=640x480..1280x720"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := valid
			current.Dependencies = append([]string(nil), valid.Dependencies...)
			test.edit(&current)
			err := Check(pkg, current)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("constraint %q did not block with exact reason: %v", test.name, err)
			}
		})
	}
}

func TestCheckFailureIncludesDetectedEvidenceAndConstraint(t *testing.T) {
	pkg := manifest.Package{Compatibility: &manifest.Compatibility{Firmware: "knulli", Architectures: []string{"aarch64"}, Devices: []string{"trimui-smart-pro"}, Resolutions: []string{"1280x720"}}}
	current := Info{Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1280x720", evidence: evidence{FirmwareRaw: "buildroot", FirmwareSource: "/etc/os-release:ID"}}
	err := Check(pkg, current)
	for _, wanted := range []string{"field=firmware", `device="trimui-smart-pro"`, `architecture="aarch64"`, `resolution="1280x720"`, `firmware_raw="buildroot"`, `firmware=""`, `firmware_source="/etc/os-release:ID"`, `requires firmware="knulli"`} {
		if err == nil || !strings.Contains(err.Error(), wanted) {
			t.Fatalf("diagnostic %q missing from %v", wanted, err)
		}
	}
}

func TestResolutionFailureIncludesSelectedSource(t *testing.T) {
	pkg := manifest.Package{Compatibility: &manifest.Compatibility{Firmware: "knulli", Architectures: []string{"aarch64"}, Devices: []string{"trimui-smart-pro"}, Resolutions: []string{"1280x720"}}}
	current := Info{Firmware: "knulli", Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1024x600", evidence: evidence{ResolutionSource: "/sys/class/graphics/fb0/mode"}}
	err := Check(pkg, current)
	if err == nil || !strings.Contains(err.Error(), `resolution_source="/sys/class/graphics/fb0/mode"`) {
		t.Fatalf("resolution source missing from diagnostic: %v", err)
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
