package updatecheck

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

func TestApplyVerifiedReleaseProposals(t *testing.T) {
	for _, scenario := range []string{"verified", "digest mismatch", "size mismatch", "wrong checksum filename", "unsafe archive", "missing launcher", "ambiguous asset", "renamed asset", "foreign URL", "patched package", "embedded payload", "nonexperimental", "unchanged", "missing publisher checksum", "upstream digest exception", "mapping collision", "missing mapping source", "missing executable", "missing mapped executable", "mapped executable", "manifest symlink", "catalogue symlink", "ancestor symlink", "nonregular manifest"} {
		t.Run(scenario, func(t *testing.T) {
			directory := t.TempDir()
			pkg := manifest.Package{
				Schema: manifest.SchemaV1, ID: "org.example.demo", Name: "Demo", Version: "1.0.0", Type: "utility", Summary: "A demo package", Repository: "https://github.com/example/demo", License: "MIT",
				Review:        manifest.Review{Status: "experimental", Notes: []string{"Historical review for 1.0.0"}, Evidence: []manifest.Evidence{{Kind: "publisher-checksum", URL: "https://github.com/example/demo/releases/download/v1.0.0/SHA256SUMS"}}},
				Release:       &manifest.Release{URL: "https://github.com/example/demo/releases/download/v1.0.0/demo-1.0.0.zip", Size: 50, InstalledSize: 10, SHA256: strings.Repeat("a", 64), Format: "zip", Immutable: true},
				Compatibility: &manifest.Compatibility{Firmware: "knulli", Architectures: []string{"aarch64"}, ABIs: []string{"linux-aarch64-glibc"}, Dependencies: []string{"libc"}},
				Install:       &manifest.Install{Destination: "/userdata/roms/tools/demo", Launcher: "demo.sh", Warning: "Experimental", AllowedWritePaths: []string{"/userdata/roms/tools/demo"}, Preserve: []string{"settings"}},
			}
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			name := "demo.sh"
			if scenario == "unsafe archive" {
				name = "../escape"
			}
			if scenario == "missing launcher" {
				name = "different.sh"
			}
			if scenario == "mapped executable" {
				name = "bundle/demo.sh"
			}
			entry, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(entry, "never execute this fixture\n"); err != nil {
				t.Fatal(err)
			}
			if scenario == "mapping collision" {
				entry, err := writer.Create("source.sh")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := io.WriteString(entry, "helper\n"); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "verified" {
				entry, err := writer.Create("notes.txt")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := io.WriteString(entry, "notes\n"); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			digestBytes := sha256.Sum256(buffer.Bytes())
			digest := hex.EncodeToString(digestBytes[:])
			asset := Asset{Name: "demo-1.2.0.zip", URL: "https://github.com/example/demo/releases/download/v1.2.0/demo-1.2.0.zip", Size: int64(buffer.Len()), Digest: "sha256:" + digest}
			checksum := digest + "  demo-1.2.0.zip\n"
			report := Report{Packages: []PackageReport{{ID: pkg.ID, Repository: pkg.Repository, CurrentVersion: "1.0.0", DiscoveredVersion: "v1.2.0", Status: "update-found", Assets: []Asset{asset, {Name: "SHA256SUMS", URL: "https://github.com/example/demo/releases/download/v1.2.0/SHA256SUMS"}}}}}
			item := &report.Packages[0]
			switch scenario {
			case "digest mismatch":
				item.Assets[0].Digest = "sha256:" + strings.Repeat("b", 64)
			case "size mismatch":
				item.Assets[0].Size++
			case "wrong checksum filename":
				checksum = digest + "  other.zip\n"
			case "ambiguous asset":
				item.Assets = append(item.Assets, asset)
			case "renamed asset":
				item.Assets[0].Name = "other.zip"
			case "foreign URL":
				item.Assets[0].URL = "https://example.net/demo-1.2.0.zip"
			case "patched package":
				pkg.Install.BinaryPatches = []manifest.BinaryPatch{{Path: "demo.sh", BeforeHex: "61", AfterHex: "62", SHA256: strings.Repeat("c", 64)}}
			case "embedded payload":
				pkg.Release.Format, pkg.Release.Base64Offset = "tar.gz", 1
			case "nonexperimental":
				pkg.Review.Status = "candidate"
				pkg.Install, pkg.Compatibility = nil, nil
			case "unchanged":
				item.Status = "current"
			case "missing publisher checksum", "upstream digest exception":
				pkg.Review.Evidence = nil
				pkg.Install.UpstreamManaged = scenario == "upstream digest exception"
			case "mapping collision":
				pkg.Install.FileMappings = map[string]string{"source.sh": "demo.sh"}
			case "missing mapping source":
				pkg.Install.FileMappings = map[string]string{"absent.sh": "unused.sh"}
			case "missing executable":
				pkg.Install.Executables = []string{"missing.sh"}
			case "missing mapped executable", "mapped executable":
				pkg.Install.FileMappings = map[string]string{"demo.sh": "bin/demo.sh"}
				pkg.Install.Launcher = "bin/demo.sh"
				pkg.Install.Executables = []string{"bin/demo.sh"}
				if scenario == "mapped executable" {
					pkg.Install.StripComponents = 1
				} else {
					pkg.Install.Executables = []string{"bin/missing.sh"}
				}
			}
			before, err := json.MarshalIndent(pkg, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			filename := filepath.Join(directory, "demo.json")
			if err := os.WriteFile(filename, before, 0600); err != nil {
				t.Fatal(err)
			}
			untouched := filepath.Join(directory, "unchanged.json")
			other := pkg
			other.ID = "org.example.other"
			otherBytes, _ := json.Marshal(other)
			if err := os.WriteFile(untouched, otherBytes, 0600); err != nil {
				t.Fatal(err)
			}
			unsafePath := strings.Contains(scenario, "symlink") || scenario == "nonregular manifest"
			var outside string
			if unsafePath {
				outside = filepath.Join(t.TempDir(), "outside.json")
				if err := os.WriteFile(outside, before, 0600); err != nil {
					t.Fatal(err)
				}
				switch scenario {
				case "manifest symlink", "nonregular manifest":
					if err := os.Remove(filename); err != nil {
						t.Fatal(err)
					}
					if scenario == "manifest symlink" {
						if err := os.Symlink(outside, filename); err != nil {
							t.Fatal(err)
						}
					} else if err := os.Mkdir(filename, 0700); err != nil {
						t.Fatal(err)
					}
				case "catalogue symlink", "ancestor symlink":
					link := filepath.Join(t.TempDir(), "linked")
					target := directory
					if scenario == "ancestor symlink" {
						target = filepath.Dir(directory)
					}
					if err := os.Symlink(target, link); err != nil {
						t.Fatal(err)
					}
					if scenario == "ancestor symlink" {
						directory = filepath.Join(link, filepath.Base(directory))
					} else {
						directory = link
					}
				}
			}
			checker := Checker{Token: "must-not-leak", Client: &http.Client{Transport: assetTransport(func(request *http.Request) (*http.Response, error) {
				if request.Header.Get("Authorization") != "" {
					t.Fatal("API credential leaked to an asset request")
				}
				var body io.Reader = bytes.NewReader(buffer.Bytes())
				if strings.HasSuffix(request.URL.Path, "/SHA256SUMS") {
					body = strings.NewReader(checksum)
				} else if request.URL.String() != asset.URL {
					return nil, fmt.Errorf("unexpected asset request: %s", request.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(body), Header: make(http.Header), Request: request}, nil
			})}}
			err = checker.Apply(context.Background(), directory, &report)
			if unsafePath {
				if err == nil {
					t.Fatal("unsafe manifest write path was accepted")
				}
				if strings.Contains(scenario, "symlink") && !strings.Contains(err.Error(), "symlink") {
					t.Fatalf("expected symlink rejection, got %v", err)
				}
				outsideBytes, readErr := os.ReadFile(outside)
				if readErr != nil || !bytes.Equal(outsideBytes, before) {
					t.Fatal("outside manifest was changed")
				}
				if scenario != "nonregular manifest" {
					targetBytes, readErr := os.ReadFile(filename)
					if readErr != nil || !bytes.Equal(targetBytes, before) {
						t.Fatal("manifest behind unsafe path was changed")
					}
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(filename)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "verified" || scenario == "upstream digest exception" || scenario == "mapped executable" {
				updated, err := manifest.Load(filename)
				if err != nil {
					t.Fatal(err)
				}
				wantSize := int64(27)
				if scenario == "verified" {
					wantSize = 33
				}
				if item.Status != "update-proposed" || updated.Version != "1.2.0" || updated.Release.URL != asset.URL || updated.Release.SHA256 != digest || updated.Release.Size != int64(buffer.Len()) || updated.Release.InstalledSize != wantSize {
					t.Fatalf("incorrect release proposal: %#v / %#v", updated.Release, item)
				}
				if !reflect.DeepEqual(updated.Compatibility, pkg.Compatibility) || !reflect.DeepEqual(updated.Install.Preserve, pkg.Install.Preserve) || updated.Review.Status != "experimental" || updated.Review.Notes[0] != pkg.Review.Notes[0] {
					t.Fatal("proposal changed runtime policy or historical evidence")
				}
				if scenario == "upstream digest exception" && !strings.Contains(updated.Install.Warning, "no publisher checksum") {
					t.Fatal("digest exception lacks a visible warning")
				}
				installedPath := "demo.sh"
				if scenario == "mapped executable" {
					installedPath = "bin/demo.sh"
				}
				fileHash := sha256.Sum256([]byte("never execute this fixture\n"))
				wantInventory := []InventoryFile{{StagedPath: "demo.sh", InstalledPath: installedPath, Size: 27, SHA256: hex.EncodeToString(fileHash[:])}}
				if scenario == "verified" {
					notesHash := sha256.Sum256([]byte("notes\n"))
					wantInventory = append(wantInventory, InventoryFile{StagedPath: "notes.txt", InstalledPath: "notes.txt", Size: 6, SHA256: hex.EncodeToString(notesHash[:])})
				}
				jsonReport, err := JSON(report)
				if err != nil {
					t.Fatal(err)
				}
				var saved Report
				if err := json.Unmarshal(jsonReport, &saved); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(saved.Packages[0].Inventory, wantInventory) {
					t.Fatalf("saved archive inventory = %#v, want %#v", saved.Packages[0].Inventory, wantInventory)
				}
				if !strings.Contains(string(Markdown(report)), fmt.Sprintf("%d regular files, %d installed bytes", len(wantInventory), wantSize)) {
					t.Fatal("Markdown report lacks the inspected inventory summary")
				}
			} else {
				if !bytes.Equal(before, after) {
					t.Fatal("blocked or unchanged package was edited")
				}
				if scenario != "unchanged" && (item.Status != "manual-update-review" || item.Error == "") {
					t.Fatalf("missing review blocker: %#v", item)
				}
				if len(item.Inventory) != 0 {
					t.Fatal("blocked proposal advertised a verified inventory")
				}
			}
			unchanged, _ := os.ReadFile(untouched)
			if !bytes.Equal(unchanged, otherBytes) {
				t.Fatal("an unrelated package changed")
			}
		})
	}
}

type assetTransport func(*http.Request) (*http.Response, error)

func (transport assetTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}
