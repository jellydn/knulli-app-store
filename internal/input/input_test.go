package input

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAutomaticControls(t *testing.T) {
	for _, layout := range []string{"knulli", "xbox", ""} {
		t.Run(layout, func(t *testing.T) {
			t.Setenv("STORE_CONTROLLER_LAYOUT", layout)
			// These are SDL logical events produced by ES a/b/x/y, not raw
			// joystick positions. Explicit expectations catch a double swap.
			buttons := []int{0, 1, 2, 3}
			if layout == "knulli" {
				buttons = []int{1, 0, 3, 2}
			}
			for _, device := range []string{"trimui-smart-pro", "magicx-zero-28"} {
				s := NewSession(t.TempDir(), device)
				s.Connect(Identity{Name: "Pad"}, true)
				if s.Mode != Normal || !s.Connected {
					t.Fatalf("first connection requires setup: %#v", s)
				}
				for i, want := range []Action{Confirm, Back, "", ""} {
					action, effect := s.HandleButton(buttons[i])
					wantEffect := NoEffect
					if i == 3 {
						wantEffect = ExportDiagnostics
					}
					if action != want || effect != wantEffect || s.Mode != Normal {
						t.Fatalf("%s ES %s: action=%s effect=%s mode=%s", device, []string{"A", "B", "X", "Y"}[i], action, effect, s.Mode)
					}
				}
				for button, want := range map[int]Action{11: Up, 12: Down, 13: Left, 14: Right, 9: PageUp, 10: PageDown} {
					if action, effect := s.HandleButton(button); action != want || effect != NoEffect {
						t.Fatalf("button %d: %s %s", button, action, effect)
					}
				}
				if action, effect := s.HandleButton(99); action != "" || effect != NoEffect {
					t.Fatal("unknown button acted")
				}
				s.SetChordAnchor(true)
				s.SetChordAnchor(false)
				if action, effect := s.HandleButton(buttons[3]); action != "" || effect != ExportDiagnostics {
					t.Fatal("released chord still quits")
				}
				s.SetChordAnchor(true)
				if action, effect := s.HandleButton(buttons[3]); action != Exit || effect != NoEffect || s.ChordHeld {
					t.Fatal("Select+Y did not exclusively quit")
				}
				if s.QuitChord() != "SELECT + Y" {
					t.Fatal(s.QuitChord())
				}
				for i, label := range []string{"A", "B", "X", "Y"} {
					if ButtonLabel(buttons[i]) != label {
						t.Fatalf("ES %s label = %s", label, ButtonLabel(buttons[i]))
					}
				}
			}
		})
	}
}

func TestOldMappingsAreIgnoredAndUntouched(t *testing.T) {
	for _, content := range []string{"", "broken JSON", `{"schema":"org.knulli.app-store/controller-mappings/v1","records":[{"identity":{"device":"trimui-smart-pro","guid":"pad"},"mapping":{"up":11,"down":12,"left":13,"right":14,"page-up":9,"page-down":10,"confirm":3,"back":2,"diagnostics":0}}]}`} {
		root := t.TempDir()
		path := filepath.Join(root, "userdata/system/configs/knulli-app-store/controller-mappings.json")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if content != "" {
			if err := os.WriteFile(path, []byte(content), 0400); err != nil {
				t.Fatal(err)
			}
		}
		for _, layout := range []string{"knulli", "xbox"} {
			t.Setenv("STORE_CONTROLLER_LAYOUT", layout)
			s := NewSession(root, "trimui-smart-pro")
			s.Connect(Identity{GUID: "pad", Name: "Pad"}, true)
			if s.Mode != Normal || !reflect.DeepEqual(s.Mapping, AutoMapping()) {
				t.Fatal("old mapping overrode defaults or required setup")
			}
			s.SetChordAnchor(true)
			s.Disconnect()
			if s.ChordHeld || s.Mode != Blocked {
				t.Fatal("disconnect retained chord or navigation")
			}
			if action, _ := s.HandleButton(12); action != "" {
				t.Fatal("disconnected controller navigated")
			}
			s.Connect(Identity{Name: "Replacement"}, false)
			if s.Mode != Normal || s.ChordHeld || !reflect.DeepEqual(s.Mapping, AutoMapping()) {
				t.Fatal("hotplug requires setup or keeps stale controls")
			}
			keyboard := NewDesktopSession(root, "trimui-smart-pro")
			if keyboard.Mode != Normal || !reflect.DeepEqual(keyboard.Mapping, KeyboardMapping()) {
				t.Fatal("desktop did not open catalogue")
			}
		}
		got, err := os.ReadFile(path)
		if content == "" {
			if !os.IsNotExist(err) {
				t.Fatal("created a mapping file")
			}
		} else if err != nil || string(got) != content {
			t.Fatal("modified the user's mapping file")
		}
		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			t.Fatal(err)
		}
		want := 1
		if content == "" {
			want = 0
		}
		if len(entries) != want {
			t.Fatal("created mapping sidecar files")
		}
	}
}

func TestDesktopKeyboardAndBlockedRecovery(t *testing.T) {
	s := NewDesktopSession(t.TempDir(), "")
	for action, button := range KeyboardMapping() {
		code, ok := s.Binding(action)
		if !ok || code != button {
			t.Fatal("missing keyboard binding")
		}
		if ButtonLabel(button) == "" {
			t.Fatal("missing key label")
		}
		got, effect := s.HandleButton(button)
		if action == Diagnostics {
			if got != "" || effect != ExportDiagnostics {
				t.Fatal("Y did not export")
			}
		} else if got != action {
			t.Fatalf("key %d = %s", button, got)
		}
	}
	if s.QuitChord() != "TAB + Y" {
		t.Fatal(s.QuitChord())
	}
	s.SetChordAnchor(true)
	if action, _ := s.HandleButton(KeyY); action != Exit {
		t.Fatal("Tab+Y did not quit")
	}
	s = NewSession(t.TempDir(), "device")
	if action, effect := s.HandleButton(s.Mapping[Confirm]); action != "" || effect != ExportDiagnostics {
		t.Fatal("blocked diagnostics inaccessible")
	}
}

func TestFooterAutomaticLabels(t *testing.T) {
	for _, layout := range []string{"knulli", "xbox", ""} {
		t.Setenv("STORE_CONTROLLER_LAYOUT", layout)
		mapping := AutoMapping()
		if got := Footer(mapping, []Hint{NewHint(Confirm), NewHint(Back), NewHint(Diagnostics), NewHint(Exit)}, QuitChord(mapping, "SDL")); got != "Confirm (A)  Back (B)  Diagnostics (Y)  Quit (SELECT + Y)" {
			t.Fatal(got)
		}
		if got := Footer(mapping, []Hint{PageHint()}, ""); got != "Page (L1/R1)" {
			t.Fatal(got)
		}
	}
	if got := Footer(KeyboardMapping(), []Hint{NewHint(Confirm), NewHint(Back), NewHint(Diagnostics), PageHint(), NewHint(Exit)}, "TAB + Y"); got != "Confirm (ENTER)  Back (ESC)  Diagnostics (Y)  Page (PGUP/PGDN)  Quit (TAB + Y)" {
		t.Fatal(got)
	}
}
