package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// A verified claim is only as wide as the evidence behind it, and evidence is
// bound to one release. Each of these loosens one of those ties, so the gate
// has to refuse rather than report the package as proven.
func TestVerifiedRequiresEvidenceForEveryDeclaredResolution(t *testing.T) {
	pkg := verifiedPackage()
	if err := pkg.Validate(); err != nil {
		t.Fatalf("verified package with matching evidence was rejected: %v", err)
	}
	pkg.Compatibility.Resolutions = []string{"640x480", "1280x720"}
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "every declared device and resolution") {
		t.Fatalf("expected an unproven resolution to be refused, got %v", err)
	}
	pkg.Compatibility.Resolutions = []string{"640x480"}
	pkg.Review.Evidence[0].PackageVersion = "9.9.9"
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "every declared device and resolution") {
		t.Fatalf("expected evidence for another version to be refused, got %v", err)
	}
	pkg.Review.Evidence = nil
	pkg.Compatibility = nil
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "every declared device and resolution") {
		t.Fatalf("expected a verified package without compatibility to be refused, got %v", err)
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

func TestBroadCompatibilityCanOnlyBeExperimental(t *testing.T) {
	pkg := validPackage()
	pkg.Compatibility.Devices = nil
	pkg.Compatibility.Resolutions = nil
	pkg.Compatibility.DeviceScope = "any"
	pkg.Compatibility.DisplayBounds = &DisplayBounds{MinimumWidth: 640, MinimumHeight: 480, MaximumWidth: 1280, MaximumHeight: 720}
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "broad device scope") || !strings.Contains(err.Error(), "display bounds") {
		t.Fatalf("non-experimental broad scope was accepted: %v", err)
	}
	pkg.Review.Status = "experimental"
	pkg.Compatibility.MinimumVersion = ""
	pkg.Install.Warning = "Unverified broad device test."
	if err := pkg.Validate(); err != nil {
		t.Fatalf("reviewed broad experiment was rejected: %v", err)
	}
}

func TestMinimumGLIBCMustBeNumericMajorMinor(t *testing.T) {
	pkg := validPackage()
	pkg.Compatibility.MinimumGLIBC = "glibc-2.34"
	if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "numeric major.minor") {
		t.Fatalf("invalid glibc version was accepted: %v", err)
	}
	pkg.Compatibility.MinimumGLIBC = "2.34"
	if err := pkg.Validate(); err != nil {
		t.Fatalf("valid glibc version was rejected: %v", err)
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

// Validate reports every problem it finds rather than the first, so one review
// round sees the whole list. These are the identity and review rules a manifest
// cannot be acted on without.
func TestValidateRejectsMalformedCoreFields(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Package)
		problem string
	}{
		{name: "wrong schema", mutate: func(p *Package) { p.Schema = "org.knulli.app-store/package-manifest/v2" }, problem: "schema must be " + SchemaV1},
		{name: "id is not reverse-domain", mutate: func(p *Package) { p.ID = "demo" }, problem: "stable reverse-domain identifier"},
		{name: "id has an upper-case segment", mutate: func(p *Package) { p.ID = "org.Example.demo" }, problem: "stable reverse-domain identifier"},
		{name: "blank name", mutate: func(p *Package) { p.Name = "   " }, problem: "name is required"},
		{name: "unknown type", mutate: func(p *Package) { p.Type = "plugin" }, problem: "type must be utility, theme, or integration"},
		{name: "blank summary", mutate: func(p *Package) { p.Summary = "\t\n" }, problem: "summary is required"},
		{name: "unknown review status", mutate: func(p *Package) { p.Review.Status = "blessed" }, problem: "review.status must be candidate"},
		{name: "repository is not https", mutate: func(p *Package) { p.Repository = "http://github.com/example/demo" }, problem: "repository must use HTTPS"},
		{name: "evidence has no kind", mutate: func(p *Package) { p.Review.Evidence = []Evidence{{URL: "https://example.com/x"}} }, problem: "needs a kind and HTTPS URL"},
		{name: "evidence is not https", mutate: func(p *Package) { p.Review.Evidence = []Evidence{{Kind: "license", URL: "http://example.com/x"}} }, problem: "needs a kind and HTTPS URL"},
		{name: "device test omits its matrix", mutate: func(p *Package) {
			p.Review.Evidence = []Evidence{{Kind: "real-device-test", URL: "https://example.com/x", Tester: "Tester", Date: "2026-09-15", PackageVersion: p.Version, Firmware: "knulli", Architecture: "aarch64", Device: "h700", Resolution: "640x480"}}
		}, problem: "real-device-test needs tester, date, package version"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pkg := validPackage()
			testCase.mutate(&pkg)
			if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), testCase.problem) {
				t.Fatalf("expected %q, got %v", testCase.problem, err)
			}
		})
	}
}

