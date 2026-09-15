package appstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

func TestServiceExposesOnlyReviewedPackagesAsActionable(t *testing.T) {
	indexPath := filepath.Join(t.TempDir(), "index.json")
	index, err := catalogForTest()
	if err != nil {
		t.Fatal(err)
	}
	if err := writeIndexForTest(index, indexPath); err != nil {
		t.Fatal(err)
	}
	service, err := Open(indexPath, installer.Manager{Root: t.TempDir(), Platform: platform.Info{
		Firmware: "knulli", Version: "scarab", Arch: "aarch64", ABI: "linux-aarch64-glibc", GLIBCVersion: "2.40", Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}, Device: "trimui-smart-pro", Resolution: "1280x720",
	}})
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatalf("expected four catalogue items, got %d", len(items))
	}
	experimental := 0
	verified := 0
	for _, item := range items {
		if item.Package.ID == "io.github.unitreign.playtime" && !item.DeviceTested {
			t.Fatalf("Smart Pro test evidence was not matched for %s", item.Package.ID)
		}
		if item.Package.Installable() {
			if item.Package.Experimental() {
				experimental++
			}
			if item.Package.Review.Status == "verified" {
				verified++
			}
			if !item.Compatible || len(item.Actions) != 1 || item.Actions[0] != Install {
				t.Fatalf("reviewed package is not installable: %#v", item)
			}
			continue
		}
		readOnlyStatus := strings.Contains(item.Compatibility, "Candidate") || strings.Contains(item.Compatibility, "installation is blocked")
		if item.Compatible || len(item.Actions) != 0 || !readOnlyStatus {
			t.Fatalf("candidate became actionable: %#v", item)
		}
	}
	if experimental != 2 || verified != 0 {
		t.Fatalf("expected two experimental and no universally verified packages, got %d and %d", experimental, verified)
	}
}

func TestApprovedCandidateRemainsReadOnly(t *testing.T) {
	pkg := installablePackage()
	pkg.Review = manifest.Review{Status: "candidate", Approval: &manifest.Approval{Provenance: "community"}}
	pkg.Release, pkg.Compatibility, pkg.Install = nil, nil, nil
	compatible, message := compatibility(pkg, platform.Info{})
	if compatible || message != "Community approved; installation is blocked by technical review" {
		t.Fatalf("unexpected approved candidate state: %v, %q", compatible, message)
	}
	if actions(Item{Package: pkg}) != nil {
		t.Fatal("approved candidate must remain read-only")
	}
}

func TestLatestKnulliMetadataAllowsOnlyExperimentalDeviceMatrix(t *testing.T) {
	root := t.TempDir()
	for name, value := range map[string]string{
		"etc/os-release":                      "NAME=Buildroot\nID=buildroot\nVERSION_ID=2025.02\nOS_NAME=\"knulli\"\nOS_VERSION=scarab\nOS_DATE=20260510\n",
		"usr/share/knulli/knulli.version":     "scarab 2026/08/19 16:06\n",
		"boot/boot/knulli.board":              "trimui-smart-pro\n",
		"sys/class/graphics/fb0/virtual_size": "1280,13107\n",
	} {
		host := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(host), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(host, []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	index, err := catalogForTest()
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(t.TempDir(), "index.json")
	if err := writeIndexForTest(index, indexPath); err != nil {
		t.Fatal(err)
	}
	detected := platform.Detect(root)
	detected.Arch = "aarch64"
	detected.ABI = "linux-aarch64-glibc"
	detected.GLIBCVersion = "2.40"
	detected.Dependencies = []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}
	if detected.Resolution != "" {
		t.Fatalf("corrupt framebuffer virtual size became compatible: %#v", detected)
	}
	detected = platform.WithResolutionCandidates(detected, append([]platform.ResolutionCandidate{{Source: "SDL renderer output", Width: 1280, Height: 720}}, detected.ResolutionCandidates...))
	service, err := Open(indexPath, installer.Manager{Root: root, Platform: detected})
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Package.Experimental() {
			if !item.Compatible || len(item.Actions) != 1 || !strings.Contains(item.Compatibility, "no minimum version is claimed") || !strings.Contains(item.Compatibility, "/etc/os-release:OS_NAME") || !strings.Contains(item.Compatibility, "source=SDL renderer output") {
				t.Fatalf("latest Knulli experimental decision is wrong: %#v", item)
			}
		}
	}
}

