package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/catalog"
	"github.com/jellydn/knulli-app-store/internal/manifest"
)

func TestWriteNewFileCreatesPrivateFileWithoutReplacingExistingPath(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "key")
	if err := writeNewFile(path, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("private key mode = %o, want 600", info.Mode().Perm())
	}
	if err := writeNewFile(path, []byte("replacement"), 0600); err == nil {
		t.Fatal("expected existing key path to be rejected")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "secret" {
		t.Fatalf("existing key was replaced: %q", data)
	}
}

const (
	testFirmware   = "knulli"
	testVersion    = "2025.1"
	testArch       = "aarch64"
	testDevice     = "h700"
	testResolution = "640x480"
)

func TestRunPrintsUsageAndRejectsAnUnknownCommand(t *testing.T) {
	cases := []struct {
		name      string
		arguments []string
		problem   string
	}{
		{name: "no arguments"},
		{name: "help", arguments: []string{"help"}},
		{name: "short help", arguments: []string{"-h"}},
		{name: "long help", arguments: []string{"--help"}},
		{name: "unknown command", arguments: []string{"frobnicate"}, problem: "unknown command"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			output, err := capture(t, func() error { return run(testCase.arguments) })
			if testCase.problem != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.problem) {
					t.Fatalf("expected %q, got %v", testCase.problem, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected success, got %v", err)
			}
			if !strings.Contains(output, "Knulli App Store installer core") {
				t.Fatalf("usage was not printed: %q", output)
			}
		})
	}
}

// Every command that takes one path has to say so itself rather than fail deep
// inside the installer, because the message is the only thing a user sees.
func TestRunRejectsMissingOrExtraArguments(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name      string
		arguments []string
		problem   string
	}{
		{name: "validate without a path", arguments: []string{"validate"}, problem: "validate requires one manifest path"},
		{name: "validate with two paths", arguments: []string{"validate", "one.json", "two.json"}, problem: "validate requires one manifest path"},
		{name: "install without a path", arguments: []string{"install"}, problem: "install requires one manifest path"},
		{name: "adopt with two paths", arguments: []string{"adopt", "one.json", "two.json"}, problem: "adopt requires one manifest path"},
		{name: "update without a path", arguments: []string{"update"}, problem: "update requires one manifest path"},
		{name: "repair without a path", arguments: []string{"repair"}, problem: "repair requires one manifest path"},
		{name: "uninstall without an id", arguments: []string{"uninstall", "-root", root}, problem: "uninstall requires one package id"},
		{name: "uninstall with two ids", arguments: []string{"uninstall", "-root", root, "one", "two"}, problem: "uninstall requires one package id"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := capture(t, func() error { return run(testCase.arguments) }); err == nil || !strings.Contains(err.Error(), testCase.problem) {
				t.Fatalf("expected %q, got %v", testCase.problem, err)
			}
		})
	}
}