// Every Validate error is accumulated, sorted, and joined, so the order of the
// rules cannot decide which problem a reviewer sees first.
func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	pkg := validPackage()
	pkg.ID = "not-reverse-domain"
	pkg.Name = ""
	pkg.Release.Size = 0
	pkg.Install.Launcher = "../escape"
	err := pkg.Validate()
	if err == nil {
		t.Fatal("expected the broken manifest to be refused")
	}
	for _, problem := range []string{"stable reverse-domain identifier", "name is required", "release size must be between", "launcher must be a safe relative path"} {
		if !strings.Contains(err.Error(), problem) {
			t.Fatalf("error omits %q: %v", problem, err)
		}
	}
	sorted := strings.Split(err.Error(), "; ")
	for index := 1; index < len(sorted); index++ {
		if sorted[index] < sorted[index-1] {
			t.Fatalf("problems are not sorted: %v", sorted)
		}
	}
}

// The package byte budget has one home, manifest.MaxPackageBytes. A manifest
// that validated on one side of the cap and was rejected on the other would
// turn a policy limit into an inconsistency, so the boundary is pinned here
// and the message has to agree with the constant it comes from.
func TestValidateBoundsReleaseAndInstalledSizeByThePackageLimit(t *testing.T) {
	limit := fmt.Sprintf("%d MiB", MaxPackageBytes>>20)
	cases := []struct {
		name    string
		mutate  func(*Package)
		problem string
	}{
		{name: "zero release size", mutate: func(p *Package) { p.Release.Size = 0 }, problem: "release size must be between 1 byte and " + limit},
		{name: "negative release size", mutate: func(p *Package) { p.Release.Size = -1 }, problem: "release size must be between 1 byte and " + limit},
		{name: "release one byte over the cap", mutate: func(p *Package) { p.Release.Size = MaxPackageBytes + 1 }, problem: "release size must be between 1 byte and " + limit},
		{name: "zero installed size", mutate: func(p *Package) { p.Release.InstalledSize = 0 }, problem: "installed size must be between 1 byte and " + limit},
		{name: "installed size one byte over the cap", mutate: func(p *Package) { p.Release.InstalledSize = MaxPackageBytes + 1 }, problem: "installed size must be between 1 byte and " + limit},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pkg := validPackage()
			testCase.mutate(&pkg)
			if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), testCase.problem) {
				t.Fatalf("expected %q, got %v", testCase.problem, err)
			}
		})
	}
	pkg := validPackage()
	pkg.Release.Size = MaxPackageBytes
	pkg.Release.InstalledSize = MaxPackageBytes
	if err := pkg.Validate(); err != nil {
		t.Fatalf("a release exactly at the cap was rejected: %v", err)
	}
}

func TestValidateRejectsReleaseMetadataThatIsNotPinnedOrComplete(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Package)
		problem string
	}{
		{name: "no version", mutate: func(p *Package) { p.Version = "" }, problem: "require version, repository, and license"},
		{name: "no license", mutate: func(p *Package) { p.License = "" }, problem: "require version, repository, and license"},
		{name: "no release", mutate: func(p *Package) { p.Release = nil }, problem: "require release metadata"},
		{name: "url belongs to another repository", mutate: func(p *Package) { p.Release.URL = "https://github.com/other/demo/releases/download/v1.0.0/demo.zip" }, problem: "version-pinned GitHub release"},
		{name: "url is not a github host", mutate: func(p *Package) { p.Release.URL = "https://example.com/demo.zip" }, problem: "version-pinned GitHub release"},
		{name: "url is not a download path", mutate: func(p *Package) { p.Release.URL = "https://github.com/example/demo/releases/tag/v1.0.0" }, problem: "version-pinned GitHub release"},
		{name: "url carries a query", mutate: func(p *Package) { p.Release.URL += "?download=1" }, problem: "version-pinned GitHub release"},
		{name: "url carries a fragment", mutate: func(p *Package) { p.Release.URL += "#asset" }, problem: "version-pinned GitHub release"},
		{name: "release is not declared immutable", mutate: func(p *Package) { p.Release.Immutable = false }, problem: "be declared immutable"},
		{name: "sha256 is truncated", mutate: func(p *Package) { p.Release.SHA256 = strings.Repeat("a", 63) }, problem: "64 lowercase hexadecimal"},
		{name: "sha256 is upper case", mutate: func(p *Package) { p.Release.SHA256 = strings.Repeat("A", 64) }, problem: "64 lowercase hexadecimal"},
		{name: "unknown archive format", mutate: func(p *Package) { p.Release.Format = "7z" }, problem: "release format must be zip or tar.gz"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pkg := validPackage()
			testCase.mutate(&pkg)
			if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), testCase.problem) {
				t.Fatalf("expected %q, got %v", testCase.problem, err)
			}
		})
	}
}

