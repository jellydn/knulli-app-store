package appstore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/catalog"
	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

type Action string

const (
	Install        Action = "install"
	Adopt          Action = "adopt"
	Update         Action = "update"
	Repair         Action = "repair"
	ForceReinstall Action = "force-reinstall"
	Uninstall      Action = "uninstall"
)

type Item struct {
	Package          manifest.Package
	Installed        bool
	InstalledVersion string
	Healthy          bool
	HealthReason     string
	DeviceTested     bool
	PreExisting      bool
	Compatible       bool
	Compatibility    string
	Actions          []Action
	RecoveryReason   string
	RecoverySummary  string
	RecoveryAllowed  bool
	RecoveryActive   bool
	RetryAction      Action
}

type Backend interface {
	Items(context.Context) ([]Item, error)
	Execute(context.Context, string, Action, func(string)) error
	ExportDiagnostics(context.Context) (string, error)
	SetPlatform(platform.Info)
}

func (s *Service) SetPlatform(info platform.Info) {
	s.manager.Platform = info
}

type Service struct {
	index   catalog.Index
	manager installer.Manager
}

func Open(indexPath string, manager installer.Manager) (*Service, error) {
	index, err := catalog.Load(indexPath)
	if err != nil {
		manager.Diagnostics.Event("catalogue_error", "path", indexPath, "error", err.Error())
		return nil, err
	}
	manager.Diagnostics.Event("catalogue_loaded", "path", indexPath, "packages", fmt.Sprint(len(index.Packages)))
	return &Service{index: index, manager: manager}, nil
}

func (s *Service) Items(ctx context.Context) ([]Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(s.index.Packages))
	for _, entry := range s.index.Packages {
		status, err := s.manager.Status(entry.ID)
		if err != nil {
			return nil, fmt.Errorf("read %s status: %w", entry.ID, err)
		}
		item := Item{Package: entry.Package, Installed: status.Installed, InstalledVersion: status.Version, Healthy: status.Healthy}
		item.DeviceTested = entry.Package.DeviceTested(s.manager.Platform.Firmware, s.manager.Platform.Arch, s.manager.Platform.Device, s.manager.Platform.Resolution)
		if len(status.Issues) > 0 {
			item.HealthReason = status.Issues[0].String()
		}
		if !item.Installed {
			item.PreExisting, err = s.manager.PreExisting(entry.Package)
			if err != nil {
				return nil, fmt.Errorf("inspect %s destination: %w", entry.ID, err)
			}
			if item.PreExisting {
				recovery, recoveryErr := s.manager.RecoveryStatus(entry.Package)
				if recoveryErr != nil {
					return nil, fmt.Errorf("inspect %s recovery state: %w", entry.ID, recoveryErr)
				}
				item.RecoveryReason = recovery.Reason
				item.RecoveryAllowed = recovery.ForceAllowed
				item.RecoveryActive = recovery.Active
			}
		}
		lifecycle, lifecycleErr := s.manager.LifecycleState(entry.ID)
		if lifecycleErr != nil {
			return nil, fmt.Errorf("read %s lifecycle state: %w", entry.ID, lifecycleErr)
		}
		item.Compatible, item.Compatibility = compatibility(entry.Package, s.manager.Platform)
		if lifecycle != nil {
			if item.PreExisting && lifecycle.RequestedOperation == string(Adopt) && lifecycle.Failure != "" {
				item.RecoveryReason = lifecycle.Failure
				item.RecoveryAllowed = lifecycle.ForceAllowed
			}
			item.RetryAction = validRetryAction(item, Action(lifecycle.RetryTarget))
		}
		if item.RecoveryReason != "" {
			preserved := 0
			if entry.Package.Install != nil {
				preserved = len(entry.Package.Install.Preserve)
			}
			item.RecoverySummary = fmt.Sprintf("Replaces reviewed app files; preserves %d declared data paths; backs up the complete existing destination for manual restore.", preserved)
		}
		s.manager.Diagnostics.Event("compatibility_decision", "package", entry.ID, "allowed", fmt.Sprint(item.Compatible), "decision", item.Compatibility)
		item.Actions = actions(item)
		if lifecycle != nil || item.HealthReason != "" {
			s.manager.LifecycleEvent(installer.LifecycleState{PackageID: entry.ID, RequestedOperation: retryOperation(lifecycle), DetectedInstallType: installType(item), RetryTarget: string(item.RetryAction), Failure: lifecycleFailure(lifecycle)}, item.HealthReason)
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Package.Name < items[j].Package.Name })
	return items, nil
}

