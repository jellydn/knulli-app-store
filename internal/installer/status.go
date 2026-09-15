package installer

import (
	"fmt"
	"os"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

type HealthIssue struct {
	Path     string
	Check    string
	Expected string
	Actual   string
}

func (issue HealthIssue) String() string {
	return fmt.Sprintf("%s: %s (expected %s, got %s)", issue.Check, issue.Path, issue.Expected, issue.Actual)
}

type Status struct {
	Installed bool
	Version   string
	Healthy   bool
	Issues    []HealthIssue
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
		destination := strings.TrimSuffix(state.Manifest.Install.Destination, "/")
		if file.Path != destination && !strings.HasPrefix(file.Path, destination+"/") {
			status.addIssue(m, state.Manifest.ID, HealthIssue{Path: file.Path, Check: "unexpected managed path", Expected: destination, Actual: file.Path})
			continue
		}
		host, err := guard.Resolve(file.Path)
		if err != nil {
			return Status{}, err
		}
		info, err := os.Stat(host)
		if os.IsNotExist(err) {
			status.addIssue(m, state.Manifest.ID, HealthIssue{Path: file.Path, Check: "missing", Expected: "regular file", Actual: "not found"})
			continue
		}
		if err != nil {
			return Status{}, err
		}
		digest, err := safefs.SHA256(host)
		if err != nil {
			return Status{}, err
		}
		if digest != file.SHA256 {
			status.addIssue(m, state.Manifest.ID, HealthIssue{Path: file.Path, Check: "content changed", Expected: shortDigest(file.SHA256), Actual: shortDigest(digest)})
		}
		if info.Mode().Perm() != os.FileMode(file.Mode).Perm() {
			status.addIssue(m, state.Manifest.ID, HealthIssue{Path: file.Path, Check: "mode changed", Expected: fmt.Sprintf("%04o", os.FileMode(file.Mode).Perm()), Actual: fmt.Sprintf("%04o", info.Mode().Perm())})
		}
	}
	m.event("package_health_checked", "package", state.Manifest.ID, "healthy", fmt.Sprint(status.Healthy), "issues", fmt.Sprint(len(status.Issues)))
	return status, nil
}

func (status *Status) addIssue(manager Manager, packageID string, issue HealthIssue) {
	status.Healthy = false
	status.Issues = append(status.Issues, issue)
	manager.event("package_health_issue", "package", packageID, "path", issue.Path, "check", issue.Check, "expected", issue.Expected, "actual", issue.Actual)
}

func shortDigest(value string) string {
	if len(value) <= 12 {
		return value
	}
	return value[:12]
}