func TestValidateRejectsIncompleteCompatibilityMetadata(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Package)
		problem string
	}{
		{name: "no compatibility", mutate: func(p *Package) { p.Compatibility = nil }, problem: "require compatibility metadata"},
		{name: "firmware is not knulli", mutate: func(p *Package) { p.Compatibility.Firmware = "batocera" }, problem: "compatibility must name Knulli"},
		{name: "no minimum version", mutate: func(p *Package) { p.Compatibility.MinimumVersion = "" }, problem: "non-experimental compatibility requires a minimum version"},
		{name: "architectures omit aarch64", mutate: func(p *Package) { p.Compatibility.Architectures = []string{"x86_64"} }, problem: "must include aarch64"},
		{name: "no device and no broad scope", mutate: func(p *Package) { p.Compatibility.Devices = nil }, problem: "at least one declared device"},
		{name: "device scope is not any", mutate: func(p *Package) { p.Compatibility.DeviceScope = "many" }, problem: "device_scope must be any when set"},
		{name: "no resolution and no display bounds", mutate: func(p *Package) { p.Compatibility.Resolutions = nil }, problem: "at least one declared resolution"},
		{name: "no runtime abi", mutate: func(p *Package) { p.Compatibility.ABIs = nil }, problem: "at least one runtime ABI"},
		{name: "no runtime dependencies", mutate: func(p *Package) { p.Compatibility.Dependencies = nil }, problem: "must declare runtime dependencies"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pkg := validPackage()
			testCase.mutate(&pkg)
			if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), testCase.problem) {
				t.Fatalf("expected %q, got %v", testCase.problem, err)
			}
		})
	}
}

// A display bounds claim replaces a resolution list, so it has to describe a
// range a detector could actually produce. An inverted or out-of-range box
// would make every detected display look either compatible or incompatible for
// the wrong reason.
func TestDisplayBoundsMustBeOrderedAndWithinLimits(t *testing.T) {
	cases := []struct {
		name   string
		bounds *DisplayBounds
	}{
		{name: "below the minimum", bounds: &DisplayBounds{MinimumWidth: 319, MinimumHeight: 200, MaximumWidth: 1280, MaximumHeight: 720}},
		{name: "height below the minimum", bounds: &DisplayBounds{MinimumWidth: 320, MinimumHeight: 199, MaximumWidth: 1280, MaximumHeight: 720}},
		{name: "maximum width below minimum width", bounds: &DisplayBounds{MinimumWidth: 640, MinimumHeight: 480, MaximumWidth: 639, MaximumHeight: 720}},
		{name: "maximum height below minimum height", bounds: &DisplayBounds{MinimumWidth: 640, MinimumHeight: 480, MaximumWidth: 1280, MaximumHeight: 479}},
		{name: "above the maximum", bounds: &DisplayBounds{MinimumWidth: 640, MinimumHeight: 480, MaximumWidth: 7681, MaximumHeight: 720}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pkg := broadExperimentalPackage()
			pkg.Compatibility.DisplayBounds = testCase.bounds
			if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "display_bounds must be ordered") {
				t.Fatalf("expected the bounds to be refused, got %v", err)
			}
		})
	}
	pkg := broadExperimentalPackage()
	pkg.Compatibility.DisplayBounds = &DisplayBounds{MinimumWidth: 320, MinimumHeight: 200, MaximumWidth: 7680, MaximumHeight: 4320}
	if err := pkg.Validate(); err != nil {
		t.Fatalf("the widest allowed bounds were rejected: %v", err)
	}
}

