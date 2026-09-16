package platform

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

type Info struct {
	Firmware     string
	Version      string
	Arch         string
	Device       string
	Resolution   string
	ABI          string
	GLIBCVersion string
	Dependencies []string

	evidence evidence
}

type evidence struct {
	FirmwareRaw      string
	FirmwareSource   string
	VersionRaw       string
	VersionSource    string
	ResolutionSource string
	Candidates       []ResolutionCandidate
}

type Field string

const (
	FieldFirmware   Field = "firmware"
	FieldVersion    Field = "version"
	FieldResolution Field = "resolution"
)

// Source is immutable provenance for one resolved platform field.
type Source struct {
	Raw      string
	Location string
}

// Evidence returns the provenance for one semantic field.
func (i Info) Evidence(field Field) Source {
	switch field {
	case FieldFirmware:
		return Source{Raw: i.evidence.FirmwareRaw, Location: i.evidence.FirmwareSource}
	case FieldVersion:
		return Source{Raw: i.evidence.VersionRaw, Location: i.evidence.VersionSource}
	case FieldResolution:
		return Source{Raw: i.Resolution, Location: i.evidence.ResolutionSource}
	default:
		return Source{}
	}
}

// ResolutionCandidates returns an independent copy of the detected candidates.
func (i Info) ResolutionCandidates() []ResolutionCandidate {
	return append([]ResolutionCandidate(nil), i.evidence.Candidates...)
}

// Clone returns an independent copy of Info's slice-backed data.
func (i Info) Clone() Info {
	i.Dependencies = append([]string(nil), i.Dependencies...)
	i.evidence.Candidates = i.ResolutionCandidates()
	return i
}

// Option adjusts one detection input before resolution. Every option is a
// no-op on the zero value.
type Option func(*request)

type request struct {
	root       string
	firmware   string
	version    string
	arch       string
	device     string
	resolution string
}

// WithFirmware overrides the normalized firmware identity.
func WithFirmware(value string) Option {
	return func(r *request) { r.firmware = value }
}

// WithVersion overrides the normalized firmware version.
func WithVersion(value string) Option {
	return func(r *request) { r.version = value }
}

// WithArch overrides the detected architecture verbatim.
func WithArch(value string) Option {
	return func(r *request) { r.arch = value }
}

// WithDevice overrides the detected device identity.
func WithDevice(value string) Option {
	return func(r *request) { r.device = value }
}

// WithResolutionOverride prepends a WIDTHxHEIGHT candidate ahead of every
// detected source.
func WithResolutionOverride(value string) Option {
	return func(r *request) { r.resolution = value }
}

// Resolve reads the platform from Knulli-owned files and applies the given
// options. It is the single entry point for detection: the returned Info
// always carries consistent normalized values and matching evidence.
func Resolve(root string, options ...Option) Info {
	request := &request{root: root}
	for _, option := range options {
		option(request)
	}
	effectiveRoot := request.root
	if effectiveRoot == "" {
		effectiveRoot = "/"
	}
	info := detect(effectiveRoot)
	if request.firmware != "" {
		info.Firmware = request.firmware
		info.evidence.FirmwareRaw = request.firmware
		info.evidence.FirmwareSource = "command-line override"
	}
	if request.version != "" {
		info.Version = request.version
		info.evidence.VersionRaw = request.version
		info.evidence.VersionSource = "command-line override"
	}
	if request.arch != "" {
		info.Arch = request.arch
		// ABI detection depends on the selected architecture. Recompute it when
		// an override describes a fixture for a different host architecture.
		info.ABI = detectABI(effectiveRoot, info.Arch)
	}
	if request.device != "" {
		info.Device = request.device
	}
	if request.resolution != "" {
		candidates := append([]ResolutionCandidate{ResolutionCandidateFromString("command-line override", request.resolution)}, info.ResolutionCandidates()...)
		info = info.WithCandidates(candidates)
	}
	return info
}

