package input

import (
	"strings"
	"testing"
)

func TestFooterLabelsComeFromTheActiveMapping(t *testing.T) {
	swapped := AutoMapping()
	swapped[Confirm], swapped[Back] = swapped[Back], swapped[Confirm]
	got := Footer(swapped, []Hint{NewHint(Confirm), NewHint(Back)}, "")
	if got != "Confirm (B)  Back (A)" {
		t.Fatalf("footer did not use the active mapping: %q", got)
	}
}

func TestFooterUsesTheCanonicalVerbForEveryAction(t *testing.T) {
	mapping := AutoMapping()
	for _, action := range Actions {
		want := Verbs[action] + " (" + ButtonLabel(mapping[action]) + ")"
		if got := Footer(mapping, []Hint{NewHint(action)}, ""); got != want {
			t.Fatalf("%s footer = %q, want %q", action, got, want)
		}
	}
}

func TestFooterRendersAHintWithoutAButton(t *testing.T) {
	if got := Footer(AutoMapping(), []Hint{{Verb: VerbAnyButton}}, ""); got != VerbAnyButton {
		t.Fatalf("buttonless hint = %q", got)
	}
	if got := Footer(AutoMapping(), nil, ""); got != "" {
		t.Fatalf("empty hint list = %q", got)
	}
}

// The quit hint is the one line that carries a chord instead of a button
// label, because no single press quits the app.
func TestFooterRendersTheQuitChordInsteadOfAButton(t *testing.T) {
	mapping := AutoMapping()
	if got := Footer(mapping, []Hint{NewHint(Exit)}, "SELECT + Y"); got != "Quit (SELECT + Y)" {
		t.Fatalf("quit hint = %q", got)
	}
	if got := Footer(mapping, []Hint{NewHint(Confirm), NewHint(Exit)}, "TAB + Y"); got != "Confirm (A)  Quit (TAB + Y)" {
		t.Fatalf("quit hint beside a button = %q", got)
	}
	// A screen with no chord to offer must not print a hint that does nothing.
	if got := Footer(mapping, []Hint{NewHint(Confirm), NewHint(Exit)}, ""); got != "Confirm (A)" {
		t.Fatalf("missing chord = %q", got)
	}
}

func TestQuitChordNamesTheAnchorOfEachInputSource(t *testing.T) {
	if got := QuitChord(KeyboardMapping(), KeyboardSourceName); got != "TAB + Y" {
		t.Fatalf("desktop quit chord = %q", got)
	}
	if got := QuitChord(AutoMapping(), "Knulli SDL_GAMECONFIG"); got != "SELECT + Y" {
		t.Fatalf("pad quit chord = %q", got)
	}
	// The trigger is whatever button the mapping gives Settings, so a
	// customized pad keeps reading its own label.
	custom := AutoMapping()
	custom[Diagnostics] = 9
	if got := QuitChord(custom, "saved controller mapping"); got != "SELECT + LEFT SHOULDER" {
		t.Fatalf("custom quit chord = %q", got)
	}
}

func TestFooterKeepsCanonicalVerbsDistinctFromButtonLabels(t *testing.T) {
	for _, verb := range []string{VerbSelect, VerbBack, VerbSettings, VerbExit} {
		if strings.Contains(verb, "SDL ") {
			t.Fatalf("the verb %q names a physical button", verb)
		}
	}
}

func TestActionLabelKeepsDirectionsDistinct(t *testing.T) {
	wants := map[Action]string{Up: "UP", Down: "DOWN", Left: "LEFT", Right: "RIGHT", Confirm: "CONFIRM"}
	for action, want := range wants {
		if got := ActionLabel(action); got != want {
			t.Fatalf("ActionLabel(%q) = %q, want %q", action, got, want)
		}
	}
}
