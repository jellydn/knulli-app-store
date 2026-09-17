package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

type fakeBackend struct {
	items      []appstore.Item
	err        error
	action     appstore.Action
	completion string
}

func (fake *fakeBackend) SetPlatform(platform.Info) {}

func (fake *fakeBackend) ExportDiagnostics(context.Context) (string, error) {
	return "/userdata/system/knulli-app-store/diagnostics/test.txt", fake.err
}

func (fake *fakeBackend) Items(context.Context) ([]appstore.Item, error) {
	return fake.items, nil
}

func (fake *fakeBackend) Execute(_ context.Context, _ string, action appstore.Action, progress func(string)) error {
	fake.action = action
	progress("Working")
	if fake.completion != "" {
		progress(fake.completion)
	}
	return fake.err
}

func TestModelKeepsRestartRequiredCompletion(t *testing.T) {
	backend := &fakeBackend{
		items:      []appstore.Item{{Package: manifest.Package{ID: "org.example.alpha", Name: "Alpha"}, Actions: []appstore.Action{appstore.Uninstall}}},
		completion: "Uninstall completed; restart required to update game list",
	}
	model := New(backend)
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	model.Select(context.Background())
	model.Select(context.Background())
	model.Select(context.Background())
	waitForModel(t, model)
	if model.Error != "" || model.Message != backend.completion {
		t.Fatalf("restart outcome was lost: message=%q error=%q", model.Message, model.Error)
	}
}

func TestModelWrapsNavigationAndRequiresConfirmation(t *testing.T) {
	backend := &fakeBackend{items: []appstore.Item{
		{Package: manifest.Package{ID: "org.example.alpha", Name: "Alpha"}},
		{Package: manifest.Package{ID: "org.example.beta", Name: "Beta"}, Actions: []appstore.Action{appstore.Install}},
	}}
	model := New(backend)
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	model.Move(-1)
	if model.Selected != 1 {
		t.Fatalf("expected wrapped selection, got %d", model.Selected)
	}
	model.Select(context.Background())
	if model.Focus != Actions {
		t.Fatalf("expected action focus, got %d", model.Focus)
	}
	model.Select(context.Background())
	if model.Focus != Confirm || model.Busy {
		t.Fatal("action must wait for explicit confirmation")
	}
	model.Select(context.Background())
	waitForModel(t, model)
	if backend.action != appstore.Install || model.Error != "" {
		t.Fatalf("unexpected operation result: %s, %s", backend.action, model.Error)
	}
}

func TestModelShowsOperationFailure(t *testing.T) {
	backend := &fakeBackend{
		items: []appstore.Item{{Package: manifest.Package{ID: "org.example.alpha", Name: "Alpha"}, Actions: []appstore.Action{appstore.Uninstall}}},
		err:   errors.New("rollback completed after download failure"),
	}
	model := New(backend)
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	model.Select(context.Background())
	model.Select(context.Background())
	model.Select(context.Background())
	waitForModel(t, model)
	if model.Error != backend.err.Error() || model.Message != "Uninstall failed" {
		t.Fatalf("failure was not clear: message=%q error=%q", model.Message, model.Error)
	}
}

func TestIssueSelectionOpensHealthBeforeActions(t *testing.T) {
	backend := &fakeBackend{items: []appstore.Item{{
		Package: manifest.Package{ID: "app.romm.grout", Name: "Grout"}, Healthy: false,
		HealthReason: "mode changed: /userdata/roms/tools/Grout/grout (expected 0755, got 0777)",
		Actions:      []appstore.Action{appstore.Repair, appstore.Uninstall},
	}}}
	model := New(backend)
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	model.Select(context.Background())
	if model.Focus != Health {
		t.Fatalf("Issue opened focus %v, want health", model.Focus)
	}
	model.Select(context.Background())
	if model.Focus != Actions {
		t.Fatalf("health confirmation opened focus %v, want actions", model.Focus)
	}
}

func TestForceReinstallRequiresTwoConfirmationSteps(t *testing.T) {
	backend := &fakeBackend{items: []appstore.Item{{
		Package: manifest.Package{ID: "org.example.alpha", Name: "Alpha"},
		Actions: []appstore.Action{appstore.Adopt, appstore.ForceReinstall},
	}}}
	model := New(backend)
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	model.Select(context.Background())
	model.Move(1)
	model.Select(context.Background())
	if model.Focus != Confirm || model.Busy {
		t.Fatalf("force reinstall skipped first review: focus=%d busy=%v", model.Focus, model.Busy)
	}
	model.Select(context.Background())
	if model.Focus != ForceConfirm || model.Busy {
		t.Fatalf("force reinstall skipped second confirmation: focus=%d busy=%v", model.Focus, model.Busy)
	}
	model.Select(context.Background())
	waitForModel(t, model)
	if backend.action != appstore.ForceReinstall {
		t.Fatalf("executed %q, want force reinstall", backend.action)
	}
}

