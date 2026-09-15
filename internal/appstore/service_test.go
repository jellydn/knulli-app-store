package appstore

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

func TestServiceExposesCandidatesAsReadOnly(t *testing.T) {
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
	if len(items) != 5 {
		t.Fatalf("expected five catalogue items, got %d", len(items))
	}
	for _, item := range items {
		if item.Compatible || len(item.Actions) != 0 || !strings.Contains(item.Compatibility, "Candidate") {
			t.Fatalf("candidate became actionable: %#v", item)
		}
	}
}

func TestActionsReflectInstallStateAndHealth(t *testing.T) {
	item := Item{Package: installablePackage(), Compatible: true}
	assertActions(t, actions(item), Install)
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
