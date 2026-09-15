package input

type Mode string

const (
	Normal      Mode = "normal"
	Setup       Mode = "setup"
	Blocked     Mode = "blocked"
	Settings    Mode = "settings"
	Calibrating Mode = "calibration"
	Review      Mode = "assignment-review"
	Preview     Mode = "preview"
)

type Effect string

const (
	NoEffect          Effect = ""
	ExportDiagnostics Effect = "export-diagnostics"
)

var (
	SetupItems    = []string{"USE DETECTED MAPPING", "TEST DETECTED MAPPING", "CUSTOMIZE", "SAFE EXIT"}
	ReviewItems   = []string{"ACCEPT", "RETRY", "START OVER", "CANCEL"}
	SettingsItems = []string{"SET UP CONTROLLER", "EXPORT DIAGNOSTICS", "RESET MAPPING", "CLOSE SETTINGS"}
)

type Session struct {
	Store           Store
	Device          string
	Identity        Identity
	Mapping         Mapping
	Source          string
	AutoSource      string
	Mode            Mode
	Connected       bool
	FirstRun        bool
	SetupIndex      int
	ReviewIndex     int
	SettingsIndex   int
	Calibration     *Calibration
	PendingSource   string
	Message         string
	ValidationError string
}

func NewSession(root, device string) *Session {
	return &Session{
		Store:    NewStore(root),
		Device:   device,
		Mapping:  AutoMapping(),
		Source:   "no controller",
		Mode:     Blocked,
		FirstRun: true,
		Message:  "No SDL GameController is available",
	}
}

func (session *Session) Connect(identity Identity, knulliMapping bool) {
	identity.Device = session.Device
	session.Identity = identity
	session.Connected = true
	session.Mapping = AutoMapping()
	session.Source = "SDL GameController mapping"
	if knulliMapping {
		session.Source = "Knulli SDL_GAMECONTROLLERCONFIG"
	}
	session.AutoSource = session.Source
	session.ValidationError = ""
	saved, found, loadErr := session.Store.Load(identity)
	if loadErr != nil {
		session.ValidationError = loadErr.Error()
	} else if found {
		session.Mapping = saved
		session.Source = "saved controller mapping"
		session.Mode = Normal
		session.FirstRun = false
		session.Message = "Saved controller mapping loaded"
		return
	}
	session.openSetup(true)
	if loadErr != nil {
		session.ValidationError = loadErr.Error()
	}
}

func (session *Session) Disconnect() {
	session.Connected = false
	session.Source = "no controller"
	session.Mode = Blocked
	session.Calibration = nil
	session.FirstRun = true
	session.Message = "No SDL GameController is available"
}

func (session *Session) OpenSetup() {
	if !session.Connected {
		session.Mode = Blocked
		return
	}
	session.openSetup(false)
}

