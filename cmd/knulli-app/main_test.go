package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteNewFileCreatesPrivateFileWithoutReplacingExistingPath(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "key")
	if err := writeNewFile(path, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("private key mode = %o, want 600", info.Mode().Perm())
	}
	if err := writeNewFile(path, []byte("replacement"), 0600); err == nil {
		t.Fatal("expected existing key path to be rejected")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "secret" {
		t.Fatalf("existing key was replaced: %q", data)
	}
}
