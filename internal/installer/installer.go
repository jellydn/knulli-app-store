package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"syscall"

	storearchive "github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

type Manager struct {
	Root     string
	Platform platform.Info
	Client   *http.Client
}

func (m Manager) Install(ctx context.Context, pkg manifest.Package) error {
	return m.apply(ctx, pkg, "install")
}

func (m Manager) Update(ctx context.Context, pkg manifest.Package) error {
	return m.apply(ctx, pkg, "update")
}

func (m Manager) Repair(ctx context.Context, pkg manifest.Package) error {
	return m.apply(ctx, pkg, "repair")
}

func (m Manager) apply(ctx context.Context, pkg manifest.Package, operation string) (result error) {
	if err := pkg.Validate(); err != nil {
		return err
	}
	if !pkg.Installable() {
		return fmt.Errorf("package %s is a candidate and cannot be installed", pkg.ID)
	}
	if err := platform.Check(pkg, m.Platform); err != nil {
		return err
	}
	if err := validateDownloadURL(pkg.Release.URL); err != nil {
		return err
	}
	baseGuard, err := safefs.NewGuard(m.root(), []string{managerPath})
	if err != nil {
		return err
	}
	managerHost, err := baseGuard.Resolve(managerPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(managerHost, "installed"), 0700); err != nil {
		return err
	}
	lock, err := acquireLock(filepath.Join(managerHost, "lock"))
	if err != nil {
		return err
	}
	defer releaseLock(lock)

	old, err := loadState(baseGuard, pkg.ID)
	if err != nil {
		return err
	}
	if operation == "install" && old != nil {
		return fmt.Errorf("package %s is already installed", pkg.ID)
	}
	if operation != "install" && old == nil {
		return fmt.Errorf("package %s is not installed", pkg.ID)
	}
	if operation == "repair" && (old.Manifest.Version != pkg.Version || old.Manifest.Release.SHA256 != pkg.Release.SHA256) {
		return fmt.Errorf("repair requires the installed release; use update for a different version")
	}
	allowed := append([]string{}, pkg.Install.AllowedWritePaths...)
	if old != nil && old.Manifest.Install != nil {
		allowed = append(allowed, old.Manifest.Install.AllowedWritePaths...)
	}
	allowed = append(allowed, managerPath)
	guard, err := safefs.NewGuard(m.root(), allowed)
	if err != nil {
		return err
	}
	destinationHost, err := guard.Resolve(pkg.Install.Destination)
	if err != nil {
		return err
	}
	available, err := safefs.AvailableBytes(destinationHost)
	if err != nil {
		return err
	}
	required := uint64(pkg.Release.Size) + uint64(pkg.Release.InstalledSize)*3
	if available < required {
		return fmt.Errorf("not enough free space: need %d bytes, have %d", required, available)
	}
	work, err := os.MkdirTemp(managerHost, "work-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	archivePath := filepath.Join(work, "release")
	client := m.Client
	if client == nil {
		client = defaultHTTPClient()
	}
	if err := download(ctx, client, *pkg.Release, archivePath); err != nil {
		return fmt.Errorf("download release: %w", err)
	}
	files, err := storearchive.Extract(archivePath, pkg.Release.Format, filepath.Join(work, "staging"), pkg.Install.StripComponents, pkg.Release.InstalledSize)
	if err != nil {
		return fmt.Errorf("extract release: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("release archive contains no installable files")
	}
	if !containsArchiveFile(files, pkg.Install.Launcher) {
		return fmt.Errorf("release archive does not contain launcher %s", pkg.Install.Launcher)
	}

	tx, err := safefs.Begin(guard, managerHost)
	if err != nil {
		return err
	}
	defer func() {
		if result != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				result = fmt.Errorf("%w; rollback also failed: %v", result, rollbackErr)
			}
		}
	}()
	state, err := m.installFiles(tx, guard, pkg, old, files)
	if err != nil {
		return err
	}
	if old != nil && old.Manifest.Install.Menu != nil && !sameMenu(old.Manifest.Install.Menu, pkg.Install.Menu) {
		if err := applyMenu(tx, guard, *old.Manifest.Install.Menu, true); err != nil {
			return err
		}
	}
	if pkg.Install.Menu != nil {
		if err := applyMenu(tx, guard, *pkg.Install.Menu, false); err != nil {
			return err
		}
	}
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
	return nil
}

func (m Manager) installFiles(tx *safefs.Transaction, guard *safefs.Guard, pkg manifest.Package, old *Installed, files []storearchive.File) (Installed, error) {
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
	preserved := stringSet(pkg.Install.Preserve)
	for _, file := range files {
		virtual := path.Join(pkg.Install.Destination, file.Relative)
		newPaths[virtual] = true
		host, err := guard.Resolve(virtual)
		if err != nil {
			return Installed{}, err
		}
		if preserved[file.Relative] {
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
		if !oldTracked[virtual] {
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
		state.Files = append(state.Files, InstalledFile{Path: virtual, SHA256: file.SHA256, Mode: uint32(file.Mode.Perm()), Preserved: preserved[file.Relative]})
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

func (m Manager) Uninstall(id string) (result error) {
	baseGuard, err := safefs.NewGuard(m.root(), []string{managerPath})
	if err != nil {
		return err
	}
	managerHost, err := baseGuard.Resolve(managerPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(managerHost, "installed"), 0700); err != nil {
		return err
	}
	lock, err := acquireLock(filepath.Join(managerHost, "lock"))
	if err != nil {
		return err
	}
	defer releaseLock(lock)
	state, err := loadState(baseGuard, id)
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
	tx, err := safefs.Begin(guard, managerHost)
	if err != nil {
		return err
	}
	defer func() {
		if result != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				result = fmt.Errorf("%w; rollback also failed: %v", result, rollbackErr)
			}
		}
	}()
	for _, file := range state.Files {
		if file.Preserved {
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
	if state.Manifest.Install.Menu != nil {
		if err := applyMenu(tx, guard, *state.Manifest.Install.Menu, true); err != nil {
			return err
		}
	}
	for _, backup := range state.Originals {
		if err := tx.Remove(backup); err != nil {
			return err
		}
	}
	if err := tx.Remove(statePath(id)); err != nil {
		return err
	}
	return tx.Commit()
}

func applyMenu(tx *safefs.Transaction, guard *safefs.Guard, menu manifest.Menu, remove bool) error {
	host, err := guard.Resolve(menu.Gamelist)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(host)
	if err != nil {
		if remove && os.IsNotExist(err) {
			return nil
		}
		if !os.IsNotExist(err) {
			return err
		}
	}
	updated, err := updateGamelist(data, menu, remove)
	if err != nil {
		return err
	}
	return tx.Write(menu.Gamelist, updated, 0644)
}

func originalPath(id, target string) string {
	digest := sha256.Sum256([]byte(target))
	return managerPath + "/originals/" + id + "/" + hex.EncodeToString(digest[:])
}

func containsArchiveFile(files []storearchive.File, relative string) bool {
	for _, file := range files {
		if file.Relative == relative {
			return true
		}
	}
	return false
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}

func sameMenu(left, right *manifest.Menu) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Gamelist == right.Gamelist && left.Path == right.Path
}

func (m Manager) root() string {
	if m.Root == "" {
		return "/"
	}
	return m.Root
}

func acquireLock(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func releaseLock(file *os.File) {
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	file.Close()
}
