package ui

import (
	"context"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/manifest"
)

func tabItem(id string, state appstore.State, actions ...appstore.Action) appstore.Item {
	return appstore.Item{
		Package: manifest.Package{ID: id, Name: id},
		Actions: actions,
		Verdict: appstore.Verdict{State: state},
	}
}

func loadedModel(t *testing.T, items ...appstore.Item) *Model {
	t.Helper()
	model := New(&fakeBackend{items: items})
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return model
}

func visibleIDs(model *Model) []string {
	ids := make([]string, 0, len(model.Items))
	for _, item := range model.Items {
		ids = append(ids, item.Package.ID)
	}
	return ids
}

// The bar is a ring, so neither direction dead-ends: right from the last tab
// reaches the first, and left from the first reaches the last.
func TestTabsWrapBothWaysAndLeadWithEverything(t *testing.T) {
	tabs := Tabs{}
	if tabs.Active() != TabAll {
		t.Fatalf("the catalogue opens on %q, want the whole list", tabs.Active().Label())
	}
	for step, want := range []Tab{TabReady, TabInstalled, TabAll} {
		tabs.Cycle(1)
		if tabs.Active() != want {
			t.Fatalf("step %d right landed on %q, want %q", step, tabs.Active().Label(), want.Label())
		}
	}
	tabs.Cycle(-1)
	if tabs.Active() != TabInstalled {
		t.Fatalf("left from the first tab landed on %q, want the last", tabs.Active().Label())
	}
}

// A tab filters on the typed verdict, so what it shows cannot drift from the
// state it claims to filter on.
func TestTabsMatchPackagesByVerdictState(t *testing.T) {
	for _, test := range []struct {
		state     appstore.State
		ready     bool
		installed bool
	}{
		{state: appstore.StateAvailable, ready: true},
		{state: appstore.StateExternal, ready: true},
		{state: appstore.StateInstalled, installed: true},
		{state: appstore.StateIssue, installed: true},
		{state: appstore.StateCandidate},
		{state: appstore.StateIncompatible},
	} {
		item := tabItem("org.example.package", test.state)
		if !TabAll.Match(item) {
			t.Fatalf("%s is hidden from the whole catalogue", test.state)
		}
		if got := TabReady.Match(item); got != test.ready {
			t.Fatalf("Ready tab on %s = %v", test.state, got)
		}
		if got := TabInstalled.Match(item); got != test.installed {
			t.Fatalf("Installed tab on %s = %v", test.state, got)
		}
	}
}

// Sideways on the catalogue walks the tab bar: the whole list stays the default
// view, and each step narrows it without reloading the index.
func TestHorizontalStepsTabsOnTheCatalogue(t *testing.T) {
	model := loadedModel(t,
		tabItem("org.example.ready", appstore.StateAvailable),
		tabItem("org.example.installed", appstore.StateInstalled),
		tabItem("org.example.candidate", appstore.StateCandidate),
	)
	if got := len(model.Items); got != 3 {
		t.Fatalf("the whole catalogue shows %d packages, want 3", got)
	}

	model.Horizontal(1)
	if model.Tab() != TabReady {
		t.Fatalf("right landed on %q, want Ready", model.Tab().Label())
	}
	if got := visibleIDs(model); len(got) != 1 || got[0] != "org.example.ready" {
		t.Fatalf("Ready tab shows %v", got)
	}

	model.Horizontal(1)
	if model.Tab() != TabInstalled {
		t.Fatalf("right landed on %q, want Installed", model.Tab().Label())
	}
	if got := visibleIDs(model); len(got) != 1 || got[0] != "org.example.installed" {
		t.Fatalf("Installed tab shows %v", got)
	}

	model.Horizontal(1)
	if model.Tab() != TabAll || len(model.Items) != 3 {
		t.Fatalf("the bar did not wrap: %q shows %d", model.Tab().Label(), len(model.Items))
	}
	model.Horizontal(-1)
	if model.Tab() != TabInstalled {
		t.Fatalf("left landed on %q, want the last tab", model.Tab().Label())
	}
}

