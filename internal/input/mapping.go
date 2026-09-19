package input

import (
	"fmt"
	"sort"
	"strings"
)

type Action string

const (
	Up          Action = "up"
	Down        Action = "down"
	Left        Action = "left"
	Right       Action = "right"
	PageUp      Action = "page-up"
	PageDown    Action = "page-down"
	Confirm     Action = "confirm"
	Back        Action = "back"
	Diagnostics Action = "diagnostics"
	Exit        Action = "exit"
)

// Actions is every action a user binds during controller setup. PageUp and
// PageDown are separate actions rather than a modifier on the directions,
// because a page is a different distance from a step: a list that outgrows its
// window is paged with the shoulders and stepped with the d-pad, and neither
// gesture has to guess which the user meant. Exit is deliberately absent:
// quitting is the Select+Y chord the input layer reads from held state, so no
// single button can end a session by accident and no mapping can claim to be
// the way out.
var Actions = []Action{Up, Down, Left, Right, PageUp, PageDown, Confirm, Back, Diagnostics}

type Mapping map[Action]int

func AutoMapping() Mapping {
	return Mapping{
		Up: 11, Down: 12, Left: 13, Right: 14,
		PageUp: 9, PageDown: 10,
		Confirm: 0, Back: 1, Diagnostics: 3,
	}
}

func (mapping Mapping) Validate() error {
	assigned := make(map[int]Action)
	for _, action := range Actions {
		button, ok := mapping[action]
		if !ok || button < 0 {
			return fmt.Errorf("required action %s has no button", action)
		}
		if other, exists := assigned[button]; exists {
			return fmt.Errorf("button %d conflicts between %s and %s", button, other, action)
		}
		assigned[button] = action
	}
	if len(mapping) != len(Actions) {
		return fmt.Errorf("mapping contains unknown actions")
	}
	return nil
}

func (mapping Mapping) Action(button int) (Action, bool) {
	for _, action := range Actions {
		if mapping[action] == button {
			return action, true
		}
	}
	return "", false
}

func (mapping Mapping) Clone() Mapping {
	result := make(Mapping, len(mapping))
	for action, button := range mapping {
		result[action] = button
	}
	return result
}

// KeepKnown drops every entry that is not a bindable action. A mapping saved
// before quitting became a chord still carries its own exit button; loading it
// through this filter migrates the record instead of rejecting it, so an
// upgraded app does not force every user back through controller setup.
func (mapping Mapping) KeepKnown() Mapping {
	result := make(Mapping, len(Actions))
	for _, action := range Actions {
		if button, ok := mapping[action]; ok {
			result[action] = button
		}
	}
	return result
}

// Missing lists the actions a mapping does not bind, in the canonical order. A
// record saved by an older build is missing whatever the action set has grown
// since it was written, which is how a stale record is told apart from a
// damaged one.
func (mapping Mapping) Missing() []Action {
	var missing []Action
	for _, action := range Actions {
		if _, ok := mapping[action]; !ok {
			missing = append(missing, action)
		}
	}
	return missing
}

// ButtonLabel names the control carrying an action, whether it is an SDL
// GameController button or a keyboard binding on a desktop run.
func ButtonLabel(button int) string {
	labels := map[int]string{
		0: "A", 1: "B", 2: "X", 3: "Y",
		4: "BACK", 5: "GUIDE", 6: "START",
		7: "LEFT STICK", 8: "RIGHT STICK", 9: "LEFT SHOULDER", 10: "RIGHT SHOULDER",
		11: "DPAD UP", 12: "DPAD DOWN", 13: "DPAD LEFT", 14: "DPAD RIGHT",
	}
	if label := labels[button]; label != "" {
		return label
	}
	if label := keyboardLabels[button]; label != "" {
		return label
	}
	return fmt.Sprintf("BUTTON %d", button)
}

type DeviceProfile struct {
	Device     string
	Resolution string
	Fallback   Mapping
	Evidence   string
}

func Profile(device string) DeviceProfile {
	switch strings.ToLower(strings.TrimSpace(device)) {
	case "trimui-smart-pro":
		return DeviceProfile{Device: "trimui-smart-pro", Resolution: "1280x720", Fallback: AutoMapping(), Evidence: "real-device App Store navigation report"}
	case "magicx-zero-28":
		return DeviceProfile{Device: "magicx-zero-28", Resolution: "640x480", Evidence: "Knulli board and display source; raw controls unavailable"}
	default:
		return DeviceProfile{Device: strings.TrimSpace(device)}
	}
}

func SortedLabels(mapping Mapping) []string {
	labels := make([]string, 0, len(Actions))
	for _, action := range Actions {
		labels = append(labels, fmt.Sprintf("%s: %s", Verb(action), ButtonLabel(mapping[action])))
	}
	sort.Strings(labels)
	return labels
}
