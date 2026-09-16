package manifest

import (
	"encoding/hex"
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
	idPattern      = regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z0-9][a-z0-9-]*){2,}$`)
	sha256Pattern  = regexp.MustCompile(`^[a-f0-9]{64}$`)
	versionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
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
	Kind           string `json:"kind"`
	URL            string `json:"url"`
	Note           string `json:"note,omitempty"`
	Tester         string `json:"tester,omitempty"`
	Date           string `json:"date,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	Firmware       string `json:"firmware,omitempty"`
	Architecture   string `json:"architecture,omitempty"`
	Device         string `json:"device,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	Result         string `json:"result,omitempty"`
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
	Firmware       string         `json:"firmware"`
	MinimumVersion string         `json:"minimum_version"`
	Architectures  []string       `json:"architectures"`
	ABIs           []string       `json:"abis,omitempty"`
	MinimumGLIBC   string         `json:"minimum_glibc,omitempty"`
	Dependencies   []string       `json:"dependencies,omitempty"`
	Devices        []string       `json:"devices,omitempty"`
	DeviceScope    string         `json:"device_scope,omitempty"`
	Resolutions    []string       `json:"resolutions,omitempty"`
	DisplayBounds  *DisplayBounds `json:"display_bounds,omitempty"`
}

type DisplayBounds struct {
	MinimumWidth  int `json:"minimum_width"`
	MinimumHeight int `json:"minimum_height"`
	MaximumWidth  int `json:"maximum_width"`
	MaximumHeight int `json:"maximum_height"`
}

type Install struct {
	Destination       string        `json:"destination"`
	StripComponents   int           `json:"strip_components,omitempty"`
	Launcher          string        `json:"launcher"`
	Executables       []string      `json:"executables,omitempty"`
	Warning           string        `json:"warning,omitempty"`
	Menu              *Menu         `json:"menu,omitempty"`
	Preserve          []string      `json:"preserve,omitempty"`
	Network           bool          `json:"network"`
	AllowedWritePaths []string      `json:"allowed_write_paths"`
	BinaryPatches     []BinaryPatch `json:"binary_patches,omitempty"`
}

type BinaryPatch struct {
	Path      string `json:"path"`
	Offset    int64  `json:"offset"`
	BeforeHex string `json:"before_hex"`
	AfterHex  string `json:"after_hex"`
	SHA256    string `json:"sha256"`
}

type Menu struct {
	Gamelist    string `json:"gamelist"`
	Path        string `json:"path"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (p Package) Installable() bool {
	return p.Review.Status == "experimental" || p.Review.Status == "installable" || p.Review.Status == "verified"
}

func (p Package) Experimental() bool {
	return p.Review.Status == "experimental"
}

func (p Package) DeviceTested(firmware, architecture, device, resolution string) bool {
	for _, item := range p.Review.Evidence {
		if item.Kind == "real-device-test" && item.Result == "passed" && item.PackageVersion == p.Version && strings.EqualFold(item.Firmware, firmware) && strings.EqualFold(item.Architecture, architecture) && strings.EqualFold(item.Device, device) && strings.EqualFold(item.Resolution, resolution) {
			return true
		}
	}
	return false
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
	if !oneOf(p.Review.Status, "candidate", "experimental", "installable", "verified") {
		problems = append(problems, "review.status must be candidate, experimental, installable, or verified")
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
		if evidence.Kind == "real-device-test" && (strings.TrimSpace(evidence.Tester) == "" || strings.TrimSpace(evidence.Date) == "" || evidence.PackageVersion == "" || evidence.Firmware == "" || evidence.Architecture == "" || evidence.Device == "" || evidence.Resolution == "" || evidence.Result != "passed") {
			problems = append(problems, fmt.Sprintf("review.evidence[%d] real-device-test needs tester, date, package version, firmware, architecture, device, resolution, and passed result", i))
		}
	}
	if p.Review.Status == "verified" && !hasVerifiedMatrixEvidence(p) {
		problems = append(problems, "verified packages require real-device-test evidence for every declared device and resolution")
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
		if p.Compatibility.Firmware != "knulli" {
			problems = append(problems, "compatibility must name Knulli")
		}
		if !p.Experimental() && p.Compatibility.MinimumVersion == "" {
			problems = append(problems, "non-experimental compatibility requires a minimum version")
		}
		if !contains(p.Compatibility.Architectures, "aarch64") {
			problems = append(problems, "initial catalogue packages must include aarch64")
		}
		if p.Compatibility.DeviceScope == "any" {
			if !p.Experimental() {
				problems = append(problems, "broad device scope is allowed only for experimental packages")
			}
		} else if p.Compatibility.DeviceScope != "" {
			problems = append(problems, "device_scope must be any when set")
		} else if len(p.Compatibility.Devices) == 0 {
			problems = append(problems, "at least one declared device or experimental broad device scope is required")
		}
		if p.Compatibility.DisplayBounds != nil {
			bounds := p.Compatibility.DisplayBounds
			if !p.Experimental() {
				problems = append(problems, "display bounds are allowed only for experimental packages")
			}
			if bounds.MinimumWidth < 320 || bounds.MinimumHeight < 200 || bounds.MaximumWidth < bounds.MinimumWidth || bounds.MaximumHeight < bounds.MinimumHeight || bounds.MaximumWidth > 7680 || bounds.MaximumHeight > 4320 {
				problems = append(problems, "display_bounds must be ordered within 320x200 and 7680x4320")
			}
		} else if len(p.Compatibility.Resolutions) == 0 {
			problems = append(problems, "at least one declared resolution or experimental display bound is required")
		}
		if len(p.Compatibility.ABIs) == 0 {
			problems = append(problems, "compatibility must declare at least one runtime ABI")
		}
		if len(p.Compatibility.Dependencies) == 0 {
			problems = append(problems, "compatibility must declare runtime dependencies")
		}
		if p.Compatibility.MinimumGLIBC != "" && !versionPattern.MatchString(p.Compatibility.MinimumGLIBC) {
			problems = append(problems, "minimum_glibc must be a numeric major.minor version")
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
	if p.Experimental() && strings.TrimSpace(p.Install.Warning) == "" {
		problems = append(problems, "experimental packages require an install warning")
	}
	executables := make(map[string]bool)
	for _, executable := range p.Install.Executables {
		if !safeRelative(executable) {
			problems = append(problems, "executables must be safe relative paths")
		}
		if executables[executable] {
			problems = append(problems, "executables must be unique")
		}
		executables[executable] = true
	}
	for _, preserve := range p.Install.Preserve {
		if !safeRelative(preserve) {
			problems = append(problems, "preserved paths must be safe relative paths")
		}
	}
	for _, patch := range p.Install.BinaryPatches {
		before, beforeErr := hex.DecodeString(patch.BeforeHex)
		after, afterErr := hex.DecodeString(patch.AfterHex)
		if !safeRelative(patch.Path) || patch.Offset < 0 || beforeErr != nil || afterErr != nil || len(before) == 0 || len(before) != len(after) || !sha256Pattern.MatchString(patch.SHA256) {
			problems = append(problems, "binary patches require a safe path, non-negative offset, equal non-empty hexadecimal bytes, and final SHA-256")
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

func hasVerifiedMatrixEvidence(pkg Package) bool {
	if pkg.Compatibility == nil {
		return false
	}
	for _, device := range pkg.Compatibility.Devices {
		for _, resolution := range pkg.Compatibility.Resolutions {
			found := false
			for _, architecture := range pkg.Compatibility.Architectures {
				if pkg.DeviceTested(pkg.Compatibility.Firmware, architecture, device, resolution) {
					found = true
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
