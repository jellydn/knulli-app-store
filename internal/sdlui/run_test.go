//go:build sdl

package sdlui

import (
	"context"
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
	"github.com/veandco/go-sdl2/sdl"
)

func TestRunRejectsToolkitMappingWizard(t *testing.T) {
	t.Setenv("INPUT_CAPTURE", "1")
	// No display is required for this guard. It must run before SDL or the
	// toolkit can create a window, write assignments or exit the process.
	t.Setenv("SDL_VIDEODRIVER", "not-a-video-driver")
	err := Run(context.Background(), &routerBackend{}, Options{})
	if err == nil || !strings.Contains(err.Error(), "INPUT_CAPTURE is not supported") {
		t.Fatalf("expected automatic-controls guard before SDL startup, got %v", err)
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
	for _, wanted := range []string{`source="SDL current display mode"`, `height="13107"`, `valid="false"`, `reason="above maximum 7680x4320"`, `event=resolution_selected`, `resolution="1280x720"`, `source="SDL renderer output"`} {
		if !strings.Contains(string(data), wanted) {
			t.Fatalf("resolution log lacks %q: %s", wanted, data)
		}
	}
}

func TestControllerLogsActionsWithoutRawButtonSpam(t *testing.T) {
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
	for _, wanted := range []string{`event=controller_semantic_action`, `action="down"`, `action="confirm"`} {
		if !strings.Contains(logText, wanted) {
			t.Fatalf("missing %s: %s", wanted, logText)
		}
	}
	if strings.Contains(logText, "controller_input") || strings.Contains(logText, "button=") {
		t.Fatal("raw button spam remains")
	}
}

// These are raw SDL events, not semantic test keys: the renderer migration
// must not introduce gabagool's face swaps or Start-to-confirm behavior.
func TestSDLControlsPreserveConfirmationContract(t *testing.T) {
	for _, layout := range []struct {
		name          string
		confirm, back uint8
		more          uint8
	}{
		{"knulli", 1, 0, 2},
		{"xbox", 0, 1, 3},
		{"", 0, 1, 3},
	} {
		t.Run(layout.name, func(t *testing.T) {
			t.Setenv("STORE_CONTROLLER_LAYOUT", layout.name)
			model, controls, logger, _ := newRouterHarness(t)
			model.Items = []appstore.Item{{
				Package: manifest.Package{ID: "org.example.utility", Name: "Utility"},
				Actions: []appstore.Action{appstore.Install},
			}}
			controller := &controllerState{instanceID: 17}
			button := func(kind uint32, which sdl.JoystickID, code uint8) bool {
				return handleEvent(context.Background(), model, &sdl.ControllerButtonEvent{
					Type: kind, Which: which, Button: code,
				}, &controller, controls, logger, InputAuto)
			}
			button(sdl.CONTROLLERBUTTONDOWN, 18, layout.confirm)
			button(sdl.CONTROLLERBUTTONUP, 17, layout.confirm)
			button(sdl.CONTROLLERBUTTONDOWN, 17, uint8(sdl.CONTROLLER_BUTTON_START))
			if model.Focus != storeui.Browse || model.Busy {
				t.Fatal("another controller, button release or Start accepted an action")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, layout.confirm)
			if model.Focus != storeui.Confirm || model.Busy {
				t.Fatal("first confirm must only open confirmation")
			}
			handleEvent(context.Background(), model, &sdl.KeyboardEvent{
				Type: sdl.KEYDOWN, Repeat: 1, Keysym: sdl.Keysym{Sym: sdl.K_RETURN},
			}, &controller, controls, logger, InputAuto)
			if model.Focus != storeui.Confirm || model.Busy {
				t.Fatal("held Enter must not execute the operation")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, layout.back)
			if model.Focus != storeui.Browse {
				t.Fatal("back did not cancel confirmation")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, layout.more)
			if model.Focus != storeui.Actions {
				t.Fatal("More did not open secondary actions")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, chordAnchorButton)
			button(sdl.CONTROLLERBUTTONUP, 17, chordAnchorButton)
			if button(sdl.CONTROLLERBUTTONDOWN, 17, layout.more) {
				t.Fatal("released Select must not leave the quit chord active")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, chordAnchorButton)
			if !button(sdl.CONTROLLERBUTTONDOWN, 17, layout.more) {
				t.Fatal("held Select+Y did not quit")
			}
		})
	}
}

func TestSDLKeyboardModeIgnoresControllerInput(t *testing.T) {
	model, _, logger, _ := newRouterHarness(t)
	controls := storeinput.NewDesktopSession(t.TempDir(), "desktop")
	controller := &controllerState{instanceID: 17}
	for _, event := range []sdl.Event{
		&sdl.ControllerButtonEvent{Type: sdl.CONTROLLERBUTTONDOWN, Which: 17, Button: chordAnchorButton},
		&sdl.ControllerDeviceEvent{Type: sdl.CONTROLLERDEVICEREMOVED, Which: 17},
	} {
		if handleEvent(context.Background(), model, event, &controller, controls, logger, InputKeyboard) {
			t.Fatal("controller input exited a keyboard session")
		}
	}
	if controller == nil || controls.Source != storeinput.KeyboardSourceName {
		t.Fatal("controller input replaced the keyboard session")
	}
}
