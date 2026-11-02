package ui

import "github.com/jellydn/knulli-app-store/internal/appstore"

// Tab is one view of the catalogue. The index lists every package the app
// knows about; a tab narrows that list to the packages a user is looking for,
// so a catalogue that outgrows one screen can still be read in the order the
// work happens.
type Tab int

const (
	// TabAll is every package the index lists. It leads because the app lands
	// there, so the default view hides nothing and a tab is always a deliberate
	// narrowing.
	TabAll Tab = iota
	// TabReady is every package with something waiting: a reviewed, compatible
	// package to install, or a destination that already holds unmanaged files
	// to adopt.
	TabReady
	// TabInstalled is every package the app manages on this device, the ones
	// failing their health check included, because repairing a package starts
	// from the same row.
	TabInstalled
)

// TabOrder is the bar's left-to-right order, which is also the order left and
// right walk. It is a slice rather than a switch so the bar and the cycle
// cannot disagree about how many tabs exist or what order they are in.
var TabOrder = []Tab{TabAll, TabReady, TabInstalled}

// Label names a tab in the bar. Labels are short because they share one row
// with nothing else, and uppercase because the whole screen is.
func (tab Tab) Label() string {
	switch tab {
	case TabReady:
		return "READY"
	case TabInstalled:
		return "INSTALLED"
	default:
		return "ALL"
	}
}

// Match reports whether a package belongs to a tab. Membership is read from the
// typed verdict rather than from the rendered state label, so a tab cannot
// drift from the state it claims to filter on.
func (tab Tab) Match(item appstore.Item) bool {
	switch tab {
	case TabReady:
		return item.Verdict.State == appstore.StateAvailable || item.Verdict.State == appstore.StateExternal
	case TabInstalled:
		return item.Verdict.State == appstore.StateInstalled || item.Verdict.State == appstore.StateIssue
	default:
		return true
	}
}

// Tabs is the catalogue's tab selection: which tab is showing, and where the
// user was in each one. It is pure state, so the switching rule is testable
// without a renderer or a device.
type Tabs struct {
	active Tab
	// at is the package each tab was last left on. A view selector that forgot
	// would send a reader back to the top of a list they had already crossed,
	// which is the one thing coming back to a tab should not do.
	at       map[view]string
	category Category
}

type view struct {
	tab      Tab
	category Category
}

type Category int

const (
	CategoryAll Category = iota
	CategoryGames
	CategoryApps
)

var CategoryOrder = []Category{CategoryAll, CategoryGames, CategoryApps}

func (category Category) Label() string {
	switch category {
	case CategoryGames:
		return "GAMES"
	case CategoryApps:
		return "APPS"
	default:
		return "ALL TYPES"
	}
}

func (category Category) Match(item appstore.Item) bool {
	switch category {
	case CategoryGames:
		return item.Package.Type == "game"
	case CategoryApps:
		return item.Package.Type == "utility" || item.Package.Type == "theme" || item.Package.Type == "integration"
	default:
		return true
	}
}

func (tabs Tabs) Category() Category { return tabs.category }

func (tabs *Tabs) CycleCategory(delta int) {
	tabs.category = Category(wrap(int(tabs.category)+delta, len(CategoryOrder)))
}

// Active is the tab whose packages the list shows.
func (tabs Tabs) Active() Tab {
	return tabs.active
}

// Remember records the package the active tab is being left on. An empty id —
// a tab with no rows — remembers nothing, so leaving an empty tab cannot erase
// where the reader stopped in the tab that has rows.
func (tabs *Tabs) Remember(id string) {
	if id == "" {
		return
	}
	if tabs.at == nil {
		tabs.at = make(map[view]string)
	}
	tabs.at[view{tabs.active, tabs.category}] = id
}

// Remembered is the package the active tab was last left on, or empty when the
// user has not been in it before.
func (tabs Tabs) Remembered() string {
	return tabs.at[view{tabs.active, tabs.category}]
}

// Cycle steps to the previous or next tab. The bar is a ring: right from the
// last tab reaches the first and left from the first reaches the last, so
// neither direction ever dead-ends on the user.
func (tabs *Tabs) Cycle(delta int) {
	if len(TabOrder) == 0 {
		return
	}
	position := 0
	for index, tab := range TabOrder {
		if tab == tabs.active {
			position = index
			break
		}
	}
	tabs.active = TabOrder[wrap(position+delta, len(TabOrder))]
}
