package catalog

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRequiresValidSignatureWhenPublicKeyIsEmbedded(t *testing.T) {
	public, private, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	embeddedPublicKeyHex = hex.EncodeToString(public)
	t.Cleanup(func() { embeddedPublicKeyHex = "" })

	index, err := Build(filepath.Join("..", "..", "catalogue", "packages"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "catalog-index.json")
	if err := Write(index, path); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected unsigned index to fail when a public key is embedded")
	}
	if err := SignFile(path, private); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Packages) != len(index.Packages) {
		t.Fatalf("signed load lost packages: %d", len(loaded.Packages))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(append([]byte{}, data...), '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected tampered index bytes to fail signature verification")
	}
}

func TestLoadRejectsSignatureFromAnotherKey(t *testing.T) {
	public, _, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	_, other, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	embeddedPublicKeyHex = hex.EncodeToString(public)
	t.Cleanup(func() { embeddedPublicKeyHex = "" })

	index, err := Build(filepath.Join("..", "..", "catalogue", "packages"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "catalog-index.json")
	if err := Write(index, path); err != nil {
		t.Fatal(err)
	}
	if err := SignFile(path, other); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected a foreign signature to fail")
	}
}
