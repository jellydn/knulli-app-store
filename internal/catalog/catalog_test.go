package catalog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRepositoryCatalogueBuildsDeterministically(t *testing.T) {
	directory := filepath.Join("..", "..", "catalogue", "packages")
	first, err := Build(directory)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Packages) != 6 || len(second.Packages) != 6 {
		t.Fatalf("expected six candidates, got %d and %d", len(first.Packages), len(second.Packages))
	}
	approved := map[string]bool{
		"app.romm.grout":               true,
		"io.github.ahmadteeb.emudrop":  true,
		"io.github.unitreign.playtime": true,
	}
	for index := range first.Packages {
		if first.Packages[index].ID != second.Packages[index].ID || first.Packages[index].ManifestSHA256 != second.Packages[index].ManifestSHA256 {
			t.Fatal("catalogue build is not deterministic")
		}
		if first.Packages[index].Package.Installable() {
			t.Fatalf("candidate unexpectedly installable: %s", first.Packages[index].ID)
		}
		approval := first.Packages[index].Package.Review.Approval
		if approved[first.Packages[index].ID] != (approval != nil && approval.Provenance == "community") {
			t.Fatalf("unexpected approval state: %s", first.Packages[index].ID)
		}
	}
}

func TestLoadRejectsChangedPackageMetadata(t *testing.T) {
	index, err := Build(filepath.Join("..", "..", "catalogue", "packages"))
	if err != nil {
		t.Fatal(err)
	}
	index.Packages[0].Package.Name = "Changed after hashing"
	data, err := json.Marshal(index)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected changed package metadata to fail digest verification")
	}
}
