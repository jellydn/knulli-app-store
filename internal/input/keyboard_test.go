package input

import (
	"strconv"
	"testing"
)

// pressKey sends the binding the current screen accepts for an action, the way
// the SDL adapter translates a keyboard key.
func pressKey(t *testing.T, session *Session, action Action) {
	t.Helper()
	code, ok := session.Binding(action)
	if !ok {
		t.Fatalf("screen %s has no binding for %s", session.Mode, action)
	}
	session.HandleButton(code)
}

func TestKeyboardMappingIsACompleteBindingSet(t *testing.T) {
	mapping := KeyboardMapping()
	if err := mapping.Validate(); err != nil {
		t.Fatalf("keyboard mapping is not usable: %v", err)
	}
	for _, action := range Actions {
		label := ButtonLabel(mapping[action])
		if label == "" || label == "BUTTON "+strconv.Itoa(mapping[action]) {
			t.Fatalf("%s carries an unnamed keyboard binding: %q", action, label)
		}
	}
	if KeyboardMapping()[Confirm] != KeyEnter || KeyboardMapping()[Back] != KeyEscape {
		t.Fatal("keyboard mapping does not use the documented keys")
	}
}

func TestKeyboardBindingsDoNotCollideWithControllerButtons(t *testing.T) {
	controller := AutoMapping()
	for _, action := range Actions {
		if controller[action] == KeyboardMapping()[action] {
			t.Fatalf("%s uses the same code for a pad button and a key", action)
		}
	}
}

func TestDesktopSessionOpensSetupAndSavesWithTheKeyboard(t *testing.T) {
	root := t.TempDir()
	session := NewDesktopSession(root, "trimui-smart-pro")
	if !session.Connected || session.Mode != Setup || !session.FirstRun {
		t.Fatalf("desktop launch did not open first-run setup: connected=%v mode=%s first=%v", session.Connected, session.Mode, session.FirstRun)
	}
	if session.Source != KeyboardSourceName || session.AutoSource != KeyboardSourceName {
		t.Fatalf("desktop source = %q / %q, want %q", session.Source, session.AutoSource, KeyboardSourceName)
	}
	if session.DetectedMapping()[Confirm] != KeyEnter || session.Mapping[Confirm] != KeyEnter {
		t.Fatalf("desktop session did not detect the keyboard: %#v", session.Mapping)
	}
	// A first run leaves through the SAFE EXIT item or the quit chord, never
	// through a lone Back press.
	if action, _ := session.HandleButton(KeyEscape); action != "" {
		t.Fatalf("first-run Back produced %q", action)
	}
	if got := session.QuitChord(); got != "TAB + Y" {
		t.Fatalf("desktop quit chord = %q", got)
	}
	session.SetChordAnchor(true)
	if action, _ := session.HandleButton(KeyY); action != Exit {
		t.Fatalf("desktop quit chord did not exit: %q", action)
	}
	pressKey(t, session, Confirm)
	if session.Mode != Normal || session.FirstRun {
		t.Fatalf("detected keyboard mapping did not open the catalogue: mode=%s", session.Mode)
	}
	saved, found, err := NewStore(root).Load(KeyboardIdentity("trimui-smart-pro"))
	if err != nil || !found || saved[Confirm] != KeyEnter {
		t.Fatalf("keyboard mapping was not saved: found=%v err=%v mapping=%#v", found, err, saved)
	}
	// A desktop run must not touch the record a physical controller owns.
	if _, found, err := NewStore(root).Load(Identity{Device: "trimui-smart-pro", GUID: "pad", Name: "Pad"}); err != nil || found {
		t.Fatalf("keyboard mapping reached a controller identity: found=%v err=%v", found, err)
	}
	reloaded := NewDesktopSession(root, "trimui-smart-pro")
	if reloaded.Mode != Normal || reloaded.FirstRun || reloaded.Source != "saved keyboard mapping" || reloaded.Message != "Saved keyboard mapping loaded" {
		t.Fatalf("saved keyboard mapping did not load with keyboard labels: mode=%s first=%v source=%q message=%q", reloaded.Mode, reloaded.FirstRun, reloaded.Source, reloaded.Message)
	}
}

func TestDesktopSessionPersistsWithoutPlatformDeviceMetadata(t *testing.T) {
	root := t.TempDir()
	session := NewDesktopSession(root, "")
	pressKey(t, session, Confirm)
	if session.Mode != Normal || session.ValidationError != "" {
		t.Fatalf("device-less desktop mapping was not saved: mode=%s error=%q", session.Mode, session.ValidationError)
	}

	reloaded := NewDesktopSession(root, "")
	if reloaded.Mode != Normal || reloaded.Source != "saved keyboard mapping" {
		t.Fatalf("device-less desktop mapping was not reloaded: mode=%s source=%q error=%q", reloaded.Mode, reloaded.Source, reloaded.ValidationError)
	}
}

func TestDesktopKeyboardCompletesCustomSetupAndPreview(t *testing.T) {
	session := NewDesktopSession(t.TempDir(), "magicx-zero-28")
	pressKey(t, session, Down)
	pressKey(t, session, Down)
	pressKey(t, session, Confirm)
	if session.Mode != Calibrating {
		t.Fatalf("customize did not start calibration: %s", session.Mode)
	}
	// One action is assigned at a time: each assignment lands on the review
	// screen, where the first item carries the calibration to the next action.
	for step := 0; session.Mode != Preview && step < 2*len(Actions); step++ {
		switch session.Mode {
		case Calibrating:
			action, ok := session.Calibration.Current()
			if !ok {
				t.Fatalf("calibration asked for no action at index %d", session.Calibration.Index)
			}
			pressKey(t, session, action)
			if session.Mode != Review {
				t.Fatalf("assigning %s did not open the review screen: %s", action, session.Mode)
			}
		case Review:
			if session.ReviewIndex != 0 {
				t.Fatalf("review did not start on the first item: index %d", session.ReviewIndex)
			}
			pressKey(t, session, Confirm)
		default:
			t.Fatalf("unexpected screen while assigning: %s", session.Mode)
		}
	}
	if session.Mode != Preview {
		t.Fatalf("calibration did not reach the preview: %s", session.Mode)
	}
	assigned := session.Calibration.Mapping
	for _, action := range Actions {
		if assigned[action] != KeyboardMapping()[action] {
			t.Fatalf("preview mapping put %s on %d, want the pressed key %d", action, assigned[action], KeyboardMapping()[action])
		}
	}
	for _, action := range Actions {
		pressKey(t, session, action)
	}
	if session.Mode != Normal || session.Source != "saved custom mapping" {
		t.Fatalf("tested keyboard mapping was not saved: mode=%s source=%s", session.Mode, session.Source)
	}
}

func TestScreenBindingFollowsTheScreenThatAcceptsIt(t *testing.T) {
	session := NewDesktopSession(t.TempDir(), "trimui-smart-pro")
	if code, ok := session.Binding(Confirm); !ok || code != KeyEnter {
		t.Fatalf("first-run setup did not offer the keyboard binding: %d %v", code, ok)
	}
	pressKey(t, session, Confirm)
	if code, ok := session.Binding(Confirm); !ok || code != KeyEnter {
		t.Fatalf("catalogue did not offer the saved keyboard binding: %d %v", code, ok)
	}
	custom := AutoMapping()
	custom[Confirm] = 42
	session.Mode = Preview
	session.Calibration = NewPreview(custom)
	if code, ok := session.Binding(Confirm); !ok || code != 42 {
		t.Fatalf("preview did not offer the mapping it tests: %d %v", code, ok)
	}
}
