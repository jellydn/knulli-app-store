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

func TestHandleExitActionQuits(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: int(storeinput.AutoMapping()[storeinput.Exit])})
	if !effects.Exit {
		t.Fatal("exit action did not exit")
	}
}

func TestHandleBackWalksFocusBeforeExiting(t *testing.T) {
	model, controls, logger, _ := newRouterHarness(t)
	model.Focus = storeui.Actions
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: int(storeinput.AutoMapping()[storeinput.Back])})
	if effects.Exit {
		t.Fatal("back from the actions focus must return to browse, not exit")
	}
	effects = Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: int(storeinput.AutoMapping()[storeinput.Back])})
	if !effects.Exit {
		t.Fatal("back from browse must exit")
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
