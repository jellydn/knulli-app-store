package sdlui

// Tests for the device-agnostic event router. This file has no build tag, so
// it runs everywhere — the router is pure Go over the UI model and input
// session; only the adapter translation needs SDL.

import (
	"context"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	"github.com/jellydn/knulli-app-store/internal/platform"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

type routerBackend struct {
	items   []appstore.Item
	exports int
}

func (router *routerBackend) Items(context.Context) ([]appstore.Item, error) {
	return router.items, nil
}

func (router *routerBackend) Execute(context.Context, string, appstore.Action, func(string)) error {
	return nil
}

func (router *routerBackend) ExportDiagnostics(context.Context) (string, error) {
	router.exports++
	return "", nil
}

func (router *routerBackend) SetPlatform(platform.Info) {}

func newRouterHarness(t *testing.T) (*storeui.Model, *storeinput.Session, *diagnostics.Log, *routerBackend) {
	t.Helper()
	logger, err := diagnostics.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	backend := &routerBackend{items: []appstore.Item{}}
	model := storeui.New(backend)
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	controls := storeinput.NewSession(t.TempDir(), "test-device")
	controls.Connect(storeinput.Identity{Name: "Test Pad"}, true)
	// Connect without a saved mapping opens the setup wizard; route through
	// the normal mode so the semantic actions are exercised.
	controls.Mode = storeinput.Normal
	return model, controls, logger, backend
}

func TestHandleQuitExits(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventQuit})
	if !effects.Exit {
		t.Fatal("quit event did not exit")
	}
}

func TestHandleButtonReportsSemanticAction(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: int(storeinput.AutoMapping()[storeinput.Down])})
	if effects.Action != storeinput.Down {
		t.Fatalf("down button reported %q, want %q", effects.Action, storeinput.Down)
	}
	if effects.Exit {
		t.Fatal("navigation must not exit")
	}
}

// Quitting is the held-Select chord: no single button ends a session, and the
// trigger keeps its own action when the anchor is not held.
func TestHandleQuitChordExitsOnlyWhileTheAnchorIsHeld(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	trigger := int(storeinput.AutoMapping()[storeinput.Diagnostics])

	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: trigger})
	if effects.Exit {
		t.Fatal("the trigger quit without the anchor held")
	}

	Handle(context.Background(), model, controls, logger, Event{Kind: EventChordAnchor, Held: true})
	effects = Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: trigger})
	if !effects.Exit {
		t.Fatal("Select+Y did not exit")
	}
}

func TestHandleReleasedAnchorHandsTheTriggerBack(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	trigger := int(storeinput.AutoMapping()[storeinput.Diagnostics])
	Handle(context.Background(), model, controls, logger, Event{Kind: EventChordAnchor, Held: true})
	Handle(context.Background(), model, controls, logger, Event{Kind: EventChordAnchor, Held: false})
	if effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: trigger}); effects.Exit {
		t.Fatal("a released anchor still quit")
	}
}

// Key repeat drives navigation only: a held Confirm must not confirm twice and
// a held Back must not unwind several screens.
func TestHandleIgnoresRepeatsForEverythingButNavigation(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	if effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyDown, Repeat: true}); effects.Action != storeinput.Down {
		t.Fatalf("repeated navigation reported %q, want %q", effects.Action, storeinput.Down)
	}
	model.Focus = storeui.Actions
	if effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyConfirm, Repeat: true}); effects.Action != "" || model.Focus != storeui.Actions {
		t.Fatalf("repeated confirm acted: action=%q focus=%v", effects.Action, model.Focus)
	}
	if effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyBack, Repeat: true}); effects.Action != "" || effects.Exit {
		t.Fatalf("repeated back acted: action=%q exit=%v", effects.Action, effects.Exit)
	}
}

func TestHandleBackReturnsThroughFocusWithoutLeavingTheApp(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	model.Focus = storeui.Actions
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: int(storeinput.AutoMapping()[storeinput.Back])})
	if effects.Exit {
		t.Fatal("back from the actions focus must return to browse, not exit")
	}
	effects = Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: int(storeinput.AutoMapping()[storeinput.Back])})
	if effects.Exit {
		t.Fatal("back from browse must not exit: quitting is the chord")
	}
}

func TestHandleControllerRemovedDisconnects(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	Handle(context.Background(), model, controls, logger, Event{Kind: EventControllerRemoved})
	if controls.Connected {
		t.Fatal("controller removal did not disconnect the session")
	}
}

