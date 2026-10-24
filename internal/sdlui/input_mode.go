package sdlui

// Input mode selection is device-agnostic, so it compiles and tests without
// cgo. The SDL adapter reads the chosen mode instead of guessing.

import "fmt"

// InputMode names the source a GUI run accepts input from.
type InputMode string

const (
	// InputAuto uses an SDL GameController. With none attached the run stays
	// blocked, which is the device rule: no catalogue with unknown controls.
	InputAuto InputMode = "auto"
	// InputKeyboard is the desktop source. The keyboard carries the semantic
	// actions, so a machine without a device can walk every screen and flow.
	InputKeyboard InputMode = "keyboard"
)

// ParseInputMode reads the -input flag. An empty value is the device default.
func ParseInputMode(value string) (InputMode, error) {
	switch InputMode(value) {
	case "":
		return InputAuto, nil
	case InputAuto:
		return InputAuto, nil
	case InputKeyboard:
		return InputKeyboard, nil
	default:
		return "", fmt.Errorf("unknown input mode %q: use %s or %s", value, InputAuto, InputKeyboard)
	}
}

// IsKeyboard reports whether the keyboard is the input source.
func (mode InputMode) IsKeyboard() bool {
	return mode == InputKeyboard
}