func TestRunValidateReportsTheReviewedStatusAndReadsNothingElse(t *testing.T) {
	directory := t.TempDir()
	path := writeManifest(t, directory, cliPackage(t, zipBytes(t, map[string]string{"run.sh": "x"})))
	output, err := capture(t, func() error { return run([]string{"validate", path}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "valid org.example.demo (installable)") {
		t.Fatalf("output = %q", output)
	}

	// A candidate is reported as one. It is not actionable, but reviewing it is
	// the whole point of the command, so it must not be an error.
	candidate := cliPackage(t, nil)
	candidate.Review.Status = "candidate"
	candidate.Release, candidate.Compatibility, candidate.Install = nil, nil, nil
	candidatePath := writeManifest(t, t.TempDir(), candidate)
	output, err = capture(t, func() error { return run([]string{"validate", candidatePath}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "(candidate)") {
		t.Fatalf("output = %q", output)
	}

	if err := run([]string{"validate", filepath.Join(directory, "absent.json")}); err == nil {
		t.Fatal("expected a missing manifest to fail")
	}
}

func TestRunCatalogueBuildsByteIdenticalIndexesAndLeavesThemUnsigned(t *testing.T) {
	directory := t.TempDir()
	first := filepath.Join(directory, "one.json")
	second := filepath.Join(directory, "two.json")
	for _, path := range []string{first, second} {
		if err := run([]string{"catalogue", "-dir", packagesDirectory(), "-output", path}); err != nil {
			t.Fatal(err)
		}
	}
	firstData, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondData, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstData, secondData) {
		t.Fatal("two builds of the same manifests differ")
	}
	if !strings.Contains(string(firstData), `"`+catalog.IndexSchemaV1+`"`) {
		t.Fatalf("index does not carry its schema: %s", firstData)
	}
	// A local build stays unsigned for review; only a device build signs.
	if _, err := os.Stat(catalog.SignaturePath(first)); !os.IsNotExist(err) {
		t.Fatalf("an unsigned build left a signature behind: %v", err)
	}
}

func TestRunCatalogueRefusesConflictingAndUnusableRequests(t *testing.T) {
	output := filepath.Join(t.TempDir(), "index.json")
	cases := []struct {
		name      string
		arguments []string
		problem   string
	}{
		{
			name:      "a directory with no manifests",
			arguments: []string{"catalogue", "-dir", t.TempDir(), "-output", output},
			problem:   "no JSON manifests found",
		},
		{
			name:      "both key flags at once",
			arguments: []string{"catalogue", "-dir", packagesDirectory(), "-output", output, "-signing-key", filepath.Join(t.TempDir(), "key"), "-generate-signing-key", filepath.Join(t.TempDir(), "other")},
			problem:   "use only one of",
		},
		{
			name:      "signing an index written to stdout",
			arguments: []string{"catalogue", "-dir", packagesDirectory(), "-output", "-", "-generate-signing-key", filepath.Join(t.TempDir(), "key")},
			problem:   "cannot sign catalogue output written to stdout",
		},
		{
			name:      "an empty output path",
			arguments: []string{"catalogue", "-dir", packagesDirectory(), "-output", ""},
			problem:   "output path is required",
		},
		{
			name:      "a key path that already exists",
			arguments: []string{"catalogue", "-dir", packagesDirectory(), "-output", output, "-generate-signing-key", existingFile(t)},
			problem:   "file exists",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := capture(t, func() error { return run(testCase.arguments) }); err == nil {
				t.Fatalf("expected the request to be refused")
			}
		})
	}
}

// The signing path is what device CI runs, so the private key mode, the public
// key mode, and the sidecar a device build would verify are all pinned here. A
// generated key signs a later index through -signing-key, so the review path and
// the device path produce the same sidecar.
func TestRunCatalogueSignsTheIndexWithAGeneratedKey(t *testing.T) {
	directory := t.TempDir()
	key := filepath.Join(directory, "signing-key")
	index := filepath.Join(directory, "index.json")
	if err := run([]string{"catalogue", "-dir", packagesDirectory(), "-output", index, "-generate-signing-key", key}); err != nil {
		t.Fatal(err)
	}
	privateInfo, err := os.Stat(key)
	if err != nil {
		t.Fatal(err)
	}
	if privateInfo.Mode().Perm() != 0600 {
		t.Fatalf("private key mode = %o, want 600", privateInfo.Mode().Perm())
	}
	publicInfo, err := os.Stat(key + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	if publicInfo.Mode().Perm() != 0644 {
		t.Fatalf("public key mode = %o, want 644", publicInfo.Mode().Perm())
	}
	assertSignedIndex(t, index, key+".pub")

	second := filepath.Join(directory, "second.json")
	if err := run([]string{"catalogue", "-dir", packagesDirectory(), "-output", second, "-signing-key", key}); err != nil {
		t.Fatal(err)
	}
	assertSignedIndex(t, second, key+".pub")
}

// The gate is the only thing standing between a half-detected device and a
// package written against a guess, so each of its four conditions is pinned on
// its own. Every fixture supplies a complete platform minus one field, so a
// condition dropped from the disjunction fails a case here instead of silently
// letting a half-detected install through.
func TestRunApplyRequiresACompleteDetectedPlatform(t *testing.T) {
	// The fixture root holds library markers but no Knulli identity, so firmware,
	// device, and version are only ever what a flag says they are.
	root := platformRoot(t)
	manifestPath := writeManifest(t, t.TempDir(), cliPackage(t, zipBytes(t, map[string]string{"run.sh": "x"})))

	withoutFlag := func(omit string) []string {
		arguments := []string{"-root", root}
		for _, field := range []struct{ flag, value string }{
			{"-firmware", testFirmware},
			{"-firmware-version", testVersion},
			{"-arch", testArch},
			{"-device", testDevice},
			{"-resolution", testResolution},
		} {
			if field.flag != omit {
				arguments = append(arguments, field.flag, field.value)
			}
		}
		return append(arguments, manifestPath)
	}

	cases := []struct {
		name   string
		omit   string
		reason string
	}{
		{name: "firmware", omit: "-firmware", reason: "no Knulli identity in the root"},
		{name: "device", omit: "-device", reason: "device identity is never inferred from the SoC family"},
		{name: "resolution", omit: "-resolution", reason: "resolution is explicit, never assumed"},
		{name: "firmware version", omit: "-firmware-version", reason: "the package names a minimum version"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := capture(t, func() error { return runApply("install", withoutFlag(testCase.omit), applyDeps{}) })
			if err == nil || !strings.Contains(err.Error(), "platform detection is incomplete") {
				t.Fatalf("expected a platform missing its %s to be refused because %s, got %v", testCase.name, testCase.reason, err)
			}
		})
	}

	// No flags at all is the same gate, and it fires before the manifest is read:
	// the path here does not exist, yet the platform complaint is what surfaces.
	_, err := capture(t, func() error { return runApply("install", []string{"-root", root, manifestPath}, applyDeps{}) })
	if err == nil || !strings.Contains(err.Error(), "platform detection is incomplete") {
		t.Fatalf("expected an entirely undetected platform to be refused, got %v", err)
	}

	// A complete platform gets past the gate: the absent manifest is reported
	// instead, proving the gate opened and no download was attempted.
	_, err = capture(t, func() error {
		return runApply("install", append(platformFlags(root), filepath.Join(root, "absent.json")), applyDeps{})
	})
	if err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("expected a complete platform to reach the manifest load, got %v", err)
	}
}

func TestRunApplyInstallsRepairsAndUninstallsAnOwnedMenuEntry(t *testing.T) {
	root := platformRoot(t)
	archive := zipBytes(t, map[string]string{"run.sh": "#!/bin/sh\necho demo\n"})
	pkg := withMenu(cliPackage(t, archive))
	manifestPath := writeManifest(t, t.TempDir(), pkg)
	deps := applyDeps{client: staticClient(archive), refreshURL: refreshServer(t, http.StatusOK)}

	output, err := capture(t, func() error { return runApply("install", append(platformFlags(root), manifestPath), deps) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "install completed for org.example.demo 1.0.0") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "game list refresh accepted") {
		t.Fatalf("expected an accepted reload, got %q", output)
	}
	launcher, err := readRoot(t, root, "/userdata/roms/tools/demo/run.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(launcher, "echo demo") {
		t.Fatalf("launcher = %q", launcher)
	}
	// The declared executable carries execute mode on the destination.
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash("userdata/roms/tools/demo/run.sh")))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Fatalf("launcher is not executable: %o", info.Mode().Perm())
	}
	gamelist, err := readRoot(t, root, "/userdata/roms/tools/gamelist.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gamelist, "./demo/run.sh") {
		t.Fatalf("gamelist = %q", gamelist)
	}

	if output, err = capture(t, func() error { return runApply("repair", append(platformFlags(root), manifestPath), deps) }); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "repair completed") {
		t.Fatalf("output = %q", output)
	}

	output, err = capture(t, func() error { return runUninstall([]string{"-root", root, pkg.ID}, deps) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "uninstalled org.example.demo") {
		t.Fatalf("output = %q", output)
	}
	if !strings.Contains(output, "game list refresh accepted") {
		t.Fatalf("expected the removal to report its reload, got %q", output)
	}
	if _, err := readRoot(t, root, "/userdata/roms/tools/demo/run.sh"); err == nil {
		t.Fatal("managed file survived uninstall")
	}
	if _, err := readRoot(t, root, "/userdata/system/knulli-app-store/installed/org.example.demo.json"); err == nil {
		t.Fatal("installed state survived uninstall")
	}
	if gamelist, err := readRoot(t, root, "/userdata/roms/tools/gamelist.xml"); err == nil && strings.Contains(gamelist, "./demo/run.sh") {
		t.Fatalf("owned menu entry survived uninstall: %q", gamelist)
	}
}

