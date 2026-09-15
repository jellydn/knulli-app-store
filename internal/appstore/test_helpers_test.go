package appstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/catalog"
	"github.com/jellydn/knulli-app-store/internal/manifest"
)

func catalogForTest() (catalog.Index, error) {
	return catalog.Build(filepath.Join("..", "..", "catalogue", "packages"))
}

func installablePackage() manifest.Package {
	return manifest.Package{
		Schema: manifest.SchemaV1, ID: "org.example.test", Name: "Test", Version: "1.0.0", Type: "utility", Summary: "Test package.",
		Repository: "https://github.com/example/test", License: "MIT", Review: manifest.Review{Status: "installable"},
		Release:       &manifest.Release{URL: "https://github.com/example/test/releases/download/v1/test.zip", SHA256: strings.Repeat("a", 64), Size: 1, InstalledSize: 1, Format: "zip", Immutable: true},
		Compatibility: &manifest.Compatibility{Firmware: "knulli", MinimumVersion: "2026.05", Architectures: []string{"aarch64"}, Devices: []string{"trimui-smart-pro"}, Resolutions: []string{"1280x720"}},
		Install:       &manifest.Install{Destination: "/userdata/roms/ports/test", Launcher: "run.sh", AllowedWritePaths: []string{"/userdata/roms/ports/test"}},
	}
}

func writeIndexForTest(index catalog.Index, path string) error {
	data, err := json.Marshal(index)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
