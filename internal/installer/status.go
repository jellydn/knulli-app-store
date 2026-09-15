package installer

import (
	"os"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

type Status struct {
	Installed bool
	Version   string
	Healthy   bool
}

func (m Manager) PreExisting(pkg manifest.Package) (bool, error) {
	if !pkg.Installable() || pkg.Install == nil {
		return false, nil
	}
	guard, err := safefs.NewGuard(m.root(), pkg.Install.AllowedWritePaths)
	if err != nil {
		return false, err
	}
	host, err := guard.Resolve(pkg.Install.Destination)
	if err != nil {
		return false, err
	}
	entries, err := os.ReadDir(host)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
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
		info, err := os.Stat(host)
		if os.IsNotExist(err) {
			status.Healthy = false
			continue
		}
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
		if info.Mode().Perm() != os.FileMode(file.Mode).Perm() {
			status.Healthy = false
		}
	}
	return status, nil
}
