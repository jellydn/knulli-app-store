package input

import (
	"strings"
	"testing"
)

func TestFooterLabelsComeFromTheActiveMapping(t *testing.T) {
	swapped := AutoMapping()
	swapped[Confirm], swapped[Back] = swapped[Back], swapped[Confirm]
	got := Footer(swapped, []Hint{NewHint(Confirm), NewHint(Back)})
	if got != "Confirm (B)  Back (A)" {
		t.Fatalf("footer did not use the active mapping: %q", got)
	}
}

func TestFooterUsesTheCanonicalVerbForEveryAction(t *testing.T) {
	mapping := AutoMapping()
	for _, action := range Actions {
		want := Verbs[action] + " (" + ButtonLabel(mapping[action]) + ")"
		if got := Footer(mapping, []Hint{NewHint(action)}); got != want {
			t.Fatalf("%s footer = %q, want %q", action, got, want)
		}
	}
}

func TestFooterRendersAHintWithoutAButton(t *testing.T) {
	if got := Footer(AutoMapping(), []Hint{{Verb: VerbAnyButton}}); got != VerbAnyButton {
		t.Fatalf("buttonless hint = %q", got)
	}
	if got := Footer(AutoMapping(), nil); got != "" {
		t.Fatalf("empty hint list = %q", got)
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
