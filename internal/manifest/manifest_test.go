package manifest

import (
	"strings"
	"testing"
)

func TestCandidateCannotContainActionableRelease(t *testing.T) {
	pkg := validPackage()
	pkg.Review.Status = "candidate"
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "candidate packages") {
		t.Fatalf("expected candidate metadata rejection, got %v", err)
	}
}

func TestVerifiedRequiresDeviceEvidence(t *testing.T) {
	pkg := validPackage()
	pkg.Review.Status = "verified"
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "real-device-test") {
		t.Fatalf("expected device evidence rejection, got %v", err)
	}
	pkg.Review.Evidence = []Evidence{{Kind: "real-device-test", URL: "https://example.com/report"}}
	if err := pkg.Validate(); err != nil {
		t.Fatalf("expected verified package to pass: %v", err)
	}
}

func TestMutableReleaseURLIsRejected(t *testing.T) {
	pkg := validPackage()
	pkg.Release.URL = "https://github.com/example/tool/releases/latest/download/tool.zip"
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "version-pinned GitHub release") {
		t.Fatalf("expected mutable URL rejection, got %v", err)
	}
}

func TestDecodeRejectsUnknownAndTrailingData(t *testing.T) {
	for _, input := range []string{
		`{"schema":"x","unknown":true}`,
		`{"schema":"x"} {"schema":"y"}`,
	} {
		if _, err := Decode(strings.NewReader(input)); err == nil {
			t.Fatalf("expected invalid JSON to fail: %s", input)
		}
	}
}

func validPackage() Package {
	return Package{
		Schema:     SchemaV1,
		ID:         "org.example.demo",
		Name:       "Demo",
		Version:    "1.0.0",
		Type:       "utility",
		Summary:    "A test package.",
		Repository: "https://github.com/example/demo",
		License:    "MIT",
		Review:     Review{Status: "installable"},
		Release: &Release{
			URL:           "https://github.com/example/demo/releases/download/v1.0.0/demo.zip",
			SHA256:        strings.Repeat("a", 64),
			Size:          10,
			InstalledSize: 10,
			Format:        "zip",
			Immutable:     true,
		},
		Compatibility: &Compatibility{
			Firmware:       "knulli",
			MinimumVersion: "2025.1",
			Architectures:  []string{"aarch64"},
			Devices:        []string{"h700"},
			Resolutions:    []string{"640x480"},
		},
		Install: &Install{
			Destination:       "/userdata/roms/tools/demo",
			Launcher:          "launch.sh",
			AllowedWritePaths: []string{"/userdata/roms/tools/demo"},
		},
	}
}
