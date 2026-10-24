package installer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jellydn/knulli-app-store/internal/gamelist"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

// Uninstall removes a managed package, restoring originals and releasing menu
// ownership. The outcome is returned after the commit; a failed uninstall
// returns a zero outcome with the error.
func (m Manager) Uninstall(ctx context.Context, id string) (OperationOutcome, error) {
	var outcome OperationOutcome
	// Removal drops the installed state, and a failed one can leave a
	// destination half-restored, so the cached result goes whatever the
	// outcome.
	defer m.statusCache.Invalidate(id)
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
	if err := m.checkUninstall(ctx, state.Manifest.Install.UninstallChecks); err != nil {
		return err
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

// Checks are read-only and fail closed. Runtime hooks are not Store-owned files:
// removing an app must never edit a shared hook or strand a reference to it.
func (m Manager) checkUninstall(ctx context.Context, checks []manifest.UninstallCheck) error {
	for _, check := range checks {
		guard, err := safefs.NewGuard(m.root(), []string{check.Path})
		if err != nil {
			return err
		}
		host, err := guard.Resolve(check.Path)
		if err != nil {
			return fmt.Errorf("cannot check uninstall safety: %w", err)
		}
		err = filepath.WalkDir(host, func(file string, entry fs.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if os.IsNotExist(walkErr) {
				return nil
			}
			if walkErr != nil {
				return walkErr
			}
			if check.Contains == "" {
				return fmt.Errorf("uninstall blocked by %s: %s", check.Path, check.Message)
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("cannot inspect non-regular hook %s", file)
			}
			input, err := os.Open(file)
			if err != nil {
				return err
			}
			const maxHookBytes = 1 << 20
			data, err := io.ReadAll(io.LimitReader(input, maxHookBytes+1))
			input.Close()
			if err != nil {
				return err
			}
			if len(data) > maxHookBytes {
				return fmt.Errorf("hook exceeds inspection limit: %s", file)
			}
			if bytes.Contains(bytes.ToLower(data), bytes.ToLower([]byte(check.Contains))) {
				return fmt.Errorf("uninstall blocked by %s: %s", check.Path, check.Message)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
