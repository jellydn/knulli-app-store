package safefs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardRejectsPathsOutsidePolicyAndSymlinkParents(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/roms/tools/demo"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := guard.Resolve("/userdata/roms/ports/outside"); err == nil {
		t.Fatal("expected path outside policy to fail")
	}
	tools := filepath.Join(root, "userdata/roms")
	if err := os.MkdirAll(tools, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(tools, "tools")); err != nil {
		t.Fatal(err)
	}
	if _, err := guard.Resolve("/userdata/roms/tools/demo/file"); err == nil {
		t.Fatal("expected symlink parent to fail")
	}
}

func TestTransactionRollsBackWritesAndDeletes(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/test"})
	if err != nil {
		t.Fatal(err)
	}
	host, err := guard.Resolve("/userdata/test/existing")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(host), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(host, []byte("before"), 0640); err != nil {
		t.Fatal(err)
	}
	tx, err := Begin(guard, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Write("/userdata/test/existing", []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.Write("/userdata/test/new", []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(host)
	if string(data) != "before" {
		t.Fatalf("existing file was not restored: %q", data)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(host), "new")); !os.IsNotExist(err) {
		t.Fatalf("new file survived rollback: %v", err)
	}
}

func TestRecoverRestoresOpenJournalAfterCrash(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/test"})
	if err != nil {
		t.Fatal(err)
	}
	host, err := guard.Resolve("/userdata/test/existing")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(host), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(host, []byte("before"), 0640); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	tx, err := Begin(guard, parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Write("/userdata/test/existing", []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.Write("/userdata/test/new", []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(host)
	if err != nil || string(data) != "after" {
		t.Fatalf("crash snapshot missing: %q, %v", data, err)
	}
	if err := Recover(root, parent); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(host)
	if err != nil || string(data) != "before" {
		t.Fatalf("open journal did not restore existing file: %q, %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(host), "new")); !os.IsNotExist(err) {
		t.Fatalf("open journal left a new file: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(parent, transactionPrefix+"*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("recovered journal directory remained: %v", matches)
	}
}

func TestPendingForPathValidatesAndMatchesOpenJournal(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/app", "/userdata/system/manager"})
	if err != nil {
		t.Fatal(err)
	}
	parent, err := guard.Resolve("/userdata/system/manager")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(parent, 0700); err != nil {
		t.Fatal(err)
	}
	tx, err := Begin(guard, parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Write("/userdata/app/run.sh", []byte("partial"), 0755); err != nil {
		t.Fatal(err)
	}
	pending, err := PendingForPath(root, parent, "/userdata/app")
	if err != nil || !pending {
		t.Fatalf("pending = %v, %v", pending, err)
	}
	pending, err = PendingForPath(root, parent, "/userdata/other")
	if err != nil || pending {
		t.Fatalf("unrelated pending = %v, %v", pending, err)
	}
}

func TestRecoverLeavesCommittedJournalMutations(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/test"})
	if err != nil {
		t.Fatal(err)
	}
	host, err := guard.Resolve("/userdata/test/existing")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(host), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(host, []byte("before"), 0640); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	tx, err := Begin(guard, parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Write("/userdata/test/existing", []byte("after"), 0600); err != nil {
		t.Fatal(err)
	}
	directory := tx.directory
	if err := tx.persistJournal(journalCommitted); err != nil {
		t.Fatal(err)
	}
	tx.closed = true
	if err := Recover(root, parent); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(host)
	if err != nil || string(data) != "after" {
		t.Fatalf("committed journal rolled back: %q, %v", data, err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("committed journal directory remained: %v", err)
	}
}

func TestRecoverRemovesJournalWithoutMutations(t *testing.T) {
	root := t.TempDir()
	parent := t.TempDir()
	directory, err := os.MkdirTemp(parent, transactionPrefix+"*")
	if err != nil {
		t.Fatal(err)
	}
	if err := Recover(root, parent); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("empty journal directory remained: %v", err)
	}
}

// An unreadable journal has to be inspected by hand, so every failure names the
// leftover directory instead of reporting only that a record could not be read.
// Recover and PendingForPath are the two ways into a journal, and the GUI shows
// the PendingForPath error as the reason a package needs attention.
func TestRecoveryFailuresNameTheLeftoverJournalDirectory(t *testing.T) {
	const destination = "/userdata/test/demo"
	entries := []struct {
		name string
		open func(root, parent string) error
	}{
		{name: "Recover", open: func(root, parent string) error { return Recover(root, parent) }},
		{name: "PendingForPath", open: func(root, parent string) error {
			_, err := PendingForPath(root, parent, destination)
			return err
		}},
	}
	for _, entry := range entries {
		root := t.TempDir()
		parent := t.TempDir()
		directory, err := os.MkdirTemp(parent, transactionPrefix+"*")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, journalName), []byte("{not json"), 0600); err != nil {
			t.Fatal(err)
		}
		err = entry.open(root, parent)
		if err == nil {
			t.Fatalf("%s accepted a corrupt journal", entry.name)
		}
		if !strings.Contains(err.Error(), directory) {
			t.Fatalf("%s error does not name the leftover directory %s: %v", entry.name, directory, err)
		}
	}
}

func TestRecoverRejectsPathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	parent := t.TempDir()
	directory, err := os.MkdirTemp(parent, transactionPrefix+"*")
	if err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "escape")
	if err := os.WriteFile(outside, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	record := journalRecord{
		Schema: journalSchemaV1,
		Status: journalStatusOpen,
		Snapshots: []journalSnapshot{{
			Host:    outside,
			Virtual: "/userdata/test/escape",
			Existed: false,
		}},
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, journalName), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := Recover(root, parent); err == nil {
		t.Fatal("expected outside journal path to fail")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside path was changed: %v", err)
	}
}
