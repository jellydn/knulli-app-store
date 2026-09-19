package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/version"
)

type Checker struct {
	Client     *http.Client
	BaseURL    string
	Token      string
	Sleep      func(time.Duration)
	MaxRetries int
}

type Report struct {
	CheckedAt string          `json:"checked_at"`
	Packages  []PackageReport `json:"packages"`
}

type PackageReport struct {
	ID                string  `json:"id"`
	Repository        string  `json:"repository"`
	CurrentVersion    string  `json:"current_version"`
	DiscoveredVersion string  `json:"discovered_version,omitempty"`
	ReleaseURL        string  `json:"release_url,omitempty"`
	ReleaseImmutable  bool    `json:"release_immutable,omitempty"`
	Status            string  `json:"status"`
	IgnoredPrerelease string  `json:"ignored_prerelease,omitempty"`
	Assets            []Asset `json:"assets,omitempty"`
	Error             string  `json:"error,omitempty"`
}

type Asset struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	Digest string `json:"digest,omitempty"`
}

type release struct {
	TagName    string         `json:"tag_name"`
	HTMLURL    string         `json:"html_url"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Immutable  bool           `json:"immutable"`
	Assets     []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

func (checker Checker) Check(ctx context.Context, packages []manifest.Package, now time.Time) Report {
	report := Report{CheckedAt: now.UTC().Format(time.RFC3339), Packages: make([]PackageReport, len(packages))}
	byRepository := make(map[string][]release)
	repositoryErrors := make(map[string]error)
	seen := make(map[string]bool)
	for _, pkg := range packages {
		if seen[pkg.Repository] {
			continue
		}
		seen[pkg.Repository] = true
		releases, err := checker.releases(ctx, pkg.Repository)
		if err != nil {
			repositoryErrors[pkg.Repository] = err
			continue
		}
		byRepository[pkg.Repository] = releases
	}
	for index, pkg := range packages {
		report.Packages[index] = assess(pkg, byRepository[pkg.Repository], repositoryErrors[pkg.Repository])
	}
	return report
}

func (checker Checker) releases(ctx context.Context, repository string) ([]release, error) {
	repositoryURL, err := url.Parse(repository)
	if err != nil || repositoryURL.Scheme != "https" || repositoryURL.Host != "github.com" {
		return nil, fmt.Errorf("repository is not a supported GitHub URL")
	}
	parts := strings.Split(strings.Trim(repositoryURL.Path, "/"), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("repository is not a supported GitHub URL")
	}
	base := strings.TrimSuffix(checker.BaseURL, "/")
	if base == "" {
		base = "https://api.github.com"
	}
	endpoint := base + "/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/releases?per_page=20"
	client := checker.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	retries := checker.MaxRetries
	if retries == 0 {
		retries = 2
	}
	for attempt := 0; ; attempt++ {
		request, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if requestErr != nil {
			return nil, requestErr
		}
		request.Header.Set("Accept", "application/vnd.github+json")
		request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if checker.Token != "" {
			request.Header.Set("Authorization", "Bearer "+checker.Token)
		}
		response, requestErr := client.Do(request)
		if requestErr != nil {
			return nil, requestErr
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 2<<20))
		response.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			var releases []release
			if err := json.Unmarshal(body, &releases); err != nil {
				return nil, fmt.Errorf("decode GitHub release metadata: %w", err)
			}
			return releases, nil
		}
		if attempt >= retries || !retryable(response.StatusCode, response.Header) {
			return nil, fmt.Errorf("GitHub release metadata returned HTTP %d", response.StatusCode)
		}
		checker.sleep(retryDelay(response.Header))
	}
}

func assess(pkg manifest.Package, releases []release, err error) PackageReport {
	result := PackageReport{ID: pkg.ID, Repository: pkg.Repository, CurrentVersion: pkg.Version, Status: "no-stable-release"}
	if err != nil {
		result.Status = "metadata-error"
		result.Error = err.Error()
		return result
	}
	var stable *release
	for index := range releases {
		item := &releases[index]
		if item.Draft {
			continue
		}
		if item.Prerelease {
			if result.IgnoredPrerelease == "" {
				result.IgnoredPrerelease = item.TagName
			}
			continue
		}
		stable = item
		break
	}
	if stable == nil {
		return result
	}
	result.DiscoveredVersion = stable.TagName
	result.ReleaseURL = stable.HTMLURL
	result.ReleaseImmutable = stable.Immutable
	result.Assets = make([]Asset, len(stable.Assets))
	for index, asset := range stable.Assets {
		result.Assets[index] = Asset(asset)
	}
	sort.Slice(result.Assets, func(i, j int) bool { return result.Assets[i].Name < result.Assets[j].Name })
	if len(stable.Assets) == 0 {
		result.Status = "missing-assets"
		return result
	}
	order, comparable := version.CompareNumeric(stable.TagName, pkg.Version)
	if comparable && order > 0 {
		result.Status = "update-found"
		return result
	}
	if !comparable && version.Normalize(pkg.Version) != version.Normalize(stable.TagName) {
		result.Status = "manual-version-review"
		return result
	}
	if comparable && order < 0 {
		result.Status = "current-newer"
		return result
	}
	result.Status = "current"
	if pkg.Release != nil && !containsAsset(result.Assets, path.Base(pkg.Release.URL)) {
		result.Status = "current-asset-missing"
	}
	return result
}

func containsAsset(assets []Asset, name string) bool {
	for _, asset := range assets {
		if asset.Name == name {
			return true
		}
	}
	return false
}

func retryable(status int, header http.Header) bool {
	return status == http.StatusTooManyRequests || status >= 500 || status == http.StatusForbidden && header.Get("X-RateLimit-Remaining") == "0"
}

func retryDelay(header http.Header) time.Duration {
	seconds, err := strconv.Atoi(header.Get("Retry-After"))
	if err != nil || seconds < 1 {
		return time.Second
	}
	if seconds > 30 {
		seconds = 30
	}
	return time.Duration(seconds) * time.Second
}

func (checker Checker) sleep(delay time.Duration) {
	if checker.Sleep != nil {
		checker.Sleep(delay)
		return
	}
	time.Sleep(delay)
}
