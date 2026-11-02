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

func TestCategoriesUsePackageTypeNotNameOrInstallState(t *testing.T) {
	for _, kind := range []string{"game", "utility", "theme", "integration"} {
		for _, state := range []appstore.State{appstore.StateAvailable, appstore.StateInstalled, appstore.StateIssue, appstore.StateCandidate, appstore.StateIncompatible} {
			item := tabItem("org.example.game-picker", state)
			item.Package.Type = kind
			if CategoryGames.Match(item) != (kind == "game") || CategoryApps.Match(item) != (kind != "game") {
				t.Fatalf("wrong category for type %s in state %s", kind, state)
			}
		}
	}

	game := tabItem("org.example.freedoom", appstore.StateAvailable)
	game.Package.Type = "game"
	picker := tabItem("org.example.random-game-picker", appstore.StateInstalled)
	picker.Package.Type = "utility"
	theme := tabItem("org.example.theme", appstore.StateAvailable)
	theme.Package.Type = "theme"
	model := loadedModel(t, game, picker, theme)
	model.Horizontal(1)
	if got := visibleIDs(model); len(got) != 1 || got[0] != game.Package.ID {
		t.Fatalf("Games shows %v", got)
	}
	model.Horizontal(1)
	if got := visibleIDs(model); len(got) != 2 || got[0] != picker.Package.ID || got[1] != theme.Package.ID {
		t.Fatalf("Apps shows %v", got)
	}
	model.Move(1)
	model.Horizontal(-1)
	model.Horizontal(1)
	if got := model.Items[model.Selected].Package.ID; got != theme.Package.ID {
		t.Fatalf("Apps forgot selection: %s", got)
	}
	game.Verdict.State = appstore.StateInstalled
	model.setCatalogue([]appstore.Item{game, picker, theme})
	model.Horizontal(-1)
	if got := visibleIDs(model); len(got) != 1 || got[0] != game.Package.ID {
		t.Fatalf("installed game left Games: %v", got)
	}
	model.CycleTab(1)
	if len(model.Items) != 0 || model.Tabs.Category() != CategoryGames {
		t.Fatal("Games + Ready must exclude installed games without changing category")
	}
	model.CycleTab(1)
	if got := visibleIDs(model); len(got) != 1 || got[0] != game.Package.ID {
		t.Fatalf("Games + Installed shows %v", got)
	}
	model.Horizontal(1)
	if got := visibleIDs(model); len(got) != 1 || got[0] != picker.Package.ID {
		t.Fatalf("Apps + Installed shows %v", got)
	}
	model.Horizontal(1)
	if model.Tabs.Category() != CategoryAll || len(model.Items) != 2 {
		t.Fatal("category did not wrap to All types within Installed")
	}
	model.Horizontal(-1)
	if model.Tabs.Category() != CategoryApps || len(model.Items) != 1 {
		t.Fatal("category did not wrap backwards")
	}
}

// Filter cycling narrows the catalogue without reloading its index.
func TestCycleStepsTabsOnTheCatalogue(t *testing.T) {
	model := loadedModel(t,
		tabItem("org.example.ready", appstore.StateAvailable),
		tabItem("org.example.installed", appstore.StateInstalled),
		tabItem("org.example.candidate", appstore.StateCandidate),
	)
	if got := len(model.Items); got != 3 {
		t.Fatalf("the whole catalogue shows %d packages, want 3", got)
	}

	model.CycleTab(1)
	if model.Tab() != TabReady {
		t.Fatalf("right landed on %q, want Ready", model.Tab().Label())
	}
	if got := visibleIDs(model); len(got) != 1 || got[0] != "org.example.ready" {
		t.Fatalf("Ready tab shows %v", got)
	}

	model.CycleTab(1)
	if model.Tab() != TabInstalled {
		t.Fatalf("right landed on %q, want Installed", model.Tab().Label())
	}
	if got := visibleIDs(model); len(got) != 1 || got[0] != "org.example.installed" {
		t.Fatalf("Installed tab shows %v", got)
	}

	model.CycleTab(1)
	if model.Tab() != TabAll || len(model.Items) != 3 {
		t.Fatalf("the bar did not wrap: %q shows %d", model.Tab().Label(), len(model.Items))
	}
	model.CycleTab(-1)
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
	model.CycleTab(1)
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
	model.CycleTab(1)
	if items := visibleIDs(model); model.Selected != 0 || items[0] != "org.example.ready" {
		t.Fatalf("a package the tab hides left the selection at %d of %v", model.Selected, items)
	}
}

// A tab remembers where it was left, so coming back to it lands where the
// reader stopped rather than at the top of a list they had already crossed.
func TestATabRemembersTheRowItWasLeftOn(t *testing.T) {
	model := loadedModel(t,
		tabItem("org.example.ready-a", appstore.StateAvailable),
		tabItem("org.example.ready-b", appstore.StateAvailable),
		tabItem("org.example.installed", appstore.StateInstalled),
	)
	model.Selected = 1 // ready-b, in the whole list
	model.CycleTab(1)
	// The Ready tab has never been open, so it carries the package the user was
	// already reading.
	if got := model.Items[model.Selected].Package.ID; got != "org.example.ready-b" {
		t.Fatalf("a tab that was never open landed on %q", got)
	}
	model.CycleTab(1)
	if got := model.Items[model.Selected].Package.ID; got != "org.example.installed" {
		t.Fatalf("stepping to the installed tab landed on %q", got)
	}
	// Ready was left on ready-b, which is not the first row, so landing there
	// again is the memory rather than the fallback.
	model.CycleTab(-1)
	if got := model.Items[model.Selected].Package.ID; got != "org.example.ready-b" {
		t.Fatalf("Ready did not remember the row it was left on: %q", got)
	}
}

// A remembered package can leave the tab while the user is elsewhere — an
// uninstall takes it out of the index — and the tab then starts at its first
// row rather than on a package it no longer shows.
func TestATabFallsBackWhenItsRememberedPackageIsGone(t *testing.T) {
	model := loadedModel(t,
		tabItem("org.example.ready-a", appstore.StateAvailable),
		tabItem("org.example.ready-b", appstore.StateAvailable),
		tabItem("org.example.installed", appstore.StateInstalled),
	)
	model.Selected = 1
	model.CycleTab(1) // Ready, carrying ready-b
	model.CycleTab(1) // Installed, remembering Ready on ready-b
	model.setCatalogue([]appstore.Item{
		tabItem("org.example.ready-a", appstore.StateAvailable),
		tabItem("org.example.installed", appstore.StateInstalled),
	})
	model.CycleTab(-1)
	if got := model.Items[model.Selected].Package.ID; got != "org.example.ready-a" {
		t.Fatalf("Ready landed on %q after its remembered package left", got)
	}
}

// A tab with nothing in it is still a tab: filter cycling steps out of it, which is
// the only way back to a list the user can act on.
func TestAnEmptyTabStillAllowsFiltering(t *testing.T) {
	model := loadedModel(t, tabItem("org.example.candidate", appstore.StateCandidate))
	model.CycleTab(1)
	if model.Tab() != TabReady || len(model.Items) != 0 {
		t.Fatalf("Ready tab shows %v", visibleIDs(model))
	}
	model.CycleTab(-1)
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
	model.CycleTab(1)
	model.Horizontal(1)
	if model.Tab() != TabAll || model.Tabs.Category() != CategoryAll {
		t.Fatal("a selector changed during an operation")
	}
}
