package ui

import (
	"context"
	"fmt"
	"time"

	"github.com/jellydn/knulli-app-store/internal/appstore"
)

type Focus int

const (
	Browse Focus = iota
	Health
	Actions
	Confirm
	ForceConfirm
)

type Model struct {
	backend appstore.Backend
	// Items is the list the catalogue shows: the packages the active tab
	// matches. catalogue keeps everything the index listed, so switching tabs
	// never reloads the index and never reorders what is already known.
	Items     []appstore.Item
	catalogue []appstore.Item
	Tabs      Tabs
	Toasts    Toasts
	Selected  int
	Action    int
	Focus     Focus
	Busy      bool
	Message   string
	Error     string
	events    chan operationEvent
}

type operationEvent struct {
	message string
	err     error
	items   []appstore.Item
	done    bool
}

func New(backend appstore.Backend) *Model {
	return &Model{backend: backend, events: make(chan operationEvent, 8)}
}

func (m *Model) Load(ctx context.Context) error {
	items, err := m.backend.Items(ctx)
	if err != nil {
		m.Error = err.Error()
		return err
	}
	m.setCatalogue(items)
	return nil
}

// setCatalogue records everything the index lists and shows what the active tab
// matches. The package the user was reading stays selected while it is still on
// screen, so a tab switch or a refresh after an operation never moves the
// selection out from under them.
func (m *Model) setCatalogue(items []appstore.Item) {
	keep := ""
	if item, ok := m.selectedItem(); ok {
		keep = item.Package.ID
	}
	m.catalogue = items
	m.filter(keep)
}

// Tab is the tab the catalogue is showing.
func (m *Model) Tab() Tab {
	return m.Tabs.Active()
}

// Total reports how many packages the index listed, before the active tab
// narrowed them, so a screen can tell an empty catalogue from an empty tab.
func (m *Model) Total() int {
	return len(m.catalogue)
}

// LiveToasts is what the notice bar says right now. Reading expires a notice
// whose moment has passed, so the bar empties itself between frames.
func (m *Model) LiveToasts() []string {
	return m.Toasts.Live(time.Now())
}

func (m *Model) Move(delta int) {
	if m.Busy || len(m.Items) == 0 {
		return
	}
	if m.Focus == Browse {
		m.Selected = wrap(m.Selected+delta, len(m.Items))
		m.Action = 0
		m.Error = ""
		return
	}
	actions := m.current().Actions
	if m.Focus == Actions && len(actions) > 0 {
		m.Action = wrap(m.Action+delta, len(actions))
	}
}

// Window returns the first and last row of the list the catalogue shows for a
// given number of rows. The window follows the selection and stops at the end
// of the list, so a short list shows no empty rows and the selection is always
// one of the rows shown. The row count belongs to the renderer, which is why it
// arrives as an argument instead of living here.
func (m *Model) Window(rows int) (int, int) {
	total := len(m.Items)
	if rows <= 0 || total == 0 {
		return 0, 0
	}
	if rows > total {
		rows = total
	}
	start := m.Selected - rows/2
	if start < 0 {
		start = 0
	}
	if start+rows > total {
		start = total - rows
	}
	return start, start + rows
}

// Horizontal moves sideways on the current screen. On the catalogue, sideways
// steps to the previous or next tab: the bar is the row under the header, and
// left and right walk it while up and down walk the rows, so a direction key
// never means two things at once. Inside a panel sideways still steps that
// panel's own buttons, because there is no tab bar there to walk.
func (m *Model) Horizontal(delta int) {
	if m.Busy {
		return
	}
	if m.Focus != Browse {
		m.Move(delta)
		return
	}
	keep := ""
	if item, ok := m.selectedItem(); ok {
		keep = item.Package.ID
	}
	m.Tabs.Cycle(delta)
	m.filter(keep)
	m.Error = ""
}

// filter shows the packages the active tab matches. A model that has never
// loaded a catalogue has nothing to narrow, so its rows are left alone: a tab
// can only hide packages the index actually listed.
func (m *Model) filter(keep string) {
	if m.catalogue == nil {
		m.clamp()
		return
	}
	visible := make([]appstore.Item, 0, len(m.catalogue))
	for _, item := range m.catalogue {
		if m.Tabs.Active().Match(item) {
			visible = append(visible, item)
		}
	}
	m.Items = visible
	m.Action = 0
	m.Selected = 0
	if keep != "" {
		for index, item := range visible {
			if item.Package.ID == keep {
				m.Selected = index
				break
			}
		}
	}
	m.clamp()
}

// Page moves the selection by a screenful and clamps at both ends: a page past
// the end of the list stops on the last row rather than wrapping to the first,
// because a reader who holds a shoulder button expects to arrive at the end.
// Paging is defined in the catalogue only; there is no second screenful of
// action buttons to move through.
func (m *Model) Page(delta, rows int) {
	if m.Busy || len(m.Items) == 0 || m.Focus != Browse || rows < 1 {
		return
	}
	next := m.Selected + delta*rows
	if next < 0 {
		next = 0
	}
	if next > len(m.Items)-1 {
		next = len(m.Items) - 1
	}
	m.Selected = next
	m.Action = 0
	m.Error = ""
}