func TestHandleControllerConnected(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	controls.Disconnect()
	identity := storeinput.Identity{Name: "Hotplug Pad", GUID: "guid"}
	Handle(context.Background(), model, controls, logger, Event{Kind: EventControllerConnected, Identity: identity, KnulliMapping: true})
	if !controls.Connected || controls.Identity.Name != identity.Name || controls.Mode != storeinput.Setup || controls.Source != "Knulli SDL_GAMECONTROLLERCONFIG" {
		t.Fatalf("controller connection was not routed: %#v", controls)
	}
}

func TestHandleKeyUsesFirstRunMapping(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	controls.Mode = storeinput.Setup
	controls.FirstRun = true
	controls.Mapping[storeinput.Down] = 99
	Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyDown})
	if controls.SetupIndex != 1 {
		t.Fatalf("first-run key did not use the portable auto mapping: index %d", controls.SetupIndex)
	}
}

func TestHandleKeyUsesNonDefaultActiveMapping(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	controls.Mapping[storeinput.Up], controls.Mapping[storeinput.Down] = controls.Mapping[storeinput.Down], controls.Mapping[storeinput.Up]
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyDown})
	if effects.Action != storeinput.Down {
		t.Fatalf("key used the default binding instead of the active mapping: action=%q", effects.Action)
	}
}

// A list that outgrows its window is crossed a screenful at a time, and the
// page stops at the end rather than flying back to the top.
func TestHandlePagesTheCatalogueByKey(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	model.Items = make([]appstore.Item, listRows*2+1)
	Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyPageDown})
	if model.Selected != listRows {
		t.Fatalf("page down selected %d, want %d", model.Selected, listRows)
	}
	if start, end := model.Window(listRows); model.Selected < start || model.Selected >= end {
		t.Fatalf("paging left the selection outside the window %d-%d", start, end)
	}
	Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyPageDown})
	Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyPageDown})
	if model.Selected != len(model.Items)-1 {
		t.Fatalf("paging past the end selected %d, want the last row", model.Selected)
	}
	Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyPageUp})
	if want := len(model.Items) - 1 - listRows; model.Selected != want {
		t.Fatalf("page up selected %d, want %d", model.Selected, want)
	}
	// A held page key is navigation like any other, so it keeps paging.
	Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyPageDown, Repeat: true})
	if model.Selected != len(model.Items)-1 {
		t.Fatalf("a repeated page key selected %d, want the last row", model.Selected)
	}
}

// The one-shot keys stay one-shot: only navigation answers a repeat, so a held
// Confirm cannot confirm twice however long the key is held down.
func TestHandleKeepsOneShotKeysOneShot(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	model.Items = make([]appstore.Item, listRows+1)
	Handle(context.Background(), model, controls, logger, Event{Kind: EventKey, Key: KeyConfirm, Repeat: true})
	if model.Focus != storeui.Browse {
		t.Fatalf("a repeated Confirm opened %v", model.Focus)
	}
}

func TestHandleModeDependentButtons(t *testing.T) {
	for _, test := range []struct {
		name       string
		prepare    func(*storeinput.Session)
		button     int
		wantMode   storeinput.Mode
		wantExport bool
	}{
		{name: "blocked diagnostics", prepare: func(controls *storeinput.Session) { controls.Disconnect() }, button: storeinput.AutoMapping()[storeinput.Confirm], wantMode: storeinput.Blocked, wantExport: true},
		{name: "calibration assignment", prepare: func(controls *storeinput.Session) {
			controls.Mode = storeinput.Calibrating
			controls.Calibration = storeinput.NewCalibration()
		}, button: 42, wantMode: storeinput.Review},
		{name: "settings diagnostics", prepare: func(controls *storeinput.Session) { controls.Mode = storeinput.Settings; controls.SettingsIndex = 1 }, button: storeinput.AutoMapping()[storeinput.Confirm], wantMode: storeinput.Settings, wantExport: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			model, controls, logger, backend := newRouterHarness(t)
			test.prepare(controls)
			effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: test.button})
			if controls.Mode != test.wantMode || effects.ExportedDiagnostics != test.wantExport || (backend.exports > 0) != test.wantExport {
				t.Fatalf("mode=%q exported=%v calls=%d", controls.Mode, effects.ExportedDiagnostics, backend.exports)
			}
		})
	}
}
