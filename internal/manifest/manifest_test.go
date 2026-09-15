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
	pkg.Review.Evidence = []Evidence{{Kind: "real-device-test", URL: "https://example.com/report", Tester: "Tester", Date: "2026-09-15", PackageVersion: pkg.Version, Firmware: "knulli", Architecture: "aarch64", Device: "h700", Resolution: "640x480", Result: "passed"}}
	if err := pkg.Validate(); err != nil {
		t.Fatalf("expected verified package to pass: %v", err)
	}
	pkg.Compatibility.Devices = append(pkg.Compatibility.Devices, "untested-device")
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "every declared device") {
		t.Fatalf("expected incomplete verified matrix rejection, got %v", err)
	}
}

func TestCommunityApprovalDoesNotMakeCandidateInstallable(t *testing.T) {
	pkg := validPackage()
	pkg.Review = Review{Status: "candidate", Approval: &Approval{Provenance: "community"}}
	pkg.Release = nil
	pkg.Compatibility = nil
	pkg.Install = nil
	if err := pkg.Validate(); err != nil {
		t.Fatalf("expected approved candidate to pass: %v", err)
	}
	if pkg.Installable() {
		t.Fatal("community approval must not make a candidate installable")
	}
}

func TestApprovalRejectsUnknownProvenance(t *testing.T) {
	pkg := validPackage()
	pkg.Review.Approval = &Approval{Provenance: "social-media"}
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "provenance") {
		t.Fatalf("expected approval provenance rejection, got %v", err)
	}
}

func TestExperimentalPackageAllowsUnknownMinimumWithWarning(t *testing.T) {
	pkg := validPackage()
	pkg.Review.Status = "experimental"
	pkg.Compatibility.MinimumVersion = ""
	pkg.Install.Warning = "Unverified device test."
	if err := pkg.Validate(); err != nil {
		t.Fatalf("expected experimental package to pass: %v", err)
	}
	pkg.Install.Warning = ""
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "install warning") {
		t.Fatalf("expected missing experimental warning rejection, got %v", err)
	}
}

func TestExecutablePathsMustBeSafeAndUnique(t *testing.T) {
	pkg := validPackage()
	pkg.Install.Executables = []string{"run.sh", "../escape", "run.sh"}
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "safe relative") || !strings.Contains(err.Error(), "unique") {
		t.Fatalf("expected executable path rejection, got %v", err)
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
