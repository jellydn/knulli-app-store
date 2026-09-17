package input

import "strings"

// A desktop run has no GameController, so the keyboard is its input source. A
// keyboard binding uses its own codes, above every SDL GameController button,
// so a key is never mistaken for a pad button and the identity is stored
// separately from a controller's mapping.
const (
	KeyEnter      = 100
	KeyEscape     = 101
	KeyY          = 102
	KeyTab        = 103
	KeyArrowUp    = 104
	KeyArrowDown  = 105
	KeyArrowLeft  = 106
	KeyArrowRight = 107
	KeyPageUp     = 108
	KeyPageDown   = 109
)

// KeyboardSourceName is the source the GUI reports for a keyboard mapping.
const KeyboardSourceName = "desktop keyboard"

const desktopDeviceIdentity = "desktop"

// keyboardLabels names each keyboard binding for the footer and the mapping
// summary, matching the "source name" shape of the controller labels.
var keyboardLabels = map[int]string{
	KeyEnter:      "ENTER",
	KeyEscape:     "ESC",
	KeyY:          "Y",
	KeyTab:        "TAB",
	KeyArrowUp:    "UP",
	KeyArrowDown:  "DOWN",
	KeyArrowLeft:  "LEFT",
	KeyArrowRight: "RIGHT",
	KeyPageUp:     "PGUP",
	KeyPageDown:   "PGDN",
}

// KeyboardMapping is the binding set a desktop run detects for a keyboard.
// KeyTab is the chord anchor rather than a bound action: holding it turns the
// next Y into the quit chord, exactly as holding Select does on a pad.
func KeyboardMapping() Mapping {
	return Mapping{
		Up: KeyArrowUp, Down: KeyArrowDown, Left: KeyArrowLeft, Right: KeyArrowRight,
		PageUp: KeyPageUp, PageDown: KeyPageDown,
		Confirm: KeyEnter, Back: KeyEscape, Diagnostics: KeyY,
	}
}

// KeyboardIdentity is the mapping-store identity of the desktop keyboard. It is
// distinct from every controller identity, so a desktop run can never overwrite
// the mapping saved for a physical controller.
func KeyboardIdentity(device string) Identity {
	if strings.TrimSpace(device) == "" {
		device = desktopDeviceIdentity
	}
	return Identity{Device: device, GUID: "desktop-keyboard", Name: "Keyboard"}
}
