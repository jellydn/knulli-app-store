//go:build sdl
// +build sdl

package sdlui

import (
	"bytes"
	"context"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func TestRenderRepresentativeStates(t *testing.T) {
	item := appstore.Item{
		Package:   manifest.Package{ID: "org.example.demo", Name: "Demo Utility", Type: "utility", Summary: "A safe package used to verify action and error layouts.", Review: manifest.Review{Status: "installable"}},
		Installed: true, InstalledVersion: "1.0.0", Healthy: false, Compatible: true,
		HealthReason: "mode changed: /userdata/roms/tools/Demo/demo (expected 0755, got 0644)", Compatibility: "Compatible with detected platform", Actions: []appstore.Action{appstore.Repair, appstore.Uninstall},
	}
	experimental := appstore.Item{
		Package: manifest.Package{
			ID: "app.romm.grout", Name: "Grout", Type: "integration", Summary: "Connects a Linux retro handheld to a RomM server.",
			Review:  manifest.Review{Status: "experimental", Approval: &manifest.Approval{Provenance: "community"}},
			Install: &manifest.Install{Warning: "Unverified. Do not use Grout updater. Update only through Knulli App Store."},
		},
		PreExisting: true, Compatible: true, Compatibility: "Experimental compatibility: Knulli identity confirmed from /etc/os-release:OS_NAME; no minimum version is claimed; device=trimui-smart-pro architecture=aarch64 resolution=1280x720 source=SDL renderer output version=scarab", Actions: []appstore.Action{appstore.Adopt},
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
	controls := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
	controls.Connected = true
	controls.Source = "Knulli SDL_GAMECONTROLLERCONFIG"
	controls.Mode = storeinput.Normal
	directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR")
	for name, model := range states {
		frame := draw(model, "TrimUI Smart Pro / 1280x720", controls)
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

func TestRenderDedicatedFirstRunSetupAtBothTargetResolutions(t *testing.T) {
	model := &storeui.Model{Items: []appstore.Item{{Package: manifest.Package{Name: "PlayTime", Review: manifest.Review{Status: "experimental"}}, Compatibility: "Blocked: MagicX runtime ABI is not yet verified"}}}
	tests := []struct {
		device   string
		header   string
		width    int
		height   int
		viewport image.Rectangle
	}{
		{device: "trimui-smart-pro", header: "TrimUI Smart Pro / 1280x720", width: 1280, height: 720, viewport: image.Rect(0, 0, 1280, 720)},
		{device: "magicx-zero-28", header: "MagicX Zero 28 / 640x480", width: 640, height: 480, viewport: image.Rect(0, 60, 640, 420)},
	}
	for _, test := range tests {
		t.Run(test.device, func(t *testing.T) {
			controls := storeinput.NewSession(t.TempDir(), test.device)
			controls.Connect(storeinput.Identity{GUID: "03000000", Name: "Runtime controller"}, true)
			frame := draw(model, test.header, controls)
			withoutCatalogue := draw(&storeui.Model{}, test.header, controls)
			if !bytes.Equal(frame.Pix, withoutCatalogue.Pix) {
				t.Fatal("catalogue content was rendered behind required setup")
			}
			output := renderOutput(frame, test.width, test.height)
			if output.Bounds() != image.Rect(0, 0, test.width, test.height) || outputRectangle(test.width, test.height) != test.viewport {
				t.Fatalf("unexpected output layout: bounds=%v viewport=%v", output.Bounds(), outputRectangle(test.width, test.height))
			}
			if directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR"); directory != "" {
				if err := os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
				if err := saveOutputScreenshot(filepath.Join(directory, test.device+"-first-run-controller-setup.png"), frame, test.width, test.height); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestRenderControllerPreviewAndBlockedState(t *testing.T) {
	model := &storeui.Model{Message: "Diagnostics saved to /userdata/system/knulli-app-store/diagnostics/report.txt"}
	blocked := storeinput.NewSession(t.TempDir(), "magicx-zero-28")
	frame := draw(model, "MagicX Zero 28 / 640x480", blocked)
	if frame.Bounds() != image.Rect(0, 0, canvasWidth, canvasHeight) {
		t.Fatalf("blocked frame failed: %v", frame.Bounds())
	}

	preview := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
	preview.Connect(storeinput.Identity{GUID: "03000000", Name: "Runtime controller"}, true)
	preview.HandleButton(storeinput.AutoMapping()[storeinput.Down])
	preview.HandleButton(storeinput.AutoMapping()[storeinput.Confirm])
	if preview.Mode != storeinput.Preview {
		t.Fatalf("test did not create preview: %s", preview.Mode)
	}
	frame = draw(&storeui.Model{}, "TrimUI Smart Pro / 1280x720", preview)
	if frame.Bounds() != image.Rect(0, 0, canvasWidth, canvasHeight) {
		t.Fatalf("preview frame failed: %v", frame.Bounds())
	}
	if directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR"); directory != "" {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		if err := saveOutputScreenshot(filepath.Join(directory, "trimui-smart-pro-controller-preview.png"), frame, 1280, 720); err != nil {
			t.Fatal(err)
		}
	}

	review := storeinput.NewSession(t.TempDir(), "magicx-zero-28")
	review.Connect(storeinput.Identity{GUID: "03000000", Name: "Runtime controller"}, true)
	review.HandleButton(storeinput.AutoMapping()[storeinput.Down])
	review.HandleButton(storeinput.AutoMapping()[storeinput.Down])
	review.HandleButton(storeinput.AutoMapping()[storeinput.Confirm])
	review.HandleButton(2)
	if review.Mode != storeinput.Review {
		t.Fatalf("test did not create assignment review: %s", review.Mode)
	}
	frame = draw(&storeui.Model{}, "MagicX Zero 28 / 640x480", review)
	output := renderOutput(frame, 640, 480)
	if output.Bounds() != image.Rect(0, 0, 640, 480) {
		t.Fatalf("assignment review frame failed: %v", output.Bounds())
	}
	if directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR"); directory != "" {
		if err := saveOutputScreenshot(filepath.Join(directory, "magicx-zero-28-controller-assignment-review.png"), frame, 640, 480); err != nil {
			t.Fatal(err)
		}
	}
}

func TestControllerSetupProgress(t *testing.T) {
	controls := storeinput.NewSession(t.TempDir(), "magicx-zero-28")
	if got := controllerSetupProgress(controls); got != "PROGRESS  0 OF 8 ACTIONS" {
		t.Fatalf("unexpected blocked progress: %q", got)
	}
	controls.Connect(storeinput.Identity{GUID: "one", Name: "Pad"}, true)
	controls.Mode = storeinput.Preview
	controls.Calibration = storeinput.NewPreview(storeinput.AutoMapping())
	controls.Calibration.Test(storeinput.AutoMapping()[storeinput.Up])
	controls.Calibration.Test(storeinput.AutoMapping()[storeinput.Confirm])
	if got := controllerSetupProgress(controls); got != "PROGRESS  2 OF 8 ACTIONS" {
		t.Fatalf("unexpected preview progress: %q", got)
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

func TestResolutionCandidateLoggingIncludesRejectionAndSelection(t *testing.T) {
	root := t.TempDir()
	logger, err := diagnostics.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	candidates := []platform.ResolutionCandidate{
		{Source: "SDL current display mode", Width: 1280, Height: 0x3333},
		{Source: "SDL renderer output", Width: 1280, Height: 720},
	}
	selected := platform.WithResolutionCandidates(platform.Info{Device: "trimui-smart-pro"}, candidates)
	logResolutionCandidates(logger, platform.AssessResolutions(candidates), selected)
	data, err := os.ReadFile(filepath.Join(root, "userdata/system/logs/knulli-app-store.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, wanted := range []string{`source="SDL current display mode"`, `height="13107"`, `valid="false"`, `reason="above maximum 7680x4320"`, `event=resolution_selected`, `resolution="1280x720"`, `source="SDL renderer output"`, `resolution_source=\"SDL renderer output\"`} {
		if !strings.Contains(string(data), wanted) {
			t.Fatalf("resolution log lacks %q: %s", wanted, data)
		}
	}
}

func TestControllerSetupLogsTransitionsWithoutRawButtonSpam(t *testing.T) {
	root := t.TempDir()
	logger, err := diagnostics.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	controls := storeinput.NewSession(root, "magicx-zero-28")
	controls.Connect(storeinput.Identity{GUID: "03000000", Name: "Runtime controller"}, true)
	model := &storeui.Model{}
	processButton(context.Background(), model, controls, storeinput.AutoMapping()[storeinput.Down], logger)
	processButton(context.Background(), model, controls, storeinput.AutoMapping()[storeinput.Confirm], logger)
	data, err := os.ReadFile(filepath.Join(root, "userdata/system/logs/knulli-app-store.log"))
	if err != nil {
		t.Fatal(err)
	}
	logText := string(data)
	if !strings.Contains(logText, `event=controller_screen_transition from="setup" to="preview"`) {
		t.Fatalf("setup transition was not logged: %s", logText)
	}
	if strings.Contains(logText, "controller_input") || strings.Contains(logText, "button=") {
		t.Fatalf("raw button spam remains in setup log: %s", logText)
	}
}

func TestSwappedConfirmBackMappingControlsCatalogueAndConfirmation(t *testing.T) {
	for _, device := range []string{"trimui-smart-pro", "magicx-zero-28"} {
		t.Run(device, func(t *testing.T) {
			root := t.TempDir()
			logger, err := diagnostics.Open(root)
			if err != nil {
				t.Fatal(err)
			}
			controls := storeinput.NewSession(root, device)
			controls.Connected = true
			controls.Mode = storeinput.Normal
			controls.Mapping = storeinput.AutoMapping()
			controls.Mapping[storeinput.Confirm], controls.Mapping[storeinput.Back] = controls.Mapping[storeinput.Back], controls.Mapping[storeinput.Confirm]
			model := &storeui.Model{Items: []appstore.Item{{
				Package:    manifest.Package{ID: "io.github.unitreign.playtime", Name: "PlayTime", Review: manifest.Review{Status: "experimental"}, Install: &manifest.Install{Warning: "Unverified experimental package."}},
				Compatible: true, Actions: []appstore.Action{appstore.Install},
			}}}

			processButton(context.Background(), model, controls, controls.Mapping[storeinput.Confirm], logger)
			if model.Focus != storeui.Actions {
				t.Fatalf("swapped Confirm did not open package actions: %v", model.Focus)
			}
			processButton(context.Background(), model, controls, controls.Mapping[storeinput.Confirm], logger)
			if model.Focus != storeui.Confirm {
				t.Fatalf("swapped Confirm did not open package confirmation: %v", model.Focus)
			}
			help := confirmationHelp(controls)
			if !strings.Contains(help, "CONFIRM (SDL B)") || !strings.Contains(help, "BACK (SDL A)") {
				t.Fatalf("confirmation did not show active physical labels: %q", help)
			}
			width, height := targetSize(device)
			frame := renderOutput(draw(model, platformHeader(device), controls), width, height)
			if frame.Bounds().Dx() == 0 {
				t.Fatal("confirmation failed to render")
			}
			if directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR"); directory != "" {
				if err := os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
				if err := saveOutputScreenshot(filepath.Join(directory, device+"-playtime-swapped-controls.png"), draw(model, platformHeader(device), controls), width, height); err != nil {
					t.Fatal(err)
				}
			}
			processButton(context.Background(), model, controls, controls.Mapping[storeinput.Back], logger)
			if model.Focus != storeui.Actions {
				t.Fatalf("swapped Back did not cancel package confirmation: %v", model.Focus)
			}
		})
	}
}

func platformHeader(device string) string {
	if device == "magicx-zero-28" {
		return "MagicX Zero 28 / 640x480"
	}
	return "TrimUI Smart Pro / 1280x720"
}

func targetSize(device string) (int, int) {
	if device == "magicx-zero-28" {
		return 640, 480
	}
	return 1280, 720
}
