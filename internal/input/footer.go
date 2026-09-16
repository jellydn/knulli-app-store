package input

import "strings"

// Hint is one instruction shown in a screen footer. The verb describes what a
// semantic action does on that screen, and the physical button name is read
// from the active mapping. A screen never names a button itself, so a
// customized controller keeps showing its own labels.
type Hint struct {
	// Action is the semantic action the hint refers to. An empty Action means
	// the screen wants a physical input without a named action, and Footer
	// renders the verb alone.
	Action Action
	// Verb is the action's meaning on the current screen.
	Verb string
}

// Canonical footer verbs are screen-independent for the actions that recur, so
// the same action reads the same way on every screen.
const (
	VerbNavigate    = "NAVIGATE"
	VerbSelect      = "SELECT"
	VerbBack        = "BACK"
	VerbSettings    = "SETTINGS"
	VerbExit        = "EXIT"
	VerbAnyButton   = "PRESS ANY BUTTON"
	VerbShownButton = "PRESS EACH SHOWN BUTTON"
)

// Verbs maps every semantic action to its canonical footer verb.
var Verbs = map[Action]string{
	Up:          VerbNavigate,
	Down:        VerbNavigate,
	Left:        VerbNavigate,
	Right:       VerbNavigate,
	Confirm:     VerbSelect,
	Back:        VerbBack,
	Diagnostics: VerbSettings,
	Exit:        VerbExit,
}

// NewHint describes an action with its canonical verb.
func NewHint(action Action) Hint {
	return Hint{Action: action, Verb: Verb(action)}
}

// Verb returns the canonical verb for an action. The footer and the controller
// mapping summary both use it, so one action never reads two ways.
func Verb(action Action) string {
	if verb, ok := Verbs[action]; ok {
		return verb
	}
	return strings.ToUpper(string(action))
}

// Footer renders one footer line as "VERB (BUTTON)" pairs. It is the single
// place a physical button label is written, which keeps every screen
// consistent and free of duplicated hints.
func Footer(mapping Mapping, hints []Hint) string {
	parts := make([]string, 0, len(hints))
	for _, hint := range hints {
		if hint.Action == "" {
			parts = append(parts, hint.Verb)
			continue
		}
		parts = append(parts, hint.Verb+" ("+ButtonLabel(mapping[hint.Action])+")")
	}
	return strings.Join(parts, "  ")
}
