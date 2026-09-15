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
	experimental := appstore.Item{
		Package: manifest.Package{
			ID: "app.romm.grout", Name: "Grout", Type: "integration", Summary: "Connects a Linux retro handheld to a RomM server.",
			Review:  manifest.Review{Status: "experimental", Approval: &manifest.Approval{Provenance: "community"}},
			Install: &manifest.Install{Warning: "Unverified. Do not use Grout updater. Update only through Knulli App Store."},
		},
		PreExisting: true, Compatible: true, Compatibility: "Experimental compatibility: Knulli identity confirmed from /etc/os-release:OS_NAME; no minimum version is claimed; device=trimui-smart-pro architecture=aarch64 resolution=1280x720 version=scarab", Actions: []appstore.Action{appstore.Adopt},
	}
	incompatible := experimental
	incompatible.PreExisting = false
	incompatible.Compatible = false
	incompatible.Actions = nil
	incompatible.Compatibility = `compatibility failed field=firmware: detected device="trimui-smart-pro" architecture="aarch64" resolution="1280x720" firmware_raw="buildroot" firmware="" firmware_source="/etc/os-release:ID"; package requires firmware="knulli"`
	emuDrop := appstore.Item{
		Package: manifest.Package{
			ID: "io.github.ahmadteeb.emudrop", Name: "EmuDrop", Type: "utility", Summary: "Browses third-party sources and downloads ROM files and artwork.",
			Review: manifest.Review{
				Status: "candidate", Approval: &manifest.Approval{Provenance: "community"},
				Notes: []string{"WARNING: ROM downloads can infringe copyright. Users must confirm that each download is lawful in their jurisdiction and that they have the required rights."},
			},
		},
		Compatibility: "Community approved; installation is blocked by technical review",
	}
	states := map[string]*storeui.Model{
		"browse":              {Items: []appstore.Item{emuDrop}},
		"compatibility-error": {Items: []appstore.Item{incompatible}},
		"confirm":             {Items: []appstore.Item{experimental}, Focus: storeui.Confirm},
		"diagnostics":         {Items: []appstore.Item{experimental}, Message: "Diagnostics saved to /userdata/system/knulli-app-store/diagnostics/knulli-app-store-diagnostics-20260915T073500Z.txt"},
		"error":               {Items: []appstore.Item{item}, Error: "SHA-256 mismatch; package files were not changed"},
		"progress":            {Items: []appstore.Item{experimental}, Busy: true, Message: "Downloading, verifying, and applying Grout"},
	}
	directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR")
	for name, model := range states {
		frame := draw(model, "TrimUI Smart Pro / 1280x720", true)
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

func TestWrapTextBreaksLongDiagnosticPaths(t *testing.T) {
	lines := wrapText("SAVED /userdata/system/knulli-app-store/diagnostics/knulli-app-store-diagnostics.txt", 20)
	for _, line := range lines {
		if len(line) > 20 {
			t.Fatalf("line is wider than the panel: %q", line)
		}
	}
}
