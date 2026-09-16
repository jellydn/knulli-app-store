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
	KeyQ          = 103
	KeyArrowUp    = 104
	KeyArrowDown  = 105
	KeyArrowLeft  = 106
	KeyArrowRight = 107
)

// KeyboardSourceName is the source the GUI reports for a keyboard mapping.
const KeyboardSourceName = "desktop keyboard"

const desktopDeviceIdentity = "desktop"

// keyboardLabels names each keyboard binding for the footer and the mapping
// summary, matching the "source name" shape of the SDL labels.
var keyboardLabels = map[int]string{
	KeyEnter:      "KEY ENTER",
	KeyEscape:     "KEY ESC",
	KeyY:          "KEY Y",
	KeyQ:          "KEY Q",
	KeyArrowUp:    "KEY UP",
	KeyArrowDown:  "KEY DOWN",
	KeyArrowLeft:  "KEY LEFT",
	KeyArrowRight: "KEY RIGHT",
}

// KeyboardMapping is the binding set a desktop run detects for a keyboard.
func KeyboardMapping() Mapping {
	return Mapping{
		Up: KeyArrowUp, Down: KeyArrowDown, Left: KeyArrowLeft, Right: KeyArrowRight,
		Confirm: KeyEnter, Back: KeyEscape, Diagnostics: KeyY, Exit: KeyQ,
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
