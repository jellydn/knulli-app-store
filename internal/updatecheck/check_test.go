package updatecheck

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

func TestReleaseAssessmentStates(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		releases string
		status   string
		ignored  string
	}{
		{name: "no update", version: "1.0.0", releases: releaseJSON("v1.0.0", false, "demo.zip"), status: "current"},
		{name: "update found", version: "1.0.0", releases: releaseJSON("v1.1.0", false, "demo.zip"), status: "update-found"},
		{name: "prerelease ignored", version: "1.0.0", releases: `[{"tag_name":"v2.0.0-beta","prerelease":true},{"tag_name":"v1.0.0","html_url":"https://github.com/example/demo/releases/tag/v1.0.0","assets":[{"name":"demo.zip","browser_download_url":"https://github.com/example/demo/releases/download/v1.0.0/demo.zip","size":10}]}]`, status: "current", ignored: "v2.0.0-beta"},
		{name: "missing release", version: "1.0.0", releases: `[]`, status: "no-stable-release"},
		{name: "missing assets", version: "1.0.0", releases: `[{"tag_name":"v1.0.0","assets":[]}]`, status: "missing-assets"},
		{name: "moved asset", version: "1.0.0", releases: releaseJSON("v1.0.0", false, "renamed.zip"), status: "current-asset-missing"},
		{name: "older stable is not an update", version: "1.13.0-alpha1", releases: releaseJSON("v1.12.2", false, "demo.zip"), status: "current-newer"},
		{name: "stable replaces same prerelease", version: "1.13.0-alpha1", releases: releaseJSON("v1.13.0", false, "demo.zip"), status: "update-found"},
		{name: "unusual version needs review", version: "scarab", releases: releaseJSON("v2", false, "demo.zip"), status: "manual-version-review"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := metadataServer(t, func(writer http.ResponseWriter, _ *http.Request) { fmt.Fprint(writer, test.releases) })
			defer server.Close()
			report := testChecker(server).Check(context.Background(), []manifest.Package{testPackage(test.version)}, time.Unix(0, 0))
			if got := report.Packages[0]; got.Status != test.status || got.IgnoredPrerelease != test.ignored {
				t.Fatalf("assessment = %#v", got)
			}
		})
	}
}

func TestMalformedMetadataIsReported(t *testing.T) {
	server := metadataServer(t, func(writer http.ResponseWriter, _ *http.Request) { fmt.Fprint(writer, `{`) })
	defer server.Close()
	report := testChecker(server).Check(context.Background(), []manifest.Package{testPackage("1.0.0")}, time.Unix(0, 0))
	if got := report.Packages[0]; got.Status != "metadata-error" || !strings.Contains(got.Error, "decode GitHub release metadata") {
		t.Fatalf("malformed metadata result = %#v", got)
	}
}

func TestRateLimitRetriesWithBoundedDelayAndToken(t *testing.T) {
	calls := 0
	server := metadataServer(t, func(writer http.ResponseWriter, request *http.Request) {
		calls++
		if request.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("token header missing")
		}
		if calls == 1 {
			writer.Header().Set("Retry-After", "99")
			writer.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(writer, releaseJSON("v1.0.0", false, "demo.zip"))
	})
	defer server.Close()
	var delays []time.Duration
	checker := testChecker(server)
	checker.Token = "test-token"
	checker.Sleep = func(delay time.Duration) { delays = append(delays, delay) }
	report := checker.Check(context.Background(), []manifest.Package{testPackage("1.0.0")}, time.Unix(0, 0))
	if calls != 2 || len(delays) != 1 || delays[0] != 30*time.Second || report.Packages[0].Status != "current" {
		t.Fatalf("retry result: calls=%d delays=%v report=%#v", calls, delays, report)
	}
}

func TestRepositoryRequestIsSharedByPackages(t *testing.T) {
	calls := 0
	server := metadataServer(t, func(writer http.ResponseWriter, _ *http.Request) {
		calls++
		fmt.Fprint(writer, releaseJSON("v1.0.0", false, "demo.zip"))
	})
	defer server.Close()
	first := testPackage("1.0.0")
	second := testPackage("0.9.0")
	second.ID = "org.example.second"
	report := testChecker(server).Check(context.Background(), []manifest.Package{first, second}, time.Unix(0, 0))
	if calls != 1 || len(report.Packages) != 2 || report.Packages[1].Status != "update-found" {
		t.Fatalf("shared repository result: calls=%d report=%#v", calls, report)
	}
}

func TestReportContainsAssetsAndManualReview(t *testing.T) {
	report := Report{
		CheckedAt: "2026-09-15T00:00:00Z",
		Packages: []PackageReport{{
			ID: "org.example.demo", Repository: "https://github.com/example/demo", CurrentVersion: "1.0.0", DiscoveredVersion: "v1.1.0", Status: "update-found",
			Assets: []Asset{{Name: "demo.zip", URL: "https://github.com/example/demo/releases/download/v1.1.0/demo.zip", Size: 42, Digest: "sha256:abc"}},
		}},
	}
	markdown := string(Markdown(report))
	for _, wanted := range []string{"update-found", "v1.1.0", "demo.zip", "42 bytes", "sha256:abc", "Manual review"} {
		if !strings.Contains(markdown, wanted) {
			t.Fatalf("report lacks %q: %s", wanted, markdown)
		}
	}
}

func testPackage(version string) manifest.Package {
	return manifest.Package{ID: "org.example.demo", Name: "Demo", Version: version, Repository: "https://github.com/example/demo", Release: &manifest.Release{URL: "https://github.com/example/demo/releases/download/v1.0.0/demo.zip"}}
}

func releaseJSON(tag string, prerelease bool, asset string) string {
	return fmt.Sprintf(`[{"tag_name":%q,"html_url":"https://github.com/example/demo/releases/tag/%s","prerelease":%t,"immutable":true,"assets":[{"name":%q,"browser_download_url":"https://github.com/example/demo/releases/download/%s/%s","size":10,"digest":"sha256:abc"}]}]`, tag, tag, prerelease, asset, tag, asset)
}

func metadataServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func testChecker(server *httptest.Server) Checker {
	return Checker{Client: server.Client(), BaseURL: server.URL, MaxRetries: 2}
}
