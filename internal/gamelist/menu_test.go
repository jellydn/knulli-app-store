package gamelist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

const testGamelist = "/userdata/roms/tools/gamelist.xml"

// newTx builds a real guard and transaction over a temp root, with the test
// gamelist as the only allowed write path.
func newTx(t *testing.T) (root string, guard *safefs.Guard, tx *safefs.Transaction) {
	t.Helper()
	root = t.TempDir()
	guard, err := safefs.NewGuard(root, []string{testGamelist})
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "tx")
	if err := os.MkdirAll(parent, 0700); err != nil {
		t.Fatal(err)
	}
	gamelistHost, err := guard.Resolve(testGamelist)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(gamelistHost), 0700); err != nil {
		t.Fatal(err)
	}
	tx, err = safefs.Begin(guard, parent)
	if err != nil {
		t.Fatal(err)
	}
	return root, guard, tx
}

func writeGamelistHost(t *testing.T, guard *safefs.Guard, data string) {
	t.Helper()
	host, err := guard.Resolve(testGamelist)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(host), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(host, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func menu(gamelist, path, name, desc string) *manifest.Menu {
	return &manifest.Menu{Gamelist: gamelist, Path: path, Name: name, Description: desc}
}

func TestDeriveFromNothing(t *testing.T) {
	menu := menu("/userdata/roms/tools/gamelist.xml", "./demo/launch.sh", "Demo", "Fixture")
	plan := Derive(false, nil, menu)
	if len(plan.Steps) != 1 || plan.Steps[0].Kind != StepAdd || plan.Steps[0].Menu != *menu {
		t.Fatalf("expected one add step, got %+v", plan.Steps)
	}
}

func TestDeriveFromOwnedEntry(t *testing.T) {
	menu := menu("/userdata/roms/tools/gamelist.xml", "./demo/launch.sh", "Demo", "Fixture")
	plan := Derive(true, menu, menu)
	if len(plan.Steps) != 1 || plan.Steps[0].Kind != StepReplace || plan.Steps[0].Previous != menu {
		t.Fatalf("expected one replace step, got %+v", plan.Steps)
	}
}

func TestDeriveMovesOwnershipToNewGamelist(t *testing.T) {
	old := menu("/userdata/roms/tools/gamelist.xml", "./demo/launch.sh", "Demo", "Fixture")
	next := menu("/userdata/roms/portmaster/gamelist.xml", "./demo/launch.sh", "Demo", "Fixture")
	plan := Derive(true, old, next)
	if len(plan.Steps) != 2 || plan.Steps[0].Kind != StepRemove || plan.Steps[1].Kind != StepAdd {
		t.Fatalf("expected remove then add, got %+v", plan.Steps)
	}
}

func TestDeriveReleasesOwnership(t *testing.T) {
	old := menu("/userdata/roms/tools/gamelist.xml", "./demo/launch.sh", "Demo", "Fixture")
	plan := Derive(true, old, nil)
	if len(plan.Steps) != 1 || plan.Steps[0].Kind != StepRemove {
		t.Fatalf("expected one remove step, got %+v", plan.Steps)
	}
}

func TestDeriveWithoutOwnershipAddsBesideUnownedEntry(t *testing.T) {
	old := menu("/userdata/roms/tools/gamelist.xml", "./demo/launch.sh", "Demo", "Fixture")
	next := menu("/userdata/roms/tools/gamelist.xml", "./demo/launch.sh", "Demo", "Fixture")
	plan := Derive(false, old, next)
	if len(plan.Steps) != 1 || plan.Steps[0].Kind != StepAdd {
		t.Fatalf("expected one add step (unowned entries are never edited), got %+v", plan.Steps)
	}
}

func TestApplyAddsEntryAndTransfersOwnership(t *testing.T) {
	_, guard, tx := newTx(t)
	entry := *menu(testGamelist, "./demo/launch.sh", "Demo", "Fixture")
	changed, owned, err := Apply(tx, guard, Plan{Steps: []Step{{Kind: StepAdd, Menu: entry}}})
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !owned {
		t.Fatalf("expected change with ownership, got changed=%v owned=%v", changed, owned)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	host, err := guard.Resolve(testGamelist)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(host)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<path>./demo/launch.sh</path>") {
		t.Fatalf("entry not written: %s", data)
	}
}

func TestApplyFailureTransfersNoOwnership(t *testing.T) {
	_, guard, tx := newTx(t)
	writeGamelistHost(t, guard, "<gameList><broken></gameList>")
	entry := *menu(testGamelist, "./demo/launch.sh", "Demo", "Fixture")
	changed, owned, err := Apply(tx, guard, Plan{Steps: []Step{{Kind: StepAdd, Menu: entry}}})
	if err == nil {
		t.Fatal("expected parse failure, got none")
	}
	if changed || owned {
		t.Fatalf("failed apply must change nothing and own nothing, got changed=%v owned=%v", changed, owned)
	}
}

func TestApplyNoOpDoesNotTransferOwnership(t *testing.T) {
	root, guard, tx := newTx(t)
	entry := *menu(testGamelist, "./demo/launch.sh", "Demo", "Fixture")
	changed, owned, err := Apply(tx, guard, Plan{Steps: []Step{{Kind: StepAdd, Menu: entry}}})
	if err != nil {
		t.Fatal(err)
	}
	if !changed || !owned {
		t.Fatalf("first add should own, got changed=%v owned=%v", changed, owned)
	}
	// A second add of the same path finds the entry and changes nothing:
	// ownership must not be claimed by a write that did not happen.
	tx2Parent := filepath.Join(root, "tx2")
	if err := os.MkdirAll(tx2Parent, 0700); err != nil {
		t.Fatal(err)
	}
	tx2, err := safefs.Begin(guard, tx2Parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	changed, owned, err = Apply(tx2, guard, Plan{Steps: []Step{{Kind: StepAdd, Menu: entry}}})
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("second add of the same entry must be a no-op")
	}
	if owned {
		t.Fatal("no-op add must not claim ownership")
	}
}