// A tab switch does not move the user off the package they were reading: the
// selection follows the package while it is still on screen, and lands on the
// first row when the new tab hides it.
func TestTabSwitchKeepsTheSelectedPackageOrLandsOnTheFirstRow(t *testing.T) {
	model := loadedModel(t,
		tabItem("org.example.installed", appstore.StateInstalled),
		tabItem("org.example.ready-a", appstore.StateAvailable),
		tabItem("org.example.ready-b", appstore.StateAvailable),
	)
	model.Selected = 2
	model.Horizontal(1)
	if items := visibleIDs(model); model.Selected != 1 || items[model.Selected] != "org.example.ready-b" {
		t.Fatalf("the tab switch lost the selected package: %v at %d", items, model.Selected)
	}

	// An installed package has no row in the Ready tab, so the selection falls
	// back to the first row the tab does show.
	model = loadedModel(t,
		tabItem("org.example.ready", appstore.StateAvailable),
		tabItem("org.example.installed", appstore.StateInstalled),
	)
	model.Selected = 1
	model.Horizontal(1)
	if items := visibleIDs(model); model.Selected != 0 || items[0] != "org.example.ready" {
		t.Fatalf("a package the tab hides left the selection at %d of %v", model.Selected, items)
	}
}

// A tab with nothing in it is still a tab: sideways steps out of it, which is
// the only way back to a list the user can act on.
func TestAnEmptyTabStillStepsSideways(t *testing.T) {
	model := loadedModel(t, tabItem("org.example.candidate", appstore.StateCandidate))
	model.Horizontal(1)
	if model.Tab() != TabReady || len(model.Items) != 0 {
		t.Fatalf("Ready tab shows %v", visibleIDs(model))
	}
	model.Horizontal(-1)
	if model.Tab() != TabAll || len(model.Items) != 1 {
		t.Fatalf("stepping out of the empty tab shows %v", visibleIDs(model))
	}
}

// Sideways means the tab bar on the catalogue and the button row inside a
// panel: there is no tab bar inside a package's own panel to walk.
func TestHorizontalStepsButtonsInsideAPanel(t *testing.T) {
	model := loadedModel(t,
		tabItem("org.example.alpha", appstore.StateInstalled, appstore.Repair, appstore.Uninstall),
		tabItem("org.example.beta", appstore.StateInstalled, appstore.Uninstall),
	)
	model.Focus = Actions
	model.Horizontal(1)
	if model.Action != 1 {
		t.Fatalf("right inside the panel selected action %d", model.Action)
	}
	if model.Tab() != TabAll {
		t.Fatalf("a panel button step changed the tab to %q", model.Tab().Label())
	}
	model.Horizontal(-1)
	if model.Action != 0 {
		t.Fatalf("left inside the panel selected action %d", model.Action)
	}
	// The rows do not move sideways either, so the two directions never fight
	// over the same selection.
	if model.Selected != 0 {
		t.Fatalf("a horizontal step moved the row selection to %d", model.Selected)
	}
}

// A refresh keeps the user where they were: the index is re-read after every
// operation, and a row that moved because another package was installed must
// not take the cursor with it.
func TestRefreshKeepsTheSelectedPackage(t *testing.T) {
	items := []appstore.Item{
		tabItem("org.example.alpha", appstore.StateInstalled),
		tabItem("org.example.beta", appstore.StateInstalled),
	}
	model := loadedModel(t, items...)
	model.Selected = 1
	model.setCatalogue(append([]appstore.Item{tabItem("org.example.gamma", appstore.StateInstalled)}, items...))
	if model.Selected != 2 || model.Items[model.Selected].Package.ID != "org.example.beta" {
		t.Fatalf("a refresh moved the selection to %d of %v", model.Selected, visibleIDs(model))
	}
}

func TestBusyModelDoesNotSwitchTabs(t *testing.T) {
	model := loadedModel(t, tabItem("org.example.alpha", appstore.StateAvailable))
	model.Busy = true
	model.Horizontal(1)
	if model.Tab() != TabAll {
		t.Fatalf("a tab switched during an operation: %q", model.Tab().Label())
	}
}
