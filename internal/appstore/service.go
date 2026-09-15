package appstore

import (
	"context"
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
	Install   Action = "install"
	Adopt     Action = "adopt"
	Update    Action = "update"
	Repair    Action = "repair"
	Uninstall Action = "uninstall"
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
	Verdict          Verdict
}

type Backend interface {
	Items(context.Context) ([]Item, error)
	Execute(context.Context, string, Action, func(string)) error
	ExportDiagnostics(context.Context) (string, error)
	SetPlatform(platform.Info)
}

func (s *Service) SetPlatform(info platform.Info) {
	s.manager = s.manager.WithPlatform(info)
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
		item, err := s.item(ctx, entry)
		if err != nil {
			return nil, err
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
	item, err := s.item(ctx, entry)
	if err != nil {
		return err
	}
	if !containsAction(item.Actions, action) {
		return fmt.Errorf("%s is not available for %s", action, id)
	}
	message := operationMessage(action, entry.Package.Name)
	progress(message)
	s.manager.Diagnostics.Event("action_selected", "package", id, "action", string(action))
	var outcome installer.OperationOutcome
	switch action {
	case Install:
		outcome, err = s.manager.Apply(ctx, installer.OpInstall, entry.Package)
	case Adopt:
		outcome, err = s.manager.Apply(ctx, installer.OpAdopt, entry.Package)
	case Update:
		outcome, err = s.manager.Apply(ctx, installer.OpUpdate, entry.Package)
	case Repair:
		outcome, err = s.manager.Apply(ctx, installer.OpRepair, entry.Package)
	case Uninstall:
		outcome, err = s.manager.Uninstall(ctx, id)
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	if err != nil {
		return err
	}
	progress(completionMessage(action, outcome))
	return nil
}

func (s *Service) ExportDiagnostics(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	lines := []string{"Platform: " + platform.Summary(s.manager.Platform()), fmt.Sprintf("Catalogue packages: %d", len(s.index.Packages))}
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

// item derives the catalogue item for one entry: status, compatibility, and
// the actions the current package state allows.
func (s *Service) item(ctx context.Context, entry catalog.Entry) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	status, err := s.manager.Status(entry.ID)
	if err != nil {
		return Item{}, fmt.Errorf("read %s status: %w", entry.ID, err)
	}
	item := Item{Package: entry.Package, Installed: status.Installed, InstalledVersion: status.Version, Healthy: status.Healthy}
	item.DeviceTested = entry.Package.DeviceTested(s.manager.Platform().Firmware, s.manager.Platform().Arch, s.manager.Platform().Device, s.manager.Platform().Resolution)
	if len(status.Issues) > 0 {
		item.HealthReason = status.Issues[0].String()
	}
	if !item.Installed {
		item.PreExisting, err = s.manager.PreExisting(entry.Package)
		if err != nil {
			return Item{}, fmt.Errorf("inspect %s destination: %w", entry.ID, err)
		}
	}
	item.Verdict = assess(entry.Package, s.manager.Platform(), status, item.PreExisting)
	item.Compatible = actionable(item.Verdict.State)
	item.Compatibility = item.Verdict.Message()
	s.manager.Diagnostics.Event("compatibility_decision", "package", entry.ID, "allowed", fmt.Sprint(item.Compatible), "decision", item.Compatibility)
	item.Actions = item.Verdict.Actions
	return item, nil
}

// actionable reports whether the verdict state allows lifecycle actions.
func actionable(state State) bool {
	return state != StateCandidate && state != StateIncompatible
}



func operationMessage(action Action, name string) string {
	if action == Adopt {
		return "Inventorying and backing up existing " + name
	}
	if action == Install || action == Update || action == Repair {
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