type ResolutionCandidate struct {
	Source string
	Width  int
	Height int
	Error  string
}

type ResolutionAssessment struct {
	ResolutionCandidate
	Valid  bool
	Reason string
}

func detect(root string) Info {
	if root == "" {
		root = "/"
	}
	arch := runtime.GOARCH
	if arch == "arm64" {
		arch = "aarch64"
	}
	info := Info{Arch: arch}
	osRelease := readKeyValues(filepath.Join(root, "etc/os-release"))
	legacyRelease := readKeyValues(filepath.Join(root, "etc/knulli-release"))
	firmwareSources := []struct {
		path  string
		key   string
		value string
	}{
		{path: "/etc/os-release", key: "OS_NAME", value: osRelease["OS_NAME"]},
		{path: "/etc/knulli-release", key: "OS_NAME", value: legacyRelease["OS_NAME"]},
		{path: "/etc/knulli-release", key: "ID", value: legacyRelease["ID"]},
		{path: "/etc/knulli-release", key: "NAME", value: legacyRelease["NAME"]},
		{path: "/etc/os-release", key: "ID", value: osRelease["ID"]},
	}
	for _, source := range firmwareSources {
		if strings.TrimSpace(source.value) != "" && info.evidence.FirmwareSource == "" {
			info.evidence.FirmwareRaw = source.value
			info.evidence.FirmwareSource = source.path + ":" + source.key
		}
		if normalized := normalizeFirmware(source.value); normalized != "" {
			info.Firmware = normalized
			info.evidence.FirmwareRaw = source.value
			info.evidence.FirmwareSource = source.path + ":" + source.key
			break
		}
	}
	versionSources := []struct {
		path  string
		value string
	}{
		{path: "/usr/share/knulli/knulli.version", value: readText(filepath.Join(root, "usr/share/knulli/knulli.version"))},
		{path: "/etc/os-release:OS_VERSION", value: osRelease["OS_VERSION"]},
		{path: "/etc/os-release:OS_DATE", value: osRelease["OS_DATE"]},
		{path: "/etc/knulli-release:VERSION_ID", value: legacyRelease["VERSION_ID"]},
	}
	for _, source := range versionSources {
		if normalized := normalizeVersion(source.value); normalized != "" {
			info.Version = normalized
			info.evidence.VersionRaw = source.value
			info.evidence.VersionSource = source.path
			break
		}
	}
	for _, devicePath := range []string{"boot/boot/knulli.board", "etc/knulli-device", "boot/batocera.board"} {
		if data, err := os.ReadFile(filepath.Join(root, devicePath)); err == nil {
			info.Device = strings.TrimSpace(string(data))
			break
		}
	}
	info = info.WithCandidates(filesystemResolutionCandidates(root))
	info.ABI = detectABI(root, info.Arch)
	info.GLIBCVersion = detectGLIBCVersion(root)
	info.Dependencies = detectDependencies(root)
	return info
}

// WithCandidates returns a copy of the info with the given resolution
// candidates and re-selects the resolution from them.
func (i Info) WithCandidates(candidates []ResolutionCandidate) Info {
	i.Resolution = ""
	i.evidence.ResolutionSource = ""
	i.evidence.Candidates = append([]ResolutionCandidate(nil), candidates...)
	for _, assessment := range AssessResolutions(candidates) {
		if assessment.Valid {
			i.Resolution = fmt.Sprintf("%dx%d", assessment.Width, assessment.Height)
			i.evidence.ResolutionSource = assessment.Source
			break
		}
	}
	return i
}

