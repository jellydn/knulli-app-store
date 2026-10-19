package sdlui

import (
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func catalogueModel() *storeui.Model { return &storeui.Model{Items: []appstore.Item{{}}} }

func TestAutomaticFooterAtEveryFocus(t *testing.T) {
	for _, layout := range []string{"knulli", "xbox", ""} {
		t.Setenv("STORE_CONTROLLER_LAYOUT", layout)
		controls := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
		controls.Connect(storeinput.Identity{Name: "Pad"}, true)
		model := catalogueModel()
		if got := footerText(model, controls); got != "Confirm (A)  Diagnostics (Y)  Quit (SELECT + Y)" {
			t.Fatal(got)
		}
		for _, focus := range []storeui.Focus{storeui.Confirm, storeui.ForceConfirm} {
			model.Focus = focus
			if got := footerText(model, controls); got != "Confirm (A)  Back (B)" {
				t.Fatal(got)
			}
		}
		model.Focus = storeui.Browse
		model.Items = make([]appstore.Item, listRows+1)
		if got := footerText(model, controls); got != "Confirm (A)  Page (L1/R1)  Diagnostics (Y)  Quit (SELECT + Y)" || len(got) > footerCharacterLimit {
			t.Fatal(got)
		}
		model.Busy = true
		if got := footerText(model, controls); got != "" {
			t.Fatal("busy footer promises rejected actions: " + got)
		}
		if got := footerText(&storeui.Model{}, controls); got != "Diagnostics (Y)  Quit (SELECT + Y)" {
			t.Fatal(got)
		}
		controls.Disconnect()
		if got := footerText(model, controls); got != "Confirm (A)  Quit (SELECT + Y)" {
			t.Fatal(got)
		}
	}
}

func TestDesktopFooterFitsAndNamesKeys(t *testing.T) {
	controls := storeinput.NewDesktopSession(t.TempDir(), "")
	model := &storeui.Model{Items: make([]appstore.Item, listRows+1)}
	got := footerText(model, controls)
	if got != "Confirm (ENTER)  Page (PGUP/PGDN)  Diagnostics (Y)  Quit (TAB + Y)" || len(got) > footerCharacterLimit || strings.Contains(got, "\n") {
		t.Fatal(got)
	}
}
