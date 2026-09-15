package appstore

import (
	"context"
	"fmt"
	"sort"

	"github.com/jellydn/knulli-app-store/internal/catalog"
	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

type Action string

const (
	Install   Action = "install"
	Update    Action = "update"
	Repair    Action = "repair"
	Uninstall Action = "uninstall"
)

type Item struct {
	Package          manifest.Package
	Installed        bool
	InstalledVersion string
	Healthy          bool
	Compatible       bool
	Compatibility    string
	Actions          []Action
}

type Backend interface {
	Items(context.Context) ([]Item, error)
	Execute(context.Context, string, Action, func(string)) error
}

type Service struct {
	index   catalog.Index
	manager installer.Manager
}

func Open(indexPath string, manager installer.Manager) (*Service, error) {
	index, err := catalog.Load(indexPath)
	if err != nil {
		return nil, err
	}
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
		item.Compatible, item.Compatibility = compatibility(entry.Package, s.manager.Platform)
		item.Actions = actions(item)
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
	progress("Starting " + string(action))
	switch action {
	case Install:
		err = s.manager.Install(ctx, entry.Package)
	case Update:
		err = s.manager.Update(ctx, entry.Package)
	case Repair:
		err = s.manager.Repair(ctx, entry.Package)
	case Uninstall:
		err = s.manager.Uninstall(id)
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	if err != nil {
		return err
	}
	progress("Completed " + string(action))
	return nil
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
	return true, "Compatible with detected platform"
}

func actions(item Item) []Action {
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
		return []Action{Install}
	}
	return nil
}

func containsAction(actions []Action, wanted Action) bool {
	for _, action := range actions {
		if action == wanted {
			return true
		}
	}
	return false
}
