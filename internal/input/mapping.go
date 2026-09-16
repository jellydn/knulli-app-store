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
	Confirm     Action = "confirm"
	Back        Action = "back"
	Diagnostics Action = "diagnostics"
	Exit        Action = "exit"
)

var Actions = []Action{Up, Down, Left, Right, Confirm, Back, Diagnostics, Exit}

type Mapping map[Action]int

func AutoMapping() Mapping {
	return Mapping{
		Up: 11, Down: 12, Left: 13, Right: 14,
		Confirm: 0, Back: 1, Diagnostics: 3, Exit: 6,
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

func ButtonLabel(button int) string {
	labels := map[int]string{
		0: "SDL A", 1: "SDL B", 2: "SDL X", 3: "SDL Y",
		4: "SDL BACK", 5: "SDL GUIDE", 6: "SDL START",
		7: "LEFT STICK", 8: "RIGHT STICK", 9: "LEFT SHOULDER", 10: "RIGHT SHOULDER",
		11: "DPAD UP", 12: "DPAD DOWN", 13: "DPAD LEFT", 14: "DPAD RIGHT",
	}
	if label := labels[button]; label != "" {
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
