package input

type Mode string

const (
	Normal  Mode = "normal"
	Blocked Mode = "blocked"
)

type Effect string

const (
	NoEffect          Effect = ""
	ExportDiagnostics Effect = "export-diagnostics"
)

type Identity struct {
	Device string
	GUID   string
	Name   string
}

// Session uses SDL's logical buttons, never raw joystick indices or saved
// custom mappings. Root is deliberately not used: old mapping files are left
// untouched, including malformed files from previous releases.
type Session struct {
	Device    string
	Identity  Identity
	Mapping   Mapping
	Source    string
	Mode      Mode
	Connected bool
	Message   string
	ChordHeld bool
}

func NewSession(_ string, device string) *Session {
	return &Session{Device: device, Mapping: AutoMapping(), Source: "no controller", Mode: Blocked, Message: "No SDL GameController is available"}
}

func NewDesktopSession(root, device string) *Session {
	session := NewSession(root, device)
	session.Mapping = KeyboardMapping()
	session.Identity = KeyboardIdentity(device)
	session.Source = KeyboardSourceName
	session.Connected = true
	session.Mode = Normal
	session.Message = ""
	return session
}

func (session *Session) Connect(identity Identity, knulliMapping bool) {
	identity.Device = session.Device
	session.Identity = identity
	session.Mapping = AutoMapping()
	session.Source = "SDL GameController mapping"
	if knulliMapping {
		session.Source = "Knulli SDL_GAMECONTROLLERCONFIG"
	}
	session.Connected = true
	session.ChordHeld = false
	session.Mode = Normal
	session.Message = ""
}

func (session *Session) Disconnect() {
	session.Connected = false
	session.Source = "no controller"
	session.Mode = Blocked
	session.ChordHeld = false
	session.Message = "No SDL GameController is available"
}

func (session *Session) SetChordAnchor(held bool) { session.ChordHeld = held }
func (session *Session) QuitChord() string        { return QuitChord(session.Mapping, session.Source) }

func (session *Session) HandleButton(button int) (Action, Effect) {
	if session.ChordHeld && button == session.Mapping[Diagnostics] {
		session.ChordHeld = false
		return Exit, NoEffect
	}
	action, ok := session.Mapping.Action(button)
	if !ok {
		return "", NoEffect
	}
	if action == Diagnostics || (session.Mode == Blocked && action == Confirm) {
		return "", ExportDiagnostics
	}
	if !session.Connected || session.Mode == Blocked {
		return "", NoEffect
	}
	return action, NoEffect
}

func (session *Session) ScreenMapping() Mapping { return session.Mapping }
func (session *Session) Binding(action Action) (int, bool) {
	code, ok := session.Mapping[action]
	return code, ok
}