func AssessResolutions(candidates []ResolutionCandidate) []ResolutionAssessment {
	result := make([]ResolutionAssessment, 0, len(candidates))
	for _, candidate := range candidates {
		assessment := ResolutionAssessment{ResolutionCandidate: candidate, Valid: true, Reason: "accepted"}
		switch {
		case candidate.Error != "":
			assessment.Valid, assessment.Reason = false, candidate.Error
		case candidate.Width < 320 || candidate.Height < 200:
			assessment.Valid, assessment.Reason = false, "below minimum 320x200"
		case candidate.Width > 7680 || candidate.Height > 4320:
			assessment.Valid, assessment.Reason = false, "above maximum 7680x4320"
		case float64(candidate.Width)/float64(candidate.Height) < 0.5 || float64(candidate.Width)/float64(candidate.Height) > 3.5:
			assessment.Valid, assessment.Reason = false, "aspect ratio is outside 1:2 to 3.5:1"
		}
		result = append(result, assessment)
	}
	return result
}

func DisplayName(device string) string {
	device = strings.TrimSpace(device)
	if strings.EqualFold(device, "trimui-smart-pro") {
		return "TrimUI Smart Pro"
	}
	if strings.EqualFold(device, "magicx-zero-28") {
		return "MagicX Zero 28"
	}
	if device == "" {
		return "Unknown device"
	}
	return "Unknown device (" + device + ")"
}

func DisplayHeader(info Info, runtimeWidth, runtimeHeight int) string {
	resolution := "size unknown"
	runtime := AssessResolutions([]ResolutionCandidate{{Source: "runtime header", Width: runtimeWidth, Height: runtimeHeight}})[0]
	if runtime.Valid {
		resolution = fmt.Sprintf("%dx%d", runtimeWidth, runtimeHeight)
	} else if info.Resolution != "" {
		resolution = info.Resolution
		if !strings.HasPrefix(info.Evidence(FieldResolution).Location, "SDL ") {
			resolution += " fallback"
		}
	}
	return DisplayName(info.Device) + " / " + resolution
}

func Check(pkg manifest.Package, current Info) error {
	wanted := pkg.Compatibility
	if wanted == nil {
		return compatibilityError("metadata", current, "package has no compatibility metadata")
	}
	if current.Firmware == "" || !strings.EqualFold(current.Firmware, wanted.Firmware) {
		return compatibilityError("firmware", current, fmt.Sprintf("package requires firmware=%q", wanted.Firmware))
	}
	if wanted.MinimumVersion != "" {
		constraint := fmt.Sprintf("package requires minimum_version=%q", wanted.MinimumVersion)
		if current.Version == "" || !comparableVersions(current.Version, wanted.MinimumVersion) {
			return compatibilityError("firmware_version", current, constraint+"; detected release ordering is unknown")
		}
		if compareVersions(current.Version, wanted.MinimumVersion) < 0 {
			return compatibilityError("firmware_version", current, constraint)
		}
	}
	if !includes(wanted.Architectures, current.Arch) {
		return compatibilityError("architecture", current, fmt.Sprintf("package allows architectures=%q", strings.Join(wanted.Architectures, ",")))
	}
	if len(wanted.ABIs) > 0 && !includes(wanted.ABIs, current.ABI) {
		return compatibilityError("abi", current, fmt.Sprintf("package requires one of abis=%q", strings.Join(wanted.ABIs, ",")))
	}
	if wanted.MinimumGLIBC != "" && (current.GLIBCVersion == "" || compareVersions(current.GLIBCVersion, wanted.MinimumGLIBC) < 0) {
		return compatibilityError("abi", current, fmt.Sprintf("package requires glibc>=%q", wanted.MinimumGLIBC))
	}
	for _, dependency := range wanted.Dependencies {
		if !includes(current.Dependencies, dependency) {
			return compatibilityError("dependency", current, fmt.Sprintf("package requires runtime dependency=%q", dependency))
		}
	}
	if current.Device == "" {
		return compatibilityError("device", current, "detected device identity is empty")
	}
	if wanted.DeviceScope != "any" && !includes(wanted.Devices, current.Device) {
		return compatibilityError("device", current, fmt.Sprintf("package allows devices=%q", strings.Join(wanted.Devices, ",")))
	}
	if current.Resolution == "" {
		return compatibilityError("resolution", current, "no validated display resolution was detected")
	}
	if wanted.DisplayBounds != nil {
		var width, height int
		if _, err := fmt.Sscanf(current.Resolution, "%dx%d", &width, &height); err != nil || width < wanted.DisplayBounds.MinimumWidth || height < wanted.DisplayBounds.MinimumHeight || width > wanted.DisplayBounds.MaximumWidth || height > wanted.DisplayBounds.MaximumHeight {
			return compatibilityError("resolution", current, fmt.Sprintf("package allows display bounds=%dx%d..%dx%d", wanted.DisplayBounds.MinimumWidth, wanted.DisplayBounds.MinimumHeight, wanted.DisplayBounds.MaximumWidth, wanted.DisplayBounds.MaximumHeight))
		}
	} else if !includes(wanted.Resolutions, current.Resolution) {
		return compatibilityError("resolution", current, fmt.Sprintf("package allows resolutions=%q", strings.Join(wanted.Resolutions, ",")))
	}
	return nil
}

