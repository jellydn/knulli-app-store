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
	if got := footerText(model, controls); got != "SELECT (SDL A)  BACK (SDL B)  SETTINGS (SDL Y)" {
		t.Fatalf("catalogue footer = %q", got)
	}
	model.Focus = storeui.Confirm
	if got := footerText(model, controls); got != "SELECT (SDL A)  BACK (SDL B)" {
		t.Fatalf("confirmation footer = %q", got)
	}
	model.Focus = storeui.ForceConfirm
	if got := footerText(model, controls); got != "SELECT (SDL A)  BACK (SDL B)" {
		t.Fatalf("force confirmation footer = %q", got)
	}
}

func TestEmptyCatalogueFooterDropsUnavailableActions(t *testing.T) {
	if got := footerText(&storeui.Model{}, normalControls(t)); got != "BACK (SDL B)  SETTINGS (SDL Y)" {
		t.Fatalf("empty catalogue footer = %q", got)
	}
}

func TestFooterLabelsFollowTheActiveMapping(t *testing.T) {
	controls := normalControls(t)
	controls.Mapping = storeinput.AutoMapping()
	controls.Mapping[storeinput.Confirm] = 2
	controls.Mapping[storeinput.Back] = 0
	if got := footerText(catalogueModel(), controls); got != "SELECT (SDL X)  BACK (SDL A)  SETTINGS (SDL Y)" {
		t.Fatalf("footer ignored the custom mapping: %q", got)
	}
}

func TestFirstRunSetupFooterOffersTheDetectedBinding(t *testing.T) {
	controls := storeinput.NewSession(t.TempDir(), "magicx-zero-28")
	controls.Connect(storeinput.Identity{GUID: "03000000", Name: "Pad"}, true)
	if controls.Mode != storeinput.Setup || !controls.FirstRun {
		t.Fatalf("connect did not open first-run setup: mode=%q", controls.Mode)
	}
	if got := footerText(catalogueModel(), controls); got != "SELECT (SDL A)  EXIT (SDL START)" {
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
