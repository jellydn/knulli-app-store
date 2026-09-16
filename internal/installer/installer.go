package installer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	storearchive "github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	"github.com/jellydn/knulli-app-store/internal/gamelist"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

type Manager struct {
	Root           string
	Client         *http.Client
	RefreshClient  *http.Client
	RefreshURL     string
	Diagnostics    *diagnostics.Log
	Now            func() time.Time
	AvailableBytes func(string) (uint64, error)

	platform platform.Info
}

type AdoptionConflictError struct {
	Path string
}

func (err *AdoptionConflictError) Error() string {
	return fmt.Sprintf("adoption backup already exists for %s; move the existing package aside before retrying", err.Path)
}

// Op names one lifecycle operation applied to a package.
type Op string

const (
	OpInstall        Op = "install"
	OpAdopt          Op = "adopt"
	OpUpdate         Op = "update"
	OpRepair         Op = "repair"
	OpForceReinstall Op = "force-reinstall"
)

// WithPlatform returns a copy of the manager bound to the given detected platform.
func (m Manager) WithPlatform(info platform.Info) Manager {
	m.platform = info
	return m
}

// Platform returns the platform this manager validates packages against.
func (m Manager) Platform() platform.Info {
	return m.platform
}

// Apply runs one package lifecycle operation: download, verify, and commit the
// reviewed release, rolling back every change on failure. The outcome is
// returned after a commit; a failed operation returns a zero outcome with the
// error.
func (m Manager) Apply(ctx context.Context, op Op, pkg manifest.Package) (OperationOutcome, error) {
	var outcome OperationOutcome
	err := m.apply(ctx, string(op), pkg, &outcome)
	return outcome, err
}

func (m Manager) apply(ctx context.Context, operation string, pkg manifest.Package, outcome *OperationOutcome) (result error) {
	m.event("operation_start", "package", pkg.ID, "action", operation)
	defer func() {
		if result != nil {
			m.event("operation_error", "package", pkg.ID, "action", operation, "error", result.Error())
		}
	}()
	// Stage 1 — preconditions: validate the request against package policy.
	if err := m.checkPreconditions(operation, pkg); err != nil {
		return err
	}
	// Stage 2 — acquire: open the manager, reconcile state, stage the release.
	staged, err := m.stage(ctx, operation, pkg)
	if err != nil {
		return err
	}
	defer staged.release()
	// Stage 3 — commit: apply the staged release inside one transaction.
	return m.commit(ctx, operation, pkg, staged, outcome)
}

// checkPreconditions validates the requested operation against pure package
// policy before any filesystem work begins.
func (m Manager) checkPreconditions(operation string, pkg manifest.Package) error {
	if err := pkg.Validate(); err != nil {
		return err
	}
	if !pkg.Installable() {
		return fmt.Errorf("package %s is a candidate and cannot be installed", pkg.ID)
	}
	if err := platform.Check(pkg, m.platform); err != nil {
		m.event("compatibility_rejected", "package", pkg.ID, "decision", err.Error())
		return err
	}
	m.event("compatibility_allowed", "package", pkg.ID, "status", pkg.Review.Status, "platform", platform.Summary(m.platform))
	return validateDownloadURL(pkg.Release.URL)
}

// managerSession is an acquired manager environment: the base guard, the
// resolved manager host path, and the held exclusive lock.
type managerSession struct {
	guard       *safefs.Guard
	managerHost string
	lock        *os.File
}

func (s *managerSession) release() {
	releaseLock(s.lock)
}

// acquireManager opens the manager directory under its base guard, takes the
// exclusive lock, and recovers any interrupted transactions.
func (m Manager) acquireManager() (*managerSession, error) {
	guard, err := safefs.NewGuard(m.root(), []string{managerPath})
	if err != nil {
		return nil, err
	}
	host, err := guard.Resolve(managerPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(host, "installed"), 0700); err != nil {
		return nil, err
	}
	lock, err := acquireLock(filepath.Join(host, "lock"))
	if err != nil {
		return nil, err
	}
	if err := safefs.Recover(m.root(), host); err != nil {
		releaseLock(lock)
		return nil, err
	}
	return &managerSession{guard: guard, managerHost: host, lock: lock}, nil
}