func (m *Model) Select(ctx context.Context) {
	if m.Busy || len(m.Items) == 0 {
		return
	}
	switch m.Focus {
	case Browse:
		if m.current().HealthReason != "" {
			m.Focus = Health
			return
		}
		if len(m.current().Actions) == 0 {
			m.Message = "No safe action is available"
			return
		}
		m.Focus = Actions
	case Health:
		if len(m.current().Actions) == 0 {
			return
		}
		m.Focus = Actions
	case Actions:
		m.Focus = Confirm
	case Confirm:
		if m.selectedAction() == appstore.ForceReinstall {
			m.Focus = ForceConfirm
			return
		}
		m.start(ctx)
	case ForceConfirm:
		m.start(ctx)
	}
}

func (m *Model) Back() bool {
	if m.Busy {
		return false
	}
	switch m.Focus {
	case ForceConfirm:
		m.Focus = Confirm
	case Confirm:
		m.Focus = Actions
	case Actions:
		m.Focus = Browse
	case Health:
		m.Focus = Browse
	default:
		// Leaving the app belongs to the quit chord, never to a single Back
		// press. With nothing left to step back to, Back does nothing.
		return false
	}
	m.Message = ""
	m.Error = ""
	return false
}

func (m *Model) ExportDiagnostics(ctx context.Context) {
	if m.Busy {
		return
	}
	path, err := m.backend.ExportDiagnostics(ctx)
	if err != nil {
		m.Error = "Export diagnostics: " + err.Error()
		return
	}
	m.Error = ""
	m.Message = "Diagnostics saved to " + path
}

func (m *Model) Poll() bool {
	changed := false
	for {
		select {
		case event := <-m.events:
			changed = true
			if event.message != "" {
				m.Message = event.message
			}
			if event.done {
				m.Busy = false
				m.Focus = Browse
				if len(event.items) > 0 {
					m.setCatalogue(event.items)
				}
				if event.err != nil {
					// A failure is a state, not a notice: it keeps the status
					// block until the user answers it.
					m.Error = event.err.Error()
				} else {
					// A completion is a notice. It says what just happened and
					// then clears itself, so a finished operation does not leave
					// the catalogue wearing a banner nothing will remove.
					m.Error = ""
					m.Message = ""
					m.Toasts.Push(event.message, time.Now())
				}
			}
		default:
			return changed
		}
	}
}

func (m *Model) current() appstore.Item {
	item, _ := m.selectedItem()
	return item
}

// selectedItem is the package under the cursor, if the visible list has one.
func (m *Model) selectedItem() (appstore.Item, bool) {
	if m.Selected < 0 || m.Selected >= len(m.Items) {
		return appstore.Item{}, false
	}
	return m.Items[m.Selected], true
}

func (m *Model) start(ctx context.Context) {
	item := m.current()
	if len(item.Actions) == 0 || m.Action >= len(item.Actions) {
		return
	}
	action := item.Actions[m.Action]
	m.Busy = true
	m.Error = ""
	m.Message = "Starting " + string(action)
	go func() {
		completion := actionName(action) + " completed"
		err := m.backend.Execute(ctx, item.Package.ID, action, func(message string) {
			completion = message
			select {
			case m.events <- operationEvent{message: message}:
			case <-ctx.Done():
			}
		})
		operationErr := err
		items, refreshErr := m.backend.Items(ctx)
		if operationErr == nil && refreshErr != nil {
			err = refreshErr
		} else {
			err = operationErr
		}
		message := completion
		if err != nil {
			message = fmt.Sprintf("%s failed", actionName(action))
		}
		select {
		case m.events <- operationEvent{message: message, err: err, items: items, done: true}:
		case <-ctx.Done():
		}
	}()
}

func (m *Model) selectedAction() appstore.Action {
	item := m.current()
	if m.Action < 0 || m.Action >= len(item.Actions) {
		return ""
	}
	return item.Actions[m.Action]
}

func (m *Model) clamp() {
	if len(m.Items) == 0 {
		m.Selected = 0
		m.Action = 0
		return
	}
	if m.Selected >= len(m.Items) {
		m.Selected = len(m.Items) - 1
	}
	if m.Selected < 0 {
		m.Selected = 0
	}
	if m.Action >= len(m.current().Actions) {
		m.Action = 0
	}
}

func wrap(value, length int) int {
	if length == 0 {
		return 0
	}
	value %= length
	if value < 0 {
		value += length
	}
	return value
}

func actionName(action appstore.Action) string {
	value := string(action)
	if value == "" {
		return "Action"
	}
	return string(value[0]-'a'+'A') + value[1:]
}
