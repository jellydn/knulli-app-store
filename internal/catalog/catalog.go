package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

const IndexSchemaV1 = "org.knulli.app-store/catalog-index/v1"

type Index struct {
	Schema   string  `json:"schema"`
	Packages []Entry `json:"packages"`
}

type Entry struct {
	ID             string           `json:"id"`
	ManifestSHA256 string           `json:"manifest_sha256"`
	Package        manifest.Package `json:"package"`
}

func Build(directory string) (Index, error) {
	paths, err := filepath.Glob(filepath.Join(directory, "*.json"))
	if err != nil {
		return Index{}, err
	}
	if len(paths) == 0 {
		return Index{}, fmt.Errorf("no JSON manifests found in %s", directory)
	}
	index := Index{Schema: IndexSchemaV1}
	seen := make(map[string]string)
	for _, manifestPath := range paths {
		pkg, err := manifest.Load(manifestPath)
		if err != nil {
			return Index{}, fmt.Errorf("%s: %w", manifestPath, err)
		}
		if first, exists := seen[pkg.ID]; exists {
			return Index{}, fmt.Errorf("duplicate package id %s in %s and %s", pkg.ID, first, manifestPath)
		}
		seen[pkg.ID] = manifestPath
		canonical, err := manifest.Canonical(pkg)
		if err != nil {
			return Index{}, err
		}
		digest := sha256.Sum256(canonical)
		index.Packages = append(index.Packages, Entry{
			ID:             pkg.ID,
			ManifestSHA256: hex.EncodeToString(digest[:]),
			Package:        pkg,
		})
	}
	sort.Slice(index.Packages, func(i, j int) bool {
		return index.Packages[i].ID < index.Packages[j].ID
	})
	return index, nil
}

func Write(index Index, output string) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if output == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}
	if strings.TrimSpace(output) == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(output), ".catalog-index-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, output)
}
