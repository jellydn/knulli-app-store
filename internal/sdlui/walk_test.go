package sdlui

import (
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func TestParseKeysReadsTheDocumentedSequence(t *testing.T) {
	keys, err := ParseKeys("down, down,enter")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	want := []Key{KeyDown, KeyDown, KeyConfirm}
	if len(keys) != len(want) {
		t.Fatalf("read %d keys, want %d", len(keys), len(want))
	}
	for index := range want {
		if keys[index] != want[index] {
			t.Fatalf("key %d = %v, want %v", index, keys[index], want[index])
		}
	}
}

func TestParseKeysRejectsAnUnknownKey(t *testing.T) {
	if _, err := ParseKeys("down,spacebar"); err == nil {
		t.Fatal("an unknown key was accepted")
	}
}

func TestParseKeysTreatsEmptyAsNoWalkthrough(t *testing.T) {
	if keys, err := ParseKeys("  "); err != nil || len(keys) != 0 {
		t.Fatalf("empty specification = %v, %v", keys, err)
	}
}

func TestKeyNamesCoverEveryKey(t *testing.T) {
	names := KeyNames()
	if len(names) != len(keySpecs) {
		t.Fatalf("KeyNames lists %d names for %d keys", len(names), len(keySpecs))
	}
	for _, name := range names {
		keys, err := ParseKeys(name)
		if err != nil || len(keys) != 1 {
			t.Fatalf("%q is not parseable: %v", name, err)
		}
		if got := KeyName(keys[0]); got != name {
			t.Fatalf("%q names %q", name, got)
		}
	}
	if got := KeyName(KeyNone); got != "none" {
		t.Fatalf("unnamed key = %q", got)
	}
}

func TestWalkFileSortsInWalkthroughOrder(t *testing.T) {
	first := WalkFile(0, KeyNone, "setup")
	second := WalkFile(1, KeyDown, "calibration")
	tenth := WalkFile(10, KeyConfirm, "catalogue")
	if first != "000-start-setup.png" || second != "001-down-calibration.png" {
		t.Fatalf("unexpected names: %q %q", first, second)
	}
	if !(first < second && second < tenth) {
		t.Fatalf("names do not sort in order: %q %q %q", first, second, tenth)
	}
}

func TestWalkStateNamesEveryScreen(t *testing.T) {
	model := &storeui.Model{Items: []appstore.Item{{Package: manifest.Package{ID: "io.github.unitreign.playtime"}, Actions: []appstore.Action{appstore.Install}}}}
	session := storeinput.NewDesktopSession(t.TempDir(), "trimui-smart-pro")

	for _, test := range []struct {
		mode  storeinput.Mode
		state string
	}{
		{storeinput.Setup, "setup"},
		{storeinput.Blocked, "blocked"},
		{storeinput.Settings, "settings"},
		{storeinput.Calibrating, "calibration"},
		{storeinput.Review, "assignment-review"},
		{storeinput.Preview, "preview"},
	} {
		session.Mode = test.mode
		if got := WalkState(model, session); got != test.state {
			t.Fatalf("mode %s is named %q, want %q", test.mode, got, test.state)
		}
	}

	session.Mode = storeinput.Normal
	for _, test := range []struct {
		focus storeui.Focus
		state string
	}{
		{storeui.Browse, "catalogue"},
		{storeui.Health, "health"},
		{storeui.Actions, "actions"},
		{storeui.Confirm, "confirm"},
		{storeui.ForceConfirm, "force-confirm"},
	} {
		model.Focus = test.focus
		if got := WalkState(model, session); got != test.state {
			t.Fatalf("focus %v is named %q, want %q", test.focus, got, test.state)
		}
	}

	model.Busy = true
	if got := WalkState(model, session); got != "progress" {
		t.Fatalf("busy screen is named %q", got)
	}
}

func TestWalkDetailRecordsTheSelectedItemAndError(t *testing.T) {
	model := &storeui.Model{
		Items: []appstore.Item{{
			Package: manifest.Package{ID: "io.github.unitreign.playtime"},
			Actions: []appstore.Action{appstore.Install, appstore.Uninstall},
		}},
		Selected: 0,
		Action:   1,
		Focus:    storeui.Confirm,
		Message:  "Install completed",
		Error:    "SHA-256 mismatch",
	}
	session := storeinput.NewDesktopSession(t.TempDir(), "trimui-smart-pro")
	session.Mode = storeinput.Normal
	session.Message = "Controller mapping tested and saved"
	detail := WalkDetail(model, session)
	for _, want := range []string{"io.github.unitreign.playtime", "confirm", "uninstall", "false", "normal", "Controller mapping tested and saved", "Install completed", "SHA-256 mismatch"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail %q does not record %q", detail, want)
		}
	}
	empty := WalkDetail(&storeui.Model{}, nil)
	if !strings.Contains(empty, "browse") || !strings.Contains(empty, "none") {
		t.Fatalf("empty detail is not named: %q", empty)
	}
}