func TestValidateRejectsUnsafeInstallMetadata(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Package)
		problem string
	}{
		{name: "no install", mutate: func(p *Package) { p.Install = nil }, problem: "require install metadata"},
		{name: "destination outside userdata", mutate: func(p *Package) { p.Install.Destination = "/opt/demo" }, problem: "install destination must be a clean path under /userdata"},
		{name: "destination is not cleaned", mutate: func(p *Package) { p.Install.Destination = "/userdata/roms/../roms/tools/demo" }, problem: "install destination must be a clean path under /userdata"},
		{name: "destination is relative", mutate: func(p *Package) { p.Install.Destination = "userdata/roms/tools/demo" }, problem: "install destination must be a clean path under /userdata"},
		{name: "negative strip components", mutate: func(p *Package) { p.Install.StripComponents = -1 }, problem: "strip_components cannot be negative"},
		{name: "launcher escapes the destination", mutate: func(p *Package) { p.Install.Launcher = "../launch.sh" }, problem: "launcher must be a safe relative path"},
		{name: "preserve path escapes the destination", mutate: func(p *Package) { p.Install.Preserve = []string{"../config.ini"} }, problem: "preserved paths must be safe relative paths"},
		{name: "no allowed write paths", mutate: func(p *Package) { p.Install.AllowedWritePaths = nil }, problem: "allowed_write_paths cannot be empty"},
		{name: "write path outside userdata", mutate: func(p *Package) { p.Install.AllowedWritePaths = append(p.Install.AllowedWritePaths, "/etc/demo") }, problem: "allowed write paths must be clean paths under /userdata"},
		{name: "write path is manager state", mutate: func(p *Package) { p.Install.AllowedWritePaths = append(p.Install.AllowedWritePaths, ManagerStatePath) }, problem: "must not overlap app-manager state"},
		{name: "write path sits inside manager state", mutate: func(p *Package) {
			p.Install.AllowedWritePaths = append(p.Install.AllowedWritePaths, ManagerStatePath+"/installed")
		}, problem: "must not overlap app-manager state"},
		{name: "write path contains manager state", mutate: func(p *Package) {
			p.Install.AllowedWritePaths = append(p.Install.AllowedWritePaths, "/userdata/system")
		}, problem: "must not overlap app-manager state"},
		// The most dangerous shape: the release would be copied straight over a
		// record the installer owns, from inside the directory that holds it.
		{name: "destination is inside manager state", mutate: func(p *Package) {
			p.Install.Destination = ManagerStatePath + "/installed/org.example.demo.json"
			p.Install.AllowedWritePaths = []string{ManagerStatePath + "/installed/org.example.demo.json"}
		}, problem: "must not overlap app-manager state"},
		{name: "destination is not covered by a write path", mutate: func(p *Package) { p.Install.AllowedWritePaths = []string{"/userdata/roms/ports/demo"} }, problem: "install destination must be covered by allowed_write_paths"},
		{name: "menu gamelist is not an allowed path", mutate: func(p *Package) {
			p.Install.Menu = &Menu{Gamelist: "/userdata/roms/other/gamelist.xml", Path: "./demo/launch.sh", Name: "Demo"}
		}, problem: "menu gamelist must be an allowed absolute path"},
		{name: "menu entry has no name", mutate: func(p *Package) {
			p.Install.Menu = &Menu{Gamelist: "/userdata/roms/tools/demo/gamelist.xml", Path: "./demo/launch.sh"}
		}, problem: "menu path and name are required"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pkg := validPackage()
			testCase.mutate(&pkg)
			if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), testCase.problem) {
				t.Fatalf("expected %q, got %v", testCase.problem, err)
			}
		})
	}
	pkg := validPackage()
	pkg.Install.Menu = &Menu{Gamelist: "/userdata/roms/tools/demo/gamelist.xml", Path: "./demo/launch.sh", Name: "Demo"}
	if err := pkg.Validate(); err != nil {
		t.Fatalf("a complete menu entry was rejected: %v", err)
	}
	// A path beside manager state is not manager state, so a package's runtime
	// directory under /userdata/system/configs stays allowed.
	pkg.Install.AllowedWritePaths = append(pkg.Install.AllowedWritePaths, "/userdata/system/configs/demo")
	if err := pkg.Validate(); err != nil {
		t.Fatalf("a write path beside manager state was rejected: %v", err)
	}
}

