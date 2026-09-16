package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jellydn/knulli-app-store/internal/safefs"
)

const lifecycleSchemaV1 = "org.knulli.app-store/lifecycle-state/v1"

type LifecycleState struct {
	Schema              string `json:"schema"`
	PackageID           string `json:"package_id"`
	RequestedOperation  string `json:"requested_operation"`
	DetectedInstallType string `json:"detected_install_type"`
	RetryTarget         string `json:"retry_target"`
	Failure             string `json:"failure,omitempty"`
	ForceAllowed        bool   `json:"force_allowed,omitempty"`
}

func (m Manager) LifecycleState(id string) (*LifecycleState, error) {
	guard, err := safefs.NewGuard(m.root(), []string{managerPath})
	if err != nil {
		return nil, err
	}
	host, err := guard.Resolve(lifecyclePath(id))
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

func (m Manager) RecordLifecycle(state LifecycleState) error {
	guard, err := safefs.NewGuard(m.root(), []string{managerPath})
	if err != nil {
		return err
	}
	host, err := guard.Resolve(lifecyclePath(state.PackageID))
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

func (m Manager) ClearLifecycle(id string) error {
	guard, err := safefs.NewGuard(m.root(), []string{managerPath})
	if err != nil {
		return err
	}
	host, err := guard.Resolve(lifecyclePath(id))
	if err != nil {
		return err
	}
	if err := os.Remove(host); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (m Manager) LifecycleEvent(state LifecycleState, healthReason string) {
	m.event("lifecycle_state", "package", state.PackageID, "requested_operation", state.RequestedOperation, "detected_install_type", state.DetectedInstallType, "selected_retry_target", state.RetryTarget, "health_reason", healthReason, "failure", state.Failure)
}

func lifecyclePath(id string) string {
	return managerPath + "/lifecycle/" + id + ".json"
}
