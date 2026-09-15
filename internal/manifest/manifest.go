package manifest

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
)

const SchemaV1 = "org.knulli.app-store/package-manifest/v1"

var (
	idPattern     = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z0-9][a-z0-9-]*){2,}$`)
	sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type Package struct {
	Schema        string         `json:"schema"`
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Version       string         `json:"version,omitempty"`
	Type          string         `json:"type"`
	Summary       string         `json:"summary"`
	Repository    string         `json:"repository,omitempty"`
	License       string         `json:"license,omitempty"`
	Review        Review         `json:"review"`
	Release       *Release       `json:"release,omitempty"`
	Compatibility *Compatibility `json:"compatibility,omitempty"`
	Install       *Install       `json:"install,omitempty"`
}

type Review struct {
	Status   string     `json:"status"`
	Approval *Approval  `json:"approval,omitempty"`
	Notes    []string   `json:"notes,omitempty"`
	Evidence []Evidence `json:"evidence,omitempty"`
}

type Approval struct {
	Provenance string `json:"provenance"`
}

type Evidence struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
	Note string `json:"note,omitempty"`
}

type Release struct {
	URL           string `json:"url"`
	SHA256        string `json:"sha256"`
	Size          int64  `json:"size"`
	InstalledSize int64  `json:"installed_size"`
	Format        string `json:"format"`
	Immutable     bool   `json:"immutable"`
}

type Compatibility struct {
	Firmware       string   `json:"firmware"`
	MinimumVersion string   `json:"minimum_version"`
	Architectures  []string `json:"architectures"`
	Devices        []string `json:"devices"`
	Resolutions    []string `json:"resolutions"`
}

type Install struct {
	Destination       string   `json:"destination"`
	StripComponents   int      `json:"strip_components,omitempty"`
	Launcher          string   `json:"launcher"`
	Menu              *Menu    `json:"menu,omitempty"`
	Preserve          []string `json:"preserve,omitempty"`
	Network           bool     `json:"network"`
	AllowedWritePaths []string `json:"allowed_write_paths"`
}

type Menu struct {
	Gamelist    string `json:"gamelist"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (p Package) Installable() bool {
	return p.Review.Status == "installable" || p.Review.Status == "verified"
}

