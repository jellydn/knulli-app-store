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
	SetupItems = []string{"USE DETECTED MAPPING", "TEST DETECTED MAPPING", "CUSTOMIZE", "SAFE EXIT"}
	// ReviewItems offers the way out of an assignment review: the first item
	// confirms the assignment the way the Confirm action does, and the last
	// one abandons it the way the Back action does. Panel text is uppercase;
	// the footer shows the same actions as "Confirm (A)  Back (B)".
	ReviewItems   = []string{"CONFIRM", "RETRY", "START OVER", "BACK"}
	SettingsItems = []string{"SET UP CONTROLLER", "EXPORT DIAGNOSTICS", "RESET MAPPING", "CLOSE SETTINGS"}
)

// Session is the controller input state for one device and controller
// identity. Message reports what the last action did; a screen's own
// instruction lives in the GUI, so a message never restates it.
type Session struct {
	Store    Store
	Device   string
	Identity Identity
	// Detected is the mapping the input source reported before the user chose
	// one. A desktop run reports the keyboard; a controller reports SDL's.
	Detected        Mapping
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
	// ChordHeld is the quit chord's anchor: the input source's own Select key
	// (the pad's SELECT button, the keyboard's Tab) is down. It is held state,
	// not a binding, so no mapping can claim to be the way out.
	ChordHeld bool
}

// sourceGameController and sourceKnulli name the SDL sources a session detects.
const (
	sourceGameController = "SDL GameController mapping"
	sourceKnulli         = "Knulli SDL_GAMECONTROLLERCONFIG"
)

func NewSession(root, device string) *Session {
	return &Session{
		Store:    NewStore(root),
		Device:   device,
		Detected: AutoMapping(),
		Mapping:  AutoMapping(),
		Source:   "no controller",
		Mode:     Blocked,
		FirstRun: true,
		Message:  "No SDL GameController is available",
	}
}

// NewDesktopSession is the session a desktop run uses instead of a controller:
// the keyboard is the detected source, so every screen and every setup step is
// reachable without a physical device. It follows the same first-run rule as a
// controller, and its mapping is stored under its own identity.
func NewDesktopSession(root, device string) *Session {
	session := NewSession(root, device)
	session.Detected = KeyboardMapping()
	session.connect(KeyboardIdentity(device), KeyboardSourceName)
	return session
}

// Connect binds the session to a controller identity and the SDL source that
// reported its mapping.
func (session *Session) Connect(identity Identity, knulliMapping bool) {
	source := sourceGameController
	if knulliMapping {
		source = sourceKnulli
	}
	session.connect(identity, source)
}

// connect binds the session to an identity and the name of the source that
// reported it, then loads the mapping saved for that identity.
func (session *Session) connect(identity Identity, source string) {
	if source != KeyboardSourceName {
		identity.Device = session.Device
	}
	session.Identity = identity
	session.Connected = true
	session.ChordHeld = false
	session.Mapping = session.detected().Clone()
	session.Source = source
	session.AutoSource = source
	session.ValidationError = ""
	saved, found, loadErr := session.Store.Load(identity)
	if loadErr != nil {
		session.ValidationError = loadErr.Error()
	} else if found {
		session.Mapping = saved
		session.Source = "saved controller mapping"
		session.Message = "Saved controller mapping loaded"
		if source == KeyboardSourceName {
			session.Source = "saved keyboard mapping"
			session.Message = "Saved keyboard mapping loaded"
		}
		session.Mode = Normal
		session.FirstRun = false
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
	session.ChordHeld = false
	session.Message = "No SDL GameController is available"
}

func (session *Session) OpenSetup() {
	if !session.Connected {
		session.Mode = Blocked
		return
	}
	session.openSetup(false)
}

// SetChordAnchor records whether the input source's Select key is down. A
// release hands the trigger button back its own action, so a stuck anchor can
// never turn an ordinary press into a quit.
func (session *Session) SetChordAnchor(held bool) {
	session.ChordHeld = held
}

// QuitChord is the chord the current screen listens for. The footer shows it
// instead of a button label, because quitting is the one action no button
// carries on its own.
func (session *Session) QuitChord() string {
	return QuitChord(session.ScreenMapping(), session.Source)
}

func (session *Session) HandleButton(button int) (Action, Effect) {
	// The quit chord outranks every screen, calibration included: a held anchor
	// plus the trigger is a way out of anywhere.
	if session.ChordHeld && button == session.activeMapping()[Diagnostics] {
		session.ChordHeld = false
		return Exit, NoEffect
	}
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
			session.Message = ""
			return "", NoEffect
		}
		return action, NoEffect
	}
}

