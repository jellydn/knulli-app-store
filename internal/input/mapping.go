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

// Required is the subset every mapping must bind: the actions that reach every
// screen and every package. Paging is deliberately outside it. A pad need not
// carry the button an optional action wants, and a user who cannot press one
// could not finish setup, so the pair is asked about rather than demanded.
var Required = []Action{Up, Down, Left, Right, Confirm, Back, Diagnostics}

// Optional is the subset a mapping may leave unbound. Skipping it costs the
// screenful jump and nothing else: up and down still move one row, a held
// direction still repeats, and the tab bar still narrows the list.
var Optional = []Action{PageUp, PageDown}

// Plan is the order one setup asks for actions: every required action, and the
// optional pair when the user asked for paging. Calibration walks a plan, so a
// setup that skipped paging never asks for a button the pad does not have.
func Plan(paging bool) []Action {
	if paging {
		return append([]Action(nil), Actions...)
	}
	plan := make([]Action, 0, len(Required))
	for _, action := range Actions {
		if !optional(action) {
			plan = append(plan, action)
		}
	}
	return plan
}

// optional reports whether an action may be left unbound.
func optional(action Action) bool {
	for _, candidate := range Optional {
		if candidate == action {
			return true
		}
	}
	return false
}

type Mapping map[Action]int

func AutoMapping() Mapping {
	return Mapping{
		Up: 11, Down: 12, Left: 13, Right: 14,
		PageUp: 9, PageDown: 10,
		Confirm: 0, Back: 1, Diagnostics: 3,
	}
}

// Validate reports whether a mapping can drive the app: every required action
// bound, no button carrying two actions, and nothing bound that is not an
// action of its own. An optional action may be absent, which is what lets a pad
// with no shoulder buttons keep a mapping it can actually use.
func (mapping Mapping) Validate() error {
	assigned := make(map[int]Action)
	for action, button := range mapping {
		if !bindable(action) {
			return fmt.Errorf("mapping contains unknown action %s", action)
		}
		if button < 0 {
			return fmt.Errorf("required action %s has no button", action)
		}
		if other, exists := assigned[button]; exists {
			return fmt.Errorf("button %d conflicts between %s and %s", button, other, action)
		}
		assigned[button] = action
	}
	for _, action := range Required {
		if _, ok := mapping[action]; !ok {
			return fmt.Errorf("required action %s has no button", action)
		}
	}
	return nil
}

// bindable reports whether an action belongs to the bindable set.
func bindable(action Action) bool {
	for _, candidate := range Actions {
		if candidate == action {
			return true
		}
	}
	return false
}

// Without returns the mapping with the given actions removed, so an optional
// binding the user never proved can be left out before the mapping is saved.
func (mapping Mapping) Without(actions ...Action) Mapping {
	result := mapping.Clone()
	for _, action := range actions {
		delete(result, action)
	}
	return result
}

// Restricted returns the mapping limited to a plan. It is how a detected layout
// becomes the mapping a setup saves: the detection reports every button the SDL
// layout knows, including the ones this pad may not carry, and the plan says
// which of them this setup asked for.
func (mapping Mapping) Restricted(plan []Action) Mapping {
	result := make(Mapping, len(plan))
	for _, action := range plan {
		if button, ok := mapping[action]; ok {
			result[action] = button
		}
	}
	return result
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

// Missing lists the required actions a mapping does not bind, in the canonical
// order. A record saved by an older build is missing whatever required action
// the set has grown since it was written, which is how a stale record is told
// apart from a damaged one. An absent optional action is not listed: a mapping
// without paging works, and reporting it as incomplete would send a pad with no
// shoulder buttons back through setup every launch.
func (mapping Mapping) Missing() []Action {
	var missing []Action
	for _, action := range Required {
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
