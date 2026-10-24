package gamelist

import (
	"fmt"
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

func TestDeriveMetadataUpdateAndSameGamelistMove(t *testing.T) {
	old := menu(testGamelist, "./demo/launch.sh", "Old name", "Old description")
	metadata := menu(testGamelist, old.Path, "New name", "New description")
	plan := Derive(true, old, metadata)
	if len(plan.Steps) != 1 || plan.Steps[0].Kind != StepReplace || plan.Steps[0].Previous != old {
		t.Fatalf("metadata update should replace the owned entry: %+v", plan.Steps)
	}
	move := menu(testGamelist, "./demo/new.sh", "New name", "New description")
	plan = Derive(true, old, move)
	if len(plan.Steps) != 2 || plan.Steps[0].Kind != StepRemove || plan.Steps[1].Kind != StepAdd {
		t.Fatalf("same-gamelist path move should remove then add: %+v", plan.Steps)
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

func TestApplyRejectsReplaceWithoutPrevious(t *testing.T) {
	_, guard, tx := newTx(t)
	entry := *menu(testGamelist, "./demo/launch.sh", "Demo", "Fixture")
	changed, owned, err := Apply(tx, guard, Plan{Steps: []Step{{Kind: StepReplace, Menu: entry}}})
	if err == nil || !strings.Contains(err.Error(), "requires previous") || changed || owned {
		t.Fatalf("malformed replace = changed %v, owned %v, error %v", changed, owned, err)
	}
}

func TestApplyMetadataUpdateAndSameGamelistMove(t *testing.T) {
	for _, test := range []struct {
		name string
		next *manifest.Menu
	}{
		{name: "metadata update", next: menu(testGamelist, "./demo/launch.sh", "New name", "New description")},
		{name: "path move", next: menu(testGamelist, "./demo/new.sh", "New name", "New description")},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, guard, tx := newTx(t)
			old := menu(testGamelist, "./demo/launch.sh", "Old name", "Old description")
			test.next.Image = "./demo/art & cover.png"
			writeGamelistHost(t, guard, "<gameList><game><path>./demo/launch.sh</path><name>Old name</name><desc>Old description</desc></game></gameList>")
			changed, owned, err := Apply(tx, guard, Derive(true, old, test.next))
			if err != nil || !changed || !owned {
				t.Fatalf("apply = changed %v, owned %v, error %v", changed, owned, err)
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
			text := string(data)
			if strings.Contains(text, "Old name") || !strings.Contains(text, test.next.Path) || !strings.Contains(text, "New description") || !strings.Contains(text, "<image>./demo/art &amp; cover.png</image>") {
				t.Fatalf("unexpected updated gamelist: %s", text)
			}
		})
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

func TestEditedArtworkIsNotReplacedOrRemoved(t *testing.T) {
	for _, action := range []string{"replace", "remove"} {
		t.Run(action, func(t *testing.T) {
			_, guard, tx := newTx(t)
			old := menu(testGamelist, "./demo/launch.sh", "Demo", "Description")
			old.Image = "./demo/original.png"
			data := "<gameList><game><path>./demo/launch.sh</path><name>Demo</name><desc>Description</desc><image>./my-cover.png</image></game></gameList>"
			writeGamelistHost(t, guard, data)
			next := menu(testGamelist, old.Path, "New name", "New description")
			if action == "remove" {
				next = nil
			}
			changed, owned, err := Apply(tx, guard, Derive(true, old, next))
			if err != nil || changed || owned {
				t.Fatalf("edited artwork not preserved: changed=%v owned=%v err=%v", changed, owned, err)
			}
			if err := tx.Commit(); err != nil {
				t.Fatal(err)
			}
			host, err := guard.Resolve(testGamelist)
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(host)
			if err != nil || string(got) != data {
				t.Fatalf("edited artwork changed: %s %v", got, err)
			}
		})
	}
}

func TestApplyFillsOnlyBlankMetadataWithoutClaimingExistingEntry(t *testing.T) {
	for _, oldOwned := range []bool{false, true} {
		for _, fields := range []string{
			"<name> </name>",
			"<name>playtime</name>",
			"<name>playtime</name><desc> </desc><image source=\"user\">./custom.png</image>",
			"<name>playtime</name><desc>My description</desc><image>./custom.png</image>",
		} {
			t.Run(fmt.Sprintf("owned=%t/%s", oldOwned, fields), func(t *testing.T) {
				_, guard, tx := newTx(t)
				data := "<gameList><game><path>./other.sh</path><name>Other</name></game><game id=\"keep\"><path>./demo/launch.sh</path>" + fields + "<playcount>17</playcount><lastplayed>20261020T150000</lastplayed><favorite>true</favorite></game></gameList>"
				writeGamelistHost(t, guard, data)
				entry := menu(testGamelist, "./demo/launch.sh", "Reviewed name", "Reviewed description")
				entry.Image = "./demo/cover.png"
				changed, owned, err := Apply(tx, guard, Derive(oldOwned, entry, entry))
				wantChange := !strings.Contains(fields, "My description")
				if err != nil || changed != wantChange || owned {
					t.Fatalf("metadata repair: changed=%v owned=%v err=%v", changed, owned, err)
				}
				if err := tx.Commit(); err != nil {
					t.Fatal(err)
				}
				host, _ := guard.Resolve(testGamelist)
				got, err := os.ReadFile(host)
				if err != nil {
					t.Fatal(err)
				}
				wantDesc := "Reviewed description"
				if strings.Contains(fields, "My description") {
					wantDesc = "My description"
				}
				wantImage := "./demo/cover.png"
				if strings.Contains(fields, "./custom.png") {
					wantImage = "./custom.png"
				}
				wantName := "playtime"
				if !strings.Contains(fields, "playtime") {
					wantName = "Reviewed name"
				}
				for _, want := range []string{"<name>" + wantName + "</name>", "<desc>" + wantDesc + "</desc>", wantImage, "<playcount>17</playcount>", "<lastplayed>20261020T150000</lastplayed>", "<favorite>true</favorite>", "id=\"keep\"", "<path>./other.sh</path>"} {
					if !strings.Contains(string(got), want) {
						t.Fatalf("lost %q: %s", want, got)
					}
				}
				if strings.Count(string(got), "<desc>") != 1 || strings.Count(string(got), "<image") != 1 {
					t.Fatalf("duplicate metadata: %s", got)
				}
			})
		}
	}
}
