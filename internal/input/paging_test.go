package input

import (
	"reflect"
	"testing"
)

// Every action belongs to exactly one half of the set, because the two lists are
// what the wizard, the validator and the store all reason about.
func TestEveryActionIsEitherRequiredOrOptional(t *testing.T) {
	seen := make(map[Action]bool, len(Actions))
	for _, action := range Required {
		if optional(action) {
			t.Fatalf("%s is required and optional at once", action)
		}
		seen[action] = true
	}
	for _, action := range Optional {
		if !optional(action) {
			t.Fatalf("%s is optional but not reported as one", action)
		}
		if seen[action] {
			t.Fatalf("%s is listed twice", action)
		}
		seen[action] = true
	}
	for _, action := range Actions {
		if !seen[action] {
			t.Fatalf("%s is neither required nor optional", action)
		}
	}
	if len(seen) != len(Actions) {
		t.Fatalf("%d actions are covered for a set of %d", len(seen), len(Actions))
	}
	if optional(Confirm) || !optional(PageUp) || !optional(PageDown) {
		t.Fatal("the optional half is not the paging pair")
	}
}

// A skipped plan drops the pair and keeps the canonical order, so the wizard
// asks for the same actions in the same order whether or not paging is in it.
func TestPlanKeepsTheCanonicalOrderAndDropsThePairWhenSkipped(t *testing.T) {
	if got := Plan(true); !reflect.DeepEqual(got, Actions) {
		t.Fatalf("the assigned plan is %v, want every action", got)
	}
	want := []Action{Up, Down, Left, Right, Confirm, Back, Diagnostics}
	if got := Plan(false); !reflect.DeepEqual(got, want) {
		t.Fatalf("the skipped plan is %v, want %v", got, want)
	}
}

// The detected path can leave the pair out. The saved mapping then binds only
// the actions the user said this pad carries, and it stays a mapping the app
// accepts.
func TestDetectedMappingCanBeSavedWithoutPaging(t *testing.T) {
	root := t.TempDir()
	identity := Identity{Device: "trimui-smart-pro", GUID: "no-shoulders", Name: "Pad"}
	session := NewSession(root, "trimui-smart-pro")
	session.Connect(identity, true)
	press(session, Confirm)
	if session.Mode != Paging {
		t.Fatalf("the detected choice skipped the paging question: %s", session.Mode)
	}
	press(session, Down)
	press(session, Confirm)
	if session.Mode != Normal || session.Source != "saved detected mapping" {
		t.Fatalf("skipping paging did not save the detected mapping: mode=%s source=%q error=%q", session.Mode, session.Source, session.ValidationError)
	}

	saved, found, err := NewStore(root).Load(identity)
	if err != nil || !found {
		t.Fatalf("the mapping was not saved: found=%v err=%v", found, err)
	}
	if err := saved.Validate(); err != nil {
		t.Fatalf("the saved mapping is not usable: %v", err)
	}
	if _, bound := saved[PageUp]; bound {
		t.Fatalf("a skipped pair was saved anyway: %#v", saved)
	}
	for _, action := range Required {
		if _, ok := saved[action]; !ok {
			t.Fatalf("the required action %s was dropped with the pair", action)
		}
	}
	// The rest of the app is untouched: the required actions still work.
	if action, _ := session.HandleButton(saved[Down]); action != Down {
		t.Fatalf("down produced %q without paging", action)
	}
	if _, ok := session.Binding(PageUp); ok {
		t.Fatal("an unbound action still reported a binding")
	}
}

// Skipping in the wizard keeps the pair out of the calibration plan, so a
// shoulderless pad is never asked for a button it does not have.
func TestCustomSetupWithoutPagingNeverAsksForThePair(t *testing.T) {
	session := NewSession(t.TempDir(), "trimui-smart-pro")
	session.Connect(Identity{GUID: "custom-skip", Name: "Pad"}, true)
	press(session, Down)
	press(session, Down)
	press(session, Confirm)
	press(session, Down)
	press(session, Confirm)
	if session.Mode != Calibrating {
		t.Fatalf("customize did not start calibration: %s", session.Mode)
	}
	if got := session.Calibration.Actions; !reflect.DeepEqual(got, Required) && !reflect.DeepEqual(got, Plan(false)) {
		t.Fatalf("the plan asks for %v, want only the required actions", got)
	}

	buttons := []int{2, 4, 5, 6, 7, 8, 9}
	for index, button := range buttons {
		session.HandleButton(button)
		if session.Mode != Review {
			t.Fatalf("action %d skipped assignment review: %s", index, session.Mode)
		}
		press(session, Confirm)
	}
	if session.Mode != Preview {
		t.Fatalf("the required assignments did not reach the preview: %s", session.Mode)
	}
	for _, button := range buttons {
		session.HandleButton(button)
	}
	if session.Mode != Normal || session.Source != "saved custom mapping" {
		t.Fatalf("the mapping was not saved: mode=%s source=%s error=%s", session.Mode, session.Source, session.ValidationError)
	}
	for _, action := range Optional {
		if _, bound := session.Mapping[action]; bound {
			t.Fatalf("%s was bound by a setup that skipped paging", action)
		}
	}
}

