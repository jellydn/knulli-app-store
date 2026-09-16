package sdlui

// Scripted keyboard walkthrough, used to verify every GUI flow on a machine
// with no device. The SDL adapter owns only the injection; the sequence, the
// screen labels and the evidence file names live here so they compile and test
// without cgo.

import (
	"fmt"
	"strings"

	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

// keySpecs names each portable key as the -keys flag writes it. The names match
// the documented desktop bindings.
var keySpecs = map[string]Key{
	"up":    KeyUp,
	"down":  KeyDown,
	"left":  KeyLeft,
	"right": KeyRight,
	"enter": KeyConfirm,
	"esc":   KeyBack,
	"y":     KeyDiagnostics,
	"q":     KeyExit,
}

// KeyName is the -keys spelling of a key, and the spelling used in an evidence
// file name.
func KeyName(key Key) string {
	for name, candidate := range keySpecs {
		if candidate == key {
			return name
		}
	}
	return "none"
}

// ParseKeys reads a comma-separated key sequence. An empty specification is an
// empty sequence, which is how a run without a walkthrough is described.
func ParseKeys(specification string) ([]Key, error) {
	specification = strings.TrimSpace(specification)
	if specification == "" {
		return nil, nil
	}
	fields := strings.Split(specification, ",")
	keys := make([]Key, 0, len(fields))
	for _, field := range fields {
		name := strings.ToLower(strings.TrimSpace(field))
		key, ok := keySpecs[name]
		if !ok {
			return nil, fmt.Errorf("unknown key %q: use %s", field, strings.Join(KeyNames(), ", "))
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// KeyNames lists every accepted key spelling, in a stable order.
func KeyNames() []string {
	return []string{"up", "down", "left", "right", "enter", "esc", "y", "q"}
}

// WalkState names the screen a rendered frame shows. The input session owns the
// controller screens and the UI model owns the catalogue screens, so a frame is
// named from both.
func WalkState(model *storeui.Model, controls *storeinput.Session) string {
	if controls != nil && controls.Mode != storeinput.Normal {
		return string(controls.Mode)
	}
	if model != nil && model.Busy {
		return "progress"
	}
	if model == nil {
		return "unknown"
	}
	switch model.Focus {
	case storeui.Health:
		return "health"
	case storeui.Actions:
		return "actions"
	case storeui.Confirm:
		return "confirm"
	case storeui.ForceConfirm:
		return "force-confirm"
	}
	return "catalogue"
}

// WalkDetail is the evidence line recorded beside a frame: what the state was
// when the frame was captured. Scripts read this instead of the pixels.
func WalkDetail(model *storeui.Model, controls *storeinput.Session) string {
	item := ""
	if model != nil && model.Selected >= 0 && model.Selected < len(model.Items) {
		item = model.Items[model.Selected].Package.ID
	}
	focus := "none"
	action := ""
	busy := "false"
	message := ""
	problem := ""
	sessionMessage := ""
	if controls != nil {
		sessionMessage = controls.Message
	}
	if model != nil {
		focus = walkFocusName(model.Focus)
		busy = fmt.Sprint(model.Busy)
		message = model.Message
		problem = model.Error
	}
	// A selected item is the only case where the action index means anything,
	// and the item guard is what makes the index safe to read.
	if item != "" && model.Action >= 0 && model.Action < len(model.Items[model.Selected].Actions) {
		action = string(model.Items[model.Selected].Actions[model.Action])
	}
	mode := "none"
	if controls != nil {
		mode = string(controls.Mode)
	}
	fields := []string{item, focus, action, busy, mode, sessionMessage, message, problem}
	for index := range fields {
		fields[index] = walkField(fields[index])
	}
	return strings.Join(fields, "\t")
}

// walkField keeps one value inside one TSV field even when an operation error
// contains a line break or tab.
func walkField(value string) string {
	return strings.NewReplacer("\t", " ", "\r", " ", "\n", " ").Replace(value)
}

func walkFocusName(focus storeui.Focus) string {
	switch focus {
	case storeui.Health:
		return "health"
	case storeui.Actions:
		return "actions"
	case storeui.Confirm:
		return "confirm"
	case storeui.ForceConfirm:
		return "force-confirm"
	default:
		return "browse"
	}
}

// WalkFile names the frame captured at a step. The step number is padded so a
// directory listing follows the walkthrough order.
func WalkFile(step int, key Key, state string) string {
	stepName := "start"
	if step > 0 {
		stepName = KeyName(key)
	}
	return fmt.Sprintf("%03d-%s-%s.png", step, stepName, state)
}

// WalkHeader names the columns of the walkthrough record. A script reads this
// record instead of the pixels: it says which screen each key reached.
const WalkHeader = "file\tstate\tkey\titem\tfocus\taction\tbusy\tmode\tsession_message\tmessage\terror\n"

// WalkShot is one piece of walkthrough evidence.
type WalkShot struct {
	File  string
	State string
	Key   Key
}

// WalkRecord is the record line for one captured frame.
func WalkRecord(shot WalkShot, detail string) string {
	return strings.Join([]string{shot.File, shot.State, KeyName(shot.Key), detail}, "\t") + "\n"
}

// WalkRunner drives a scripted walkthrough one frame at a time. It captures the
// opening frame, the screen each key produced, and the screen an operation
// finished on, then stops once the sequence has run out and the last screen has
// settled.
type WalkRunner struct {
	walker         *Walker
	step           int
	settle         int
	idle           int
	pendingCapture bool
	wasBusy        bool
}

// NewWalkRunner starts a walkthrough. A nil runner means the run is
// interactive. settleFrames is how many idle frames to render after the last
// key before the run ends.
func NewWalkRunner(keys []Key, settleFrames int) *WalkRunner {
	walker := NewWalker(keys)
	if walker == nil {
		return nil
	}
	if settleFrames < 1 {
		settleFrames = 1
	}
	return &WalkRunner{walker: walker, settle: settleFrames, pendingCapture: true}
}

// Capture reports the evidence for this frame, if this frame is evidence. It
// records the screen a key produced and the screen an operation finished on.
func (runner *WalkRunner) Capture(state string, busy bool) (WalkShot, bool) {
	if runner == nil {
		return WalkShot{}, false
	}
	finished := runner.wasBusy && !busy
	runner.wasBusy = busy
	if !runner.pendingCapture && !finished {
		return WalkShot{}, false
	}
	runner.pendingCapture = false
	shot := WalkShot{File: WalkFile(runner.step, runner.walker.Last(), state), State: state, Key: runner.walker.Last()}
	runner.step++
	return shot, true
}

// Step reports the next key to send. A key waits for any operation in front of
// it, so a walkthrough never overtakes the work it started.
func (runner *WalkRunner) Step(busy bool) (Key, bool) {
	if runner == nil || busy {
		return KeyNone, false
	}
	key, ok := runner.walker.Pending()
	if !ok {
		return KeyNone, false
	}
	runner.walker.Advance()
	runner.pendingCapture = true
	runner.idle = 0
	return key, true
}

// Done reports that every key has been sent and the last screen has settled.
func (runner *WalkRunner) Done(busy bool) bool {
	if runner == nil {
		return false
	}
	if !runner.walker.Done() || busy {
		runner.idle = 0
		return false
	}
	runner.idle++
	return runner.idle >= runner.settle
}

// Sent counts the keys sent so far, so a stuck walkthrough can report how far
// it got.
func (runner *WalkRunner) Sent() int {
	if runner == nil {
		return 0
	}
	return runner.walker.Sent()
}

// Steps counts the frames captured so far.
func (runner *WalkRunner) Steps() int {
	if runner == nil {
		return 0
	}
	return runner.step
}

// Walker steps a scripted key sequence. It never sends a key while an operation
// is in progress, so a download or a diagnostic export finishes before the next
// key, which keeps a walkthrough reproducible.
type Walker struct {
	keys  []Key
	index int
	sent  int
	last  Key
}

// NewWalker starts a walkthrough over a key sequence. A nil walker means the
// run is interactive.
func NewWalker(keys []Key) *Walker {
	if len(keys) == 0 {
		return nil
	}
	return &Walker{keys: keys, last: KeyNone}
}

// Pending reports the next key without consuming it.
func (walker *Walker) Pending() (Key, bool) {
	if walker == nil || walker.index >= len(walker.keys) {
		return KeyNone, false
	}
	return walker.keys[walker.index], true
}

// Advance consumes the pending key and remembers it for the evidence name.
func (walker *Walker) Advance() {
	if walker == nil || walker.index >= len(walker.keys) {
		return
	}
	walker.last = walker.keys[walker.index]
	walker.index++
	walker.sent++
}

// Done reports that every key has been sent.
func (walker *Walker) Done() bool {
	return walker == nil || walker.index >= len(walker.keys)
}

// Last is the key that produced the current screen.
func (walker *Walker) Last() Key {
	if walker == nil {
		return KeyNone
	}
	return walker.last
}

// Sent counts the keys consumed so far.
func (walker *Walker) Sent() int {
	if walker == nil {
		return 0
	}
	return walker.sent
}
