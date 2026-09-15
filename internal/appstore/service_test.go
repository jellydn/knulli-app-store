package appstore

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

func TestServiceExposesOnlyExperimentalPackagesAsActionable(t *testing.T) {
	indexPath := filepath.Join(t.TempDir(), "index.json")
	index, err := catalogForTest()
	if err != nil {
		t.Fatal(err)
	}
	if err := writeIndexForTest(index, indexPath); err != nil {
		t.Fatal(err)
	}
	service, err := Open(indexPath, installer.Manager{Root: t.TempDir(), Platform: platform.Info{
		Firmware: "knulli", Version: "2026.05", Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1280x720",
	}})
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 6 {
		t.Fatalf("expected six catalogue items, got %d", len(items))
	}
	experimental := 0
	for _, item := range items {
		if item.Package.Experimental() {
			experimental++
			if !item.Compatible || len(item.Actions) != 1 || item.Actions[0] != Install {
				t.Fatalf("experimental package is not installable: %#v", item)
			}
			continue
		}
		readOnlyStatus := strings.Contains(item.Compatibility, "Candidate") || strings.Contains(item.Compatibility, "installation is blocked")
		if item.Compatible || len(item.Actions) != 0 || !readOnlyStatus {
			t.Fatalf("candidate became actionable: %#v", item)
		}
	}
	if experimental != 2 {
		t.Fatalf("expected two experimental packages, got %d", experimental)
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
		"usr/share/knulli/knulli.version":     "scarab 2026/05/10 14:23\n",
		"boot/boot/knulli.board":              "trimui-smart-pro\n",
		"sys/class/graphics/fb0/virtual_size": "1280,720\n",
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
			if !item.Compatible || len(item.Actions) != 1 || !strings.Contains(item.Compatibility, "no minimum version is claimed") || !strings.Contains(item.Compatibility, "/etc/os-release:OS_NAME") {
				t.Fatalf("latest Knulli experimental decision is wrong: %#v", item)
			}
		}
	}
}

func TestActionsReflectInstallStateAndHealth(t *testing.T) {
	item := Item{Package: installablePackage(), Compatible: true}
	assertActions(t, actions(item), Install)
	item.PreExisting = true
	assertActions(t, actions(item), Adopt)
	item.PreExisting = false
	item.Installed = true
	item.InstalledVersion = item.Package.Version
	assertActions(t, actions(item), Uninstall, Repair)
	item.InstalledVersion = "0.9.0"
	assertActions(t, actions(item), Update, Uninstall, Repair)
	item.Compatible = false
	assertActions(t, actions(item), Uninstall)
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
