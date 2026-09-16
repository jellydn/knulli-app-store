package sdlui

// Device-agnostic event handling: the SDL adapter translates SDL events into
// Event values, and Handle applies them to the UI model and input session.
// Everything here compiles without cgo, so the event routing is testable on
// any platform.

import (
	"context"
	"fmt"

	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

// EventKind names one device-level input event.
type EventKind int

const (
	// EventNone is the absence of a translated event.
	EventNone EventKind = iota
	// EventQuit asks the UI to exit.
	EventQuit
	// EventButton is one button press, already mapped to a raw button code.
	// The active controller produces it.
	EventButton
	// EventKey is one portable keyboard key.
	EventKey
	// EventControllerConnected means a controller became active.
	EventControllerConnected
	// EventControllerRemoved means the active controller went away.
	EventControllerRemoved
)

type Key int

const (
	KeyNone Key = iota
	KeyUp
	KeyLeft
	KeyDown
	KeyRight
	KeyConfirm
	KeyBack
	KeyDiagnostics
	KeyExit
)

// Event is one translated device event.
type Event struct {
	Kind          EventKind
	Button        int // raw button code; only for EventButton
	Key           Key
	Identity      storeinput.Identity
	KnulliMapping bool
}

// Effects reports what Handle did with an event.
type Effects struct {
	// Exit requests application shutdown.
	Exit bool
	// Action is the semantic action dispatched to the model, if any.
	Action storeinput.Action
	// ExportedDiagnostics reports that a diagnostics export was triggered.
	ExportedDiagnostics bool
}

// Handle applies one device event to the model and input session. The
// adapter owns the device: it decides which events exist and which raw
// button codes they carry; Handle owns everything that happens next.
func Handle(ctx context.Context, model *storeui.Model, controls *storeinput.Session, logger *diagnostics.Log, event Event) Effects {
	switch event.Kind {
	case EventQuit:
		return Effects{Exit: true}
	case EventControllerConnected:
		beforeMode, beforeSource := controls.Mode, controls.Source
		controls.Connect(event.Identity, event.KnulliMapping)
		logger.Event("controller_connected", "device", controls.Device, "name", event.Identity.Name, "guid", event.Identity.GUID, "mapping_source", controls.Source)
		logControllerTransition(logger, controls, beforeMode, beforeSource)
		if controls.ValidationError != "" {
			logger.Event("controller_mapping_validation_failed", "error", controls.ValidationError)
		}
		return Effects{}
	case EventControllerRemoved:
		beforeMode, beforeSource := controls.Mode, controls.Source
		controls.Disconnect()
		logControllerTransition(logger, controls, beforeMode, beforeSource)
		return Effects{}
	case EventButton:
		return processButton(ctx, model, controls, event.Button, logger)
	case EventKey:
		action := keyAction(event.Key)
		if action == "" {
			return Effects{}
		}
		mapping := controls.Mapping
		if controls.FirstRun {
			mapping = storeinput.AutoMapping()
		}
		return processButton(ctx, model, controls, int(mapping[action]), logger)
	}
	return Effects{}
}

func keyAction(key Key) storeinput.Action {
	switch key {
	case KeyUp:
		return storeinput.Up
	case KeyLeft:
		return storeinput.Left
	case KeyDown:
		return storeinput.Down
	case KeyRight:
		return storeinput.Right
	case KeyConfirm:
		return storeinput.Confirm
	case KeyBack:
		return storeinput.Back
	case KeyDiagnostics:
		return storeinput.Diagnostics
	case KeyExit:
		return storeinput.Exit
	default:
		return ""
	}
}
func processButton(ctx context.Context, model *storeui.Model, controls *storeinput.Session, button int, logger *diagnostics.Log) Effects {
	beforeMode := controls.Mode
	beforeSource := controls.Source
	beforeError := controls.ValidationError
	beforeIndex, beforeTested := setupProgress(controls)
	action, effect := controls.HandleButton(button)
	afterIndex, afterTested := setupProgress(controls)
	logControllerTransition(logger, controls, beforeMode, beforeSource)
	if afterIndex != beforeIndex && afterIndex >= 0 {
		logger.Event("controller_setup_progress", "completed", fmt.Sprint(afterIndex), "total", fmt.Sprint(len(storeinput.Actions)))
	}
	if afterTested != beforeTested && afterTested >= 0 {
		logger.Event("controller_preview_progress", "tested", fmt.Sprint(afterTested), "total", fmt.Sprint(len(storeinput.Actions)))
	}
	if controls.ValidationError != "" && controls.ValidationError != beforeError {
		logger.Event("controller_mapping_validation_failed", "error", controls.ValidationError)
	}
	if action != "" {
		logger.Event("controller_semantic_action", "action", string(action), "screen", string(beforeMode))
	}
	effects := Effects{Action: action}
	if effect == storeinput.ExportDiagnostics {
		logger.Event("controller_setup_action", "action", "export-diagnostics")
		model.ExportDiagnostics(ctx)
		effects.ExportedDiagnostics = true
	}
	effects.Exit = dispatchAction(ctx, model, action)
	return effects
}

func logControllerTransition(logger *diagnostics.Log, controls *storeinput.Session, beforeMode storeinput.Mode, beforeSource string) {
	if controls.Mode != beforeMode || controls.Source != beforeSource {
		logger.Event("controller_screen_transition", "from", string(beforeMode), "to", string(controls.Mode), "mapping_source", controls.Source, "message", controls.Message)
	}
}

func setupProgress(controls *storeinput.Session) (int, int) {
	if controls.Calibration == nil {
		return -1, -1
	}
	return controls.Calibration.Index, len(controls.Calibration.Tested)
}

func dispatchAction(ctx context.Context, model *storeui.Model, action storeinput.Action) bool {
	switch action {
	case storeinput.Up, storeinput.Left:
		model.Move(-1)
	case storeinput.Down, storeinput.Right:
		model.Move(1)
	case storeinput.Confirm:
		model.Select(ctx)
	case storeinput.Back:
		return model.Back()
	case storeinput.Exit:
		return true
	}
	return false
}
