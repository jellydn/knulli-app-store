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
	Firmware             string
	FirmwareRaw          string
	FirmwareSource       string
	Version              string
	VersionRaw           string
	VersionSource        string
	Arch                 string
	Device               string
	Resolution           string
	ResolutionSource     string
	ResolutionCandidates []ResolutionCandidate
	ABI                  string
	GLIBCVersion         string
	Dependencies         []string
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

func Detect(root string) Info {
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
		if strings.TrimSpace(source.value) != "" && info.FirmwareSource == "" {
			info.FirmwareRaw = source.value
			info.FirmwareSource = source.path + ":" + source.key
		}
		if normalized := normalizeFirmware(source.value); normalized != "" {
			info.Firmware = normalized
			info.FirmwareRaw = source.value
			info.FirmwareSource = source.path + ":" + source.key
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
			info.VersionRaw = source.value
			info.VersionSource = source.path
			break
		}
	}
	for _, devicePath := range []string{"boot/boot/knulli.board", "etc/knulli-device", "boot/batocera.board"} {
		if data, err := os.ReadFile(filepath.Join(root, devicePath)); err == nil {
			info.Device = strings.TrimSpace(string(data))
			break
		}
	}
	info.ResolutionCandidates = filesystemResolutionCandidates(root)
	info = WithResolutionCandidates(info, info.ResolutionCandidates)
	info.ABI = detectABI(root, info.Arch)
	info.GLIBCVersion = detectGLIBCVersion(root)
	info.Dependencies = detectDependencies(root)
	return info
}

func WithResolutionCandidates(info Info, candidates []ResolutionCandidate) Info {
	info.Resolution = ""
	info.ResolutionSource = ""
	info.ResolutionCandidates = append([]ResolutionCandidate(nil), candidates...)
	for _, assessment := range AssessResolutions(candidates) {
		if assessment.Valid {
			info.Resolution = fmt.Sprintf("%dx%d", assessment.Width, assessment.Height)
			info.ResolutionSource = assessment.Source
			break
		}
	}
	return info
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
		if !strings.HasPrefix(info.ResolutionSource, "SDL ") {
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
		info.Device, info.Arch, info.ABI, info.GLIBCVersion, strings.Join(info.Dependencies, ","), info.Resolution, info.ResolutionSource, info.FirmwareRaw, info.Firmware, info.FirmwareSource, info.VersionRaw, info.Version, info.VersionSource)
}

func compatibilityError(field string, current Info, constraint string) error {
	detected := fmt.Sprintf("device=%q architecture=%q abi=%q glibc=%q dependencies=%q resolution=%q resolution_source=%q firmware_raw=%q firmware=%q firmware_source=%q",
		current.Device, current.Arch, current.ABI, current.GLIBCVersion, strings.Join(current.Dependencies, ","), current.Resolution, current.ResolutionSource, current.FirmwareRaw, current.Firmware, current.FirmwareSource)
	if field == "firmware_version" {
		detected += fmt.Sprintf(" version_raw=%q version=%q version_source=%q", current.VersionRaw, current.Version, current.VersionSource)
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
