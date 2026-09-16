package input

import (
	"strings"
	"testing"
)

func TestReviewChoicesUseTheCanonicalVerbs(t *testing.T) {
	if len(ReviewItems) < 2 {
		t.Fatalf("review needs a confirming and an abandoning choice: %#v", ReviewItems)
	}
	if got := ReviewItems[0]; got != strings.ToUpper(Verb(Confirm)) {
		t.Fatalf("the confirming choice is %q, want the confirm verb %q", got, Verb(Confirm))
	}
	if got := ReviewItems[len(ReviewItems)-1]; got != strings.ToUpper(Verb(Back)) {
		t.Fatalf("the abandoning choice is %q, want the back verb %q", got, Verb(Back))
	}
}

func TestScreenMappingMatchesTheScreenThatDrawsIt(t *testing.T) {
	session := NewSession(t.TempDir(), "trimui-smart-pro")
	session.Connected = true
	session.FirstRun = false
	session.Mode = Normal
	swapped := AutoMapping()
	swapped[Confirm], swapped[Back] = swapped[Back], swapped[Confirm]
	session.Mapping = swapped
	if got := session.ScreenMapping(); got[Confirm] != swapped[Confirm] || got[Back] != swapped[Back] {
		t.Fatalf("the catalogue screen must offer the session mapping: %#v", got)
	}
	// A disconnect returns to the detected mapping, so the blocked screen has
	// to label that one and not the saved swap.
	session.Disconnect()
	detected := AutoMapping()
	if got := session.ScreenMapping(); got[Confirm] != detected[Confirm] || got[Back] != detected[Back] {
		t.Fatalf("the blocked screen must offer the detected mapping: %#v", got)
	}
}

func TestMenuChoicesAreUppercaseLabels(t *testing.T) {
	for _, items := range [][]string{SetupItems, ReviewItems, SettingsItems} {
		for _, item := range items {
			if item == "" || item != strings.ToUpper(item) {
				t.Fatalf("menu choice %q is not an uppercase label", item)
			}
		}
	}
}
