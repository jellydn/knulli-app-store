package installer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	storearchive "github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

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
