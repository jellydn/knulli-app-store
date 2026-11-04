//go:build sdl

package sdlui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
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
	"golang.org/x/image/math/fixed"
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
	filtered := &storeui.Model{Items: []appstore.Item{installed}}
	filtered.Tabs.CycleCategory(2)
	filtered.Tabs.Cycle(2)
	states["filtered"] = filtered
	emptyFiltered := storeui.New(&routerBackend{items: []appstore.Item{item}})
	if err := emptyFiltered.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	emptyFiltered.Horizontal(1)
	emptyFiltered.CycleTab(2)
	states["empty-filtered"] = emptyFiltered
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
	}{
		{"320x240", 320, 240}, {"320x200", 320, 200}, {"480x320", 480, 320},
		{"639x480", 639, 480}, {"640x480", 640, 480}, {"641x480", 641, 480},
		{"640x479", 640, 479}, {"640x481", 640, 481},
		{"720x480", 720, 480}, {"720x720", 720, 720},
		{"480x640", 480, 640}, {"640x960", 640, 960}, {"320x640", 320, 640},
		{"1280x720", 1280, 720}, {"1920x1080", 1920, 1080},
		{"1920x1200", 1920, 1200}, {"2560x1080", 2560, 1080},
	} {
		w, h := canvasSize(size.width, size.height)
		canvas, err := newConsoleCanvas(w, h)
		if err != nil {
			t.Fatal(err)
		}
		controls := storeinput.NewSession(t.TempDir(), "trimui-smart-pro")
		controls.Connected, controls.Mode = true, storeinput.Normal
		for mode, session := range map[string]*storeinput.Session{"": controls, "-keyboard": storeinput.NewDesktopSession(t.TempDir(), "")} {
			for name, model := range states {
				if model.Busy && !canvas.compact && fitText(canvas.body, model.Message, canvas.width-48) != model.Message {
					t.Fatalf("%s %s: progress was truncated", size.name, name)
				}
				frame := canvas.draw(model, "KNULLI", session)
				if frame.Bounds() != image.Rect(0, 0, w, h) {
					t.Fatal(frame.Bounds())
				}
				for i, group := range canvas.groups {
					if !group.bounds.In(image.Rect(8, canvas.footerTop()+8, w-8, h-8)) {
						t.Fatalf("%s %s%s: hint outside reserved footer: %v", size.name, name, mode, group.bounds)
					}
					for _, other := range canvas.groups[:i] {
						if group.bounds.Overlaps(other.bounds) || (group.bounds.Min.Y == other.bounds.Min.Y && group.bounds.Min.X-other.bounds.Max.X < 8) {
							t.Fatalf("%s %s%s: hints overlap or lost their gap: %v / %v", size.name, name, mode, group.bounds, other.bounds)
						}
					}
				}
				// Long action/key labels must leave the footer's safe margins
				// intact, including Manage existing on the narrow keyboard view.
				margin := 8
				if !canvas.compact && model.Focus == storeui.Browse {
					margin = 16
				}
				for y := canvas.footerTop(); y < h; y++ {
					for x := 0; x < margin; x++ {
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

func TestCatalogueArtworkDoesNotRequireInstallation(t *testing.T) {
	canvas, err := newConsoleCanvas(640, 480)
	if err != nil {
		t.Fatal(err)
	}
	defer canvas.close()
	for _, test := range []struct {
		id      string
		artwork bool
	}{
		{"app.romm.grout", true},
		{"com.akouzoukos.apotris", true},
		{"io.github.jhhoward.catacombs", true},
		{"org.retroshelf.knulli", false},
	} {
		t.Run(test.id, func(t *testing.T) {
			pkg, err := manifest.Load("../../catalogue/packages/" + test.id + ".json")
			if err != nil {
				t.Fatal(err)
			}
			pkg.Install = nil
			model := &storeui.Model{Items: []appstore.Item{{Package: pkg}}}
			actual := canvas.draw(model, "", nil)
			model.Items[0].Package.ID = "unknown.package.id"
			fallback := canvas.draw(model, "", nil)
			if differs := !bytes.Equal(actual.Pix, fallback.Pix); differs != test.artwork {
				t.Fatalf("artwork differs from initials fallback=%v, want %v", differs, test.artwork)
			}
		})
	}
}

func TestConsoleStatusKeepsContextualControlsVisible(t *testing.T) {
	canvas, err := newConsoleCanvas(640, 480)
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
	canvas, err := newConsoleCanvas(640, 480)
	if err != nil {
		t.Fatal(err)
	}
	defer canvas.close()
	value := fitText(canvas.title, "星のカービィ — A very long title with asymmetric characters", 210)
	if !utf8.ValidString(value) || font.MeasureString(canvas.title, value).Ceil() > 210 {
		t.Fatalf("invalid fitted label %q", value)
	}
}

func TestHeaderShowsFilterIntersectionWithoutInputHints(t *testing.T) {
	for _, size := range [][2]int{{320, 240}, {320, 200}, {480, 320}, {639, 480}, {640, 480}, {641, 480}, {640, 479}, {640, 481}, {720, 480}, {720, 720}, {480, 640}, {640, 960}, {320, 640}, {1280, 720}, {1920, 1080}, {1920, 1200}, {2560, 1080}} {
		t.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(t *testing.T) {
			w, h := canvasSize(size[0], size[1])
			canvas, err := newConsoleCanvas(w, h)
			if err != nil {
				t.Fatal(err)
			}
			defer canvas.close()
			for category, categoryLabel := range []string{"ALL TYPES", "GAMES", "APPS"} {
				for status, statusLabel := range []string{"ALL STATES", "READY", "INSTALLED"} {
					// Empty views must retain both filters too, including the
					// longest real labels rather than only short Games / Ready.
					model := &storeui.Model{}
					model.Tabs.CycleCategory(category)
					model.Tabs.Cycle(status)
					pad := canvas.draw(model, "", nil)
					keyboard := canvas.draw(model, "", storeinput.NewDesktopSession("", ""))
					bottom := 96
					if canvas.compact {
						bottom = 4 + canvas.title.Metrics().Height.Ceil() + 4 + canvas.step + 2
					}
					bounds := image.Rect(0, 0, w, bottom)
					if !renderedText(keyboard, canvas, categoryLabel+" / "+statusLabel, bounds, palette.background, palette.accent) {
						t.Fatalf("current intersection missing or clipped: %s / %s", categoryLabel, statusLabel)
					}
					if !bytes.Equal(pad.Pix[:bottom*pad.Stride], keyboard.Pix[:bottom*keyboard.Stride]) {
						t.Fatal("header depends on input bindings; instructions belong in footer")
					}
				}
			}
		})
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

func TestFooterKeepsLongestActionsReadable(t *testing.T) {
	for _, size := range [][2]int{{320, 240}, {400, 480}, {480, 320}, {639, 480}, {640, 480}, {641, 480}, {720, 720}, {1280, 720}, {2560, 1080}} {
		for _, policy := range []string{"knulli", "xbox", "keyboard"} {
			t.Run(fmt.Sprintf("%dx%d/%s", size[0], size[1], policy), func(t *testing.T) {
				t.Setenv("STORE_CONTROLLER_LAYOUT", policy)
				w, h := canvasSize(size[0], size[1])
				canvas, err := newConsoleCanvas(w, h)
				if err != nil {
					t.Fatal(err)
				}
				defer canvas.close()
				controls := storeinput.NewSession("", "")
				controls.Connect(storeinput.Identity{Name: "Pad"}, true)
				if policy == "keyboard" {
					controls = storeinput.NewDesktopSession("", "")
				}
				for _, action := range []appstore.Action{appstore.Adopt, appstore.ForceReinstall, appstore.CheckUpdate} {
					model := &storeui.Model{Items: []appstore.Item{{Package: manifest.Package{Name: "Long action test"}, Actions: []appstore.Action{action}}}}
					frame := canvas.draw(model, "KNULLI", controls)
					for _, label := range []string{actionLabel(action), "Browse", "Type", "Status", "More", "Quit"} {
						if !renderedText(frame, canvas, label, image.Rect(0, canvas.footerTop(), w, h), palette.panel, palette.text) {
							t.Fatalf("complete footer label %q is missing or obscured", label)
						}
					}
					if canvas.compact {
						item := model.Items[0]
						if !renderedText(frame, canvas, trustLabel(item)+" / "+installState(item), image.Rect(0, 0, w, canvas.statusTop()), palette.background, statusColor(item)) {
							t.Fatal("footer crowded out selected-package trust/install state")
						}
					}
				}
			})
		}
	}
}

// Match independently rasterized text, so clipping or a neighbouring hint
// painting over a label fails even if layout bookkeeping says it fits.
func renderedText(frame *image.RGBA, canvas *consoleCanvas, value string, bounds image.Rectangle, background, ink color.Color) bool {
	expected := image.NewRGBA(image.Rect(0, 0, font.MeasureString(canvas.body, value).Ceil(), canvas.body.Metrics().Height.Ceil()))
	fill(expected, expected.Bounds(), background)
	drawer := font.Drawer{Dst: expected, Src: image.NewUniform(ink), Face: canvas.body, Dot: fixed.P(0, canvas.body.Metrics().Ascent.Ceil())}
	drawer.DrawString(value)
	middle := expected.Bounds().Dy() / 2
	signature := expected.Pix[middle*expected.Stride : (middle+1)*expected.Stride]
	for y := bounds.Min.Y; y+expected.Bounds().Dy() <= bounds.Max.Y; y++ {
		strip := frame.Pix[(y+middle)*frame.Stride : (y+middle+1)*frame.Stride]
		for offset := 0; offset+expected.Stride <= len(strip); {
			index := bytes.Index(strip[offset:], signature)
			if index < 0 {
				break
			}
			left := offset + index
			offset = left + 4
			matches := true
			for row := 0; row < expected.Bounds().Dy(); row++ {
				start := (y+row)*frame.Stride + left
				if !bytes.Equal(frame.Pix[start:start+expected.Stride], expected.Pix[row*expected.Stride:(row+1)*expected.Stride]) {
					matches = false
					break
				}
			}
			if matches {
				return true
			}
		}
	}
	return false
}
