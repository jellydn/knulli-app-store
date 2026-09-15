//go:build sdl
// +build sdl

package sdlui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func TestRenderRepresentativeStates(t *testing.T) {
	item := appstore.Item{
		Package:   manifest.Package{ID: "org.example.demo", Name: "Demo Utility", Type: "utility", Summary: "A safe package used to verify action and error layouts.", Review: manifest.Review{Status: "installable"}},
		Installed: true, InstalledVersion: "1.0.0", Healthy: false, Compatible: true,
		Compatibility: "Compatible with detected platform", Actions: []appstore.Action{appstore.Repair, appstore.Uninstall},
	}
	states := map[string]*storeui.Model{
		"confirm": {Items: []appstore.Item{item}, Focus: storeui.Confirm},
		"error":   {Items: []appstore.Item{item}, Error: "SHA-256 mismatch; package files were not changed"},
	}
	directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR")
	for name, model := range states {
		frame := draw(model, "KNULLI 2026.05 / AARCH64 / TRIMUI-SMART-PRO / 1280X720", true)
		if frame.Bounds().Dx() != canvasWidth || frame.Bounds().Dy() != canvasHeight {
			t.Fatalf("%s frame has unexpected bounds %v", name, frame.Bounds())
		}
		if directory != "" {
			if err := os.MkdirAll(directory, 0755); err != nil {
				t.Fatal(err)
			}
			if err := saveScreenshot(filepath.Join(directory, "trimui-smart-pro-gui-"+name+".png"), frame); err != nil {
				t.Fatal(err)
			}
		}
	}
}