func (p Package) Validate() error {
	var problems []string
	if p.Schema != SchemaV1 {
		problems = append(problems, "schema must be "+SchemaV1)
	}
	if !idPattern.MatchString(p.ID) {
		problems = append(problems, "id must be a stable reverse-domain identifier")
	}
	if strings.TrimSpace(p.Name) == "" {
		problems = append(problems, "name is required")
	}
	if !oneOf(p.Type, "utility", "theme", "integration") {
		problems = append(problems, "type must be utility, theme, or integration")
	}
	if strings.TrimSpace(p.Summary) == "" {
		problems = append(problems, "summary is required")
	}
	if !oneOf(p.Review.Status, "candidate", "installable", "verified") {
		problems = append(problems, "review.status must be candidate, installable, or verified")
	}
	if p.Review.Approval != nil {
		if !oneOf(p.Review.Approval.Provenance, "community", "maintainer") {
			problems = append(problems, "review.approval.provenance must be community or maintainer")
		}
	}
	for i, evidence := range p.Review.Evidence {
		if !isHTTPS(evidence.URL) || strings.TrimSpace(evidence.Kind) == "" {
			problems = append(problems, fmt.Sprintf("review.evidence[%d] needs a kind and HTTPS URL", i))
		}
	}
	if p.Review.Status == "verified" && !hasDeviceEvidence(p.Review.Evidence) {
		problems = append(problems, "verified packages require real-device-test evidence")
	}
	if p.Repository != "" && !isHTTPS(p.Repository) {
		problems = append(problems, "repository must use HTTPS")
	}
	if p.Installable() {
		problems = append(problems, p.validateInstallable()...)
	} else if p.Release != nil || p.Compatibility != nil || p.Install != nil {
		problems = append(problems, "candidate packages must not contain actionable release or install metadata")
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func (p Package) validateInstallable() []string {
	var problems []string
	if p.Version == "" || p.Repository == "" || p.License == "" {
		problems = append(problems, "installable packages require version, repository, and license")
	}
	if p.Release == nil {
		problems = append(problems, "installable packages require release metadata")
	} else {
		if !immutableReleaseURL(p.Release.URL, p.Repository) || !p.Release.Immutable {
			problems = append(problems, "release URL must be a version-pinned GitHub release for the declared repository and be declared immutable")
		}
		if !sha256Pattern.MatchString(p.Release.SHA256) {
			problems = append(problems, "release SHA-256 must be 64 lowercase hexadecimal characters")
		}
		if p.Release.Size <= 0 || p.Release.Size > 512<<20 {
			problems = append(problems, "release size must be between 1 byte and 512 MiB")
		}
		if p.Release.InstalledSize <= 0 || p.Release.InstalledSize > 512<<20 {
			problems = append(problems, "installed size must be between 1 byte and 512 MiB")
		}
		if !oneOf(p.Release.Format, "zip", "tar.gz") {
			problems = append(problems, "release format must be zip or tar.gz")
		}
	}
	if p.Compatibility == nil {
		problems = append(problems, "installable packages require compatibility metadata")
	} else {
		if p.Compatibility.Firmware != "knulli" || p.Compatibility.MinimumVersion == "" {
			problems = append(problems, "compatibility must name Knulli and a minimum version")
		}
		if !contains(p.Compatibility.Architectures, "aarch64") {
			problems = append(problems, "initial catalogue packages must include aarch64")
		}
		if len(p.Compatibility.Devices) == 0 {
			problems = append(problems, "at least one tested device is required")
		}
		if len(p.Compatibility.Resolutions) == 0 {
			problems = append(problems, "at least one tested resolution is required")
		}
	}
	if p.Install == nil {
		return append(problems, "installable packages require install metadata")
	}
	if !safeAbsolute(p.Install.Destination) || !under(p.Install.Destination, "/userdata") {
		problems = append(problems, "install destination must be a clean path under /userdata")
	}
	if p.Install.StripComponents < 0 {
		problems = append(problems, "strip_components cannot be negative")
	}
	if !safeRelative(p.Install.Launcher) {
		problems = append(problems, "launcher must be a safe relative path")
	}
	for _, preserve := range p.Install.Preserve {
		if !safeRelative(preserve) {
			problems = append(problems, "preserved paths must be safe relative paths")
		}
	}
	if len(p.Install.AllowedWritePaths) == 0 {
		problems = append(problems, "allowed_write_paths cannot be empty")
	}
	for _, allowed := range p.Install.AllowedWritePaths {
		if !safeAbsolute(allowed) || !under(allowed, "/userdata") {
			problems = append(problems, "allowed write paths must be clean paths under /userdata")
		}
		if under("/userdata/system/knulli-app-store", allowed) {
			problems = append(problems, "package write paths must not include app-manager state")
		}
	}
	if !coveredBy(p.Install.Destination, p.Install.AllowedWritePaths) {
		problems = append(problems, "install destination must be covered by allowed_write_paths")
	}
	if p.Install.Menu != nil {
		if !safeAbsolute(p.Install.Menu.Gamelist) || !coveredBy(p.Install.Menu.Gamelist, p.Install.AllowedWritePaths) {
			problems = append(problems, "menu gamelist must be an allowed absolute path")
		}
		if p.Install.Menu.Path == "" || p.Install.Menu.Name == "" {
			problems = append(problems, "menu path and name are required")
		}
	}
	return problems
}

func immutableReleaseURL(raw, repository string) bool {
	u, err := url.Parse(raw)
	repo, repoErr := url.Parse(strings.TrimSuffix(repository, "/"))
	if err != nil || repoErr != nil || u.Scheme != "https" || u.Host != "github.com" || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if repo.Scheme != "https" || repo.Host != "github.com" || repo.RawQuery != "" || repo.Fragment != "" {
		return false
	}
	prefix := strings.TrimSuffix(repo.Path, "/") + "/releases/download/"
	if !strings.HasPrefix(u.Path, prefix) {
		return false
	}
	remainder := strings.TrimPrefix(u.Path, prefix)
	parts := strings.Split(remainder, "/")
	return len(parts) >= 2 && parts[0] != "" && parts[len(parts)-1] != "" && !strings.EqualFold(parts[0], "latest")
}

func isHTTPS(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != ""
}

func safeAbsolute(value string) bool {
	return strings.HasPrefix(value, "/") && path.Clean(value) == value && value != "/"
}

func safeRelative(value string) bool {
	return value != "" && !strings.HasPrefix(value, "/") && path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

func under(value, parent string) bool {
	return value == parent || strings.HasPrefix(value, parent+"/")
}

func coveredBy(value string, allowed []string) bool {
	for _, parent := range allowed {
		if under(value, parent) {
			return true
		}
	}
	return false
}

func oneOf(value string, values ...string) bool {
	return contains(values, value)
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func hasDeviceEvidence(evidence []Evidence) bool {
	for _, item := range evidence {
		if item.Kind == "real-device-test" {
			return true
		}
	}
	return false
}