// The manager-state rule is deliberately symmetric, so the helper it reads is
// pinned directly and not only through the rule that calls it. The neighbour
// case matters: a prefix match without the separator is a different directory.
func TestOverlapsCoversBothDirections(t *testing.T) {
	cases := []struct {
		name  string
		left  string
		right string
		want  bool
	}{
		{name: "identical", left: ManagerStatePath, right: ManagerStatePath, want: true},
		{name: "left is inside", left: ManagerStatePath + "/installed", right: ManagerStatePath, want: true},
		{name: "right is inside", left: ManagerStatePath, right: ManagerStatePath + "/installed", want: true},
		{name: "left contains", left: "/userdata", right: ManagerStatePath, want: true},
		{name: "right contains", left: ManagerStatePath, right: "/userdata", want: true},
		{name: "neighbour with a shared prefix", left: "/userdata/system/knulli-app-store-other", right: ManagerStatePath, want: false},
		{name: "sibling directory", left: "/userdata/system/configs/demo", right: ManagerStatePath, want: false},
		{name: "unrelated", left: "/userdata/roms/tools/demo", right: ManagerStatePath, want: false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := overlaps(testCase.left, testCase.right); got != testCase.want {
				t.Fatalf("overlaps(%q, %q) = %v, want %v", testCase.left, testCase.right, got, testCase.want)
			}
		})
	}
}

// A binary patch rewrites bytes in a downloaded binary, so the manifest has to
// pin both what it expects to find and what the result must hash to. These
// cases are the whole admission test for that field.
func TestBinaryPatchesMustBePinnedAndReproducible(t *testing.T) {
	complete := BinaryPatch{Path: "demo", Offset: 4, BeforeHex: "0a0b", AfterHex: "0c0d", SHA256: strings.Repeat("b", 64)}
	pkg := validPackage()
	pkg.Install.BinaryPatches = []BinaryPatch{complete}
	if err := pkg.Validate(); err != nil {
		t.Fatalf("a complete patch was rejected: %v", err)
	}
	cases := []struct {
		name  string
		patch BinaryPatch
	}{
		{name: "unsafe target path", patch: BinaryPatch{Path: "../demo", Offset: 4, BeforeHex: "0a0b", AfterHex: "0c0d", SHA256: strings.Repeat("b", 64)}},
		{name: "negative offset", patch: BinaryPatch{Path: "demo", Offset: -1, BeforeHex: "0a0b", AfterHex: "0c0d", SHA256: strings.Repeat("b", 64)}},
		{name: "before is not hexadecimal", patch: BinaryPatch{Path: "demo", Offset: 4, BeforeHex: "zz", AfterHex: "0c0d", SHA256: strings.Repeat("b", 64)}},
		{name: "after is not hexadecimal", patch: BinaryPatch{Path: "demo", Offset: 4, BeforeHex: "0a0b", AfterHex: "zz", SHA256: strings.Repeat("b", 64)}},
		{name: "no expected bytes", patch: BinaryPatch{Path: "demo", Offset: 4, BeforeHex: "", AfterHex: "", SHA256: strings.Repeat("b", 64)}},
		{name: "replacement changes the length", patch: BinaryPatch{Path: "demo", Offset: 4, BeforeHex: "0a0b", AfterHex: "0c", SHA256: strings.Repeat("b", 64)}},
		{name: "no final hash", patch: BinaryPatch{Path: "demo", Offset: 4, BeforeHex: "0a0b", AfterHex: "0c0d"}},
		{name: "final hash is malformed", patch: BinaryPatch{Path: "demo", Offset: 4, BeforeHex: "0a0b", AfterHex: "0c0d", SHA256: "short"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pkg := validPackage()
			pkg.Install.BinaryPatches = []BinaryPatch{testCase.patch}
			if err := pkg.Validate(); err == nil || !strings.Contains(err.Error(), "binary patches require") {
				t.Fatalf("expected the patch to be refused, got %v", err)
			}
		})
	}
}

