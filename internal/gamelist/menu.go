// Package gamelist owns EmulationStation menu ownership: which gamelist
// entries the installer may create, replace, or remove, and applying that
// effect inside a transaction. Entries this installer does not own are never
// edited or deleted.
package gamelist

import (
	"fmt"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

// StepKind names one gamelist edit.
type StepKind string

const (
	StepAdd     StepKind = "add"
	StepReplace StepKind = "replace"
	StepRemove  StepKind = "remove"
)

// Step is one gamelist edit: the entry to write (add, replace) or remove,
// plus, for replace, the exact owned entry it replaces.
type Step struct {
	Kind     StepKind
	Menu     manifest.Menu
	Previous *manifest.Menu
}

// Plan is the complete menu effect of one lifecycle operation, derived
// without touching the filesystem. Applying it is the only place gamelist
// files change.
type Plan struct {
	Steps []Step
}

// Derive builds the menu effect of applying menu over a previously owned
// entry. oldOwned and oldMenu describe the ownership recorded in the
// installed state; menu is the manifest's menu, nil when the release has
// none. Ownership is honored exactly: a release without a menu releases
// ownership, a release with the same gamelist and path replaces in place,
// and anything else is a remove of the owned entry followed by an add.
func Derive(oldOwned bool, oldMenu, menu *manifest.Menu) Plan {
	var plan Plan
	previous := ownedEntry(oldOwned, oldMenu)
	if previous != nil && (menu == nil || !sameEntry(*previous, *menu)) {
		plan.Steps = append(plan.Steps, Step{Kind: StepRemove, Menu: *previous})
	}
	if menu == nil {
		return plan
	}
	if previous != nil && sameEntry(*previous, *menu) {
		plan.Steps = append(plan.Steps, Step{Kind: StepReplace, Menu: *menu, Previous: previous})
	} else {
		plan.Steps = append(plan.Steps, Step{Kind: StepAdd, Menu: *menu})
	}
	return plan
}

// ownedEntry returns the entry this installer owns, or nil.
func ownedEntry(owned bool, menu *manifest.Menu) *manifest.Menu {
	if !owned || menu == nil {
		return nil
	}
	return menu
}

// sameEntry reports whether two menu entries address the same slot: the same
// gamelist file and the same path. Name and description may change in place.
func sameEntry(left, right manifest.Menu) bool {
	return left.Gamelist == right.Gamelist && left.Path == right.Path
}

// Apply executes the plan inside the transaction. It reports whether any
// gamelist file changed and whether the installer owns the package's menu
// entry afterwards. Ownership transfers only when the owning write actually
// happened: an add or replace that found nothing to do leaves the entry
// unowned, and a removal always releases ownership.
func Apply(tx *safefs.Transaction, guard *safefs.Guard, plan Plan) (changed bool, owned bool, err error) {
	for _, step := range plan.Steps {
		stepChanged, err := applyStep(tx, guard, step)
		if err != nil {
			return changed, false, err
		}
		changed = changed || stepChanged
		switch step.Kind {
		case StepAdd, StepReplace:
			owned = stepChanged
		case StepRemove:
			owned = false
		}
	}
	return changed, owned, nil
}

func applyStep(tx *safefs.Transaction, guard *safefs.Guard, step Step) (bool, error) {
	switch step.Kind {
	case StepAdd:
		return addEntry(tx, guard, step.Menu)
	case StepReplace:
		return replaceEntry(tx, guard, *step.Previous, step.Menu)
	case StepRemove:
		return removeEntry(tx, guard, step.Menu)
	}
	return false, fmt.Errorf("unknown menu step %q", step.Kind)
}
