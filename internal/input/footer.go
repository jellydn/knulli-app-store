package input

import "strings"

// Hint is one instruction shown in a screen footer. The verb describes what a
// semantic action does on that screen, and the physical button name is read
// from the automatic binding set.
type Hint struct {
	// Action is the semantic action the hint refers to.
	Action Action
	// Verb is the action's meaning on the current screen.
	Verb string
	// With is the second action a paired hint names. Paging is one verb with
	// two directions, so its hint reads "Page (PGUP/PGDN)" instead of stating
	// the same verb twice with a different button each time.
	With Action
}

// Canonical footer verbs are screen-independent for the actions that recur, so
// the same action reads the same way on every screen. A verb is the action's
// name in title case; the button hint beside it carries the physical label.
const (
	VerbNavigate    = "Navigate"
	VerbPage        = "Page"
	VerbSelect      = "Confirm"
	VerbBack        = "Back"
	VerbDiagnostics = "Diagnostics"
	VerbExit        = "Quit"
)

// Verbs maps every semantic action to its canonical footer verb.
var Verbs = map[Action]string{
	Up:          VerbNavigate,
	Down:        VerbNavigate,
	Left:        VerbNavigate,
	Right:       VerbNavigate,
	PageUp:      VerbPage,
	PageDown:    VerbPage,
	Confirm:     VerbSelect,
	Back:        VerbBack,
	Diagnostics: VerbDiagnostics,
	Exit:        VerbExit,
}

// PageHint is the one hint that names two buttons, because paging is a single
// verb with two directions. Both actions must be bound for it to render.
func PageHint() Hint {
	return Hint{Action: PageUp, With: PageDown, Verb: VerbPage}
}

// NewHint describes an action with its canonical verb.
func NewHint(action Action) Hint {
	return Hint{Action: action, Verb: Verb(action)}
}

// Verb returns the canonical footer verb for an action.
func Verb(action Action) string {
	if verb, ok := Verbs[action]; ok {
		return verb
	}
	return strings.ToUpper(string(action))
}

// Footer renders one footer line as "VERB (BUTTON)" pairs. It is the single
// place a physical button label is written, which keeps every screen
// consistent and free of duplicated hints.
//
// The quit hint is the one action no button carries on its own, so the caller
// passes the chord it is actually listening for ("SELECT + Y", "TAB + Y"). An
// empty chord leaves that hint out rather than printing a button that does
// nothing.
func Footer(mapping Mapping, hints []Hint, chord string) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		switch {
		case hint.Action == Exit:
			if chord != "" {
				parts = append(parts, hint.Verb+" ("+chord+")")
			}
		case hint.With != "":
			// A paired hint names both buttons or neither: half a pair would
			// promise a direction the current screen does not accept.
			first, hasFirst := mapping[hint.Action]
			second, hasSecond := mapping[hint.With]
			if !hasFirst || !hasSecond {
				continue
			}
			parts = append(parts, hint.Verb+" ("+ButtonLabel(first)+"/"+ButtonLabel(second)+")")
		default:
			parts = append(parts, hint.Verb+" ("+ButtonLabel(mapping[hint.Action])+")")
		}
	}
	return strings.Join(parts, "  ")
}

// QuitChord names the held anchor that turns the next trigger press into the
// quit, in the order the user presses it. The anchor is the input source's own
// Select key — the pad's SELECT button or the keyboard's Tab — and the trigger
// is the Y button used for diagnostics.
func QuitChord(mapping Mapping, source string) string {
	anchor := "SELECT"
	if source == KeyboardSourceName {
		anchor = "TAB"
	}
	trigger, ok := mapping[Diagnostics]
	if !ok {
		return ""
	}
	return anchor + " + " + ButtonLabel(trigger)
}