func (s *Service) Execute(ctx context.Context, id string, action Action, progress func(string)) error {
	entry, found := s.find(id)
	if !found {
		return fmt.Errorf("package %s is not in the catalogue", id)
	}
	items, err := s.Items(ctx)
	if err != nil {
		return err
	}
	var selected *Item
	for index := range items {
		if items[index].Package.ID == id {
			selected = &items[index]
			break
		}
	}
	if selected == nil || !containsAction(selected.Actions, action) {
		return fmt.Errorf("%s is not available for %s", action, id)
	}
	message := operationMessage(action, entry.Package.Name)
	progress(message)
	s.manager.Diagnostics.Event("action_selected", "package", id, "action", string(action))
	lifecycle := installer.LifecycleState{PackageID: id, RequestedOperation: string(action), DetectedInstallType: installType(*selected), RetryTarget: string(action)}
	if err := s.manager.RecordLifecycle(lifecycle); err != nil {
		return fmt.Errorf("record lifecycle state: %w", err)
	}
	s.manager.LifecycleEvent(lifecycle, selected.HealthReason)
	manager := s.manager
	outcome := installer.OperationOutcome{}
	manager.Outcome = func(value installer.OperationOutcome) { outcome = value }
	switch action {
	case Install:
		err = manager.Install(ctx, entry.Package)
	case Adopt:
		err = manager.Adopt(ctx, entry.Package)
	case Update:
		err = manager.Update(ctx, entry.Package)
	case Repair:
		err = manager.Repair(ctx, entry.Package)
	case ForceReinstall:
		err = manager.ForceReinstall(ctx, entry.Package)
	case Uninstall:
		err = manager.UninstallContext(ctx, id)
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	if err != nil {
		lifecycle.Failure = err.Error()
		lifecycle.ForceAllowed = action == Adopt && recoverableAdoptionFailure(err)
		if stateErr := s.manager.RecordLifecycle(lifecycle); stateErr != nil {
			return fmt.Errorf("%w; record retry state: %v", err, stateErr)
		}
		s.manager.LifecycleEvent(lifecycle, selected.HealthReason)
		return err
	}
	if err := s.manager.ClearLifecycle(id); err != nil {
		return fmt.Errorf("clear lifecycle state after completed %s: %w", action, err)
	}
	progress(completionMessage(action, outcome))
	return nil
}

func (s *Service) ExportDiagnostics(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	lines := []string{"Platform: " + platform.Summary(s.manager.Platform), fmt.Sprintf("Catalogue packages: %d", len(s.index.Packages))}
	for _, entry := range s.index.Packages {
		lines = append(lines, strings.Join([]string{"Package:", entry.ID, entry.Package.Version, entry.Package.Review.Status}, " "))
	}
	path, err := s.manager.Diagnostics.Export(lines)
	if err != nil {
		return "", err
	}
	s.manager.Diagnostics.Event("diagnostics_exported", "path", path)
	return path, nil
}

func (s *Service) find(id string) (catalog.Entry, bool) {
	for _, entry := range s.index.Packages {
		if entry.ID == id {
			return entry, true
		}
	}
	return catalog.Entry{}, false
}

func compatibility(pkg manifest.Package, current platform.Info) (bool, string) {
	if !pkg.Installable() {
		if pkg.Review.Approval != nil {
			return false, "Community approved; installation is blocked by technical review"
		}
		return false, "Candidate: compatibility is not approved"
	}
	if err := platform.Check(pkg, current); err != nil {
		return false, err.Error()
	}
	if pkg.Experimental() {
		return true, fmt.Sprintf("Experimental compatibility: Knulli identity confirmed from %s; no minimum version is claimed; device=%s architecture=%s resolution=%s source=%s version=%s", current.FirmwareSource, current.Device, current.Arch, current.Resolution, current.ResolutionSource, current.Version)
	}
	return true, "Compatible with detected platform"
}

func actions(item Item) []Action {
	if item.RetryAction != "" {
		base := actionsWithoutRetry(item)
		result := []Action{item.RetryAction}
		for _, action := range base {
			if action != item.RetryAction {
				result = append(result, action)
			}
		}
		return result
	}
	return actionsWithoutRetry(item)
}

func actionsWithoutRetry(item Item) []Action {
	if item.Installed {
		result := []Action{Uninstall}
		if item.Package.Installable() && item.Compatible {
			if item.InstalledVersion != item.Package.Version {
				result = append([]Action{Update}, result...)
			}
			result = append(result, Repair)
		}
		return result
	}
	if item.Package.Installable() && item.Compatible {
		if item.PreExisting {
			if item.RecoveryActive {
				return nil
			}
			if item.RecoveryAllowed {
				return []Action{Adopt, ForceReinstall}
			}
			return []Action{Adopt}
		}
		return []Action{Install}
	}
	return nil
}

func validRetryAction(item Item, retry Action) Action {
	switch retry {
	case Install:
		if !item.Installed && (!item.PreExisting || item.RecoveryReason != "") && item.Package.Installable() && item.Compatible {
			return retry
		}
	case Adopt:
		if !item.Installed && item.PreExisting && !item.RecoveryActive && item.Package.Installable() && item.Compatible {
			return retry
		}
	case Update:
		if item.Installed && item.InstalledVersion != item.Package.Version && item.Compatible {
			return retry
		}
	case Repair:
		if item.Installed && item.Compatible {
			return retry
		}
	case ForceReinstall:
		if !item.Installed && item.PreExisting && !item.RecoveryActive && item.Compatible {
			return retry
		}
	case Uninstall:
		if item.Installed {
			return retry
		}
	}
	return ""
}

func installType(item Item) string {
	if item.Installed {
		return "managed"
	}
	if item.RecoveryActive || (item.PreExisting && item.RecoveryReason != "") {
		return "stale-or-partial"
	}
	if item.PreExisting {
		return "external"
	}
	return "absent"
}

func retryOperation(state *installer.LifecycleState) string {
	if state == nil {
		return ""
	}
	return state.RequestedOperation
}

func lifecycleFailure(state *installer.LifecycleState) string {
	if state == nil {
		return ""
	}
	return state.Failure
}

func recoverableAdoptionFailure(err error) bool {
	var conflict *installer.AdoptionConflictError
	return errors.As(err, &conflict)
}

func operationMessage(action Action, name string) string {
	if action == Adopt {
		return "Inventorying and backing up existing " + name
	}
	if action == Install || action == Update || action == Repair || action == ForceReinstall {
		return "Downloading, verifying, and applying " + name
	}
	return "Removing managed files and restoring backups for " + name
}

func completionMessage(action Action, outcome installer.OperationOutcome) string {
	message := actionLabel(action) + " completed"
	if outcome.GameListRefreshAccepted {
		return message + "; game list refresh requested"
	}
	if outcome.RestartRequired {
		return message + "; restart required to update game list"
	}
	return message
}

func actionLabel(action Action) string {
	if action == Adopt {
		return "Manage existing install"
	}
	if action == ForceReinstall {
		return "Force reinstall"
	}
	value := string(action)
	if value == "" {
		return "Action"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func containsAction(actions []Action, wanted Action) bool {
	for _, action := range actions {
		if action == wanted {
			return true
		}
	}
	return false
}