func TestImmutableReleaseURLRequiresAPinnedGithubDownloadForTheSameRepository(t *testing.T) {
	const repository = "https://github.com/example/demo"
	cases := []struct {
		name   string
		raw    string
		repo   string
		accept bool
	}{
		{name: "pinned asset", raw: repository + "/releases/download/v1.0.0/demo.zip", repo: repository, accept: true},
		{name: "asset nested below the tag", raw: repository + "/releases/download/v1.0.0/nested/demo.zip", repo: repository, accept: true},
		{name: "repository with a trailing slash", raw: repository + "/releases/download/v1.0.0/demo.zip", repo: repository + "/", accept: true},
		{name: "latest alias", raw: repository + "/releases/latest/download/demo.zip", repo: repository},
		{name: "no asset after the tag", raw: repository + "/releases/download/v1.0.0", repo: repository},
		{name: "release page rather than an asset", raw: repository + "/releases/tag/v1.0.0", repo: repository},
		{name: "not a release path at all", raw: repository + "/archive/v1.0.0.zip", repo: repository},
		{name: "http scheme", raw: "http://github.com/example/demo/releases/download/v1.0.0/demo.zip", repo: repository},
		{name: "another host", raw: "https://example.com/example/demo/releases/download/v1.0.0/demo.zip", repo: repository},
		{name: "another repository", raw: "https://github.com/other/demo/releases/download/v1.0.0/demo.zip", repo: repository},
		{name: "query in the url", raw: repository + "/releases/download/v1.0.0/demo.zip?x=1", repo: repository},
		{name: "fragment in the url", raw: repository + "/releases/download/v1.0.0/demo.zip#x", repo: repository},
		{name: "repository is not https", raw: repository + "/releases/download/v1.0.0/demo.zip", repo: "http://github.com/example/demo"},
		{name: "repository is another host", raw: repository + "/releases/download/v1.0.0/demo.zip", repo: "https://gitlab.com/example/demo"},
		{name: "repository carries a query", raw: repository + "/releases/download/v1.0.0/demo.zip", repo: repository + "?ref=main"},
		{name: "repository does not parse", raw: repository + "/releases/download/v1.0.0/demo.zip", repo: "://bad"},
		{name: "url does not parse", raw: "https://[::1", repo: repository},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := immutableReleaseURL(testCase.raw, testCase.repo); got != testCase.accept {
				t.Fatalf("immutableReleaseURL(%q, %q) = %v, want %v", testCase.raw, testCase.repo, got, testCase.accept)
			}
		})
	}
}

// The catalogue digest is computed over Canonical, so the encoding has to be
// stable and compact: two builds of the same manifest must hash the same, and a
// decoded manifest must re-encode to the same bytes.
func TestCanonicalEncodesStablyAndRoundTrips(t *testing.T) {
	pkg := validPackage()
	first, err := Canonical(pkg)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Canonical(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("canonical encoding is not stable")
	}
	if bytes.ContainsAny(first, "\n\t") {
		t.Fatalf("canonical encoding is not compact: %s", first)
	}
	var decoded Package
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatalf("canonical encoding does not decode: %v", err)
	}
	again, err := Canonical(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, again) {
		t.Fatalf("canonical encoding does not round-trip:\n%s\n%s", first, again)
	}
}

func TestLoadReadsAValidManifestAndReportsAMissingOne(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "demo.json")
	data, err := json.Marshal(validPackage())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	pkg, err := Load(path)
	if err != nil {
		t.Fatalf("loading a valid manifest failed: %v", err)
	}
	if pkg.ID != "org.example.demo" || pkg.Version != "1.0.0" {
		t.Fatalf("loaded the wrong manifest: %#v", pkg)
	}
	if _, err := Load(filepath.Join(directory, "absent.json")); err == nil {
		t.Fatal("expected a missing manifest to report an error")
	}
	// A file that decodes but fails policy is refused by the same entry point,
	// so a caller cannot bypass validation by loading from disk.
	if err := os.WriteFile(path, []byte(`{"schema":"`+SchemaV1+`"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected an incomplete manifest to be rejected")
	}
}

func verifiedPackage() Package {
	pkg := validPackage()
	pkg.Review.Status = "verified"
	pkg.Review.Evidence = []Evidence{{
		Kind: "real-device-test", URL: "https://example.com/report", Tester: "Tester", Date: "2026-09-15",
		PackageVersion: pkg.Version, Firmware: "knulli", Architecture: "aarch64", Device: "h700", Resolution: "640x480", Result: "passed",
	}}
	return pkg
}

// broadExperimentalPackage is the only shape allowed to declare a display range
// instead of an exact resolution list.
func broadExperimentalPackage() Package {
	pkg := validPackage()
	pkg.Review.Status = "experimental"
	pkg.Compatibility.MinimumVersion = ""
	pkg.Compatibility.Devices = nil
	pkg.Compatibility.Resolutions = nil
	pkg.Compatibility.DeviceScope = "any"
	pkg.Install.Warning = "Unverified broad device test."
	return pkg
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
			ABIs:           []string{"linux-aarch64-glibc"},
			Dependencies:   []string{"sdl2"},
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
