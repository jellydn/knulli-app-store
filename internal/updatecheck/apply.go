package updatecheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
	"github.com/jellydn/knulli-app-store/internal/version"
)

// Apply prepares proposals, not approvals. Historical review evidence is retained
// at its original version; unsupported changes stay in the report for a reviewer.
func (checker Checker) Apply(ctx context.Context, directory string, report *Report) error {
	directory, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	guard, err := safefs.NewGuard("/", []string{directory})
	if err != nil {
		return err
	}
	if _, err := guard.Resolve(directory); err != nil {
		return err
	}
	paths, err := filepath.Glob(filepath.Join(directory, "*.json"))
	if err != nil {
		return err
	}
	for _, filename := range paths {
		if _, err := guard.Resolve(filename); err != nil {
			return err
		}
		info, err := os.Lstat(filename)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("manifest is not a regular file: %s", filename)
		}
		pkg, err := manifest.Load(filename)
		if err != nil {
			return err
		}
		for index := range report.Packages {
			item := &report.Packages[index]
			if item.ID != pkg.ID || item.Status != "update-found" {
				continue
			}
			updated, err := checker.propose(ctx, pkg, item)
			if err != nil {
				item.Status = "manual-update-review"
				item.Error = err.Error()
				continue
			}
			// Keep unrelated manifest values, including optional fields, intact.
			data, err := os.ReadFile(filename)
			if err != nil {
				return err
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(data, &fields); err != nil {
				return err
			}
			fields["version"], _ = json.Marshal(updated.Version)
			fields["release"], _ = json.Marshal(updated.Release)
			fields["review"], _ = json.Marshal(updated.Review)
			if updated.Install.Warning != pkg.Install.Warning {
				var install map[string]json.RawMessage
				if err := json.Unmarshal(fields["install"], &install); err != nil {
					return err
				}
				install["warning"], _ = json.Marshal(updated.Install.Warning)
				fields["install"], _ = json.Marshal(install)
			}
			data, err = json.MarshalIndent(fields, "", "  ")
			if err != nil {
				return err
			}
			if _, err := guard.Resolve(filename); err != nil {
				return err
			}
			if err := safefs.AtomicWrite(filename, append(data, '\n'), info.Mode().Perm()); err != nil {
				return err
			}
			item.Status = "update-proposed"
		}
	}
	return nil
}

