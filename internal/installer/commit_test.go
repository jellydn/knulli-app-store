package installer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/safefs"
)

// A rollback that cannot undo the operation must not replace the original
// failure. The GUI shows one reason, and "rollback also failed" is what tells an
// operator the destination is in neither the old nor the new state.
func TestRollbackFoldsAFailedRollbackIntoTheOperationError(t *testing.T) {
	root := t.TempDir()
	baseGuard, err := safefs.NewGuard(root, []string{managerPath})
	if err != nil {
		t.Fatal(err)
	}
	managerHost, err := baseGuard.Resolve(managerPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(managerHost, 0700); err != nil {
		t.Fatal(err)
	}
	guard, err := safefs.NewGuard(root, []string{"/userdata/roms/tools/demo", managerPath})
	if err != nil {
		t.Fatal(err)
	}
	tx, err := safefs.Begin(guard, managerHost)
	if err != nil {
		t.Fatal(err)
	}
	virtual := "/userdata/roms/tools/demo/launch.sh"
	if err := os.MkdirAll(filepath.Join(root, "userdata/roms/tools/demo"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := tx.Write(virtual, []byte("reviewed launcher"), 0755); err != nil {
		t.Fatal(err)
	}
	// Nothing existed at that path, so the rollback removes it. A non-empty
	// directory in its place makes the removal fail, which is the case this
	// branch exists for: a destination left in a state the rollback cannot
	// describe.
	host, err := guard.Resolve(virtual)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(host); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(host, "unexpected"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(host, "unexpected", "blocker"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	result := errors.New("install failed: SHA-256 mismatch")
	Manager{Root: root}.rollback("org.example.demo", "install", tx, guard, nil, &result)
	if !strings.Contains(result.Error(), "install failed: SHA-256 mismatch") {
		t.Fatalf("rollback dropped the operation error: %v", result)
	}
	if !strings.Contains(result.Error(), "rollback also failed") {
		t.Fatalf("rollback hid its own failure: %v", result)
	}
}