func Summary(info Info) string {
	return fmt.Sprintf("device=%q architecture=%q abi=%q glibc=%q dependencies=%q resolution=%q resolution_source=%q firmware_raw=%q firmware=%q firmware_source=%q version_raw=%q version=%q version_source=%q",
		info.Device, info.Arch, info.ABI, info.GLIBCVersion, strings.Join(info.Dependencies, ","), info.Resolution, info.Evidence(FieldResolution).Location, info.Evidence(FieldFirmware).Raw, info.Firmware, info.Evidence(FieldFirmware).Location, info.Evidence(FieldVersion).Raw, info.Version, info.Evidence(FieldVersion).Location)
}

func compatibilityError(field string, current Info, constraint string) error {
	detected := fmt.Sprintf("device=%q architecture=%q abi=%q glibc=%q dependencies=%q resolution=%q resolution_source=%q firmware_raw=%q firmware=%q firmware_source=%q",
		current.Device, current.Arch, current.ABI, current.GLIBCVersion, strings.Join(current.Dependencies, ","), current.Resolution, current.Evidence(FieldResolution).Location, current.Evidence(FieldFirmware).Raw, current.Firmware, current.Evidence(FieldFirmware).Location)
	if field == "firmware_version" {
		detected += fmt.Sprintf(" version_raw=%q version=%q version_source=%q", current.Evidence(FieldVersion).Raw, current.Version, current.Evidence(FieldVersion).Location)
	}
	return fmt.Errorf("compatibility failed field=%s: detected %s; %s", field, detected, constraint)
}

func detectABI(root, architecture string) string {
	if architecture != "aarch64" {
		return ""
	}
	for _, name := range []string{"lib/ld-linux-aarch64.so.1", "lib64/ld-linux-aarch64.so.1"} {
		if info, err := os.Stat(filepath.Join(root, name)); err == nil && info.Mode().IsRegular() {
			return "linux-aarch64-glibc"
		}
	}
	return ""
}

func detectGLIBCVersion(root string) string {
	matcher := regexp.MustCompile(`GLIBC_([0-9]+\.[0-9]+)`)
	for _, name := range []string{"lib/libc.so.6", "lib64/libc.so.6", "usr/lib/libc.so.6"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		latest := ""
		for _, match := range matcher.FindAllSubmatch(data, -1) {
			version := string(match[1])
			if latest == "" || compareVersions(version, latest) > 0 {
				latest = version
			}
		}
		return latest
	}
	return ""
}

