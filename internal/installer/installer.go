// Package installer owns every side effect a package operation performs. This
// file is the orchestrator: the manager, the lifecycle entry point, and the
// three-stage apply that delegates to stage.go and commit.go. The rest of the
// package is split by concern, so a change lands next to the behaviour it
// affects: staging and the manager session in stage.go, the transaction that
// commits a release in commit.go, removal in uninstall.go, destination
// inventory in adoption.go, force-reinstall backups in recovery.go, the
// executable, patch, and path helpers in files.go, the installed-state record
// and health checks in state.go and status.go, and the health-check cache in
// cache.go.
package installer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

type Manager struct {
	Root           string
	Client         *http.Client
	RefreshClient  *http.Client
	RefreshURL     string
	Diagnostics    *diagnostics.Log
	Now            func() time.Time
	AvailableBytes func(string) (uint64, error)

	platform    platform.Info
	statusCache *StatusCache
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
	m.platform = info.Clone()
	return m
}

// Platform returns the platform this manager validates packages against.
func (m Manager) Platform() platform.Info {
	return m.platform
}

// WithStatusCache returns a copy of the manager that reuses health-check
// results for packages whose recorded state has not changed. A manager built
// without one verifies every managed file on every call.
func (m Manager) WithStatusCache(cache *StatusCache) Manager {
	m.statusCache = cache
	return m
}

// Apply runs one package lifecycle operation: download, verify, and commit the
// reviewed release, rolling back every change on failure. The outcome is
// returned after a commit; a failed operation returns a zero outcome with the
// error.
func (m Manager) Apply(ctx context.Context, op Op, pkg manifest.Package) (OperationOutcome, error) {
	switch op {
	case OpInstall, OpAdopt, OpUpdate, OpRepair, OpForceReinstall:
	default:
		return OperationOutcome{}, fmt.Errorf("unsupported lifecycle operation %q", op)
	}
	var outcome OperationOutcome
	// The operation can change what a health check sees, and a failed one can
	// leave a destination half-written before the rollback, so the cached result
	// goes whatever the outcome.
	defer m.statusCache.Invalidate(pkg.ID)
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

func (m Manager) event(name string, fields ...string) {
	if m.Diagnostics != nil {
		m.Diagnostics.Event(name, fields...)
	}
}

func (m Manager) root() string {
	if m.Root == "" {
		return "/"
	}
	return m.Root
}