func (checker Checker) propose(ctx context.Context, pkg manifest.Package, item *PackageReport) (manifest.Package, error) {
	if pkg.Review.Status != "experimental" || pkg.Release == nil || pkg.Install == nil ||
		len(pkg.Install.BinaryPatches) != 0 || pkg.Release.Base64Offset != 0 {
		return pkg, fmt.Errorf("requires manual release, patch, embedded payload, or technical-status review")
	}
	order, comparable := version.CompareNumeric(item.DiscoveredVersion, pkg.Version)
	if !comparable || order <= 0 || item.CurrentVersion != pkg.Version || item.Repository != pkg.Repository {
		return pkg, fmt.Errorf("report does not describe a newer version of this package")
	}
	asset, err := matchingAsset(*item, path.Base(pkg.Release.URL), pkg.Version)
	if err != nil {
		return pkg, err
	}
	digest := strings.TrimPrefix(asset.Digest, "sha256:")
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size || asset.Digest != "sha256:"+digest || asset.Size <= 0 || asset.Size > manifest.MaxPackageBytes {
		return pkg, fmt.Errorf("release asset needs a SHA-256 API digest and a size within the package limit")
	}
	var checksum *Asset
	for index := len(pkg.Review.Evidence) - 1; index >= 0; index-- {
		evidence := pkg.Review.Evidence[index]
		if evidence.Kind != "publisher-checksum" {
			continue
		}
		found, err := matchingAsset(*item, path.Base(evidence.URL), pkg.Version)
		if err != nil {
			return pkg, fmt.Errorf("publisher checksum: %w", err)
		}
		checksum = &found
		break
	}
	if checksum == nil && !pkg.Install.UpstreamManaged {
		return pkg, fmt.Errorf("no reviewed publisher checksum source; GitHub digest alone is allowed only for upstream-managed experimental packages")
	}
	temporary, err := os.MkdirTemp("", "catalogue-proposal-*")
	if err != nil {
		return pkg, err
	}
	defer os.RemoveAll(temporary)
	filename := filepath.Join(temporary, "release")
	file, err := os.Create(filename)
	if err != nil {
		return pkg, err
	}
	hash := sha256.New()
	size, downloadErr := checker.download(ctx, asset.URL, io.MultiWriter(file, hash), manifest.MaxPackageBytes)
	closeErr := file.Close()
	if downloadErr != nil {
		return pkg, downloadErr
	}
	if closeErr != nil {
		return pkg, closeErr
	}
	if size != asset.Size || hex.EncodeToString(hash.Sum(nil)) != digest {
		return pkg, fmt.Errorf("downloaded bytes do not match the pinned GitHub size and digest")
	}
	if checksum != nil {
		var body strings.Builder
		if _, err := checker.download(ctx, checksum.URL, &body, 1<<20); err != nil {
			return pkg, err
		}
		matched := false
		for _, line := range strings.Split(body.String(), "\n") {
			fields := strings.Fields(line)
			if len(fields) == 2 && fields[0] == digest && strings.TrimPrefix(fields[1], "*") == asset.Name {
				matched = true
			}
		}
		if !matched {
			return pkg, fmt.Errorf("publisher checksum does not match the exact asset name and SHA-256")
		}
	}
	files, err := archive.Extract(ctx, filename, pkg.Release.Format, filepath.Join(temporary, "stage"), pkg.Install.StripComponents, manifest.MaxPackageBytes)
	if err != nil {
		return pkg, err
	}
	var installedSize int64
	paths := make(map[string]bool)
	mapped := 0
	inventory := make([]InventoryFile, 0, len(files))
	for _, file := range files {
		installedSize += file.Size
		name := file.Relative
		if target, ok := pkg.Install.FileMappings[name]; ok {
			name = target
			mapped++
		}
		if paths[name] {
			return pkg, fmt.Errorf("file mapping collides with release path %s", name)
		}
		paths[name] = true
		inventory = append(inventory, InventoryFile{StagedPath: file.Relative, InstalledPath: name, Size: file.Size, SHA256: file.SHA256})
	}
	if installedSize == 0 {
		return pkg, fmt.Errorf("archive contains no regular-file bytes")
	}
	if mapped != len(pkg.Install.FileMappings) {
		return pkg, fmt.Errorf("file mapping source is absent from release archive")
	}
	for _, required := range append([]string{pkg.Install.Launcher}, pkg.Install.Executables...) {
		if !paths[required] {
			return pkg, fmt.Errorf("release archive is missing required file %s", required)
		}
	}
	pkg.Version = version.Normalize(item.DiscoveredVersion)
	release := *pkg.Release
	release.URL, release.SHA256, release.Size, release.InstalledSize = asset.URL, digest, size, installedSize
	pkg.Release = &release
	pkg.Review.Notes = append(pkg.Review.Notes, "Automated release proposal: downloaded size/SHA-256 and bounded archive inspection passed. Earlier runtime, dependency, path and device evidence is historical; review this version before merging. No upstream code was executed.")
	pkg.Review.Evidence = append(pkg.Review.Evidence, manifest.Evidence{Kind: "release-asset", URL: asset.URL, Note: "Automated proposal verified downloaded bytes against GitHub API SHA-256: " + digest})
	pkg.Review.Evidence = append(pkg.Review.Evidence, manifest.Evidence{Kind: "release-api", URL: "https://api.github.com/repos/" + strings.TrimPrefix(pkg.Repository, "https://github.com/") + "/releases/tags/" + url.PathEscape(item.DiscoveredVersion), Note: "Asset size and digest used for this proposal; downloaded bytes were independently hashed. GitHub account and API trust, not independent publisher authentication."})
	if checksum != nil {
		pkg.Review.Evidence = append(pkg.Review.Evidence, manifest.Evidence{Kind: "publisher-checksum", URL: checksum.URL})
	} else {
		install := *pkg.Install
		if !strings.Contains(install.Warning, "GitHub digest only for this version; no publisher checksum.") {
			install.Warning += " GitHub digest only for this version; no publisher checksum."
		}
		pkg.Install = &install
	}
	if err := pkg.Validate(); err != nil {
		return pkg, err
	}
	item.Inventory = inventory
	return pkg, nil
}

func matchingAsset(item PackageReport, oldName, oldVersion string) (Asset, error) {
	name := strings.ReplaceAll(oldName, oldVersion, version.Normalize(item.DiscoveredVersion))
	var matches []Asset
	for _, asset := range item.Assets {
		if asset.Name != name {
			continue
		}
		prefix := strings.TrimSuffix(item.Repository, "/") + "/releases/download/" + url.PathEscape(item.DiscoveredVersion) + "/"
		if asset.URL != prefix+url.PathEscape(asset.Name) {
			continue
		}
		matches = append(matches, asset)
	}
	if len(matches) != 1 {
		return Asset{}, fmt.Errorf("expected one version-pinned asset matching %q; found %d", name, len(matches))
	}
	return matches[0], nil
}

func (checker Checker) download(ctx context.Context, address string, destination io.Writer, limit int64) (int64, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return 0, err
	}
	client := checker.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	secureClient := *client
	secureClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if request.URL.Scheme != "https" {
			return fmt.Errorf("asset download redirected away from HTTPS")
		}
		if client.CheckRedirect != nil {
			return client.CheckRedirect(request, via)
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many asset redirects")
		}
		return nil
	}
	// Public assets need no credential. Never send the API token to asset hosts.
	response, err := secureClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("asset download returned HTTP %d", response.StatusCode)
	}
	if response.Request != nil && response.Request.URL.Scheme != "https" {
		return 0, fmt.Errorf("asset download redirected away from HTTPS")
	}
	size, err := io.Copy(destination, io.LimitReader(response.Body, limit+1))
	if err == nil && size > limit {
		err = fmt.Errorf("asset exceeds %d byte limit", limit)
	}
	return size, err
}
