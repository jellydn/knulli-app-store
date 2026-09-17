//go:build sdl

package sdlui

import (
	"bytes"
	"context"
	"fmt"
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
	issue := appstore.Item{
		Package:   manifest.Package{ID: "org.example.demo", Name: "Demo Utility", Type: "utility", Summary: "A safe package used to verify action and error layouts.", Review: manifest.Review{Status: "installable"}},
		Installed: true, InstalledVersion: "1.0.0", Healthy: false, Compatible: true,
		HealthReason: "content changed: /userdata/roms/tools/Grout/grout (expected 39b5ba053913620aea2db051c2fad2fa0bf05b59c2cc88bcb01734dd048882e7, got 31da1b650f285f47c27e1c94567c50e2b25a59893bde831aa175d722de269adb)", Compatibility: "Compatible with detected platform", Actions: []appstore.Action{appstore.Repair, appstore.Uninstall},
		Verdict: appstore.Verdict{State: appstore.StateIssue, Reasons: []appstore.Reason{{Kind: appstore.ReasonHealth, Detail: "content changed"}}},
	}
	verified := appstore.Item{
		Package: manifest.Package{
			ID: "app.romm.grout", Name: "Grout", Type: "integration", Summary: "Connects a Linux retro handheld to a RomM server.",
			Version: "5.1.0.0", Review: manifest.Review{Status: "verified", Approval: &manifest.Approval{Provenance: "community"}},
			Install: &manifest.Install{Warning: "Do not use Grout updater. Update only through Knulli App Store."},
		},
		Installed: true, InstalledVersion: "5.1.0.0", Healthy: true, DeviceTested: true, Compatible: true, Compatibility: "Compatible with detected platform", Actions: []appstore.Action{appstore.Uninstall, appstore.Repair}, Verdict: appstore.Verdict{State: appstore.StateInstalled},
	}
	experimental := appstore.Item{
		Package: manifest.Package{
			ID: "io.github.unitreign.playtime", Name: "PlayTime", Version: "1.0.0", Type: "utility", Summary: "Tracks game play time on Knulli.",
			Review: manifest.Review{Status: "experimental", Approval: &manifest.Approval{Provenance: "community"}}, Install: &manifest.Install{Warning: "Unverified experimental test. Existing files are backed up."},
		},
		DeviceTested: true, Compatible: true, Compatibility: "Experimental compatibility on this device; no minimum version is claimed", Actions: []appstore.Action{appstore.Install}, Verdict: appstore.Verdict{State: appstore.StateAvailable, Reasons: []appstore.Reason{{Kind: appstore.ReasonReview, Detail: "Experimental compatibility on this device; no minimum version is claimed"}}},
	}
	external := experimental
	external.PreExisting = true
	external.Actions = []appstore.Action{appstore.Adopt}
	external.Verdict.State = appstore.StateExternal
	recovery := external
	recovery.RecoveryReason = "adoption backup already exists for /userdata/roms/tools/PlayTime/playtime"
	recovery.RecoverySummary = "Replaces reviewed app files; preserves declared data; backs up the complete existing destination for manual restore."
	recovery.Actions = []appstore.Action{appstore.Adopt, appstore.ForceReinstall}
	incompatible := experimental
	incompatible.Compatible = false
	incompatible.Actions = nil
	incompatible.Compatibility = `compatibility failed field=firmware: detected device="trimui-smart-pro" architecture="aarch64" resolution="1280x720" firmware_raw="buildroot" firmware="" firmware_source="/etc/os-release:ID"; package requires firmware="knulli"`
	incompatible.Verdict = appstore.Verdict{State: appstore.StateIncompatible, Reasons: []appstore.Reason{{Kind: appstore.ReasonPlatform, Detail: incompatible.Compatibility}}}
	candidate := appstore.Item{
		Package:       manifest.Package{ID: "io.github.example.candidate", Name: "Candidate Tool", Type: "utility", Summary: "Metadata is still under review.", Review: manifest.Review{Status: "candidate"}},
		Compatibility: "Candidate: compatibility is not approved",
		Verdict:       appstore.Verdict{State: appstore.StateCandidate, Reasons: []appstore.Reason{{Kind: appstore.ReasonReview, Detail: "Candidate: compatibility is not approved"}}},
	}
	states := map[string]*storeui.Model{
		"catalogue":           {Items: []appstore.Item{verified, experimental, candidate}},
		"details":             {Items: []appstore.Item{experimental}, Focus: storeui.Actions},
		"installed-healthy":   {Items: []appstore.Item{verified}},
		"verified":            {Items: []appstore.Item{verified}, Focus: storeui.Actions},
		"experimental":        {Items: []appstore.Item{experimental}, Focus: storeui.Actions},
		"candidate":           {Items: []appstore.Item{candidate}, Focus: storeui.Actions},
		"external":            {Items: []appstore.Item{external}, Focus: storeui.Actions},
		"external-confirm":    {Items: []appstore.Item{external}, Focus: storeui.Confirm},
		"recovery":            {Items: []appstore.Item{recovery}, Focus: storeui.Actions, Error: recovery.RecoveryReason},
		"force-review":        {Items: []appstore.Item{recovery}, Focus: storeui.Confirm, Action: 1},
		"force-confirm":       {Items: []appstore.Item{recovery}, Focus: storeui.ForceConfirm, Action: 1},
		"issue":               {Items: []appstore.Item{issue}},
		"issue-details":       {Items: []appstore.Item{issue}, Focus: storeui.Health},
		"compatibility-error": {Items: []appstore.Item{incompatible}, Focus: storeui.Actions},
		"confirm":             {Items: []appstore.Item{experimental}, Focus: storeui.Confirm},
		"verified-confirm":    {Items: []appstore.Item{{Package: verified.Package, Compatible: true, Actions: []appstore.Action{appstore.Install}}}, Focus: storeui.Confirm},
		"diagnostics":         {Items: []appstore.Item{experimental}, Message: "Diagnostics saved to /userdata/system/knulli-app-store/diagnostics/knulli-app-store-diagnostics-20260915T073500Z.txt"},
		"error":               {Items: []appstore.Item{issue}, Error: "SHA-256 mismatch; package files were not changed"},
		"progress":            {Items: []appstore.Item{experimental}, Busy: true, Message: "Downloading, verifying, and applying Grout"},
	}
	directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR")
	for _, device := range []string{"trimui-smart-pro", "magicx-zero-28"} {
		controls := storeinput.NewSession(t.TempDir(), device)
		controls.Connected = true
		controls.Source = "Knulli SDL_GAMECONTROLLERCONFIG"
		controls.Mode = storeinput.Normal
		controls.Message = ""
		width, height := targetSize(device)
		for name, model := range states {
			for index := range model.Items {
				if model.Items[index].Package.ID == "io.github.unitreign.playtime" {
					model.Items[index].DeviceTested = device == "trimui-smart-pro"
				}
			}
			frame := draw(model, platformHeader(device), controls)
			output := renderOutput(frame, width, height)
			if frame.Bounds() != image.Rect(0, 0, canvasWidth, canvasHeight) || output.Bounds() != image.Rect(0, 0, width, height) {
				t.Fatalf("%s/%s has unexpected bounds: frame=%v output=%v", device, name, frame.Bounds(), output.Bounds())
			}
			if directory != "" {
				if err := os.MkdirAll(directory, 0755); err != nil {
					t.Fatal(err)
				}
				if err := saveOutputScreenshot(filepath.Join(directory, device+"-gui-"+name+".png"), frame, width, height); err != nil {
					t.Fatal(err)
				}
			}
		}
		controls.Mode = storeinput.Settings
		settings := draw(&storeui.Model{}, platformHeader(device), controls)
		if output := renderOutput(settings, width, height); output.Bounds() != image.Rect(0, 0, width, height) {
			t.Fatalf("%s settings has unexpected bounds %v", device, output.Bounds())
		}
		if directory != "" {
			if err := saveOutputScreenshot(filepath.Join(directory, device+"-gui-settings.png"), settings, width, height); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// catalogueItem is a verified package with two safe actions, used to prove the
// action row stays visible under every kind of status block.
func catalogueItem() appstore.Item {
	return appstore.Item{
		Package:    manifest.Package{ID: "app.romm.grout", Name: "Grout", Version: "5.2.0.0", Type: "integration", Review: manifest.Review{Status: "verified"}},
		Compatible: true, Compatibility: "Compatible with detected platform",
		Actions: []appstore.Action{appstore.Install, appstore.Repair},
	}
}

func TestFooterCharacterLimitTracksTheCanvasWidth(t *testing.T) {
	if want := (canvasWidth - 2*panelLeft) / 7; footerCharacterLimit != want {
		t.Fatalf("footer limit %d does not match the %dpx canvas (%d)", footerCharacterLimit, canvasWidth, want)
	}
}

func TestStatusBlockSitsAboveTheActionRow(t *testing.T) {
	labelTop := actionLabelBaseline - 11
	if statusBoxBottom >= labelTop {
		t.Fatalf("status block (bottom %d) reaches the action row (label top %d)", statusBoxBottom, labelTop)
	}
	if statusBoxBottom-statusBaseline-(statusLines-1)*15 < 0 {
		t.Fatal("status block is shorter than the lines it must show")
	}
}

func TestStatusBlockNeverCoversTheActionRow(t *testing.T) {
	recovery := catalogueItem()
	recovery.RecoveryReason = "adoption backup already exists for /userdata/roms/tools/Grout/grout"
	recovery.RecoverySummary = "Replaces reviewed app files; preserves declared data."
	recovery.Actions = []appstore.Action{appstore.Adopt, appstore.ForceReinstall}
	states := map[string]*storeui.Model{
		"error":    {Items: []appstore.Item{catalogueItem()}, Error: "Install failed: checksum mismatch on /userdata/roms/tools/Grout/grout"},
		"message":  {Items: []appstore.Item{catalogueItem()}, Message: "Diagnostics saved to /userdata/system/knulli-app-store/diagnostics/knulli-app-store-diagnostics-20260915T073500Z.txt"},
		"recovery": {Items: []appstore.Item{recovery}, Focus: storeui.Actions, Error: recovery.RecoveryReason},
	}
	for name, model := range states {
		t.Run(name, func(t *testing.T) {
			frame := draw(model, platformHeader("trimui-smart-pro"), nil)
			painted := 0
			for y := actionRowBaseline; y < actionRowBaseline+22; y++ {
				for x := panelInset; x < panelRight; x++ {
					if frame.RGBAAt(x, y) == palette.selected {
						painted++
					}
				}
			}
			if painted == 0 {
				t.Fatal("the status block covered the action buttons")
			}
		})
	}
}

func TestFooterIsTheOnlyHintOnEveryScreen(t *testing.T) {
	item := appstore.Item{Compatible: true, Actions: []appstore.Action{appstore.Install}}
	normal := func() *storeinput.Session {
		controls := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
		controls.Connected = true
		controls.FirstRun = false
		controls.Mode = storeinput.Normal
		return controls
	}
	mode := func(value storeinput.Mode, prepare func(*storeinput.Session)) *storeinput.Session {
		controls := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
		controls.Connected = true
		controls.FirstRun = value == storeinput.Setup
		controls.Mode = value
		if prepare != nil {
			prepare(controls)
		}
		return controls
	}
	type screen struct {
		name     string
		model    *storeui.Model
		controls *storeinput.Session
	}
	screens := []screen{
		{name: "catalogue", model: &storeui.Model{Items: []appstore.Item{item}, Focus: storeui.Browse}, controls: normal()},
		{name: "health", model: &storeui.Model{Items: []appstore.Item{item}, Focus: storeui.Health}, controls: normal()},
		{name: "actions", model: &storeui.Model{Items: []appstore.Item{item}, Focus: storeui.Actions}, controls: normal()},
		{name: "confirmation", model: &storeui.Model{Items: []appstore.Item{item}, Focus: storeui.Confirm}, controls: normal()},
		{name: "force-confirmation", model: &storeui.Model{Items: []appstore.Item{item}, Focus: storeui.ForceConfirm}, controls: normal()},
		{name: "blocked", model: &storeui.Model{}, controls: mode(storeinput.Blocked, nil)},
		{name: "settings", model: &storeui.Model{}, controls: mode(storeinput.Settings, nil)},
		{name: "review", model: &storeui.Model{}, controls: mode(storeinput.Review, func(controls *storeinput.Session) { controls.Calibration = storeinput.NewCalibration() })},
		{name: "calibrating", model: &storeui.Model{}, controls: mode(storeinput.Calibrating, func(controls *storeinput.Session) { controls.Calibration = storeinput.NewCalibration() })},
		{name: "preview", model: &storeui.Model{}, controls: mode(storeinput.Preview, func(controls *storeinput.Session) {
			controls.Calibration = storeinput.NewPreview(storeinput.AutoMapping())
		})},
	}
	for _, current := range screens {
		t.Run(current.name, func(t *testing.T) {
			frame := draw(current.model, platformHeader("trimui-smart-pro"), current.controls)
			// The gap between the panel and the footer must stay empty, so no
			// screen can squeeze a second hint next to the shared footer line.
			for y := panelBottom; y < footerBaseline-11; y++ {
				for x := 0; x < canvasWidth; x++ {
					if got := frame.RGBAAt(x, y); got != palette.background {
						t.Fatalf("pixel (%d,%d) is %#v, want background between panel and footer", x, y, got)
					}
				}
			}
			painted := false
			for y := footerBaseline - 11; y <= footerBaseline && !painted; y++ {
				for x := 0; x < canvasWidth; x++ {
					if frame.RGBAAt(x, y) == palette.muted {
						painted = true
						break
					}
				}
			}
			if !painted {
				t.Fatal("the shared footer hint was not painted below the panel")
			}
		})
	}
}

func TestHeaderIncludesBrandMark(t *testing.T) {
	frame := draw(&storeui.Model{}, platformHeader("trimui-smart-pro"), nil)
	if frame.RGBAAt(19, 12) != palette.accent {
		t.Fatalf("filled brand tile missing accent: got %#v", frame.RGBAAt(19, 12))
	}
	if frame.RGBAAt(25, 12) != palette.accent {
		t.Fatalf("outlined brand tile missing accent border: got %#v", frame.RGBAAt(25, 12))
	}
	if frame.RGBAAt(27, 12) != palette.background {
		t.Fatalf("outlined brand tile missing inner background: got %#v", frame.RGBAAt(27, 12))
	}
}

func TestCompactLabelsAndRequiredNotices(t *testing.T) {
	verified := manifest.Package{Review: manifest.Review{Status: "verified"}, Install: &manifest.Install{Warning: "Do not use the updater."}}
	experimental := manifest.Package{Review: manifest.Review{Status: "experimental"}, Install: &manifest.Install{Warning: "Unverified test."}}
	if trustLabel(appstore.Item{Package: verified}) != "VERIFIED" || trustLabel(appstore.Item{Package: experimental}) != "EXPERIMENTAL" || trustLabel(appstore.Item{Package: experimental, DeviceTested: true}) != "DEVICE TESTED" {
		t.Fatal("trust labels do not separate verified and experimental packages")
	}
	if got := installState(appstore.Item{Package: verified, Verdict: appstore.Verdict{State: appstore.StateInstalled}}); got != "INSTALLED" {
		t.Fatalf("healthy install label = %q", got)
	}
	if got := installState(appstore.Item{Package: verified, Verdict: appstore.Verdict{State: appstore.StateIssue}}); got != "ISSUE" {
		t.Fatalf("unhealthy install label = %q", got)
	}
	if got := installState(appstore.Item{Package: experimental, Verdict: appstore.Verdict{State: appstore.StateExternal}}); got != "EXTERNAL" || actionLabel(appstore.Adopt) != "Manage existing" {
		t.Fatalf("external install labels are not concise: state=%q action=%q", got, actionLabel(appstore.Adopt))
	}
	for _, pkg := range []manifest.Package{verified, experimental} {
		if pkg.Install.Warning == "" || len(wrapText(strings.ToUpper(pkg.Install.Warning), 40)) > 2 {
			t.Fatalf("required package notice is not concise and visible: %#v", pkg.Install)
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
	if directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR"); directory != "" {
		if err := os.MkdirAll(directory, 0755); err != nil {
			t.Fatal(err)
		}
		if err := saveOutputScreenshot(filepath.Join(directory, "magicx-zero-28-controller-blocked.png"), frame, 640, 480); err != nil {
			t.Fatal(err)
		}
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
	if got := controllerSetupProgress(controls); got != fmt.Sprintf("PROGRESS  0 OF %d ACTIONS", len(storeinput.Actions)) {
		t.Fatalf("unexpected blocked progress: %q", got)
	}
	controls.Connect(storeinput.Identity{GUID: "one", Name: "Pad"}, true)
	controls.Mode = storeinput.Preview
	controls.Calibration = storeinput.NewPreview(storeinput.AutoMapping())
	controls.Calibration.Test(storeinput.AutoMapping()[storeinput.Up])
	controls.Calibration.Test(storeinput.AutoMapping()[storeinput.Confirm])
	if got := controllerSetupProgress(controls); got != fmt.Sprintf("PROGRESS  2 OF %d ACTIONS", len(storeinput.Actions)) {
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
	selected := platform.Info{Device: "trimui-smart-pro"}.WithCandidates(candidates)
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
			help := footerText(model, controls)
			if !strings.Contains(help, "Confirm (B)") || !strings.Contains(help, "Back (A)") {
				t.Fatalf("footer did not show active physical labels: %q", help)
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

// countMutedInHeading counts the muted pixels in the heading line to the right
// of the word CATALOGUE, which is where the list reports its range.
func countMutedInHeading(frame *image.RGBA) int {
	muted := 0
	for column := 100; column < listRight; column++ {
		for baseline := 62 - glyphHeight; baseline <= 62; baseline++ {
			if frame.RGBAAt(column, baseline) == palette.muted {
				muted++
			}
		}
	}
	return muted
}

// A catalogue longer than the window shows one window of rows, marks the
// selected row with a solid accent ring around an accent wash, and reports where
// the window sits in the list.
func TestLongCatalogueShowsOneWindowOfRows(t *testing.T) {
	model := &storeui.Model{}
	for index := 0; index < listRows+3; index++ {
		model.Items = append(model.Items, catalogueItem())
	}
	model.Selected = len(model.Items) - 1
	controls := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
	controls.Connected = true
	controls.Mode = storeinput.Normal
	frame := draw(model, platformHeader("trimui-smart-pro"), controls)
	start, end := model.Window(listRows)
	row := listRowRectangle(model.Selected - start)
	for column := row.Min.X; column < row.Max.X; column++ {
		if frame.RGBAAt(column, row.Min.Y) != palette.accent || frame.RGBAAt(column, row.Max.Y-1) != palette.accent {
			t.Fatal("the selected row has no accent ring")
		}
	}
	wash := frame.RGBAAt(row.Max.X-4, row.Min.Y+2)
	if wash != blend(palette.accent, palette.panel, selectionWash) {
		t.Fatalf("the selected row is not washed with the accent: %v", wash)
	}
	if wash == palette.selected {
		t.Fatal("the selected row still reads as a filled button")
	}
	// Nothing is painted below the window, so the list never claims rows it does
	// not show. The panel background is what is left there.
	for y := listBottom; y < panelBottom; y++ {
		if frame.RGBAAt(listLeft, y) != palette.panel {
			t.Fatalf("the list painted row content at y=%d, outside its window", y)
		}
	}
	if muted := countMutedInHeading(frame); muted == 0 {
		t.Fatalf("the overflowing list did not report its range %d-%d of %d", start+1, end, len(model.Items))
	}
}

// A list that fits is fully shown, so a range would only repeat the count and
// the heading stays bare.
func TestShortCatalogueReportsNoRange(t *testing.T) {
	model := &storeui.Model{Items: []appstore.Item{catalogueItem()}}
	frame := draw(model, platformHeader("trimui-smart-pro"), nil)
	if muted := countMutedInHeading(frame); muted != 0 {
		t.Fatalf("a catalogue that fits reported a range (%d pixels)", muted)
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
