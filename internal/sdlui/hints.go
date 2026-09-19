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
	// Quitting is a chord, so its hint carries the chord rather than a button
	// label. It is only offered where the app can actually be left.
	chord := ""
	if controls != nil {
		chord = controls.QuitChord()
	}
	return storeinput.Footer(footerMapping(controls), hints, chord)
}

// footerMapping is the mapping the current screen actually accepts, so the
// footer labels never disagree with the buttons that work.
func footerMapping(controls *storeinput.Session) storeinput.Mapping {
	if controls == nil {
		return storeinput.AutoMapping()
	}
	return controls.ScreenMapping()
}

// paged reports whether the catalogue holds more rows than the list can show,
// which is the one case where paging moves anything.
func paged(model *storeui.Model) bool {
	return len(model.Items) > listRows
}

// footerHints selects the instructions for the current screen. A screen picks
// semantic actions only; the verbs come from the canonical table in the input
// package and the button labels from the active mapping.
func footerHints(model *storeui.Model, controls *storeinput.Session) []storeinput.Hint {
	confirm := storeinput.NewHint(storeinput.Confirm)
	back := storeinput.NewHint(storeinput.Back)
	quit := storeinput.NewHint(storeinput.Exit)
	if controls == nil || controls.Mode == storeinput.Normal {
		if model.Busy {
			return nil
		}
		if len(model.Items) == 0 {
			return []storeinput.Hint{storeinput.NewHint(storeinput.Diagnostics), quit}
		}
		if model.Focus == storeui.Confirm || model.Focus == storeui.ForceConfirm {
			return []storeinput.Hint{confirm, back}
		}
		// The catalogue is the top of the app: there is nothing behind it, so
		// Back is not offered and quitting is the chord. Paging is offered only
		// when the list is longer than the window, so a short catalogue never
		// advertises a gesture that would move nothing.
		hints := []storeinput.Hint{confirm}
		if paged(model) {
			hints = append(hints, storeinput.PageHint())
		}
		return append(hints, storeinput.NewHint(storeinput.Diagnostics), quit)
	}
	switch controls.Mode {
	case storeinput.Blocked:
		return []storeinput.Hint{confirm, quit}
	case storeinput.Setup:
		if controls.FirstRun {
			return []storeinput.Hint{confirm, quit}
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
