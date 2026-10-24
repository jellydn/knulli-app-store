package appstore

import (
	"context"
	"fmt"

	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/manifest"
)

const StoreID = "io.github.jellydn.knulli-app-store"
const CheckUpdate Action = "check for updates"

// Build variables are stamped alongside the catalogue key in device builds.
var StoreVersion = "development"
var StoreTarget string

type selfUpdate struct {
	version string
	target  string
	pending *installer.StoreUpdate
	restart bool
}

func (s *Service) EnableSelfUpdate(version, target string) {
	s.store = &selfUpdate{version: version, target: target}
}

func (s *Service) storeItem() Item {
	store := s.store
	version := store.version
	actions := []Action{CheckUpdate}
	summary := "Update this Store from official GitHub releases. Installed apps and settings stay unchanged."
	warning := "Checks official GitHub releases, including experimental builds. Updating replaces the Store binaries and bundled catalogue, not your installed apps or settings. Exit and reopen the Store after updating."
	state := StateInstalled
	if store.pending != nil {
		version = store.pending.Version
		actions = []Action{Update, CheckUpdate}
		state = StateAvailable
	}
	if store.restart {
		actions = nil
		state = StateInstalled
		warning = "Store update completed. Exit and reopen the Store to use the new version."
		summary = warning
	}
	return Item{
		Package:   manifest.Package{ID: StoreID, Name: "Knulli App Store", Version: version, Type: "utility", Summary: summary, Repository: "https://github.com/jellydn/knulli-app-store", Review: manifest.Review{Status: "experimental"}, Install: &manifest.Install{Warning: warning}},
		Installed: true, InstalledVersion: store.version, Healthy: true, Compatible: true,
		Compatibility: warning, Actions: actions, Verdict: Verdict{State: state, Actions: actions},
	}
}

func (s *Service) executeStore(ctx context.Context, action Action, progress func(string)) error {
	store := s.store
	if !containsAction(s.storeItem().Actions, action) {
		return fmt.Errorf("%s is not available for the Store", action)
	}
	if action == CheckUpdate {
		// A failed recheck must not leave an older offer enabled.
		store.pending = nil
		progress("Checking official Store releases")
		update, err := s.manager.CheckStoreUpdate(ctx, store.target, store.version)
		if err != nil {
			return err
		}
		store.pending = update
		if update == nil {
			progress("Store is up to date")
		} else {
			progress("Store update available: " + update.Version + "; select Update to confirm")
		}
		return nil
	}
	progress("Downloading and verifying the Store update")
	if err := s.manager.UpdateStore(ctx, *store.pending); err != nil {
		return err
	}
	store.restart = true
	progress("Store update completed; exit and reopen the Store")
	return nil
}
