package installer

import (
	"context"
	"fmt"
	"os"
	"path"
	"sort"

	storearchive "github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/gamelist"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

// commit applies the staged release inside one transaction: files, menu
// ownership, and the installed-state record are written together and rolled
// back as one on failure.
func (m Manager) commit(ctx context.Context, operation string, pkg manifest.Package, staged *stagedRelease, outcome *OperationOutcome) (result error) {
	guard := staged.guard
	old := staged.old
	tx, err := safefs.Begin(guard, staged.session.managerHost)
	if err != nil {
		return err
	}
	m.event("transaction_begin", "package", pkg.ID, "action", operation)
	recoveryBackupPath := ""
	defer m.rollback(pkg.ID, operation, tx, guard, &recoveryBackupPath, &result)
	var state Installed
	if operation == "adopt" {
		state, err = m.adoptFiles(tx, guard, pkg, staged.files, staged.existing)
	} else {
		if operation == "force-reinstall" {
			recoveryBackupPath = m.recoveryBackupDirectory(pkg)
			backupErr := m.backupRecoveryDestination(tx, pkg, staged.existing, recoveryBackupPath)
			if backupErr != nil {
				return backupErr
			}
			m.event("recovery_backup_complete", "package", pkg.ID, "path", recoveryBackupPath, "files", fmt.Sprint(len(staged.existing)))
		}
		state, err = m.installFiles(tx, guard, pkg, old, staged.files, operation == "force-reinstall")
	}
	if err != nil {
		return err
	}
	m.event("backup_complete", "package", pkg.ID, "originals", fmt.Sprint(len(state.Originals)))
	var previous *manifest.Menu
	if old != nil && old.MenuOwned && old.Manifest.Install.Menu != nil {
		owned := *old.Manifest.Install.Menu
		previous = &owned
	}
	menuChanged, menuOwned, err := gamelist.Apply(tx, guard, gamelist.Derive(previous != nil, previous, pkg.Install.Menu))
	if err != nil {
		return err
	}
	state.MenuOwned = menuOwned
	gameListChanged := menuChanged
	stateData, err := encodeState(state)
	if err != nil {
		return err
	}
	if err := tx.Write(statePath(pkg.ID), stateData, 0600); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	*outcome = m.outcome(ctx, gameListChanged)
	m.event("operation_complete", "package", pkg.ID, "action", operation)
	return nil
}

func (m Manager) installFiles(tx *safefs.Transaction, guard *safefs.Guard, pkg manifest.Package, old *Installed, files []storearchive.File, recovery bool) (Installed, error) {
	state := Installed{Schema: "org.knulli.app-store/installed-state/v1", Manifest: pkg, Originals: make(map[string]string)}
	oldTracked := make(map[string]bool)
	if old != nil {
		for key, value := range old.Originals {
			state.Originals[key] = value
		}
		for _, file := range old.Files {
			oldTracked[file.Path] = true
		}
	}
	newPaths := make(map[string]bool)
	for _, file := range files {
		virtual := path.Join(pkg.Install.Destination, file.Relative)
		newPaths[virtual] = true
		host, err := guard.Resolve(virtual)
		if err != nil {
			return Installed{}, err
		}
		preserved := isPreserved(file.Relative, pkg.Install.Preserve)
		if preserved {
			if info, err := os.Stat(host); err == nil {
				digest, err := safefs.SHA256(host)
				if err != nil {
					return Installed{}, err
				}
				state.Files = append(state.Files, InstalledFile{Path: virtual, SHA256: digest, Mode: uint32(info.Mode().Perm()), Preserved: true})
				continue
			} else if !os.IsNotExist(err) {
				return Installed{}, err
			}
		}
		if !recovery && !oldTracked[virtual] {
			if info, err := os.Stat(host); err == nil {
				backup := originalPath(pkg.ID, virtual)
				if _, exists := state.Originals[virtual]; !exists {
					if err := tx.Copy(host, backup, info.Mode()); err != nil {
						return Installed{}, err
					}
					state.Originals[virtual] = backup
				}
			} else if !os.IsNotExist(err) {
				return Installed{}, err
			}
		}
		if err := tx.Copy(file.Path, virtual, file.Mode); err != nil {
			return Installed{}, err
		}
		installedInfo, err := os.Stat(host)
		if err != nil {
			return Installed{}, err
		}
		state.Files = append(state.Files, InstalledFile{Path: virtual, SHA256: file.SHA256, Mode: uint32(installedInfo.Mode().Perm()), Preserved: preserved})
	}
	if old != nil {
		for _, stale := range old.Files {
			if newPaths[stale.Path] || stale.Preserved {
				continue
			}
			if backup, exists := state.Originals[stale.Path]; exists {
				backupHost, err := guard.Resolve(backup)
				if err != nil {
					return Installed{}, err
				}
				info, err := os.Stat(backupHost)
				if err != nil {
					return Installed{}, err
				}
				if err := tx.Copy(backupHost, stale.Path, info.Mode()); err != nil {
					return Installed{}, err
				}
				if err := tx.Remove(backup); err != nil {
					return Installed{}, err
				}
				delete(state.Originals, stale.Path)
			} else if err := tx.Remove(stale.Path); err != nil {
				return Installed{}, err
			}
		}
	}
	sort.Slice(state.Files, func(i, j int) bool { return state.Files[i].Path < state.Files[j].Path })
	return state, nil
}

func (m Manager) adoptFiles(tx *safefs.Transaction, guard *safefs.Guard, pkg manifest.Package, files []storearchive.File, existing []existingFile) (Installed, error) {
	state := Installed{Schema: "org.knulli.app-store/installed-state/v1", Manifest: pkg, Originals: make(map[string]string)}
	existingByPath := make(map[string]existingFile, len(existing))
	for _, file := range existing {
		existingByPath[file.Virtual] = file
	}
	for _, releaseFile := range files {
		virtual := path.Join(pkg.Install.Destination, releaseFile.Relative)
		existingFile, found := existingByPath[virtual]
		preserved := isPreserved(releaseFile.Relative, pkg.Install.Preserve)
		managed := found && existingFile.SHA256 == releaseFile.SHA256 && !preserved
		if found {
			if err := backupExisting(tx, guard, pkg.ID, existingFile, &state); err != nil {
				return Installed{}, err
			}
		}
		mode := releaseFile.Mode.Perm()
		if managed {
			if err := tx.Chmod(virtual, releaseFile.Mode); err != nil {
				return Installed{}, err
			}
			info, err := os.Stat(existingFile.Host)
			if err != nil {
				return Installed{}, err
			}
			mode = info.Mode().Perm()
		} else if found {
			mode = existingFile.Mode.Perm()
		}
		state.Files = append(state.Files, InstalledFile{Path: virtual, SHA256: releaseFile.SHA256, Mode: uint32(mode), Preserved: preserved, Unmanaged: !managed})
		delete(existingByPath, virtual)
	}
	for _, file := range existingByPath {
		if err := backupExisting(tx, guard, pkg.ID, file, &state); err != nil {
			return Installed{}, err
		}
	}
	sort.Slice(state.Files, func(i, j int) bool { return state.Files[i].Path < state.Files[j].Path })
	m.event("adoption_complete", "package", pkg.ID, "managed_files", fmt.Sprint(len(state.Files)), "backups", fmt.Sprint(len(state.Originals)))
	return state, nil
}

func backupExisting(tx *safefs.Transaction, guard *safefs.Guard, id string, file existingFile, state *Installed) error {
	backup := originalPath(id, file.Virtual)
	backupHost, err := guard.Resolve(backup)
	if err != nil {
		return err
	}
	if _, err := os.Stat(backupHost); err == nil {
		return &AdoptionConflictError{Path: file.Virtual}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := tx.Copy(file.Host, backup, file.Mode); err != nil {
		return err
	}
	state.Originals[file.Virtual] = backup
	return nil
}

// rollback rolls the transaction back when the operation failed, folding any
// rollback failure into the operation error.
func (m Manager) rollback(id, action string, tx *safefs.Transaction, guard *safefs.Guard, cleanupPath *string, result *error) {
	if *result == nil {
		return
	}
	m.event("rollback_start", "package", id, "action", action)
	if rollbackErr := tx.Rollback(); rollbackErr != nil {
		*result = fmt.Errorf("%w; rollback also failed: %v", *result, rollbackErr)
		m.event("rollback_error", "package", id, "error", rollbackErr.Error())
	} else {
		m.event("rollback_complete", "package", id)
	}
	if guard != nil && cleanupPath != nil && *cleanupPath != "" {
		if host, err := guard.Resolve(*cleanupPath); err == nil {
			_ = os.RemoveAll(host)
		}
	}
}