// A mapping may bind a button the pad turns out not to have. The preview is
// finished by the required actions, and an optional binding the user never
// pressed is left out of what gets saved, so a mapping never claims a button
// the pad has not proved.
func TestAnOptionalButtonTheUserNeverPressedIsNotSaved(t *testing.T) {
	root := t.TempDir()
	identity := Identity{Device: "trimui-smart-pro", GUID: "unproved", Name: "Pad"}
	session := NewSession(root, "trimui-smart-pro")
	session.Connect(identity, true)
	press(session, Down)
	press(session, Confirm)
	press(session, Confirm)
	if session.Mode != Preview {
		t.Fatalf("the detected test did not open the preview: %s", session.Mode)
	}
	for _, action := range Required {
		press(session, action)
	}
	if session.Mode != Normal {
		t.Fatalf("the required actions did not finish the preview: mode=%s error=%q", session.Mode, session.ValidationError)
	}
	if _, bound := session.Mapping[PageUp]; bound {
		t.Fatal("an unproved button was kept in the session mapping")
	}
	saved, found, err := NewStore(root).Load(identity)
	if err != nil || !found {
		t.Fatalf("the mapping was not saved: found=%v err=%v", found, err)
	}
	for _, action := range Optional {
		if _, bound := saved[action]; bound {
			t.Fatalf("%s was saved without ever being pressed", action)
		}
	}
}

// Back abandons the question rather than answering it, so the setup choice can
// be changed before anything is bound or written.
func TestPagingQuestionBackReturnsToTheSetupChoices(t *testing.T) {
	session := NewSession(t.TempDir(), "trimui-smart-pro")
	session.Connect(Identity{GUID: "back", Name: "Pad"}, true)
	press(session, Confirm)
	if session.Mode != Paging {
		t.Fatalf("the setup choice did not ask about paging: %s", session.Mode)
	}
	press(session, Back)
	if session.Mode != Setup {
		t.Fatalf("Back left the question on %s", session.Mode)
	}
	if session.Paging {
		t.Fatal("Back recorded an answer")
	}
	// Safe Exit is not a mapping, so it never asks the question at all.
	press(session, Down)
	press(session, Down)
	press(session, Down)
	if action, _ := session.HandleButton(AutoMapping()[Confirm]); action != Exit {
		t.Fatalf("safe exit asked a question and produced %q", action)
	}
}

// A record without the optional pair is a working mapping, not a stale one: the
// pad it belongs to simply has no shoulder buttons, and the app keeps it.
func TestRecordWithoutPagingStillDrivesTheApp(t *testing.T) {
	root := t.TempDir()
	identity := Identity{Device: "magicx-zero-28", GUID: "seven", Name: "Pad"}
	seven := AutoMapping().Without(PageUp, PageDown)
	if err := NewStore(root).Save(identity, seven); err != nil {
		t.Fatal(err)
	}
	session := NewSession(root, identity.Device)
	session.Connect(identity, true)
	if session.Mode != Normal || session.Source != "saved controller mapping" || session.ValidationError != "" {
		t.Fatalf("a mapping without paging was refused: mode=%s source=%q error=%q", session.Mode, session.Source, session.ValidationError)
	}
	if _, bound := session.Mapping[PageUp]; bound {
		t.Fatal("paging appeared in a mapping that never bound it")
	}
	if action, _ := session.HandleButton(seven[Down]); action != Down {
		t.Fatalf("down produced %q", action)
	}
	session.HandleButton(seven[Diagnostics])
	if session.Mode != Settings {
		t.Fatalf("Settings did not open without paging: %s", session.Mode)
	}
}

// The validator is what keeps a partially bound mapping usable: the required
// half must be there, and a mapping may say nothing about the optional half.
func TestValidationRequiresTheRequiredHalfOnly(t *testing.T) {
	full := AutoMapping()
	if err := full.Validate(); err != nil {
		t.Fatalf("the full mapping is not valid: %v", err)
	}
	if err := full.Without(PageUp, PageDown).Validate(); err != nil {
		t.Fatalf("a mapping without paging is not valid: %v", err)
	}
	if err := full.Without(Diagnostics).Validate(); err == nil {
		t.Fatal("a mapping missing a required action was accepted")
	}
	unknown := full.Clone()
	unknown[Action("rewind")] = 20
	if err := unknown.Validate(); err == nil {
		t.Fatal("a mapping with an unknown action was accepted")
	}
}
