package installer

import (
	"os"

	"github.com/jellydn/knulli-app-store/internal/safefs"
)

type Status struct {
	Installed bool
	Version   string
	Healthy   bool
}

func (m Manager) Status(id string) (Status, error) {
	baseGuard, err := safefs.NewGuard(m.root(), []string{managerPath})
	if err != nil {
		return Status{}, err
	}
	state, err := loadState(baseGuard, id)
	if err != nil || state == nil {
		return Status{}, err
	}
	allowed := append([]string{}, state.Manifest.Install.AllowedWritePaths...)
	allowed = append(allowed, managerPath)
	guard, err := safefs.NewGuard(m.root(), allowed)
	if err != nil {
		return Status{}, err
	}
	status := Status{Installed: true, Version: state.Manifest.Version, Healthy: true}
	for _, file := range state.Files {
		if file.Preserved {
			continue
		}
		host, err := guard.Resolve(file.Path)
		if err != nil {
			return Status{}, err
		}
		digest, err := safefs.SHA256(host)
		if os.IsNotExist(err) || (err == nil && digest != file.SHA256) {
			status.Healthy = false
			continue
		}
		if err != nil {
			return Status{}, err
		}
	}
	return status, nil
}
