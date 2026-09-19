package installer

import (
	"context"
	"fmt"
	"os"

	"github.com/jellydn/knulli-app-store/internal/gamelist"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

// Uninstall removes a managed package, restoring originals and releasing menu
// ownership. The outcome is returned after the commit; a failed uninstall
// returns a zero outcome with the error.
func (m Manager) Uninstall(ctx context.Context, id string) (OperationOutcome, error) {
	var outcome OperationOutcome
	err := m.uninstall(ctx, id, &outcome)
	return outcome, err
}

func (m Manager) uninstall(ctx context.Context, id string, outcome *OperationOutcome) (result error) {
	m.event("operation_start", "package", id, "action", "uninstall")
	defer func() {
		if result != nil {
			m.event("operation_error", "package", id, "action", "uninstall", "error", result.Error())
		}
	}()
	session, err := m.acquireManager()
	if err != nil {
		return err
	}
	defer session.release()
	state, err := loadState(session.guard, id)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("package %s is not installed", id)
	}
	allowed := append([]string{}, state.Manifest.Install.AllowedWritePaths...)
	allowed = append(allowed, managerPath)
	guard, err := safefs.NewGuard(m.root(), allowed)
	if err != nil {
		return err
	}
	tx, err := safefs.Begin(guard, session.managerHost)
	if err != nil {
		return err
	}
	m.event("transaction_begin", "package", id, "action", "uninstall")
	defer m.rollback(id, "uninstall", tx, nil, nil, &result)
	for _, file := range state.Files {
		if file.Preserved || file.Unmanaged {
			continue
		}
		if backup, exists := state.Originals[file.Path]; exists {
			backupHost, err := guard.Resolve(backup)
			if err != nil {
				return err
			}
			info, err := os.Stat(backupHost)
			if err != nil {
				return err
			}
			if err := tx.Copy(backupHost, file.Path, info.Mode()); err != nil {
				return err
			}
		} else if err := tx.Remove(file.Path); err != nil {
			return err
		}
	}
	gameListChanged := false
	if state.MenuOwned && state.Manifest.Install.Menu != nil {
		menu := *state.Manifest.Install.Menu
		changed, _, err := gamelist.Apply(tx, guard, gamelist.Derive(true, &menu, nil))
		if err != nil {
			return err
		}
		gameListChanged = changed
	}
	for _, backup := range state.Originals {
		if err := tx.Remove(backup); err != nil {
			return err
		}
	}
	if err := tx.Remove(statePath(id)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	*outcome = m.outcome(ctx, gameListChanged)
	m.event("operation_complete", "package", id, "action", "uninstall")
	return nil
}
