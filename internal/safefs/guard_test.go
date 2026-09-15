package safefs

import (
	"os"
	"path/filepath"
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
