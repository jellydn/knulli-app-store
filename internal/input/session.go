package input

import "time"

type Mode string

const (
	Normal      Mode = "normal"
	Startup     Mode = "startup"
	Settings    Mode = "settings"
	Calibrating Mode = "calibration"
	Preview     Mode = "preview"
)

type Effect string

const (
	NoEffect          Effect = ""
	ExportDiagnostics Effect = "export-diagnostics"
)

var SettingsItems = []string{"SET UP CONTROLLER", "EXPORT DIAGNOSTICS", "RESET TO AUTO", "CLOSE SETTINGS"}

type Session struct {
	Store           Store
	Device          string
	Identity        Identity
	Mapping         Mapping
	Source          string
	AutoSource      string
	Mode            Mode
	Connected       bool
	SettingsIndex   int
	Calibration     *Calibration
	Deadline        time.Time
	Message         string
	ValidationError string
}

func NewSession(root, device string) *Session {
	return &Session{Store: NewStore(root), Device: device, Mapping: AutoMapping(), Source: "no controller", Mode: Normal}
}

func (session *Session) Connect(identity Identity, knulliMapping bool, now time.Time) {
	identity.Device = session.Device
	session.Identity = identity
	session.Connected = true
	session.Mapping = AutoMapping()
	session.Source = "SDL GameController auto mapping"
	if knulliMapping {
		session.Source = "Knulli SDL_GAMECONTROLLERCONFIG"
	}
	session.AutoSource = session.Source
	session.ValidationError = ""
	saved, found, err := session.Store.Load(identity)
	if err != nil {
		session.ValidationError = err.Error()
	} else if found {
		session.Mapping = saved
		session.Source = "saved calibration"
		session.Mode = Normal
		session.Message = "Saved controller mapping loaded"
		return
	}
	session.Mode = Startup
	session.Deadline = now.Add(8 * time.Second)
	session.Message = "Press any controller button to start setup, or wait for automatic mapping"
}

func (session *Session) Disconnect() {
	session.Connected = false
	session.Source = "no controller"
	session.Mode = Normal
	session.Calibration = nil
	session.Message = "Controller disconnected"
}

func (session *Session) Tick(now time.Time) {
	if session.Deadline.IsZero() || now.Before(session.Deadline) {
		return
	}
	switch session.Mode {
	case Startup:
		session.Mode = Normal
		session.Message = "Automatic controller mapping active"
	case Calibrating, Preview:
		session.Mode = Settings
		session.Calibration = nil
		session.Message = "Controller setup timed out; previous mapping kept"
	}
	session.Deadline = time.Time{}
}

func (session *Session) HandleButton(button int, now time.Time) (Action, Effect) {
	if !session.Connected {
		return "", NoEffect
	}
	switch session.Mode {
	case Startup:
		session.startCalibration(now)
		return "", NoEffect
	case Calibrating:
		current, _ := session.Calibration.Current()
		if current != Exit && button == session.Mapping[Exit] {
			session.cancelCalibration()
			return "", NoEffect
		}
		if err := session.Calibration.Assign(button); err == nil {
			session.ValidationError = ""
			session.Deadline = now.Add(20 * time.Second)
			if session.Calibration.Preview {
				session.Mode = Preview
				session.Deadline = now.Add(30 * time.Second)
				session.Message = "Test every mapped control to save"
			}
		} else {
			session.ValidationError = session.Calibration.Error
		}
		return "", NoEffect
	case Preview:
		if session.Calibration.Test(button) {
			if err := session.Store.Save(session.Identity, session.Calibration.Mapping); err != nil {
				session.ValidationError = err.Error()
				session.Mode = Settings
			} else {
				session.Mapping = session.Calibration.Mapping.Clone()
				session.Source = "saved calibration"
				session.Mode = Normal
				session.Message = "Controller mapping tested and saved"
			}
			session.Calibration = nil
			session.Deadline = time.Time{}
		}
		return "", NoEffect
	case Settings:
		return session.handleSettings(button, now)
	default:
		action, ok := session.Mapping.Action(button)
		if !ok {
			return "", NoEffect
		}
		if action == Diagnostics {
			session.Mode = Settings
			session.SettingsIndex = 0
			session.Message = "Controller settings"
			return "", NoEffect
		}
		return action, NoEffect
	}
}

func (session *Session) handleSettings(button int, now time.Time) (Action, Effect) {
	action, ok := session.Mapping.Action(button)
	if !ok {
		return "", NoEffect
	}
	switch action {
	case Up, Left:
		session.SettingsIndex = wrap(session.SettingsIndex-1, len(SettingsItems))
	case Down, Right:
		session.SettingsIndex = wrap(session.SettingsIndex+1, len(SettingsItems))
	case Back, Exit:
		session.Mode = Normal
	case Confirm:
		switch session.SettingsIndex {
		case 0:
			session.startCalibration(now)
		case 1:
			return "", ExportDiagnostics
		case 2:
			if err := session.Store.Reset(session.Identity); err != nil {
				session.ValidationError = err.Error()
			} else {
				session.Mapping = AutoMapping()
				session.Source = session.AutoSource
				session.Message = "Saved mapping removed; automatic mapping active"
			}
		case 3:
			session.Mode = Normal
		}
	}
	return "", NoEffect
}

func (session *Session) startCalibration(now time.Time) {
	session.Mode = Calibrating
	session.Calibration = NewCalibration()
	session.ValidationError = ""
	session.Deadline = now.Add(20 * time.Second)
	session.Message = "Assign each semantic action; the current Exit control cancels until the Exit step"
}

func (session *Session) cancelCalibration() {
	session.Mode = Settings
	session.Calibration = nil
	session.Deadline = time.Time{}
	session.Message = "Controller setup cancelled; previous mapping kept"
}

func wrap(value, length int) int {
	value %= length
	if value < 0 {
		value += length
	}
	return value
}
