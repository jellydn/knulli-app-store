package installer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

type RecoveryStatus struct {
	Reason       string
	ForceAllowed bool
	Active       bool
}

func (m Manager) RecoveryStatus(pkg manifest.Package) (RecoveryStatus, error) {
	if !pkg.Installable() || pkg.Install == nil {
		return RecoveryStatus{}, nil
	}
	guard, err := safefs.NewGuard(m.root(), []string{managerPath})
	if err != nil {
		return RecoveryStatus{}, err
	}
	managerHost, err := guard.Resolve(managerPath)
	if err != nil {
		return RecoveryStatus{}, err
	}
	if err := os.MkdirAll(managerHost, 0700); err != nil {
		return RecoveryStatus{}, err
	}
	lock, err := acquireLock(filepath.Join(managerHost, "lock"))
	if err != nil {
		if strings.Contains(err.Error(), "another package operation is active") {
			return RecoveryStatus{Reason: err.Error(), Active: true}, nil
		}
		return RecoveryStatus{}, err
	}
	releaseLock(lock)
	pending, err := safefs.PendingForPath(m.root(), managerHost, pkg.Install.Destination)
	if err != nil {
		return RecoveryStatus{}, fmt.Errorf("inspect transaction journal: %w", err)
	}
	if pending {
		return RecoveryStatus{
			Reason:       "Interrupted package transaction detected; the next operation will roll it back before changing files",
			ForceAllowed: true,
		}, nil
	}
	return RecoveryStatus{}, nil
}

// PreExisting reports whether the destination holds something this installer
// did not write. Adoption inventories regular files, so an empty directory
// skeleton left behind by a rolled-back write is an absent package, not an
// external installation: reporting it as external offers Manage existing for
// a copy adoption can never find. Anything else at the destination still
// needs the user to move it aside first. This stops at the first entry
// instead of hashing the destination like adoption does, because the
// catalogue asks every package on every load.
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
	info, err := os.Lstat(host)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.IsDir() {
		return true, nil
	}
	found := false
	err = fs.WalkDir(os.DirFS(host), ".", func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		found = true
		return fs.SkipAll
	})
	if err != nil {
		return false, err
	}
	return found, nil
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
			status.addIssue(m, state.Manifest.ID, HealthIssue{Path: file.Path, Check: "content changed", Expected: file.SHA256, Actual: digest})
		}
		if issue := modeIssue(file, info.Mode()); issue != nil {
			status.addIssue(m, state.Manifest.ID, *issue)
		}
	}
	m.event("package_health_checked", "package", state.Manifest.ID, "healthy", fmt.Sprint(status.Healthy), "issues", fmt.Sprint(len(status.Issues)))
	return status, nil
}

// modeIssue reports a permission regression. The destination filesystem owns
// the mode bits and a Knulli SD card can report 0777 for a file the installer
// requested as 0755 or 0644, so only the read, write, and execute capabilities
// the recorded mode granted are required back; wider bits are not an issue.
func modeIssue(file InstalledFile, actual os.FileMode) *HealthIssue {
	expected := os.FileMode(file.Mode).Perm()
	for _, class := range [...]os.FileMode{0444, 0222, 0111} {
		if expected&class != 0 && actual.Perm()&class == 0 {
			return &HealthIssue{Path: file.Path, Check: "mode changed", Expected: fmt.Sprintf("%04o", expected), Actual: fmt.Sprintf("%04o", actual.Perm())}
		}
	}
	return nil
}

func (status *Status) addIssue(manager Manager, packageID string, issue HealthIssue) {
	status.Healthy = false
	status.Issues = append(status.Issues, issue)
	manager.event("package_health_issue", "package", packageID, "path", issue.Path, "check", issue.Check, "expected", issue.Expected, "actual", issue.Actual)
}
