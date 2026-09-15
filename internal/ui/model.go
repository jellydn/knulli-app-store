package ui

import (
	"context"
	"fmt"

	"github.com/jellydn/knulli-app-store/internal/appstore"
)

type Focus int

const (
	Browse Focus = iota
	Actions
	Confirm
	ForceConfirm
)

type Model struct {
	backend  appstore.Backend
	Items    []appstore.Item
	Selected int
	Action   int
	Focus    Focus
	Busy     bool
	Message  string
	Error    string
	events   chan operationEvent
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
	m.Items = items
	m.clamp()
	return nil
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

func (m *Model) Select(ctx context.Context) {
	if m.Busy || len(m.Items) == 0 {
		return
	}
	switch m.Focus {
	case Browse:
		if len(m.current().Actions) == 0 {
			m.Message = "No safe action is available"
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
	default:
		return true
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
					m.Items = event.items
					m.clamp()
				}
				if event.err != nil {
					m.Error = event.err.Error()
				} else {
					m.Error = ""
				}
			}
		default:
			return changed
		}
	}
}

func (m *Model) current() appstore.Item {
	if len(m.Items) == 0 {
		return appstore.Item{}
	}
	return m.Items[m.Selected]
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