// A refused reload is not a failed operation. The package is committed, so the
// command has to say a restart is needed rather than report an error, and the
// files have to still be on the card.
func TestRunApplyReportsRestartRequiredWhenTheReloadIsRefused(t *testing.T) {
	cases := []struct {
		name    string
		refresh func(t *testing.T) string
	}{
		{name: "the reload returns an error status", refresh: func(t *testing.T) string { return refreshServer(t, http.StatusInternalServerError) }},
		{name: "nothing is listening", refresh: closedServerURL},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := platformRoot(t)
			archive := zipBytes(t, map[string]string{"run.sh": "#!/bin/sh\n"})
			manifestPath := writeManifest(t, t.TempDir(), withMenu(cliPackage(t, archive)))
			deps := applyDeps{client: staticClient(archive), refreshURL: testCase.refresh(t)}
			output, err := capture(t, func() error { return runApply("install", append(platformFlags(root), manifestPath), deps) })
			if err != nil {
				t.Fatalf("a refused reload must not fail the operation: %v", err)
			}
			if !strings.Contains(output, "restart required to update game list") {
				t.Fatalf("output = %q", output)
			}
			if _, err := readRoot(t, root, "/userdata/roms/tools/demo/run.sh"); err != nil {
				t.Fatalf("the committed operation was rolled back: %v", err)
			}
		})
	}
}