func (session *Session) handleBlocked(button int) (Action, Effect) {
	// Keyboard still reaches this path when no GameController exists. Confirm
	// exports diagnostics and the quit chord (checked before this screen) is
	// the way out; nothing here can end the session on its own. Catalogue
	// navigation stays closed.
	action, ok := session.activeMapping().Action(button)
	if !ok {
		return "", NoEffect
	}
	if action == Confirm {
		return "", ExportDiagnostics
	}
	return "", NoEffect
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
	case Back:
		if session.FirstRun {
			// A first run has nothing to go back to. Leaving is the quit chord
			// or the SAFE EXIT item, both deliberate choices.
			return "", NoEffect
		}
		session.Mode = Settings
	case Confirm:
		switch session.SetupIndex {
		case 0:
			if session.saveMapping(session.detected(), "saved detected mapping") {
				session.Message = "Detected mapping saved"
			}
		case 1:
			session.startPreview(session.detected(), "tested detected mapping")
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
		session.Message = ""
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
	case Back:
		session.cancelCalibration()
	case Confirm:
		switch session.ReviewIndex {
		case 0:
			if session.Calibration.Preview {
				session.Mode = Preview
				session.PendingSource = "saved custom mapping"
				session.Message = ""
			} else {
				session.Mode = Calibrating
				session.Message = ""
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
	case Back:
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
				session.Mapping = session.detected().Clone()
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
	session.Message = ""
}

func (session *Session) startCalibration() {
	session.Mode = Calibrating
	session.Calibration = NewCalibration()
	session.PendingSource = "saved custom mapping"
	session.ValidationError = ""
	session.Message = ""
}

func (session *Session) startPreview(mapping Mapping, source string) {
	session.Mode = Preview
	session.Calibration = NewPreview(mapping)
	session.PendingSource = source
	session.ValidationError = ""
	session.Message = ""
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
	session.Message = ""
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

// ScreenMapping is the mapping the current screen accepts, so a screen can
// label its controls without disagreeing with the buttons that work. The
// blocked and setup screens offer the detected mapping on a first run, the
// preview accepts only the mapping it is testing, and every other screen uses
// the session mapping.
func (session *Session) ScreenMapping() Mapping {
	switch session.Mode {
	case Blocked, Setup:
		return session.activeMapping()
	case Preview:
		if session.Calibration != nil {
			return session.Calibration.Mapping
		}
		return session.Mapping
	default:
		return session.Mapping
	}
}

// Binding returns the code that carries an action on the current screen. A
// caller translating a keyboard key into a binding uses this instead of reading
// a mapping directly, so a key reaches the same screen a controller would.
func (session *Session) Binding(action Action) (int, bool) {
	code, ok := session.ScreenMapping()[action]
	return code, ok
}

// DetectedMapping is the mapping the input source reported before the user
// chose one.
func (session *Session) DetectedMapping() Mapping {
	return session.detected()
}

func (session *Session) detected() Mapping {
	if session.Detected == nil {
		return AutoMapping()
	}
	return session.Detected
}

func (session *Session) activeMapping() Mapping {
	if session.FirstRun {
		return session.detected()
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
