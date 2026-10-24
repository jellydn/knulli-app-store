package input

import (
	"fmt"
	"os"
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

var Actions = []Action{Up, Down, Left, Right, PageUp, PageDown, Confirm, Back, Diagnostics}

type Mapping map[Action]int

// AutoMapping consumes logical SDL GameController events unchanged. Knulli's
// default controller.py maps ES a/b/x/y to SDL b/a/y/x. Ports Xbox mode and
// desktop SDL use direct letters. Only bindings change, never SDL events.
func AutoMapping() Mapping {
	mapping := Mapping{Up: 11, Down: 12, Left: 13, Right: 14, PageUp: 9, PageDown: 10, Confirm: 0, Back: 1, Diagnostics: 3}
	if os.Getenv("STORE_CONTROLLER_LAYOUT") == "knulli" {
		mapping[Confirm], mapping[Back], mapping[Diagnostics] = 1, 0, 2
	}
	return mapping
}

func (mapping Mapping) Action(button int) (Action, bool) {
	for _, action := range Actions {
		if bound, ok := mapping[action]; ok && bound == button {
			return action, true
		}
	}
	return "", false
}

func ButtonLabel(button int) string {
	face := []string{"A", "B", "X", "Y"}
	if os.Getenv("STORE_CONTROLLER_LAYOUT") == "knulli" {
		face = []string{"B", "A", "Y", "X"}
	}
	if button >= 0 && button < len(face) {
		return face[button]
	}
	labels := map[int]string{4: "SELECT", 5: "GUIDE", 6: "START", 7: "LEFT STICK", 8: "RIGHT STICK", 9: "L1", 10: "R1", 11: "DPAD UP", 12: "DPAD DOWN", 13: "DPAD LEFT", 14: "DPAD RIGHT"}
	if label := labels[button]; label != "" {
		return label
	}
	if label := keyboardLabels[button]; label != "" {
		return label
	}
	return fmt.Sprintf("BUTTON %d", button)
}
