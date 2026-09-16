package sdlui

// Footer selection is device-agnostic: it compiles without cgo so every
// screen's hint list stays testable. draw.go paints the one footer line it
// returns and nothing else draws a button hint.

import (
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

// footerCharacterLimit is the widest footer that fits the canvas: 640 pixels
// minus a 16 pixel margin either side, at the built-in face's 7 pixel advance.
const footerCharacterLimit = 86

// footerText renders the single footer line for the current screen.
func footerText(model *storeui.Model, controls *storeinput.Session) string {
	hints := footerHints(model, controls)
	if len(hints) == 0 {
		return ""
	}
	return storeinput.Footer(footerMapping(controls), hints)
}

// footerMapping is the mapping the current screen actually accepts, so the
// footer labels never disagree with the buttons that work.
func footerMapping(controls *storeinput.Session) storeinput.Mapping {
	if controls == nil {
		return storeinput.AutoMapping()
	}
	return controls.ScreenMapping()
}

// footerHints selects the instructions for the current screen. A screen picks
// semantic actions only; the verbs come from the canonical table in the input
// package and the button labels from the active mapping.
func footerHints(model *storeui.Model, controls *storeinput.Session) []storeinput.Hint {
	confirm := storeinput.NewHint(storeinput.Confirm)
	back := storeinput.NewHint(storeinput.Back)
	if controls == nil || controls.Mode == storeinput.Normal {
		if len(model.Items) == 0 {
			return []storeinput.Hint{back, storeinput.NewHint(storeinput.Diagnostics)}
		}
		if model.Focus == storeui.Confirm || model.Focus == storeui.ForceConfirm {
			return []storeinput.Hint{confirm, back}
		}
		return []storeinput.Hint{confirm, back, storeinput.NewHint(storeinput.Diagnostics)}
	}
	switch controls.Mode {
	case storeinput.Blocked:
		return []storeinput.Hint{confirm, back}
	case storeinput.Setup:
		if controls.FirstRun {
			return []storeinput.Hint{confirm, storeinput.NewHint(storeinput.Exit)}
		}
		return []storeinput.Hint{confirm, back}
	case storeinput.Settings, storeinput.Review:
		return []storeinput.Hint{confirm, back}
	case storeinput.Calibrating:
		return []storeinput.Hint{{Verb: storeinput.VerbAnyButton}}
	case storeinput.Preview:
		return []storeinput.Hint{{Verb: storeinput.VerbShownButton}}
	default:
		return []storeinput.Hint{back, storeinput.NewHint(storeinput.Diagnostics)}
	}
}
