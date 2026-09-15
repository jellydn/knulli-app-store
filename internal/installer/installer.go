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
	"strings"
	"syscall"

	storearchive "github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

type Manager struct {
	Root          string
	Platform      platform.Info
	Client        *http.Client
	RefreshClient *http.Client
	RefreshURL    string
	Diagnostics   *diagnostics.Log
	Outcome       func(OperationOutcome)
}

func (m Manager) Install(ctx context.Context, pkg manifest.Package) error {
	return m.apply(ctx, pkg, "install")
}

func (m Manager) Adopt(ctx context.Context, pkg manifest.Package) error {
	return m.apply(ctx, pkg, "adopt")
}

func (m Manager) Update(ctx context.Context, pkg manifest.Package) error {
	return m.apply(ctx, pkg, "update")
}

func (m Manager) Repair(ctx context.Context, pkg manifest.Package) error {
	return m.apply(ctx, pkg, "repair")
}

func (m Manager) apply(ctx context.Context, pkg manifest.Package, operation string) (result error) {
	m.event("operation_start", "package", pkg.ID, "action", operation)
	defer func() {
		if result != nil {
			m.event("operation_error", "package", pkg.ID, "action", operation, "error", result.Error())
		}
	}()
	if err := pkg.Validate(); err != nil {
		return err
	}
	if !pkg.Installable() {
		return fmt.Errorf("package %s is a candidate and cannot be installed", pkg.ID)
	}
	if err := platform.Check(pkg, m.Platform); err != nil {
		m.event("compatibility_rejected", "package", pkg.ID, "decision", err.Error())
		return err
	}
	m.event("compatibility_allowed", "package", pkg.ID, "status", pkg.Review.Status, "platform", platform.Summary(m.Platform))
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
	if err := safefs.Recover(m.root(), managerHost); err != nil {
		return err
	}

	old, err := loadState(baseGuard, pkg.ID)
	if err != nil {
		return err
	}
	if (operation == "install" || operation == "adopt") && old != nil {
		return fmt.Errorf("package %s is already installed", pkg.ID)
	}
	if operation != "install" && operation != "adopt" && old == nil {
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
	var existing []existingFile
	var existingBytes uint64
	if old == nil {
		existing, existingBytes, err = inventoryExisting(destinationHost, pkg.Install.Destination)
		if err != nil {
			return fmt.Errorf("inventory pre-existing installation: %w", err)
		}
		m.event("adoption_inventory", "package", pkg.ID, "files", fmt.Sprint(len(existing)), "bytes", fmt.Sprint(existingBytes))
	}
	if operation == "install" && len(existing) > 0 {
		return fmt.Errorf("an external installation exists; use Manage existing")
	}
	if operation == "adopt" && len(existing) == 0 {
		return fmt.Errorf("no external installation was found; use Install")
	}
	available, err := safefs.AvailableBytes(destinationHost)
	if err != nil {
		return err
	}
	required := uint64(pkg.Release.Size) + uint64(pkg.Release.InstalledSize)*3 + existingBytes*2
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
	m.event("download_start", "package", pkg.ID, "url", pkg.Release.URL, "expected_bytes", fmt.Sprint(pkg.Release.Size))
	if err := download(ctx, client, *pkg.Release, archivePath); err != nil {
		return fmt.Errorf("download release: %w", err)
	}
	m.event("download_verified", "package", pkg.ID, "bytes", fmt.Sprint(pkg.Release.Size), "sha256", pkg.Release.SHA256)
	files, err := storearchive.Extract(archivePath, pkg.Release.Format, filepath.Join(work, "staging"), pkg.Install.StripComponents, pkg.Release.InstalledSize)
	if err != nil {
		return fmt.Errorf("extract release: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("release archive contains no installable files")
	}
	m.event("extraction_complete", "package", pkg.ID, "format", pkg.Release.Format, "files", fmt.Sprint(len(files)))
	if err := applyExecutableModes(files, pkg.Install.Executables); err != nil {
		return err
	}
	if !containsArchiveFile(files, pkg.Install.Launcher) {
		return fmt.Errorf("release archive does not contain launcher %s", pkg.Install.Launcher)
	}

	tx, err := safefs.Begin(guard, managerHost)
	if err != nil {
		return err
	}
	m.event("transaction_begin", "package", pkg.ID, "action", operation)
	defer func() {
		if result != nil {
			m.event("rollback_start", "package", pkg.ID, "action", operation)
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				result = fmt.Errorf("%w; rollback also failed: %v", result, rollbackErr)
				m.event("rollback_error", "package", pkg.ID, "error", rollbackErr.Error())
			} else {
				m.event("rollback_complete", "package", pkg.ID)
			}
		}
	}()
	var state Installed
	if operation == "adopt" {
		state, err = m.adoptFiles(tx, guard, pkg, files, existing)
	} else {
		state, err = m.installFiles(tx, guard, pkg, old, files)
	}
	if err != nil {
		return err
	}
	m.event("backup_complete", "package", pkg.ID, "originals", fmt.Sprint(len(state.Originals)))
	gameListChanged := false
	if old != nil && old.MenuOwned && old.Manifest.Install.Menu != nil && !sameMenu(old.Manifest.Install.Menu, pkg.Install.Menu) {
		changed, menuErr := removeMenu(tx, guard, *old.Manifest.Install.Menu)
		if menuErr != nil {
			return menuErr
		}
		gameListChanged = gameListChanged || changed
	}
	if pkg.Install.Menu != nil {
		if old != nil && old.MenuOwned && old.Manifest.Install.Menu != nil && sameMenu(old.Manifest.Install.Menu, pkg.Install.Menu) {
			changed, menuErr := replaceMenu(tx, guard, *old.Manifest.Install.Menu, *pkg.Install.Menu)
			if menuErr != nil {
				return menuErr
			}
			state.MenuOwned = changed
			gameListChanged = gameListChanged || changed
		} else {
			owned, menuErr := addMenu(tx, guard, *pkg.Install.Menu)
			if menuErr != nil {
				return menuErr
			}
			state.MenuOwned = owned
			gameListChanged = gameListChanged || owned
		}
	}
	if old != nil && old.MenuOwned && pkg.Install.Menu == nil {
		state.MenuOwned = false
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
	m.reportOutcome(ctx, gameListChanged)
	m.event("operation_complete", "package", pkg.ID, "action", operation)
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
		managed := found && existingFile.SHA256 == releaseFile.SHA256 && existingFile.Mode.Perm() == releaseFile.Mode.Perm() && !preserved
		if found && !managed {
			if err := backupExisting(tx, guard, pkg.ID, existingFile, &state); err != nil {
				return Installed{}, err
			}
		}
		state.Files = append(state.Files, InstalledFile{Path: virtual, SHA256: releaseFile.SHA256, Mode: uint32(releaseFile.Mode.Perm()), Preserved: preserved, Unmanaged: !managed})
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
		return fmt.Errorf("adoption backup already exists for %s; move the existing package aside before retrying", file.Virtual)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := tx.Copy(file.Host, backup, file.Mode); err != nil {
		return err
	}
	state.Originals[file.Virtual] = backup
	return nil
}

func (m Manager) Uninstall(id string) error {
	return m.UninstallContext(context.Background(), id)
}

func (m Manager) UninstallContext(ctx context.Context, id string) (result error) {
	m.event("operation_start", "package", id, "action", "uninstall")
	defer func() {
		if result != nil {
			m.event("operation_error", "package", id, "action", "uninstall", "error", result.Error())
		}
	}()
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
	if err := safefs.Recover(m.root(), managerHost); err != nil {
		return err
	}
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
	m.event("transaction_begin", "package", id, "action", "uninstall")
	defer func() {
		if result != nil {
			m.event("rollback_start", "package", id, "action", "uninstall")
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				result = fmt.Errorf("%w; rollback also failed: %v", result, rollbackErr)
				m.event("rollback_error", "package", id, "error", rollbackErr.Error())
			} else {
				m.event("rollback_complete", "package", id)
			}
		}
	}()
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
		changed, menuErr := removeMenu(tx, guard, *state.Manifest.Install.Menu)
		if menuErr != nil {
			return menuErr
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
	m.reportOutcome(ctx, gameListChanged)
	m.event("operation_complete", "package", id, "action", "uninstall")
	return nil
}

func (m Manager) event(name string, fields ...string) {
	if m.Diagnostics != nil {
		m.Diagnostics.Event(name, fields...)
	}
}

const maximumAdoptionBytes = 512 << 20

type existingFile struct {
	Virtual string
	Host    string
	Mode    os.FileMode
	SHA256  string
}

func inventoryExisting(destinationHost, destination string) ([]existingFile, uint64, error) {
	info, err := os.Lstat(destinationHost)
	if os.IsNotExist(err) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	if !info.IsDir() {
		return nil, 0, fmt.Errorf("destination is not a directory; move it aside before retrying")
	}
	var files []existingFile
	var total uint64
	err = filepath.Walk(destinationHost, func(host string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("refusing non-regular pre-existing path %s; move the package directory aside before retrying", host)
		}
		relative, err := filepath.Rel(destinationHost, host)
		if err != nil {
			return err
		}
		size := uint64(info.Size())
		if size > maximumAdoptionBytes || total > maximumAdoptionBytes-size {
			return fmt.Errorf("pre-existing package exceeds the 512 MiB adoption limit; move it aside before retrying")
		}
		total += size
		digest, err := safefs.SHA256(host)
		if err != nil {
			return err
		}
		files = append(files, existingFile{Virtual: path.Join(destination, filepath.ToSlash(relative)), Host: host, Mode: info.Mode(), SHA256: digest})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Virtual < files[j].Virtual })
	return files, total, err
}

func applyExecutableModes(files []storearchive.File, executables []string) error {
	wanted := stringSet(executables)
	for index := range files {
		if wanted[files[index].Relative] {
			files[index].Mode = 0755
			delete(wanted, files[index].Relative)
		}
	}
	for executable := range wanted {
		return fmt.Errorf("release archive does not contain declared executable %s", executable)
	}
	return nil
}

func isPreserved(relative string, preserved []string) bool {
	for _, entry := range preserved {
		if relative == entry || strings.HasPrefix(relative, entry+"/") {
			return true
		}
	}
	return false
}

func addMenu(tx *safefs.Transaction, guard *safefs.Guard, menu manifest.Menu) (bool, error) {
	return changeMenu(tx, guard, menu.Gamelist, func(data []byte) ([]byte, bool, error) {
		return addMenuEntry(data, menu)
	})
}

func replaceMenu(tx *safefs.Transaction, guard *safefs.Guard, oldMenu, newMenu manifest.Menu) (bool, error) {
	return changeMenu(tx, guard, newMenu.Gamelist, func(data []byte) ([]byte, bool, error) {
		return replaceOwnedMenuEntry(data, oldMenu, newMenu)
	})
}

func removeMenu(tx *safefs.Transaction, guard *safefs.Guard, menu manifest.Menu) (bool, error) {
	return changeMenu(tx, guard, menu.Gamelist, func(data []byte) ([]byte, bool, error) {
		return removeOwnedMenuEntry(data, menu)
	})
}

func changeMenu(tx *safefs.Transaction, guard *safefs.Guard, gamelist string, change func([]byte) ([]byte, bool, error)) (bool, error) {
	host, err := guard.Resolve(gamelist)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(host)
	if err != nil {
		if !os.IsNotExist(err) {
			return false, err
		}
	}
	updated, changed, err := change(data)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	return true, tx.Write(gamelist, updated, 0644)
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
