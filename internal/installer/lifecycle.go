package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jellydn/knulli-app-store/internal/safefs"
)

const lifecycleSchemaV1 = "org.knulli.app-store/lifecycle-state/v1"

// LifecycleState records the last operation requested for a package, so a retry
// can stay bound to the operation that failed instead of becoming a different
// one, and so a fresh install failure is never retried as adoption.
type LifecycleState struct {
	Schema              string `json:"schema"`
	PackageID           string `json:"package_id"`
	RequestedOperation  string `json:"requested_operation"`
	DetectedInstallType string `json:"detected_install_type"`
	RetryTarget         string `json:"retry_target"`
	Failure             string `json:"failure,omitempty"`
	ForceAllowed        bool   `json:"force_allowed,omitempty"`
}

// lifecycleStore owns the lifecycle record: its schema, its path below manager
// state, and its atomic write. It is deliberately not a method set on Manager,
// because the record is a small, separately testable store: every retry rule
// added to the manager used to widen the manager's own surface for a file it
// only ever read and wrote.
type lifecycleStore struct {
	guard *safefs.Guard
}

// newLifecycleStore opens manager state under its own guard. Manager state is
// the only path this store may touch, whatever paths a package declares.
func newLifecycleStore(root string) (lifecycleStore, error) {
	guard, err := safefs.NewGuard(root, []string{managerPath})
	if err != nil {
		return lifecycleStore{}, err
	}
	return lifecycleStore{guard: guard}, nil
}

// Load reads the record for a package. A package with no record reports no
// context to retry rather than an error, and a record that names a different
// package or schema is refused instead of being trusted.
func (store lifecycleStore) Load(id string) (*LifecycleState, error) {
	host, err := store.guard.Resolve(lifecyclePath(id))
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(host)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state LifecycleState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("read lifecycle state: %w", err)
	}
	if state.Schema != lifecycleSchemaV1 || state.PackageID != id {
		return nil, fmt.Errorf("lifecycle state has an invalid schema or package id")
	}
	return &state, nil
}

// Save writes the record, stamping the schema so a caller cannot forget it.
func (store lifecycleStore) Save(state LifecycleState) error {
	host, err := store.guard.Resolve(lifecyclePath(state.PackageID))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(host), 0700); err != nil {
		return err
	}
	state.Schema = lifecycleSchemaV1
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return safefs.AtomicWrite(host, append(data, '\n'), 0600)
}

// Clear removes the record of a finished operation. An absent record is already
// clear, so clearing twice is not an error.
func (store lifecycleStore) Clear(id string) error {
	host, err := store.guard.Resolve(lifecyclePath(id))
	if err != nil {
		return err
	}
	if err := os.Remove(host); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// LifecycleState reads the retry context recorded for a package.
func (m Manager) LifecycleState(id string) (*LifecycleState, error) {
	store, err := newLifecycleStore(m.root())
	if err != nil {
		return nil, err
	}
	return store.Load(id)
}

// RecordLifecycle records the requested operation and the retry target that
// stays safe if it fails.
func (m Manager) RecordLifecycle(state LifecycleState) error {
	store, err := newLifecycleStore(m.root())
	if err != nil {
		return err
	}
	return store.Save(state)
}

// ClearLifecycle drops the retry context of a finished operation.
func (m Manager) ClearLifecycle(id string) error {
	store, err := newLifecycleStore(m.root())
	if err != nil {
		return err
	}
	return store.Clear(id)
}

// LifecycleEvent logs the recorded retry context beside the health reason that
// accompanied it.
func (m Manager) LifecycleEvent(state LifecycleState, healthReason string) {
	m.event("lifecycle_state", "package", state.PackageID, "requested_operation", state.RequestedOperation, "detected_install_type", state.DetectedInstallType, "selected_retry_target", state.RetryTarget, "health_reason", healthReason, "failure", state.Failure)
}

func lifecyclePath(id string) string {
	return managerPath + "/lifecycle/" + id + ".json"
}
