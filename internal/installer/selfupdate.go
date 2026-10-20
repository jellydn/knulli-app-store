package installer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	storearchive "github.com/jellydn/knulli-app-store/internal/archive"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/safefs"
)

const StoreDirectory = "/userdata/roms/ports/knulli-app-store"
const storeReleaseBase = "https://github.com/jellydn/knulli-app-store/releases/download/"

type StoreUpdate struct {
	Version string
	Release manifest.Release
}

// CheckStoreUpdate includes the project's CI prereleases. GitHub's HTTPS API
// supplies the digest; an unsigned checksum fetched from another host is not trusted.
func (m Manager) CheckStoreUpdate(ctx context.Context, target, current string) (*StoreUpdate, error) {
	if target != "trimui-smart-pro" && target != "magicx-zero-28" {
		return nil, fmt.Errorf("unknown Store build target %q", target)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/jellydn/knulli-app-store/releases?per_page=20", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	client := m.Client
	if client == nil {
		client = defaultHTTPClient()
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Request == nil || response.Request.URL.Scheme != "https" {
		return nil, fmt.Errorf("store update check failed: %s", response.Status)
	}
	var releases []struct {
		Tag    string `json:"tag_name"`
		Draft  bool   `json:"draft"`
		Assets []struct {
			Name   string `json:"name"`
			URL    string `json:"browser_download_url"`
			Digest string `json:"digest"`
			Size   int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("read Store releases: %w", err)
	}
	prefix := "knulli-app-store-" + target + "-experimental-"
	for _, release := range releases {
		if release.Draft {
			continue
		}
		for _, asset := range release.Assets {
			if !strings.HasPrefix(asset.Name, prefix) || !strings.HasSuffix(asset.Name, ".zip") {
				continue
			}
			if asset.Name == prefix+current+".zip" || release.Tag == current {
				return nil, nil
			}
			if release.Tag == "" || strings.ContainsAny(release.Tag, "/\\?#") || asset.URL != storeReleaseBase+release.Tag+"/"+asset.Name {
				return nil, fmt.Errorf("store asset is not a pinned official release")
			}
			digest := strings.TrimPrefix(asset.Digest, "sha256:")
			decoded, err := hex.DecodeString(digest)
			if !strings.HasPrefix(asset.Digest, "sha256:") || err != nil || len(decoded) != 32 || asset.Size <= 0 || asset.Size > manifest.MaxPackageBytes {
				return nil, fmt.Errorf("store asset lacks a valid SHA-256 or byte size")
			}
			return &StoreUpdate{Version: release.Tag, Release: manifest.Release{URL: asset.URL, SHA256: digest, Size: asset.Size, InstalledSize: manifest.MaxPackageBytes, Format: "zip", Immutable: true}}, nil
		}
	}
	return nil, fmt.Errorf("no published Store asset for %s", target)
}

// RecoverTransactions runs before loading the signed catalogue: an interrupted
// self-update may have changed its key-bearing binary but not its catalogue yet.
func (m Manager) RecoverTransactions() error {
	session, err := m.acquireManager()
	if err != nil {
		return err
	}
	session.release()
	return nil
}

// UpdateStore replaces only the seven packaged Store files. Atomic renames keep
// the running Linux executable alive; the next launch uses the new binary/key pair.
func (m Manager) UpdateStore(ctx context.Context, update StoreUpdate) (err error) {
	if !strings.HasPrefix(update.Release.URL, storeReleaseBase) || validateDownloadURL(update.Release.URL) != nil || update.Release.Size <= 0 {
		return fmt.Errorf("store update must use a pinned official release")
	}
	session, err := m.acquireManager()
	if err != nil {
		return err
	}
	defer session.release()
	guard, err := safefs.NewGuard(m.root(), []string{StoreDirectory, "/userdata/roms/ports/Knulli App Store.sh", "/userdata/roms/ports/images/Knulli App Store.png"})
	if err != nil {
		return err
	}
	destination, err := guard.Resolve(StoreDirectory)
	if err != nil {
		return err
	}
	availableBytes := m.AvailableBytes
	if availableBytes == nil {
		availableBytes = safefs.AvailableBytes
	}
	available, err := availableBytes(destination)
	if err != nil {
		return err
	}
	if available < uint64(update.Release.Size) {
		return fmt.Errorf("not enough free space to download the Store update")
	}
	work, err := os.MkdirTemp(session.managerHost, "store-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	client := m.Client
	if client == nil {
		client = defaultHTTPClient()
	}
	archivePath := filepath.Join(work, "release.zip")
	if err := download(ctx, client, update.Release, archivePath); err != nil {
		return err
	}
	files, err := storearchive.Extract(ctx, archivePath, "zip", filepath.Join(work, "staging"), 0, manifest.MaxPackageBytes)
	if err != nil {
		return err
	}
	wanted := map[string]os.FileMode{
		"Knulli App Store.sh":                     0755,
		"images/Knulli App Store.png":             0644,
		"knulli-app-store/knulli-app":             0755,
		"knulli-app-store/knulli-app-ui":          0755,
		"knulli-app-store/catalog-index.json":     0644,
		"knulli-app-store/catalog-index.json.sig": 0644,
		"knulli-app-store/README.txt":             0644,
	}
	if len(files) != len(wanted) {
		return fmt.Errorf("unexpected Store archive inventory")
	}
	var required uint64
	for _, file := range files {
		if _, ok := wanted[file.Relative]; !ok {
			return fmt.Errorf("unexpected Store archive path %q", file.Relative)
		}
		required += uint64(file.Size)
		host, err := guard.Resolve("/userdata/roms/ports/" + file.Relative)
		if err != nil {
			return err
		}
		info, err := os.Lstat(host)
		if err == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("store destination is not a regular file: %s", file.Relative)
			}
			required += uint64(info.Size())
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	available, err = availableBytes(destination)
	if err != nil {
		return err
	}
	if available < required {
		return fmt.Errorf("not enough free space for Store update and rollback: need %d bytes", required)
	}
	transaction, err := safefs.Begin(guard, session.managerHost)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("%w; Store rollback: %v", err, rollbackErr)
			}
		}
	}()
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := transaction.Copy(file.Path, "/userdata/roms/ports/"+file.Relative, wanted[file.Relative]); err != nil {
			return err
		}
	}
	return transaction.Commit()
}