func (session *Session) HandleButton(button int) (Action, Effect) {
	if !session.Connected || session.Mode == Blocked {
		return session.handleBlocked(button)
	}
	switch session.Mode {
	case Setup:
		return session.handleSetup(button)
	case Calibrating:
		return session.handleCalibration(button)
	case Review:
		return session.handleReview(button)
	case Preview:
		return session.handlePreview(button)
	case Settings:
		return session.handleSettings(button)
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

func (session *Session) handleBlocked(button int) (Action, Effect) {
	// Keyboard still reaches this path when no GameController exists.
	// Confirm exports diagnostics. Back and Exit leave. Catalogue navigation stays closed.
	action, ok := session.activeMapping().Action(button)
	if !ok {
		return "", NoEffect
	}
	switch action {
	case Confirm:
		return "", ExportDiagnostics
	case Back, Exit:
		return Exit, NoEffect
	default:
		return "", NoEffect
	}
}

func (session *Session) handleSetup(button int) (Action, Effect) {
	controls := session.activeMapping()
	action, ok := controls.Action(button)
	if !ok {
		return "", NoEffect
	}
	switch action {
	case Up, Left:
		session.SetupIndex = wrap(session.SetupIndex-1, len(SetupItems))
	case Down, Right:
		session.SetupIndex = wrap(session.SetupIndex+1, len(SetupItems))
	case Back, Exit:
		if session.FirstRun {
			return Exit, NoEffect
		}
		session.Mode = Settings
	case Confirm:
		switch session.SetupIndex {
		case 0:
			if session.saveMapping(AutoMapping(), "saved detected mapping") {
				session.Message = "Detected mapping saved"
			}
		case 1:
			session.startPreview(AutoMapping(), "tested detected mapping")
		case 2:
			session.startCalibration()
		case 3:
			return Exit, NoEffect
		}
	}
	return "", NoEffect
}

func (session *Session) handleCalibration(button int) (Action, Effect) {
	if err := session.Calibration.Assign(button); err == nil {
		session.ValidationError = ""
		session.Mode = Review
		session.ReviewIndex = 0
		session.Message = "Detected physical control: " + ButtonLabel(button)
	} else {
		session.ValidationError = session.Calibration.Error
		session.Message = "Retry with a different physical button"
	}
	return "", NoEffect
}

func (session *Session) handleReview(button int) (Action, Effect) {
	action, ok := session.Mapping.Action(button)
	if !ok {
		return "", NoEffect
	}
	switch action {
	case Up, Left:
		session.ReviewIndex = wrap(session.ReviewIndex-1, len(ReviewItems))
	case Down, Right:
		session.ReviewIndex = wrap(session.ReviewIndex+1, len(ReviewItems))
	case Back, Exit:
		session.cancelCalibration()
	case Confirm:
		switch session.ReviewIndex {
		case 0:
			if session.Calibration.Preview {
				session.Mode = Preview
				session.PendingSource = "saved custom mapping"
				session.Message = "Test every mapped action before save"
			} else {
				session.Mode = Calibrating
				session.Message = "Press one physical button for the shown action"
			}
		case 1:
			session.retryAssignment()
		case 2:
			session.startCalibration()
			session.Message = "Controller setup started over"
		case 3:
			session.cancelCalibration()
		}
	}
	return "", NoEffect
}

func (session *Session) handlePreview(button int) (Action, Effect) {
	if session.Calibration.Test(button) {
		if session.saveMapping(session.Calibration.Mapping, session.PendingSource) {
			session.Message = "Controller mapping tested and saved"
		}
		session.Calibration = nil
	}
	return "", NoEffect
}

func (session *Session) handleSettings(button int) (Action, Effect) {
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
			session.openSetup(false)
		case 1:
			return "", ExportDiagnostics
		case 2:
			if err := session.Store.Reset(session.Identity); err != nil {
				session.ValidationError = err.Error()
			} else {
				session.Mapping = AutoMapping()
				session.Source = session.AutoSource
				session.openSetup(true)
				session.Message = "Saved mapping removed; choose and test a mapping"
			}
		case 3:
			session.Mode = Normal
		}
	}
	return "", NoEffect
}

func (session *Session) openSetup(firstRun bool) {
	session.Mode = Setup
	session.FirstRun = firstRun
	session.SetupIndex = 0
	session.Calibration = nil
	session.ValidationError = ""
	session.Message = "Choose how to configure this controller"
}

func (session *Session) startCalibration() {
	session.Mode = Calibrating
	session.Calibration = NewCalibration()
	session.PendingSource = "saved custom mapping"
	session.ValidationError = ""
	session.Message = "Press one physical button for the shown action"
}

func (session *Session) startPreview(mapping Mapping, source string) {
	session.Mode = Preview
	session.Calibration = NewPreview(mapping)
	session.PendingSource = source
	session.ValidationError = ""
	session.Message = "Test every mapped action before save"
}

func (session *Session) retryAssignment() {
	if session.Calibration.Index > 0 {
		session.Calibration.Index--
		action := Actions[session.Calibration.Index]
		delete(session.Calibration.Mapping, action)
		session.Calibration.Preview = false
	}
	session.Mode = Calibrating
	session.ValidationError = ""
	session.Message = "Retry: press a different physical button"
}

func (session *Session) cancelCalibration() {
	session.Calibration = nil
	session.ValidationError = ""
	if session.FirstRun {
		session.Mode = Setup
	} else {
		session.Mode = Settings
	}
	session.Message = "Controller setup cancelled; previous mapping kept"
}

func (session *Session) saveMapping(mapping Mapping, source string) bool {
	if err := session.Store.Save(session.Identity, mapping); err != nil {
		session.ValidationError = err.Error()
		session.Mode = Setup
		return false
	}
	session.Mapping = mapping.Clone()
	session.Source = source
	session.Mode = Normal
	session.FirstRun = false
	session.ValidationError = ""
	return true
}

func (session *Session) activeMapping() Mapping {
	if session.FirstRun {
		return AutoMapping()
	}
	return session.Mapping
}

func wrap(value, length int) int {
	value %= length
	if value < 0 {
		value += length
	}
	return value
}
