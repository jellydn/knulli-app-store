package safefs

import (
	"os"
	"path/filepath"
	"testing"
)

// AtomicWrite is what every persisted record goes through, so the properties
// that matter are the ones a crash could otherwise break: the parent exists, the
// bytes land complete, the mode is the requested one, and no temporary file is
// left behind for a later run to mistake for state.
func TestAtomicWriteCreatesParentsAndLeavesNoTemporaryFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "userdata/system/manager/installed/demo.json")
	if err := AtomicWrite(path, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "first" {
		t.Fatalf("content = %q, %v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
	// Replacing an existing record must be the same operation, not an append.
	if err := AtomicWrite(path, []byte("second"), 0640); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(path)
	if err != nil || string(data) != "second" {
		t.Fatalf("content = %q, %v", data, err)
	}
	info, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".write-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Fatalf("temporary files remained: %v", leftovers)
	}
}

// A parent that cannot be created has to be reported rather than truncated
// through, because the caller decides whether a failed record is survivable.
func TestAtomicWriteReportsAParentThatCannotBeCreated(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(filepath.Join(blocker, "child", "record.json"), []byte("x"), 0600); err == nil {
		t.Fatal("expected a file in the parent path to be refused")
	}
}

func TestCopyCarriesContentsAndModeAndReportsAMissingSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.bin")
	if err := os.WriteFile(source, []byte("payload"), 0755); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(root, "nested/deeper/destination.bin")
	if err := Copy(source, destination, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(destination)
	if err != nil || string(data) != "payload" {
		t.Fatalf("content = %q, %v", data, err)
	}
	info, err := os.Stat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("mode = %o, want 755", info.Mode().Perm())
	}
	// A copy over an existing file replaces it whole, which is what restoring a
	// snapshot relies on.
	if err := os.WriteFile(destination, []byte("stale and longer"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Copy(source, destination, 0644); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(destination)
	if err != nil || string(data) != "payload" {
		t.Fatalf("content = %q, %v", data, err)
	}
	if err := Copy(filepath.Join(root, "absent.bin"), filepath.Join(root, "out.bin"), 0644); err == nil {
		t.Fatal("expected a missing source to be reported")
	}
}

func TestSHA256HashesContentsAndReportsAMissingFile(t *testing.T) {
	// The digest of "abc", so the test pins the function rather than a length.
	path := filepath.Join(t.TempDir(), "abc.txt")
	if err := os.WriteFile(path, []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}
	digest, err := SHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	if digest != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("digest = %s", digest)
	}
	if _, err := SHA256(filepath.Join(t.TempDir(), "absent.txt")); err == nil {
		t.Fatal("expected a missing file to be reported")
	}
}

func TestAvailableBytesReportsSpaceAndWalksUpFromAMissingPath(t *testing.T) {
	root := t.TempDir()
	available, err := AvailableBytes(root)
	if err != nil {
		t.Fatal(err)
	}
	if available == 0 {
		t.Fatal("a mounted temporary directory reported no free space")
	}
	// Staging is created before the destination exists, so a path that is not
	// there yet has to answer from its nearest existing parent.
	missing := filepath.Join(root, "userdata/system/manager/work-1/release")
	nested, err := AvailableBytes(missing)
	if err != nil {
		t.Fatal(err)
	}
	if nested == 0 {
		t.Fatalf("a missing path reported no free space")
	}
}

// The parent walk only continues while the failure is "not there yet". Any
// other failure is a real answer from the kernel and has to be reported, or the
// free-space gate would compare a package against nothing. A path containing a
// NUL byte is used because the kernel rejects it deterministically, where a
// permission-based fixture would behave differently when the suite runs as root.
func TestAvailableBytesReportsAPathTheKernelRejects(t *testing.T) {
	if _, err := AvailableBytes("/userdata\x00/card"); err == nil {
		t.Fatal("expected a path the kernel rejects to be reported")
	}
}