func detectDependencies(root string) []string {
	wanted := map[string][]string{
		"sdl2":       {"usr/lib/libSDL2-2.0.so.0", "lib/libSDL2-2.0.so.0"},
		"sdl2-image": {"usr/lib/libSDL2_image-2.0.so.0", "lib/libSDL2_image-2.0.so.0"},
		"sdl2-ttf":   {"usr/lib/libSDL2_ttf-2.0.so.0", "lib/libSDL2_ttf-2.0.so.0"},
		"libc":       {"lib/libc.so.6", "lib64/libc.so.6", "usr/lib/libc.so.6"},
		"libresolv":  {"lib/libresolv.so.2", "lib64/libresolv.so.2", "usr/lib/libresolv.so.2"},
		"libpthread": {"lib/libpthread.so.0", "lib64/libpthread.so.0", "usr/lib/libpthread.so.0"},
	}
	var found []string
	for dependency, paths := range wanted {
		for _, name := range paths {
			if _, err := os.Stat(filepath.Join(root, name)); err == nil {
				found = append(found, dependency)
				break
			}
		}
	}
	sort.Strings(found)
	return found
}

var versionPart = regexp.MustCompile(`[0-9]+|[a-zA-Z]+`)
var numericVersion = regexp.MustCompile(`^[0-9]`)
var resolutionNumbers = regexp.MustCompile(`([0-9]+)[x,]([0-9]+)`)

func normalizeFirmware(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "knulli") {
		return "knulli"
	}
	return ""
}

func normalizeVersion(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func comparableVersions(left, right string) bool {
	return strings.EqualFold(left, right) || (numericVersion.MatchString(left) && numericVersion.MatchString(right))
}

func compareVersions(left, right string) int {
	a := versionPart.FindAllString(strings.ToLower(left), -1)
	b := versionPart.FindAllString(strings.ToLower(right), -1)
	for index := 0; index < len(a) || index < len(b); index++ {
		if index >= len(a) {
			return -1
		}
		if index >= len(b) {
			return 1
		}
		leftNumber, leftErr := strconv.Atoi(a[index])
		rightNumber, rightErr := strconv.Atoi(b[index])
		var result int
		if leftErr == nil && rightErr == nil {
			result = leftNumber - rightNumber
		} else {
			result = strings.Compare(a[index], b[index])
		}
		if result < 0 {
			return -1
		}
		if result > 0 {
			return 1
		}
	}
	return 0
}

func includes(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(value, wanted) {
			return true
		}
	}
	return false
}

func readKeyValues(path string) map[string]string {
	values := make(map[string]string)
	file, err := os.Open(path)
	if err != nil {
		return values
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, found := strings.Cut(scanner.Text(), "=")
		if found {
			values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return values
}

func readText(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func filesystemResolutionCandidates(root string) []ResolutionCandidate {
	sources := []struct {
		path   string
		source string
	}{
		{path: "sys/class/graphics/fb0/mode", source: "/sys/class/graphics/fb0/mode"},
		{path: "sys/class/graphics/fb0/modes", source: "/sys/class/graphics/fb0/modes"},
		{path: "sys/class/graphics/fb0/virtual_size", source: "/sys/class/graphics/fb0/virtual_size"},
	}
	var candidates []ResolutionCandidate
	for _, item := range sources {
		value := readText(filepath.Join(root, item.path))
		if value == "" {
			continue
		}
		candidates = append(candidates, ResolutionCandidateFromString(item.source, value))
	}
	return candidates
}

func ResolutionCandidateFromString(source, value string) ResolutionCandidate {
	matches := resolutionNumbers.FindStringSubmatch(strings.TrimSpace(value))
	if len(matches) != 3 {
		return ResolutionCandidate{Source: source, Error: "value does not contain WIDTHxHEIGHT"}
	}
	width, widthErr := strconv.Atoi(matches[1])
	height, heightErr := strconv.Atoi(matches[2])
	if widthErr != nil || heightErr != nil {
		return ResolutionCandidate{Source: source, Error: "width or height is not an integer"}
	}
	return ResolutionCandidate{Source: source, Width: width, Height: height}
}
