//go:build sdl

package sdlui

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/catalog"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
	"golang.org/x/image/font"
)

func TestRenderRepresentativeStates(t *testing.T) {
	pkg, err := manifest.Load("../../catalogue/packages/io.github.jellydn.random-game-picker.json")
	if err != nil {
		t.Fatal(err)
	}
	item := appstore.Item{Package: pkg, Compatible: true, Actions: []appstore.Action{appstore.Install}, Verdict: appstore.Verdict{State: appstore.StateAvailable}}
	installed := item
	installed.Installed, installed.Healthy = true, true
	installed.Actions = []appstore.Action{appstore.Repair, appstore.Uninstall}
	installed.Verdict.State = appstore.StateInstalled
	issue := installed
	issue.HealthReason = "Managed file differs: random-game-picker. Repair restores the reviewed release."
	issue.Verdict.State = appstore.StateIssue
	recovery := item
	recovery.Actions = []appstore.Action{appstore.Adopt, appstore.ForceReinstall}
	recovery.RecoveryReason = "An existing backup requires manual review."
	recovery.RecoverySummary = "Replaces reviewed app files, preserves declared data and backs up the complete existing destination for manual restore."
	longTitle := item
	longTitle.Package.Name = "Asymmetric package title with a long final word — settings"
	longTitle.Package.Summary = strings.Repeat("Long package text must stay in the detail pane. ", 8)
	update := installed
	update.InstalledVersion = "0.2.1"
	update.Actions = []appstore.Action{appstore.Update, appstore.Uninstall, appstore.Repair}
	storeUpdate := installed
	storeUpdate.Package.ID, storeUpdate.Package.Name = appstore.StoreID, "Knulli App Store"
	storeUpdate.Actions = []appstore.Action{appstore.CheckUpdate}
	states := map[string]*storeui.Model{
		"catalogue":     {Items: []appstore.Item{item}},
		"details":       {Items: []appstore.Item{item}, Focus: storeui.Actions},
		"confirm":       {Items: []appstore.Item{item}, Focus: storeui.Confirm},
		"installed":     {Items: []appstore.Item{installed}},
		"repair":        {Items: []appstore.Item{installed}, Focus: storeui.Actions},
		"issue":         {Items: []appstore.Item{issue}, Focus: storeui.Health},
		"force-review":  {Items: []appstore.Item{recovery}, Focus: storeui.Confirm, Action: 1},
		"force-confirm": {Items: []appstore.Item{recovery}, Focus: storeui.ForceConfirm, Action: 1},
		"error":         {Items: []appstore.Item{item}, Error: "SHA-256 mismatch. Package files were not changed."},
		"progress":      {Items: []appstore.Item{item}, Busy: true, Message: "Downloading: 37%"},
		"manage":        {Items: []appstore.Item{recovery}},
		"update":        {Items: []appstore.Item{update}},
		"store-update":  {Items: []appstore.Item{storeUpdate}},
		"long-title":    {Items: []appstore.Item{longTitle}},
		"empty":         {},
	}
	notice := &storeui.Model{Items: []appstore.Item{installed}}
	notice.Toasts.Push("Install completed; game list refresh requested", time.Now())
	states["notice"] = notice
	for name, message := range map[string]string{
		"preparing": "Preparing installation", "verifying": "Verifying release",
		"extracting": "Extracting release", "installing": "Installing files: 66% (2/3)",
		"finalizing": "Finalizing installation",
	} {
		states[name] = &storeui.Model{Items: []appstore.Item{item}, Busy: true, Message: message}
	}
	index, err := catalog.Build("../../catalogue/packages")
	if err != nil {
		t.Fatal(err)
	}
	library := make([]appstore.Item, 0, len(index.Packages))
	for _, entry := range index.Packages {
		library = append(library, appstore.Item{Package: entry.Package})
	}
	sort.Slice(library, func(i, j int) bool { return library[i].Package.Name < library[j].Package.Name })
	for selected := range library {
		if library[selected].Package.ID == pkg.ID {
			library[selected] = item
			states["catalogue"] = &storeui.Model{Items: library, Selected: selected}
		}
	}
	states["library-last"] = &storeui.Model{Items: library, Selected: len(library) - 1}
	for _, size := range []struct {
		name          string
		width, height int
	}{{"640x480", 640, 480}, {"1280x720", 1280, 720}} {
		canvas, err := newConsoleCanvas(logicalWidth(size.width, size.height))
		if err != nil {
			t.Fatal(err)
		}
		controls := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
		controls.Connected, controls.Mode = true, storeinput.Normal
		for mode, session := range map[string]*storeinput.Session{"": controls, "-keyboard": storeinput.NewDesktopSession(t.TempDir(), "")} {
			for name, model := range states {
				if model.Busy && fitText(canvas.body, model.Message, canvas.width-48) != model.Message {
					t.Fatalf("%s %s: progress was truncated", size.name, name)
				}
				frame := canvas.draw(model, "KNULLI", session)
				if frame.Bounds() != image.Rect(0, 0, logicalWidth(size.width, size.height), 480) {
					t.Fatal(frame.Bounds())
				}
				// Long action/key labels must leave the footer's safe margins
				// intact, including Manage existing on the narrow keyboard view.
				for y := 427; y < 480; y++ {
					for x := 0; x < 16; x++ {
						if frame.RGBAAt(x, y) != palette.panel || frame.RGBAAt(canvas.width-1-x, y) != palette.panel {
							t.Fatalf("%s %s%s: footer escaped safe margins", size.name, name, mode)
						}
					}
				}
				if directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR"); directory != "" {
					if err := os.MkdirAll(directory, 0755); err != nil {
						t.Fatal(err)
					}
					if err := saveOutputScreenshot(filepath.Join(directory, size.name+"-"+name+mode+".png"), frame, size.width, size.height); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		canvas.close()
	}
}

func TestConsoleStatusKeepsContextualControlsVisible(t *testing.T) {
	canvas, err := newConsoleCanvas(640)
	if err != nil {
		t.Fatal(err)
	}
	defer canvas.close()
	item := appstore.Item{Package: manifest.Package{Name: "Package"}, Actions: []appstore.Action{appstore.Install}}
	model := &storeui.Model{Items: []appstore.Item{item}}
	baseline := canvas.draw(model, "", nil)
	model.Error = "Checksum mismatch; nothing was changed"
	failed := canvas.draw(model, "", nil)
	for y := 427; y < 480; y++ {
		if !bytes.Equal(baseline.Pix[y*baseline.Stride:(y+1)*baseline.Stride], failed.Pix[y*failed.Stride:(y+1)*failed.Stride]) {
			t.Fatal("error hid contextual controls")
		}
	}
	model.Error = ""
	model.Toasts.Push("Expired notice", time.Now().Add(-storeui.ToastTTL))
	if !bytes.Equal(baseline.Pix, canvas.draw(model, "", nil).Pix) {
		t.Fatal("expired notice still shown")
	}
}

func TestConsoleLabelsFitWithoutBreakingUTF8(t *testing.T) {
	canvas, err := newConsoleCanvas(640)
	if err != nil {
		t.Fatal(err)
	}
	defer canvas.close()
	value := fitText(canvas.title, "星のカービィ — A very long title with asymmetric characters", 210)
	if !utf8.ValidString(value) || font.MeasureString(canvas.title, value).Ceil() > 210 {
		t.Fatalf("invalid fitted label %q", value)
	}
}

func TestTrustLabelsRequireExactDeviceEvidence(t *testing.T) {
	pkg := manifest.Package{Review: manifest.Review{Status: "verified"}, Install: &manifest.Install{}}
	if trustLabel(appstore.Item{Package: pkg}) != "UNTESTED" {
		t.Fatal("unverified hardware received tested badge")
	}
	if trustLabel(appstore.Item{Package: pkg, DeviceTested: true}) != "VERIFIED" {
		t.Fatal("exact device evidence missing")
	}
	if actionLabel(appstore.Adopt) != "Manage existing" {
		t.Fatal("adoption vocabulary changed")
	}
}
