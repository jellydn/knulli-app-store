package input

import "fmt"

type Calibration struct {
	Mapping Mapping
	Index   int
	Preview bool
	Tested  map[Action]bool
	Error   string
}

func NewCalibration() *Calibration {
	return &Calibration{Mapping: make(Mapping), Tested: make(map[Action]bool)}
}

func NewPreview(mapping Mapping) *Calibration {
	return &Calibration{Mapping: mapping.Clone(), Preview: true, Tested: make(map[Action]bool)}
}

func (calibration *Calibration) Current() (Action, bool) {
	if calibration.Preview || calibration.Index >= len(Actions) {
		return "", false
	}
	return Actions[calibration.Index], true
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
	if calibration.Index == len(Actions) {
		if err := calibration.Mapping.Validate(); err != nil {
			calibration.Error = err.Error()
			return err
		}
		calibration.Preview = true
	}
	return nil
}

func (calibration *Calibration) Test(button int) bool {
	action, ok := calibration.Mapping.Action(button)
	if !ok || !calibration.Preview {
		return false
	}
	calibration.Tested[action] = true
	return len(calibration.Tested) == len(Actions)
}
