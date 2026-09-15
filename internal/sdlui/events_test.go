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
	items []appstore.Item
}

func (router *routerBackend) Items(context.Context) ([]appstore.Item, error) {
	return router.items, nil
}

func (router *routerBackend) Execute(context.Context, string, appstore.Action, func(string)) error {
	return nil
}

func (router *routerBackend) ExportDiagnostics(context.Context) (string, error) {
	return "", nil
}

func (router *routerBackend) SetPlatform(platform.Info) {}

func newRouterHarness(t *testing.T) (*storeui.Model, *storeinput.Session, *diagnostics.Log) {
	t.Helper()
	logger, err := diagnostics.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	model := storeui.New(&routerBackend{items: []appstore.Item{}})
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	controls := storeinput.NewSession(t.TempDir(), "test-device")
	controls.Connect(storeinput.Identity{Name: "Test Pad"}, true)
	// Connect without a saved mapping opens the setup wizard; route through
	// the normal mode so the semantic actions are exercised.
	controls.Mode = storeinput.Normal
	return model, controls, logger
}

func TestHandleQuitExits(t *testing.T) {
	model, controls, logger := newRouterHarness(t)
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventQuit})
	if !effects.Exit {
		t.Fatal("quit event did not exit")
	}
}

func TestHandleButtonReportsSemanticAction(t *testing.T) {
	model, controls, logger := newRouterHarness(t)
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: int(storeinput.AutoMapping()[storeinput.Down])})
	if effects.Action != storeinput.Down {
		t.Fatalf("down button reported %q, want %q", effects.Action, storeinput.Down)
	}
	if effects.Exit {
		t.Fatal("navigation must not exit")
	}
}

func TestHandleExitActionQuits(t *testing.T) {
	model, controls, logger := newRouterHarness(t)
	effects := Handle(context.Background(), model, controls, logger, Event{Kind: EventButton, Button: int(storeinput.AutoMapping()[storeinput.Exit])})
	if !effects.Exit {
		t.Fatal("exit action did not exit")
	}
}

func TestHandleBackWalksFocusBeforeExiting(t *testing.T) {
	model, controls, logger := newRouterHarness(t)
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
	model, controls, logger := newRouterHarness(t)
	Handle(context.Background(), model, controls, logger, Event{Kind: EventControllerRemoved})
	if controls.Connected {
		t.Fatal("controller removal did not disconnect the session")
	}
}