// stagedRelease is an acquired manager session with a verified release staged
// in a temporary work directory.
type stagedRelease struct {
	session  *managerSession
	guard    *safefs.Guard
	old      *Installed
	existing []existingFile
	files    []storearchive.File
	work     string
}

// release discards the staging directory and releases the manager lock.
func (s *stagedRelease) release() {
	os.RemoveAll(s.work)
	s.session.release()
}

// stage acquires the manager, reconciles the requested operation with the
// installed state, and downloads, verifies, and extracts the release. Any
// failure cleans up the staging directory and the session.
func (m Manager) stage(ctx context.Context, operation string, pkg manifest.Package) (*stagedRelease, error) {
	session, err := m.acquireManager()
	if err != nil {
		return nil, err
	}
	staged := &stagedRelease{session: session}
	failed := true
	defer func() {
		if failed {
			staged.release()
		}
	}()

	old, err := loadState(session.guard, pkg.ID)
	if err != nil {
		return nil, err
	}
	if (operation == "install" || operation == "adopt") && old != nil {
		return nil, fmt.Errorf("package %s is already installed", pkg.ID)
	}
	if operation != "install" && operation != "adopt" && operation != "force-reinstall" && old == nil {
		return nil, fmt.Errorf("package %s is not installed", pkg.ID)
	}
	if operation == "repair" && (old.Manifest.Version != pkg.Version || old.Manifest.Release.SHA256 != pkg.Release.SHA256) {
		return nil, fmt.Errorf("repair requires the installed release; use update for a different version")
	}
	allowed := append([]string{}, pkg.Install.AllowedWritePaths...)
	if old != nil && old.Manifest.Install != nil {
		allowed = append(allowed, old.Manifest.Install.AllowedWritePaths...)
	}
	allowed = append(allowed, managerPath)
	guard, err := safefs.NewGuard(m.root(), allowed)
	if err != nil {
		return nil, err
	}
	destinationHost, err := guard.Resolve(pkg.Install.Destination)
	if err != nil {
		return nil, err
	}
	var existing []existingFile
	var existingBytes uint64
	if old == nil || operation == "force-reinstall" {
		existing, existingBytes, err = inventoryExisting(destinationHost, pkg.Install.Destination)
		if err != nil {
			return nil, fmt.Errorf("inventory pre-existing installation: %w", err)
		}
		m.event("adoption_inventory", "package", pkg.ID, "files", fmt.Sprint(len(existing)), "bytes", fmt.Sprint(existingBytes))
	}
	if operation == "install" && len(existing) > 0 {
		return nil, fmt.Errorf("an external installation exists; use Manage existing")
	}
	if operation == "adopt" && len(existing) == 0 {
		return nil, fmt.Errorf("no external installation was found; use Install")
	}
	if operation == "force-reinstall" && len(existing) == 0 && old == nil {
		return nil, fmt.Errorf("no stale, partial, or external installation was found; use Install")
	}
	availableBytes := safefs.AvailableBytes
	if m.AvailableBytes != nil {
		availableBytes = m.AvailableBytes
	}
	available, err := availableBytes(destinationHost)
	if err != nil {
		return nil, err
	}
	required := uint64(pkg.Release.Size) + uint64(pkg.Release.InstalledSize)*3 + existingBytes*2
	if available < required {
		return nil, fmt.Errorf("not enough free space: need %d bytes, have %d", required, available)
	}
	work, err := os.MkdirTemp(session.managerHost, "work-*")
	if err != nil {
		return nil, err
	}
	staged.work = work
	archivePath := filepath.Join(work, "release")
	client := m.Client
	if client == nil {
		client = defaultHTTPClient()
	}
	m.event("download_start", "package", pkg.ID, "url", pkg.Release.URL, "expected_bytes", fmt.Sprint(pkg.Release.Size))
	if err := download(ctx, client, *pkg.Release, archivePath); err != nil {
		return nil, fmt.Errorf("download release: %w", err)
	}
	m.event("download_verified", "package", pkg.ID, "bytes", fmt.Sprint(pkg.Release.Size), "sha256", pkg.Release.SHA256)
	files, err := storearchive.Extract(archivePath, pkg.Release.Format, filepath.Join(work, "staging"), pkg.Install.StripComponents, pkg.Release.InstalledSize)
	if err != nil {
		return nil, fmt.Errorf("extract release: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("release archive contains no installable files")
	}
	m.event("extraction_complete", "package", pkg.ID, "format", pkg.Release.Format, "files", fmt.Sprint(len(files)))
	if err := applyExecutableModes(files, pkg.Install.Executables); err != nil {
		return nil, err
	}
	if err := applyBinaryPatches(files, pkg.Install.BinaryPatches); err != nil {
		return nil, err
	}
	if !containsArchiveFile(files, pkg.Install.Launcher) {
		return nil, fmt.Errorf("release archive does not contain launcher %s", pkg.Install.Launcher)
	}
	staged.guard = guard
	staged.old = old
	staged.existing = existing
	staged.files = files
	failed = false
	return staged, nil
}

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

type recoveryBackup struct {
	Schema    string               `json:"schema"`
	PackageID string               `json:"package_id"`
	CreatedAt string               `json:"created_at"`
	Files     []recoveryBackupFile `json:"files"`
}

type recoveryBackupFile struct {
	OriginalPath string `json:"original_path"`
	BackupPath   string `json:"backup_path"`
	SHA256       string `json:"sha256"`
	Mode         uint32 `json:"mode"`
	Size         int64  `json:"size"`
}

func (m Manager) recoveryBackupDirectory(pkg manifest.Package) string {
	now := time.Now().UTC()
	if m.Now != nil {
		now = m.Now().UTC()
	}
	return managerPath + "/recovery-backups/" + pkg.ID + "/" + now.Format("20060102T150405.000000000Z")
}

func (m Manager) backupRecoveryDestination(tx *safefs.Transaction, pkg manifest.Package, existing []existingFile, directory string) error {
	now := time.Now().UTC()
	if m.Now != nil {
		now = m.Now().UTC()
	}
	record := recoveryBackup{
		Schema:    "org.knulli.app-store/recovery-backup/v1",
		PackageID: pkg.ID,
		CreatedAt: now.Format(time.RFC3339Nano),
		Files:     make([]recoveryBackupFile, 0, len(existing)),
	}
	for _, file := range existing {
		digest := sha256.Sum256([]byte(file.Virtual))
		backup := directory + "/files/" + hex.EncodeToString(digest[:])
		if err := tx.Copy(file.Host, backup, file.Mode); err != nil {
			return fmt.Errorf("create recovery backup for %s: %w", file.Virtual, err)
		}
		info, err := os.Stat(file.Host)
		if err != nil {
			return err
		}
		record.Files = append(record.Files, recoveryBackupFile{
			OriginalPath: file.Virtual,
			BackupPath:   backup,
			SHA256:       file.SHA256,
			Mode:         uint32(file.Mode.Perm()),
			Size:         info.Size(),
		})
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	manifestPath := directory + "/manifest.json"
	if err := tx.Write(manifestPath, append(data, '\n'), 0600); err != nil {
		return err
	}
	return nil
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

func applyBinaryPatches(files []storearchive.File, patches []manifest.BinaryPatch) error {
	byPath := make(map[string]*storearchive.File, len(files))
	for index := range files {
		byPath[files[index].Relative] = &files[index]
	}
	for _, patch := range patches {
		file := byPath[patch.Path]
		if file == nil {
			return fmt.Errorf("binary patch target is absent: %s", patch.Path)
		}
		before, _ := hex.DecodeString(patch.BeforeHex)
		after, _ := hex.DecodeString(patch.AfterHex)
		data, err := os.ReadFile(file.Path)
		if err != nil {
			return err
		}
		end := patch.Offset + int64(len(before))
		if patch.Offset < 0 || end > int64(len(data)) || !bytes.Equal(data[patch.Offset:end], before) {
			return fmt.Errorf("binary patch source mismatch for %s at offset %d", patch.Path, patch.Offset)
		}
		copy(data[patch.Offset:end], after)
		digest := sha256.Sum256(data)
		actual := hex.EncodeToString(digest[:])
		if actual != patch.SHA256 {
			return fmt.Errorf("binary patch result mismatch for %s: expected %s, got %s", patch.Path, patch.SHA256, actual)
		}
		if err := os.WriteFile(file.Path, data, file.Mode); err != nil {
			return err
		}
		file.SHA256 = actual
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
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
			return nil, fmt.Errorf("another package operation is active")
		}
		return nil, err
	}
	return file, nil
}

func releaseLock(file *os.File) {
	syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	file.Close()
}