func TestWalkerSendsEveryKeyOnceAndStops(t *testing.T) {
	walker := NewWalker([]Key{KeyDown, KeyConfirm})
	if walker.Done() {
		t.Fatal("a fresh walker is already done")
	}
	for step, want := range []Key{KeyDown, KeyConfirm} {
		key, ok := walker.Pending()
		if !ok || key != want {
			t.Fatalf("step %d pending = %v, %v", step, key, ok)
		}
		walker.Advance()
		if walker.Last() != want {
			t.Fatalf("step %d left %v pending", step, walker.Last())
		}
	}
	if !walker.Done() || walker.Sent() != 2 {
		t.Fatalf("walker did not finish: done=%v sent=%d", walker.Done(), walker.Sent())
	}
	if key, ok := walker.Pending(); ok || key != KeyNone {
		t.Fatalf("finished walker still offers %v", key)
	}
	walker.Advance()
	if walker.Sent() != 2 {
		t.Fatalf("advancing past the end sent another key: %d", walker.Sent())
	}
}

func TestWalkRunnerCapturesEachKeyAndWaitsForWork(t *testing.T) {
	runner := NewWalkRunner([]Key{KeyConfirm, KeyDown}, 2)
	if runner == nil {
		t.Fatal("a key sequence produced no runner")
	}
	// The opening frame is evidence before any key is sent.
	frame, ok := runner.Capture("setup", false)
	if !ok || frame.File != "000-start-setup.png" || frame.Key != KeyNone {
		t.Fatalf("opening frame = %#v, %v", frame, ok)
	}
	if _, ok := runner.Capture("setup", false); ok {
		t.Fatal("the same frame was captured twice")
	}
	if key, ok := runner.Step(false); !ok || key != KeyConfirm {
		t.Fatalf("first key = %v, %v", key, ok)
	}
	// A busy operation holds the next key until it finishes.
	if key, ok := runner.Step(true); ok {
		t.Fatalf("a key was sent during an operation: %v", key)
	}
	frame, ok = runner.Capture("progress", true)
	if !ok || frame.File != "001-enter-progress.png" {
		t.Fatalf("progress frame = %#v, %v", frame, ok)
	}
	// The screen an operation finished on is evidence too.
	frame, ok = runner.Capture("catalogue", false)
	if !ok || frame.File != "002-enter-catalogue.png" {
		t.Fatalf("completion frame = %#v, %v", frame, ok)
	}
	if key, ok := runner.Step(false); !ok || key != KeyDown {
		t.Fatalf("second key = %v, %v", key, ok)
	}
	if _, ok := runner.Capture("confirm", false); !ok {
		t.Fatal("the key's screen was not captured")
	}
	if runner.Done(true) {
		t.Fatal("a busy run reported completion")
	}
	if runner.Done(false) {
		t.Fatal("the run completed before the screen settled")
	}
	if !runner.Done(false) {
		t.Fatal("the settled run did not complete")
	}
	if runner.Sent() != 2 || runner.Steps() != 4 {
		t.Fatalf("runner sent %d keys and captured %d frames", runner.Sent(), runner.Steps())
	}
}

func TestNilWalkRunnerIsInert(t *testing.T) {
	runner := NewWalkRunner(nil, 3)
	if runner != nil {
		t.Fatal("an empty sequence produced a runner")
	}
	if _, ok := runner.Capture("catalogue", false); ok {
		t.Fatal("a nil runner captured a frame")
	}
	if key, ok := runner.Step(false); ok {
		t.Fatalf("a nil runner sent %v", key)
	}
	if runner.Done(false) || runner.Sent() != 0 || runner.Steps() != 0 {
		t.Fatal("a nil runner is not inert")
	}
}

func TestWalkRecordCarriesTheEvidenceColumns(t *testing.T) {
	shot := WalkShot{File: "001-enter-catalogue.png", State: "catalogue", Key: KeyConfirm}
	record := WalkRecord(shot, "io.github.unitreign.playtime\tbrowse\tinstall\tfalse\tnormal\t\t\t")
	fields := strings.Split(strings.TrimSuffix(record, "\n"), "\t")
	if len(fields) != len(strings.Split(strings.TrimSuffix(WalkHeader, "\n"), "\t")) {
		t.Fatalf("record has %d columns for the %d column header: %q", len(fields), len(strings.Split(WalkHeader, "\t")), record)
	}
	if fields[1] != "catalogue" || fields[2] != "enter" || fields[3] != "io.github.unitreign.playtime" {
		t.Fatalf("unexpected record: %q", record)
	}
}

func TestEmptyKeySequenceHasNoWalker(t *testing.T) {
	walker := NewWalker(nil)
	if walker != nil {
		t.Fatal("an empty sequence produced a walker")
	}
	if !walker.Done() || walker.Sent() != 0 || walker.Last() != KeyNone {
		t.Fatal("a nil walker is not inert")
	}
	if key, ok := walker.Pending(); ok || key != KeyNone {
		t.Fatalf("a nil walker offered %v", key)
	}
	walker.Advance()
}