func TestRunApplyRefusesInstallOverAnExternalCopyAndAdoptsItInstead(t *testing.T) {
	root := platformRoot(t)
	archive := zipBytes(t, map[string]string{"run.sh": "#!/bin/sh\n"})
	manifestPath := writeManifest(t, t.TempDir(), cliPackage(t, archive))
	deps := applyDeps{client: staticClient(archive)}
	existing := filepath.Join(root, filepath.FromSlash("userdata/roms/tools/demo/run.sh"))
	if err := os.MkdirAll(filepath.Dir(existing), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existing, []byte("hand-installed"), 0755); err != nil {
		t.Fatal(err)
	}

	_, err := capture(t, func() error { return runApply("install", append(platformFlags(root), manifestPath), deps) })
	if err == nil || !strings.Contains(err.Error(), "use Manage existing") {
		t.Fatalf("expected install over an external copy to be refused, got %v", err)
	}

	output, err := capture(t, func() error { return runApply("adopt", append(platformFlags(root), manifestPath), deps) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "adopt completed") {
		t.Fatalf("output = %q", output)
	}
	// Adoption takes ownership without reinstalling, so the bytes that were
	// already there are still there when the reviewed copy does not match.
	content, err := readRoot(t, root, "/userdata/roms/tools/demo/run.sh")
	if err != nil {
		t.Fatal(err)
	}
	if content != "hand-installed" {
		t.Fatalf("adoption overwrote the existing file: %q", content)
	}
	if _, err := readRoot(t, root, "/userdata/system/knulli-app-store/installed/org.example.demo.json"); err != nil {
		t.Fatalf("adoption recorded no state: %v", err)
	}
}

func TestRunApplyRefusesAdoptWhenThereIsNothingToManage(t *testing.T) {
	root := platformRoot(t)
	archive := zipBytes(t, map[string]string{"run.sh": "#!/bin/sh\n"})
	manifestPath := writeManifest(t, t.TempDir(), cliPackage(t, archive))
	_, err := capture(t, func() error {
		return runApply("adopt", append(platformFlags(root), manifestPath), applyDeps{client: staticClient(archive)})
	})
	if err == nil || !strings.Contains(err.Error(), "use Install") {
		t.Fatalf("expected adopt without an external copy to be refused, got %v", err)
	}
}

func TestRunApplyGatesRepairAndUpdateOnTheInstalledRelease(t *testing.T) {
	root := platformRoot(t)
	firstArchive := zipBytes(t, map[string]string{"run.sh": "#!/bin/sh\n# v1\n"})
	firstPath := writeManifest(t, t.TempDir(), cliPackage(t, firstArchive))
	if _, err := capture(t, func() error {
		return runApply("install", append(platformFlags(root), firstPath), applyDeps{client: staticClient(firstArchive)})
	}); err != nil {
		t.Fatal(err)
	}

	secondArchive := zipBytes(t, map[string]string{"run.sh": "#!/bin/sh\n# v2\n"})
	second := cliPackage(t, secondArchive)
	second.Version = "2.0.0"
	second.Release.URL = "https://github.com/example/demo/releases/download/v2.0.0/demo.zip"
	secondPath := writeManifest(t, t.TempDir(), second)

	// Repair may only reapply the release that is actually installed.
	_, err := capture(t, func() error {
		return runApply("repair", append(platformFlags(root), secondPath), applyDeps{client: staticClient(secondArchive)})
	})
	if err == nil || !strings.Contains(err.Error(), "repair requires the installed release") {
		t.Fatalf("expected a mismatched repair to be refused, got %v", err)
	}

	updateDeps := applyDeps{client: staticClient(secondArchive)}
	output, err := capture(t, func() error { return runApply("update", append(platformFlags(root), secondPath), updateDeps) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "update completed for org.example.demo 2.0.0") {
		t.Fatalf("output = %q", output)
	}
	content, err := readRoot(t, root, "/userdata/roms/tools/demo/run.sh")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, "# v2") {
		t.Fatalf("update did not apply the new release: %q", content)
	}

	// Once that release is the installed one, repair accepts it.
	if _, err := capture(t, func() error {
		return runApply("repair", append(platformFlags(root), secondPath), updateDeps)
	}); err != nil {
		t.Fatal(err)
	}
}

// This walks the whole dispatch rather than runUninstall directly. The package
// owns no menu entry, so nothing asks Knulli to reload and the fixed loopback
// port is never contacted, which keeps the case independent of a developer's
// running EmulationStation.
func TestRunUninstallsAMenuLessPackageWithoutAskingForAReload(t *testing.T) {
	root := platformRoot(t)
	archive := zipBytes(t, map[string]string{"run.sh": "#!/bin/sh\n"})
	pkg := cliPackage(t, archive)
	manifestPath := writeManifest(t, t.TempDir(), pkg)
	if _, err := capture(t, func() error {
		return runApply("install", append(platformFlags(root), manifestPath), applyDeps{client: staticClient(archive)})
	}); err != nil {
		t.Fatal(err)
	}

	output, err := capture(t, func() error { return run([]string{"uninstall", "-root", root, pkg.ID}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "uninstalled org.example.demo") {
		t.Fatalf("output = %q", output)
	}
	if strings.Contains(output, "game list refresh accepted") || strings.Contains(output, "restart required") {
		t.Fatalf("a menu-less uninstall reported a reload: %q", output)
	}
	if _, err := readRoot(t, root, "/userdata/roms/tools/demo/run.sh"); err == nil {
		t.Fatal("managed file survived uninstall")
	}
}

func TestRunUninstallReportsAPackageThatIsNotInstalled(t *testing.T) {
	root := t.TempDir()
	_, err := capture(t, func() error { return run([]string{"uninstall", "-root", root, "org.example.absent"}) })
	if err == nil || !strings.Contains(err.Error(), "is not installed") {
		t.Fatalf("expected a missing package to be reported, got %v", err)
	}
}

// capture runs one command with stdout redirected, so a case can assert what the
// user would have seen. Tests do not run in parallel, because the redirect is
// process-wide.
func capture(t *testing.T, command func() error) (string, error) {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = writer
	commandErr := command()
	os.Stdout = original
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	return string(data), commandErr
}

// platformRoot writes the two library markers platform detection reads, so a
// command run against it looks like a reviewed device root rather than a bare
// temporary directory. Firmware, version, device identity, and resolution still
// arrive as flags, which is the documented way to fill values that detection
// cannot read from a fixture.
func platformRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"lib/ld-linux-aarch64.so.1", "lib/libSDL2-2.0.so.0"} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func platformFlags(root string) []string {
	return []string{"-root", root, "-firmware", testFirmware, "-firmware-version", testVersion, "-arch", testArch, "-device", testDevice, "-resolution", testResolution}
}

func packagesDirectory() string {
	return filepath.Join("..", "..", "catalogue", "packages")
}

// zipBytes builds the release archive a manifest points at.
func zipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

// cliPackage is an installable manifest whose release is the given archive. The
// installed size is read back out of the archive, so a fixture cannot declare a
// budget the extraction would reject. Each caller gets its own copy.
func cliPackage(t *testing.T, archive []byte) manifest.Package {
	t.Helper()
	var installedSize int64
	if len(archive) > 0 {
		reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range reader.File {
			installedSize += int64(item.UncompressedSize64)
		}
	}
	return manifest.Package{
		Schema: manifest.SchemaV1, ID: "org.example.demo", Name: "Demo", Version: "1.0.0",
		Type: "utility", Summary: "A test package.",
		Repository: "https://github.com/example/demo", License: "MIT",
		Review: manifest.Review{Status: "installable"},
		Release: &manifest.Release{
			URL:    "https://github.com/example/demo/releases/download/v1.0.0/demo.zip",
			SHA256: digestOf(archive), Size: int64(len(archive)), InstalledSize: installedSize,
			Format: "zip", Immutable: true,
		},
		Compatibility: &manifest.Compatibility{
			Firmware: testFirmware, MinimumVersion: testVersion,
			Architectures: []string{testArch}, ABIs: []string{"linux-aarch64-glibc"},
			Dependencies: []string{"sdl2"}, Devices: []string{testDevice}, Resolutions: []string{testResolution},
		},
		Install: &manifest.Install{
			Destination: "/userdata/roms/tools/demo", Launcher: "run.sh",
			Executables: []string{"run.sh"},
			// The destination has to be covered by a declared write path, so the
			// list repeats it rather than describing anything wider.
			AllowedWritePaths: []string{"/userdata/roms/tools/demo"},
		},
	}
}

// withMenu adds the one gamelist entry the installer may own, which is what makes
// a committed operation ask Knulli to reload the game list.
func withMenu(pkg manifest.Package) manifest.Package {
	pkg.Install.AllowedWritePaths = append(pkg.Install.AllowedWritePaths, "/userdata/roms/tools/gamelist.xml")
	pkg.Install.Menu = &manifest.Menu{Gamelist: "/userdata/roms/tools/gamelist.xml", Path: "./demo/run.sh", Name: "Demo"}
	return pkg
}

func writeManifest(t *testing.T, directory string, pkg manifest.Package) string {
	t.Helper()
	path := filepath.Join(directory, pkg.ID+".json")
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func digestOf(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// staticTransport answers the pinned github.com release URL with the archive
// bytes. A manifest cannot point its release anywhere but a version-pinned
// github.com URL, so a test cannot serve the download over loopback by editing
// the manifest; it replaces the client instead, which is the seam
// installer.Manager exposes for exactly this reason.
type staticTransport struct {
	archive []byte
}

func (transport staticTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		ContentLength: int64(len(transport.archive)),
		Body:          io.NopCloser(bytes.NewReader(transport.archive)),
		Request:       request,
		Header:        make(http.Header),
	}, nil
}

func staticClient(archive []byte) *http.Client {
	return &http.Client{Transport: staticTransport{archive: archive}}
}

// refreshServer is a stand-in for EmulationStation's reload endpoint.
func refreshServer(t *testing.T, status int) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// closedServerURL returns an address that accepts nothing, so the reload fails
// at the connection rather than at the response status.
func closedServerURL(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := "http://" + listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}

func existingFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "taken")
	if err := os.WriteFile(path, []byte("taken"), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// readRoot reads a file by its device path, so a case can assert on the tree the
// way the installer sees it instead of rebuilding the path arithmetic.
func readRoot(t *testing.T, root, virtual string) (string, error) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(virtual, "/"))))
	return string(data), err
}

// assertSignedIndex verifies the sidecar the way a device build would: the index
// bytes, the signature beside them, and the public key compiled into the binary.
func assertSignedIndex(t *testing.T, indexPath, publicPath string) {
	t.Helper()
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(publicPath)
	if err != nil {
		t.Fatal(err)
	}
	public, err := catalog.ParsePublicKey(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	signatureData, err := os.ReadFile(catalog.SignaturePath(indexPath))
	if err != nil {
		t.Fatal(err)
	}
	var signature catalog.Signature
	if err := json.Unmarshal(signatureData, &signature); err != nil {
		t.Fatal(err)
	}
	if err := catalog.Verify(indexData, signature, public); err != nil {
		t.Fatalf("the signed index does not verify: %v", err)
	}
}
