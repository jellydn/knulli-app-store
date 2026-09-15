package catalog

import (
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
	if len(first.Packages) != 5 || len(second.Packages) != 5 {
		t.Fatalf("expected five candidates, got %d and %d", len(first.Packages), len(second.Packages))
	}
	for index := range first.Packages {
		if first.Packages[index].ID != second.Packages[index].ID || first.Packages[index].ManifestSHA256 != second.Packages[index].ManifestSHA256 {
			t.Fatal("catalogue build is not deterministic")
		}
		if first.Packages[index].Package.Installable() {
			t.Fatalf("candidate unexpectedly installable: %s", first.Packages[index].ID)
		}
	}
}