func TestMagicXAllowsOnlyPlayTimeExperimentalPackage(t *testing.T) {
	index, err := catalogForTest()
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(t.TempDir(), "index.json")
	if err := writeIndexForTest(index, indexPath); err != nil {
		t.Fatal(err)
	}
	service, err := Open(indexPath, installer.Manager{Root: t.TempDir(), Platform: platform.Info{
		Firmware: "knulli", Version: "scarab 2026/08/19 16:06", Arch: "aarch64", ABI: "linux-aarch64-glibc", GLIBCVersion: "2.40", Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}, Device: "magicx-zero-28", Resolution: "640x480",
		FirmwareRaw: "knulli", FirmwareSource: "/etc/os-release:OS_NAME", VersionRaw: "scarab 2026/08/19 16:06", VersionSource: "/usr/share/knulli/knulli.version", ResolutionSource: "SDL renderer output",
	}})
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		switch item.Package.ID {
		case "io.github.unitreign.playtime":
			if !item.Compatible || item.DeviceTested || !item.Package.Experimental() || len(item.Actions) != 1 || item.Actions[0] != Install || !strings.Contains(strings.ToLower(item.Package.Install.Warning), "unverified") {
				t.Fatalf("PlayTime was not offered as an unverified MagicX experiment: %#v", item)
			}
		case "app.romm.grout":
			if !item.Compatible || item.DeviceTested || len(item.Actions) != 1 || item.Actions[0] != Install {
				t.Fatalf("Grout was not offered as an unverified MagicX experiment: %#v", item)
			}
		}
	}
}

func TestActionsReflectInstallStateAndHealth(t *testing.T) {
	item := Item{Package: installablePackage(), Compatible: true}
	assertActions(t, actions(item), Install)
	item.PreExisting = true
	assertActions(t, actions(item), Adopt)
	item.RecoveryReason = "checksum mismatch"
	assertActions(t, actions(item), Adopt)
	item.RecoveryAllowed = true
	assertActions(t, actions(item), Adopt, ForceReinstall)
	item.RecoveryActive = true
	assertActions(t, actions(item))
	item.RecoveryReason = ""
	item.RecoveryAllowed = false
	item.RecoveryActive = false
	item.PreExisting = false
	item.Installed = true
	item.InstalledVersion = item.Package.Version
	item.Healthy = true
	assertActions(t, actions(item), Uninstall, Repair)
	item.Healthy = false
	assertActions(t, actions(item), Uninstall, Repair)
	item.InstalledVersion = "0.9.0"
	assertActions(t, actions(item), Update, Uninstall, Repair)
	item.Compatible = false
	assertActions(t, actions(item), Uninstall)
}

func TestOnlyRecoverableAdoptionFailuresAllowForceReinstall(t *testing.T) {
	for _, test := range []struct {
		err     error
		allowed bool
	}{
		{err: &installer.AdoptionConflictError{Path: "/userdata/roms/tools/demo/run.sh"}, allowed: true},
		{err: errors.New("download release: SHA-256 mismatch"), allowed: false},
		{err: errors.New("compatibility failed field=architecture"), allowed: false},
		{err: errors.New("not enough free space"), allowed: false},
		{err: errors.New("archive path escapes destination"), allowed: false},
		{err: errors.New("catalogue signature is invalid"), allowed: false},
	} {
		if got := recoverableAdoptionFailure(test.err); got != test.allowed {
			t.Fatalf("recoverableAdoptionFailure(%q) = %v, want %v", test.err, got, test.allowed)
		}
	}
}

func TestCompletionMessageReportsRefreshOrRestartPrecisely(t *testing.T) {
	if got := completionMessage(Uninstall, installer.OperationOutcome{GameListRefreshAccepted: true}); got != "Uninstall completed; game list refresh requested" {
		t.Fatalf("refresh message = %q", got)
	}
	if got := completionMessage(Adopt, installer.OperationOutcome{RestartRequired: true}); got != "Manage existing install completed; restart required to update game list" {
		t.Fatalf("restart message = %q", got)
	}
}

func assertActions(t *testing.T, got []Action, wanted ...Action) {
	t.Helper()
	if len(got) != len(wanted) {
		t.Fatalf("actions = %v, wanted %v", got, wanted)
	}
	for index := range got {
		if got[index] != wanted[index] {
			t.Fatalf("actions = %v, wanted %v", got, wanted)
		}
	}
}
