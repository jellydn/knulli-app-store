package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The lifecycle record is a store in its own right, so it is exercised without
// running an operation: a retry rule should not need a package to be installed
// before it can be tested.
func TestLifecycleStoreRoundTripsAndStampsTheSchema(t *testing.T) {
	root := t.TempDir()
	store, err := newLifecycleStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(LifecycleState{PackageID: "org.example.demo", RequestedOperation: "install", DetectedInstallType: "absent", RetryTarget: "install"}); err != nil {
		t.Fatal(err)
	}

	got, err := store.Load("org.example.demo")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("a saved record did not read back")
	}
	if got.Schema != lifecycleSchemaV1 {
		t.Fatalf("schema = %q, want %q", got.Schema, lifecycleSchemaV1)
	}
	if got.RequestedOperation != "install" || got.RetryTarget != "install" || got.DetectedInstallType != "absent" {
		t.Fatalf("unexpected retry context: %#v", got)
	}
	if _, err := os.Stat(filepath.Join(root, strings.TrimPrefix(lifecyclePath("org.example.demo"), "/"))); err != nil {
		t.Fatalf("the record did not land below manager state: %v", err)
	}
}

// A package with no record is a package with nothing to retry, not an error.
func TestLifecycleStoreReportsNoRecordAsNoContext(t *testing.T) {
	store, err := newLifecycleStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Load("org.example.demo")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("an absent record read as %#v", got)
	}
}

// A record filed under one package that names another, or declares a schema the
// store does not know, is refused rather than obeyed. This is the check that
// stops a copied or hand-edited record from steering a retry.
func TestLifecycleStoreRefusesARecordItCannotTrust(t *testing.T) {
	catalogue := []struct {
		name string
		body string
	}{
		{name: "names another package", body: `{"schema":"` + lifecycleSchemaV1 + `","package_id":"org.example.other"}`},
		{name: "declares an unknown schema", body: `{"schema":"org.example/other/v1","package_id":"org.example.demo"}`},
	}
	for _, item := range catalogue {
		root := t.TempDir()
		store, err := newLifecycleStore(root)
		if err != nil {
			t.Fatal(err)
		}
		// Placed at the path the read will use, so the refusal proves the check
		// rather than an absent file.
		host := filepath.Join(root, strings.TrimPrefix(lifecyclePath("org.example.demo"), "/"))
		if err := os.MkdirAll(filepath.Dir(host), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(host, []byte(item.body), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Load("org.example.demo"); err == nil {
			t.Fatalf("a record that %s was accepted", item.name)
		}
	}
}

func TestLifecycleStoreClearsOnceAndStaysCleared(t *testing.T) {
	root := t.TempDir()
	store, err := newLifecycleStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(LifecycleState{PackageID: "org.example.demo", RequestedOperation: "repair", RetryTarget: "repair"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Clear("org.example.demo"); err != nil {
		t.Fatal(err)
	}
	if got, err := store.Load("org.example.demo"); err != nil || got != nil {
		t.Fatalf("a cleared record read as %#v, %v", got, err)
	}
	if err := store.Clear("org.example.demo"); err != nil {
		t.Fatalf("clearing an already clear record failed: %v", err)
	}
}

// The manager keeps its own surface, so callers do not learn about the store.
func TestManagerLifecycleMethodsUseTheStore(t *testing.T) {
	root := t.TempDir()
	manager := Manager{Root: root}
	if err := manager.RecordLifecycle(LifecycleState{PackageID: "org.example.demo", RequestedOperation: "update", RetryTarget: "update"}); err != nil {
		t.Fatal(err)
	}
	got, err := manager.LifecycleState("org.example.demo")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.RequestedOperation != "update" {
		t.Fatalf("manager read %#v", got)
	}
	if err := manager.ClearLifecycle("org.example.demo"); err != nil {
		t.Fatal(err)
	}
	if got, err := manager.LifecycleState("org.example.demo"); err != nil || got != nil {
		t.Fatalf("manager read %#v after clearing, %v", got, err)
	}
}
