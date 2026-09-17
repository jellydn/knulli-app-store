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
	if len(first.Packages) != 5 || len(second.Packages) != 5 {
		t.Fatalf("expected five packages, got %d and %d", len(first.Packages), len(second.Packages))
	}
	// RetSend is published by the App Store maintainer's own fork, so its
	// recorded provenance is the maintainer rather than the community.
	provenance := map[string]string{
		"app.romm.grout":                         "community",
		"io.github.jellydn.retsend":              "maintainer",
		"io.github.misantronic.raofflineproxy":   "community",
		"io.github.tomtombombadil.pocketcurator": "community",
		"io.github.unitreign.playtime":           "community",
	}
	experimental := 0
	verified := 0
	for index := range first.Packages {
		if first.Packages[index].ID != second.Packages[index].ID || first.Packages[index].ManifestSHA256 != second.Packages[index].ManifestSHA256 {
			t.Fatal("catalogue build is not deterministic")
		}
		if first.Packages[index].Package.Experimental() {
			experimental++
		} else if first.Packages[index].Package.Review.Status == "verified" {
			verified++
		} else if first.Packages[index].Package.Installable() {
			t.Fatalf("non-experimental package unexpectedly installable: %s", first.Packages[index].ID)
		}
		approval := first.Packages[index].Package.Review.Approval
		wanted, known := provenance[first.Packages[index].ID]
		if !known || approval == nil || approval.Provenance != wanted {
			t.Fatalf("unexpected approval state: %s", first.Packages[index].ID)
		}
	}
	if experimental != 3 || verified != 0 {
		t.Fatalf("expected three broad experimental packages and no current device-verified release, got %d and %d", experimental, verified)
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
