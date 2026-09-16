package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

const managerPath = "/userdata/system/knulli-app-store"

type Installed struct {
	Schema    string            `json:"schema"`
	Manifest  manifest.Package  `json:"manifest"`
	Files     []InstalledFile   `json:"files"`
	Originals map[string]string `json:"originals,omitempty"`
	MenuOwned bool              `json:"menu_owned,omitempty"`
}

type InstalledFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
	// Size and Modified are the signature of the exact bytes SHA256 was
	// verified against. They are recorded only when SHA256 is the digest of
	// this destination file, so a health check can skip hashing a file that
	// still matches them instead of re-reading every managed file on every
	// catalogue load. State written before this signature existed has no
	// Modified value, so its first check hashes and then records them.
	Size      int64  `json:"size,omitempty"`
	Modified  string `json:"modified,omitempty"`
	Preserved bool   `json:"preserved,omitempty"`
	Unmanaged bool   `json:"unmanaged,omitempty"`
}

// verificationSignature captures the metadata a health check compares before
// hashing a file again: size and modification time, at nanosecond precision
// where the filesystem supports it.
func verificationSignature(info os.FileInfo) (int64, string) {
	return info.Size(), info.ModTime().UTC().Format(time.RFC3339Nano)
}

// verifiedAgainst reports whether the destination still matches the signature
// the recorded SHA256 was verified against, so its bytes need not be read
// again.
func (file InstalledFile) verifiedAgainst(info os.FileInfo) bool {
	if file.SHA256 == "" || file.Modified == "" {
		return false
	}
	size, modified := verificationSignature(info)
	return file.Size == size && file.Modified == modified
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
	state.Manifest = migrateLegacyManifest(state.Manifest)
	if err := state.Manifest.Validate(); err != nil || !state.Manifest.Installable() {
		return nil, fmt.Errorf("installed state contains an invalid manifest")
	}
	return &state, nil
}

func migrateLegacyManifest(pkg manifest.Package) manifest.Package {
	if pkg.Compatibility == nil {
		return pkg
	}
	if len(pkg.Compatibility.ABIs) == 0 && len(pkg.Compatibility.Architectures) == 1 && pkg.Compatibility.Architectures[0] == "aarch64" {
		pkg.Compatibility.ABIs = []string{"linux-aarch64-glibc"}
	}
	if len(pkg.Compatibility.Dependencies) == 0 {
		pkg.Compatibility.Dependencies = []string{"libc"}
	}
	return pkg
}

func encodeState(state Installed) ([]byte, error) {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
