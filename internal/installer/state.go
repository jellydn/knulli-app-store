package installer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

const managerPath = "/userdata/system/knulli-app-store"

type Installed struct {
	Schema    string            `json:"schema"`
	Manifest  manifest.Package  `json:"manifest"`
	Files     []InstalledFile   `json:"files"`
	Originals map[string]string `json:"originals,omitempty"`
}

type InstalledFile struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Mode      uint32 `json:"mode"`
	Preserved bool   `json:"preserved,omitempty"`
}

func statePath(id string) string {
	return managerPath + "/installed/" + id + ".json"
}

func loadState(guard *safefs.Guard, id string) (*Installed, error) {
	host, err := guard.Resolve(statePath(id))
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
	var state Installed
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("read installed state: %w", err)
	}
	if state.Schema != "org.knulli.app-store/installed-state/v1" || state.Manifest.ID != id {
		return nil, fmt.Errorf("installed state has an invalid schema or package id")
	}
	if err := state.Manifest.Validate(); err != nil || !state.Manifest.Installable() {
		return nil, fmt.Errorf("installed state contains an invalid manifest")
	}
	return &state, nil
}

func encodeState(state Installed) ([]byte, error) {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