func TestModelDoesNotOfferCandidateAction(t *testing.T) {
	backend := &fakeBackend{items: []appstore.Item{{Package: manifest.Package{ID: "org.example.candidate", Name: "Candidate"}}}}
	model := New(backend)
	if err := model.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	model.Select(context.Background())
	if model.Focus != Browse || model.Message != "No safe action is available" {
		t.Fatal("candidate should remain read-only")
	}
}

func TestModelExportsDiagnosticsPath(t *testing.T) {
	model := New(&fakeBackend{})
	model.ExportDiagnostics(context.Background())
	if model.Error != "" || !strings.Contains(model.Message, "/userdata/system/knulli-app-store/diagnostics/test.txt") {
		t.Fatalf("diagnostics result was not useful: message=%q error=%q", model.Message, model.Error)
	}
}

// The list window is a function of the selection and the row count, so a list
// that fits shows every row and a longer one keeps the selection on screen.
func TestWindowFollowsTheSelectionAndStopsAtTheEndOfTheList(t *testing.T) {
	model := &Model{Items: make([]appstore.Item, 12)}
	for _, test := range []struct{ selected, rows, start, end int }{
		{selected: 0, rows: 5, start: 0, end: 5},
		{selected: 3, rows: 5, start: 1, end: 6},
		{selected: 5, rows: 5, start: 3, end: 8},
		{selected: 11, rows: 5, start: 7, end: 12},
		{selected: 11, rows: 20, start: 0, end: 12},
	} {
		model.Selected = test.selected
		start, end := model.Window(test.rows)
		if start != test.start || end != test.end {
			t.Fatalf("selection %d with %d rows: window %d-%d, want %d-%d", test.selected, test.rows, start, end, test.start, test.end)
		}
		if test.selected < start || test.selected >= end {
			t.Fatalf("selection %d is outside the window %d-%d", test.selected, start, end)
		}
	}
	empty := &Model{}
	if start, end := empty.Window(5); start != 0 || end != 0 {
		t.Fatalf("empty catalogue window = %d-%d", start, end)
	}
	if start, end := model.Window(0); start != 0 || end != 0 {
		t.Fatalf("window without rows = %d-%d", start, end)
	}
}

// A page is a screenful, and it clamps instead of wrapping: holding the key at
// the end of the list stays on the last row rather than flying back to the top.
func TestPageMovesByAScreenfulAndClampsAtBothEnds(t *testing.T) {
	model := &Model{Items: make([]appstore.Item, 12), Focus: Browse}
	model.Page(1, 5)
	if model.Selected != 5 {
		t.Fatalf("one page down selected %d, want 5", model.Selected)
	}
	model.Page(1, 5)
	model.Page(1, 5)
	if model.Selected != 11 {
		t.Fatalf("paging past the end selected %d, want the last row", model.Selected)
	}
	model.Page(-1, 5)
	if model.Selected != 6 {
		t.Fatalf("one page up selected %d, want 6", model.Selected)
	}
	model.Page(-1, 5)
	model.Page(-1, 5)
	if model.Selected != 0 {
		t.Fatalf("paging past the start selected %d, want the first row", model.Selected)
	}
	// Paging belongs to the catalogue. There is no second screenful of action
	// buttons, and a busy model is not accepting navigation at all.
	model.Focus = Actions
	model.Page(1, 5)
	if model.Selected != 0 {
		t.Fatalf("paging moved a non-catalogue screen to %d", model.Selected)
	}
	model.Focus = Browse
	model.Busy = true
	model.Page(1, 5)
	if model.Selected != 0 {
		t.Fatalf("paging moved a busy model to %d", model.Selected)
	}
	model.Busy = false
	model.Page(1, 0)
	if model.Selected != 0 {
		t.Fatalf("paging by no rows moved the selection to %d", model.Selected)
	}
}

func waitForModel(t *testing.T, model *Model) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for model.Busy && time.Now().Before(deadline) {
		model.Poll()
		time.Sleep(time.Millisecond)
	}
	model.Poll()
	if model.Busy {
		t.Fatal("operation did not finish")
	}
}
