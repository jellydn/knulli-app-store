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
