package input

import (
	"fmt"
	"strings"
)

type Calibration struct {
	Mapping Mapping
	// Actions is what this calibration asks for, in order: the plan the setup
	// chose, so a setup that skipped paging is never asked for a button the pad
	// does not carry.
	Actions []Action
	Index   int
	Preview bool
	Tested  map[Action]bool
	Error   string
}

// NewCalibration starts an assignment walk over a plan: every required action,
// and the optional pair when the user asked for paging.
func NewCalibration(paging bool) *Calibration {
	return &Calibration{Mapping: make(Mapping), Actions: Plan(paging), Tested: make(map[Action]bool)}
}

func NewPreview(mapping Mapping) *Calibration {
	return &Calibration{Mapping: mapping.Clone(), Preview: true, Tested: make(map[Action]bool)}
}

func (calibration *Calibration) Current() (Action, bool) {
	if calibration.Preview || calibration.Index >= len(calibration.Actions) {
		return "", false
	}
	return calibration.Actions[calibration.Index], true
}

func (calibration *Calibration) Assign(button int) error {
	action, ok := calibration.Current()
	if !ok {
		return fmt.Errorf("calibration is not accepting assignments")
	}
	for assignedAction, assignedButton := range calibration.Mapping {
		if assignedButton == button {
			calibration.Error = fmt.Sprintf("%s is already assigned to %s; try another button", ButtonLabel(button), ActionLabel(assignedAction))
			return fmt.Errorf("button conflict")
		}
	}
	calibration.Mapping[action] = button
	calibration.Index++
	calibration.Error = ""
	if calibration.Index == len(calibration.Actions) {
		if err := calibration.Mapping.Validate(); err != nil {
			calibration.Error = err.Error()
			return err
		}
		calibration.Preview = true
	}
	return nil
}

// Test records that the user pressed a button bound to an action. The preview is
// finished once every required action has been pressed: an optional action does
// not hold it open, because a pad that does not carry the button could not get
// there, and the user already decided the pair was worth asking for. Anything
// optional that was never pressed is dropped here, so what gets saved is only
// what this pad proved it has.
func (calibration *Calibration) Test(button int) bool {
	action, ok := calibration.Mapping.Action(button)
	if !ok || !calibration.Preview {
		return false
	}
	calibration.Tested[action] = true
	if calibration.RequiredTested() < len(Required) {
		return false
	}
	calibration.Mapping = calibration.Mapping.Without(calibration.Untested()...)
	return true
}

// RequiredTested counts how many required actions the preview has seen, which
// is the number its progress line reports.
func (calibration *Calibration) RequiredTested() int {
	count := 0
	for _, action := range Required {
		if calibration.Tested[action] {
			count++
		}
	}
	return count
}

// Untested lists the optional actions the preview binds that the user never
// pressed, in the canonical order.
func (calibration *Calibration) Untested() []Action {
	var untested []Action
	for _, action := range Optional {
		if _, bound := calibration.Mapping[action]; bound && !calibration.Tested[action] {
			untested = append(untested, action)
		}
	}
	return untested
}

// UntestedLabels names the optional actions a preview is about to leave out, so
// a screen can say what saving now will cost. Names come from the canonical
// label table, so the line reads exactly like the mapping summary above it.
func UntestedLabels(calibration *Calibration) string {
	if calibration == nil {
		return ""
	}
	labels := make([]string, 0, len(Optional))
	for _, action := range calibration.Untested() {
		labels = append(labels, ActionLabel(action))
	}
	return strings.Join(labels, ", ")
}
