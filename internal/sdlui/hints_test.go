package sdlui

// Footer tests. They are tag-free because the hint selection is device
// agnostic; only the painting in draw.go needs SDL.

import (
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func catalogueModel() *storeui.Model {
	return &storeui.Model{Items: []appstore.Item{{}}}
}

func normalControls(t *testing.T) *storeinput.Session {
	t.Helper()
	controls := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
	controls.Connected = true
	controls.FirstRun = false
	controls.Mode = storeinput.Normal
	return controls
}

func TestCatalogueFooterUsesOneCanonicalHint(t *testing.T) {
	controls := normalControls(t)
	model := catalogueModel()
	// The catalogue is the top of the app: nothing sits behind it, so Back is
	// absent and the way out is the Select chord.
	if got := footerText(model, controls); got != "Confirm (SOUTH)  Settings (NORTH)  Quit (SELECT + NORTH)" {
		t.Fatalf("catalogue footer = %q", got)
	}
	model.Focus = storeui.Confirm
	if got := footerText(model, controls); got != "Confirm (SOUTH)  Back (EAST)" {
		t.Fatalf("confirmation footer = %q", got)
	}
	model.Focus = storeui.ForceConfirm
	if got := footerText(model, controls); got != "Confirm (SOUTH)  Back (EAST)" {
		t.Fatalf("force confirmation footer = %q", got)
	}
}

func TestEmptyCatalogueFooterDropsUnavailableActions(t *testing.T) {
	if got := footerText(&storeui.Model{}, normalControls(t)); got != "Settings (NORTH)  Quit (SELECT + NORTH)" {
		t.Fatalf("empty catalogue footer = %q", got)
	}
}

func TestBusyCatalogueDoesNotAdvertiseRejectedActions(t *testing.T) {
	model := catalogueModel()
	model.Busy = true
	if got := footerText(model, normalControls(t)); got != "" {
		t.Fatalf("busy catalogue footer = %q, want no actions", got)
	}
}

func TestFooterLabelsFollowTheActiveMapping(t *testing.T) {
	controls := normalControls(t)
	controls.Mapping = storeinput.AutoMapping()
	controls.Mapping[storeinput.Confirm] = 2
	controls.Mapping[storeinput.Back] = 0
	if got := footerText(catalogueModel(), controls); got != "Confirm (WEST)  Settings (NORTH)  Quit (SELECT + NORTH)" {
		t.Fatalf("footer ignored the custom mapping: %q", got)
	}
}

func TestFirstRunSetupFooterOffersTheDetectedBinding(t *testing.T) {
	controls := storeinput.NewSession(t.TempDir(), "magicx-zero-28")
	controls.Connect(storeinput.Identity{GUID: "03000000", Name: "Pad"}, true)
	if controls.Mode != storeinput.Setup || !controls.FirstRun {
		t.Fatalf("connect did not open first-run setup: mode=%q", controls.Mode)
	}
	if got := footerText(catalogueModel(), controls); got != "Confirm (SOUTH)  Quit (SELECT + NORTH)" {
		t.Fatalf("first-run footer = %q", got)
	}
}

func TestEveryScreenShowsOneFooterLine(t *testing.T) {
	model := catalogueModel()
	sessions := map[string]*storeinput.Session{}
	blocked := storeinput.NewSession(t.TempDir(), "magicx-zero-28")
	sessions["blocked"] = blocked
	firstRun := storeinput.NewSession(t.TempDir(), "magicx-zero-28")
	firstRun.Connect(storeinput.Identity{GUID: "03000000", Name: "Pad"}, true)
	sessions["setup"] = firstRun
	for name, mode := range map[string]storeinput.Mode{
		"optional-setup": storeinput.Setup,
		"paging":         storeinput.Paging,
		"settings":       storeinput.Settings,
		"review":         storeinput.Review,
		"calibrating":    storeinput.Calibrating,
		"preview":        storeinput.Preview,
	} {
		controls := storeinput.NewSession(t.TempDir(), "magicx-zero-28")
		controls.Connected = true
		controls.FirstRun = mode == storeinput.Setup
		controls.Mode = mode
		sessions[name] = controls
	}
	for name, controls := range sessions {
		footer := footerText(model, controls)
		if footer == "" {
			t.Fatalf("%s screen has no footer hint", name)
		}
		if strings.Contains(footer, "\n") {
			t.Fatalf("%s footer is not one line: %q", name, footer)
		}
	}
}

// Paging is offered only when the list is longer than the window: a catalogue
// that fits never advertises a gesture that would move nothing.
func TestCatalogueOffersPagingOnlyWhenTheListOutgrowsTheWindow(t *testing.T) {
	controls := normalControls(t)
	if got := footerText(catalogueModel(), controls); strings.Contains(got, "Page") {
		t.Fatalf("a catalogue that fits advertised paging: %q", got)
	}
	long := &storeui.Model{Items: make([]appstore.Item, listRows+1)}
	want := "Confirm (SOUTH)  Page (L1/R1)  Settings (NORTH)  Quit (SELECT + NORTH)"
	if got := footerText(long, controls); got != want {
		t.Fatalf("paged catalogue footer = %q, want %q", got, want)
	}
	if len(want) > footerCharacterLimit {
		t.Fatalf("the paged catalogue footer is %d characters, wider than the canvas allows (%d)", len(want), footerCharacterLimit)
	}
	// Paging belongs to the catalogue list, not to the screens with four fixed
	// choices of their own.
	long.Focus = storeui.Confirm
	if got := footerText(long, controls); strings.Contains(got, "Page") {
		t.Fatalf("a confirmation offered paging: %q", got)
	}
}

// A mapping that skipped paging binds neither button, so the catalogue must not
// promise a gesture the pad cannot make: the paired hint names both buttons or
// neither.
func TestCatalogueDropsThePageHintWhenPagingIsNotBound(t *testing.T) {
	controls := normalControls(t)
	controls.Mapping = storeinput.AutoMapping().Without(storeinput.PageUp, storeinput.PageDown)
	long := &storeui.Model{Items: make([]appstore.Item, listRows+1)}
	got := footerText(long, controls)
	if strings.Contains(got, "Page") {
		t.Fatalf("a mapping without paging still advertised paging: %q", got)
	}
	if want := "Confirm (SOUTH)  Settings (NORTH)  Quit (SELECT + NORTH)"; got != want {
		t.Fatalf("footer without paging = %q, want %q", got, want)
	}
}

func TestFooterFitsTheCanvasWithTheWidestLabels(t *testing.T) {
	controls := normalControls(t)
	controls.Mapping = storeinput.AutoMapping()
	controls.Mapping[storeinput.Confirm] = 10
	controls.Mapping[storeinput.Back] = 9
	controls.Mapping[storeinput.Diagnostics] = 10
	if footer := footerText(catalogueModel(), controls); len(footer) > footerCharacterLimit {
		t.Fatalf("footer is wider than the canvas: %q", footer)
	}
}
