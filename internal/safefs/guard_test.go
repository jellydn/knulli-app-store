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

// A recovery that cannot undo an open journal has to report the failure and
// leave the journal in place, because the destination is in neither state and
// only an operator can decide what to keep.
func TestRecoverReportsAFailedRollback(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/test"})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "userdata/test"), 0755); err != nil {
		t.Fatal(err)
	}
	parent := t.TempDir()
	tx, err := Begin(guard, parent)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Write("/userdata/test/new", []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	// Nothing existed at that path, so the rollback removes it. Replace it with
	// a non-empty directory so the removal cannot succeed.
	host, err := guard.Resolve("/userdata/test/new")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(host); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(host, "child"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(host, "child", "blocker"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	err = Recover(root, parent)
	if err == nil {
		t.Fatal("expected the failed rollback to be reported")
	}
	if !strings.Contains(err.Error(), "roll back transaction journal") || !strings.Contains(err.Error(), tx.directory) {
		t.Fatalf("recovery error does not name the journal it could not roll back: %v", err)
	}
	if _, statErr := os.Stat(tx.directory); statErr != nil {
		t.Fatalf("journal directory should remain for manual inspection: %v", statErr)
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

// A guard is built from a root that has to exist, because the root is what every
// resolved path is measured against. A policy that cannot be read is refused
// here rather than during a write.
func TestNewGuardRejectsAnUnusableRootOrPolicy(t *testing.T) {
	root := t.TempDir()
	cases := []struct {
		name    string
		root    string
		allowed []string
		problem string
	}{
		{name: "root does not exist", root: filepath.Join(root, "absent"), allowed: []string{"/userdata/app"}, problem: "resolve filesystem root"},
		{name: "no allowed paths", root: root, allowed: nil, problem: "at least one allowed path"},
		{name: "allowed path is relative", root: root, allowed: []string{"userdata/app"}, problem: "invalid allowed path"},
		{name: "allowed path is not cleaned", root: root, allowed: []string{"/userdata/roms/../roms/app"}, problem: "invalid allowed path"},
		{name: "allowed path is the root", root: root, allowed: []string{"/"}, problem: "invalid allowed path"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := NewGuard(testCase.root, testCase.allowed); err == nil || !strings.Contains(err.Error(), testCase.problem) {
				t.Fatalf("expected %q, got %v", testCase.problem, err)
			}
		})
	}
}

// Virtual is the inverse of Resolve, and it enforces the same policy in the
// other direction: a host path the guard did not hand out is not one it accepts.
func TestGuardVirtualMapsHostPathsBackAndEnforcesTheSamePolicy(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/app"})
	if err != nil {
		t.Fatal(err)
	}
	host, err := guard.Resolve("/userdata/app/run.sh")
	if err != nil {
		t.Fatal(err)
	}
	virtual, err := guard.Virtual(host)
	if err != nil {
		t.Fatal(err)
	}
	if virtual != "/userdata/app/run.sh" {
		t.Fatalf("Virtual returned %q", virtual)
	}
	if _, err := guard.Virtual(filepath.Join(root, "userdata/other/run.sh")); err == nil {
		t.Fatal("expected a host path outside the allowed paths to be refused")
	}
	if _, err := guard.Virtual(filepath.Join(filepath.Dir(root), "outside/run.sh")); err == nil {
		t.Fatal("expected a host path outside the root to be refused")
	}
}

// Create, chmod, and remove are the three mutations the installer performs on
// an existing file, so a committed transaction has to leave all three applied
// and drop its journal.
func TestTransactionAppliesCopyChmodAndRemoveThenCommits(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/app"})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source.bin")
	if err := os.WriteFile(source, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}
	dropped, err := guard.Resolve("/userdata/app/dropped.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dropped), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dropped, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}

	parent := t.TempDir()
	tx, err := Begin(guard, parent)
	if err != nil {
		t.Fatal(err)
	}
	directory := tx.directory
	if err := tx.Copy(source, "/userdata/app/copied.bin", 0644); err != nil {
		t.Fatal(err)
	}
	if err := tx.Chmod("/userdata/app/copied.bin", 0755); err != nil {
		t.Fatal(err)
	}
	if err := tx.Remove("/userdata/app/dropped.txt"); err != nil {
		t.Fatal(err)
	}
	// Removing a path that was never there is already the desired state.
	if err := tx.Remove("/userdata/app/never-existed.txt"); err != nil {
		t.Fatalf("removing an absent path failed: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	copied, err := guard.Resolve("/userdata/app/copied.bin")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(copied)
	if err != nil || string(data) != "payload" {
		t.Fatalf("copied content = %q, %v", data, err)
	}
	info, err := os.Stat(copied)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("copied mode = %o, want 755", info.Mode().Perm())
	}
	if _, err := os.Stat(dropped); !os.IsNotExist(err) {
		t.Fatalf("removed path survived the commit: %v", err)
	}
	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("committed journal directory remained: %v", err)
	}
}

// Once a transaction is closed it must not accept another mutation, because the
// caller would otherwise believe a change it made after commit had been
// snapshotted and could be rolled back.
func TestTransactionRefusesMutationsAfterItCloses(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/app"})
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source.bin")
	if err := os.WriteFile(source, []byte("payload"), 0644); err != nil {
		t.Fatal(err)
	}
	tx, err := Begin(guard, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func() error{
		"write":  func() error { return tx.Write("/userdata/app/after.txt", []byte("x"), 0644) },
		"copy":   func() error { return tx.Copy(source, "/userdata/app/after.bin", 0644) },
		"chmod":  func() error { return tx.Chmod("/userdata/app/after.txt", 0600) },
		"remove": func() error { return tx.Remove("/userdata/app/after.txt") },
		"commit": tx.Commit,
	}
	for name, mutate := range mutations {
		if err := mutate(); err == nil || !strings.Contains(err.Error(), "transaction is closed") {
			t.Fatalf("%s after commit: expected a closed-transaction error, got %v", name, err)
		}
	}
	// A rollback after a commit is a no-op, not a failure: there is nothing left
	// to undo and the caller may not know which of the two already ran.
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback after commit reported %v", err)
	}
}

// A transaction has to refuse a destination it cannot describe as a single
// regular file. A directory there would otherwise be replaced by a copy that
// silently discards its contents.
func TestTransactionRefusesANonRegularDestination(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/app"})
	if err != nil {
		t.Fatal(err)
	}
	host, err := guard.Resolve("/userdata/app/blocked")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(host, 0755); err != nil {
		t.Fatal(err)
	}
	tx, err := Begin(guard, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	err = tx.Write("/userdata/app/blocked", []byte("x"), 0644)
	if err == nil || !strings.Contains(err.Error(), "non-regular file") {
		t.Fatalf("expected a non-regular destination to be refused, got %v", err)
	}
	// A path outside the declared policy is refused before any snapshot.
	if err := tx.Write("/userdata/other/file", []byte("x"), 0644); err == nil || !strings.Contains(err.Error(), "outside declared write paths") {
		t.Fatalf("expected a path outside the policy to be refused, got %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
}

// A journal directory is created inside an existing parent, so a missing parent
// has to be an error rather than an implicitly created tree: the parent is the
// manager directory whose absence means the lock was never taken.
func TestBeginRequiresAnExistingJournalParent(t *testing.T) {
	root := t.TempDir()
	guard, err := NewGuard(root, []string{"/userdata/app"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Begin(guard, filepath.Join(t.TempDir(), "absent", "child")); err == nil {
		t.Fatal("expected a missing journal parent to be refused")
	}
}

// journalEntries are the two ways into a journal directory. Every journal
// reading rule has to hold for both, because a rule enforced on only one of
// them is a rule the GUI can bypass: PendingForPath is what decides whether a
// package is shown as needing attention.
func journalEntries(destination string) []struct {
	name string
	open func(root, parent string) error
} {
	return []struct {
		name string
		open func(root, parent string) error
	}{
		{name: "Recover", open: func(root, parent string) error { return Recover(root, parent) }},
		{name: "PendingForPath", open: func(root, parent string) error {
			_, err := PendingForPath(root, parent, destination)
			return err
		}},
	}
}

// writeJournal places one journal record in a fresh directory and returns it.
func writeJournal(t *testing.T, parent string, record journalRecord) string {
	t.Helper()
	directory, err := os.MkdirTemp(parent, transactionPrefix+"*")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, journalName), data, 0600); err != nil {
		t.Fatal(err)
	}
	return directory
}

// A journal whose status the code does not recognise has to stop recovery. The
// tempting alternative, treating anything that is not open as committed, would
// skip the rollback and leave exactly the half-written destination the journal
// exists to prevent.
func TestUnknownJournalStatusStopsBothEntryPoints(t *testing.T) {
	const destination = "/userdata/test/demo"
	for _, entry := range journalEntries(destination) {
		t.Run(entry.name, func(t *testing.T) {
			root := t.TempDir()
			parent := t.TempDir()
			directory := writeJournal(t, parent, journalRecord{
				Schema: journalSchemaV1,
				Status: "half-written",
				Snapshots: []journalSnapshot{{
					Host:    journalHost(t, root, destination+"/run.sh"),
					Virtual: destination + "/run.sh",
				}},
			})
			err := entry.open(root, parent)
			if err == nil || !strings.Contains(err.Error(), "unknown transaction journal status") {
				t.Fatalf("expected an unknown status to be refused, got %v", err)
			}
			if !strings.Contains(err.Error(), directory) {
				t.Fatalf("error does not name the journal %s: %v", directory, err)
			}
		})
	}
}

// journalHost is the host path a journal records for a virtual path. It goes
// through the guard because that is what a real journal holds: the recorded host
// is a resolved path under the resolved root, and recovery rejects anything
// else before it reads a backup name.
func journalHost(t *testing.T, root, virtual string) string {
	t.Helper()
	guard, err := NewGuard(root, []string{"/userdata/test/demo"})
	if err != nil {
		t.Fatal(err)
	}
	host, err := guard.Resolve(virtual)
	if err != nil {
		t.Fatal(err)
	}
	return host
}

// A journal is untrusted input: it is read after a crash, from a directory any
// process could have written. A backup name with a separator in it would make
// the restore read from outside the journal directory.
func TestJournalRejectsABackupNameThatEscapesItsDirectory(t *testing.T) {
	const destination = "/userdata/test/demo"
	for _, entry := range journalEntries(destination) {
		t.Run(entry.name, func(t *testing.T) {
			root := t.TempDir()
			parent := t.TempDir()
			directory := writeJournal(t, parent, journalRecord{
				Schema: journalSchemaV1,
				Status: journalStatusOpen,
				Snapshots: []journalSnapshot{{
					Host:    journalHost(t, root, destination+"/run.sh"),
					Virtual: destination + "/run.sh",
					Backup:  "../../escape",
					Mode:    0644,
					Existed: true,
				}},
			})
			err := entry.open(root, parent)
			if err == nil || !strings.Contains(err.Error(), "invalid journal backup name") {
				t.Fatalf("expected an escaping backup name to be refused, got %v", err)
			}
			if !strings.Contains(err.Error(), directory) {
				t.Fatalf("error does not name the journal %s: %v", directory, err)
			}
		})
	}
}

// Recovery scans a directory it does not exclusively own, so a plain file that
// merely matches the journal prefix is skipped rather than read as a journal or
// deleted.
func TestRecoverSkipsEntriesThatAreNotJournalDirectories(t *testing.T) {
	root := t.TempDir()
	parent := t.TempDir()
	stray := filepath.Join(parent, transactionPrefix+"stray")
	if err := os.WriteFile(stray, []byte("not a journal"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Recover(root, parent); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stray); err != nil {
		t.Fatalf("a non-directory match was removed: %v", err)
	}
}
