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

// A customized pad that swaps the face buttons must keep showing compass
// labels for the bindings it saved, not the default AutoMapping letters.
// This is the device path that matters when lettering differs across shells:
// the footer must name the physical positions the user actually assigned.
func TestCustomizedMappingFooterUsesCompassLabelsFromSavedBindings(t *testing.T) {
	root := t.TempDir()
	identity := storeinput.Identity{GUID: "custom-compass", Name: "TSP Pad"}
	controls := storeinput.NewSession(root, "trimui-smart-pro")
	controls.Connect(identity, true)

	// First-run setup: CUSTOMIZE, then SKIP PAGING so only required actions
	// are assigned. Navigate with the detected (Auto) mapping while FirstRun.
	pressDetected := func(action storeinput.Action) {
		t.Helper()
		button := storeinput.AutoMapping()[action]
		controls.HandleButton(button)
	}
	pressDetected(storeinput.Down)
	pressDetected(storeinput.Down)
	pressDetected(storeinput.Confirm) // CUSTOMIZE
	if controls.Mode != storeinput.Paging {
		t.Fatalf("customize did not open the paging question: mode=%s", controls.Mode)
	}
	pressDetected(storeinput.Down)
	pressDetected(storeinput.Confirm) // SKIP PAGING
	if controls.Mode != storeinput.Calibrating {
		t.Fatalf("customize did not start calibration: mode=%s", controls.Mode)
	}

	// Required plan order: up, down, left, right, confirm, back, diagnostics.
	// Face buttons are deliberately swapped vs AutoMapping so the footer cannot
	// pass by accident: Confirm=EAST, Back=SOUTH, Settings=WEST.
	buttons := []int{
		11, // up
		12, // down
		13, // left
		14, // right
		1,  // confirm → EAST
		0,  // back → SOUTH
		2,  // diagnostics → WEST
	}
	for index, button := range buttons {
		controls.HandleButton(button)
		if controls.Mode != storeinput.Review {
			t.Fatalf("action %d skipped assignment review: mode=%s", index, controls.Mode)
		}
		pressDetected(storeinput.Confirm)
	}
	if controls.Mode != storeinput.Preview {
		t.Fatalf("custom mapping skipped preview: mode=%s error=%q", controls.Mode, controls.ValidationError)
	}
	for _, button := range buttons {
		controls.HandleButton(button)
	}
	if controls.Mode != storeinput.Normal || controls.Source != "saved custom mapping" {
		t.Fatalf("custom mapping was not saved: mode=%s source=%q error=%q", controls.Mode, controls.Source, controls.ValidationError)
	}
	if controls.Mapping[storeinput.Confirm] != 1 || controls.Mapping[storeinput.Back] != 0 || controls.Mapping[storeinput.Diagnostics] != 2 {
		t.Fatalf("saved face bindings are wrong: %#v", controls.Mapping)
	}

	// Catalogue footer must name the saved compass positions, including the
	// quit chord whose trigger is the saved Settings binding (WEST).
	if got := footerText(catalogueModel(), controls); got != "Confirm (EAST)  Settings (WEST)  Quit (SELECT + WEST)" {
		t.Fatalf("catalogue footer after customize = %q", got)
	}
	model := catalogueModel()
	model.Focus = storeui.Confirm
	if got := footerText(model, controls); got != "Confirm (EAST)  Back (SOUTH)" {
		t.Fatalf("confirmation footer after customize = %q", got)
	}

	// A later launch loads the same record and must keep the same labels.
	reloaded := storeinput.NewSession(root, "trimui-smart-pro")
	reloaded.Connect(identity, true)
	if reloaded.Mode != storeinput.Normal || reloaded.Source != "saved controller mapping" {
		t.Fatalf("saved custom mapping did not load: mode=%s source=%q", reloaded.Mode, reloaded.Source)
	}
	if got := footerText(catalogueModel(), reloaded); got != "Confirm (EAST)  Settings (WEST)  Quit (SELECT + WEST)" {
		t.Fatalf("reloaded catalogue footer = %q", got)
	}
	// Mapping summary labels are compass directions too, never A/B/X/Y.
	// Only the bound actions are checked: SortedLabels walks the full action
	// set, and an unbound optional button would read as the zero index.
	summary := map[storeinput.Action]string{}
	for action, button := range reloaded.Mapping {
		summary[action] = storeinput.ButtonLabel(button)
	}
	if summary[storeinput.Confirm] != "EAST" || summary[storeinput.Back] != "SOUTH" || summary[storeinput.Diagnostics] != "WEST" {
		t.Fatalf("saved face summary labels = %#v", summary)
	}
	for action, label := range summary {
		for _, letter := range []string{"A", "B", "X", "Y"} {
			if label == letter {
				t.Fatalf("%s summary still uses a letter face label: %q", action, label)
			}
		}
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
