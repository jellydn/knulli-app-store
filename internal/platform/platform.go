package platform

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/manifest"
)

type Info struct {
	Firmware       string
	FirmwareRaw    string
	FirmwareSource string
	Version        string
	VersionRaw     string
	VersionSource  string
	Arch           string
	Device         string
	Resolution     string
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
	if data, err := os.ReadFile(filepath.Join(root, "sys/class/graphics/fb0/virtual_size")); err == nil {
		info.Resolution = strings.ReplaceAll(strings.TrimSpace(string(data)), ",", "x")
	}
	return info
}

func DisplayName(device string) string {
	device = strings.TrimSpace(device)
	if strings.EqualFold(device, "trimui-smart-pro") {
		return "TrimUI Smart Pro"
	}
	if device == "" {
		return "Unknown device"
	}
	return "Unknown device (" + device + ")"
}

func DisplayHeader(info Info, runtimeWidth, runtimeHeight int) string {
	resolution := "size unknown"
	if runtimeWidth > 0 && runtimeHeight > 0 {
		resolution = fmt.Sprintf("%dx%d", runtimeWidth, runtimeHeight)
	} else if resolutionPattern.MatchString(info.Resolution) {
		resolution = info.Resolution + " fallback"
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
	if !includes(wanted.Devices, current.Device) {
		return compatibilityError("device", current, fmt.Sprintf("package allows devices=%q", strings.Join(wanted.Devices, ",")))
	}
	if !includes(wanted.Resolutions, current.Resolution) {
		return compatibilityError("resolution", current, fmt.Sprintf("package allows resolutions=%q", strings.Join(wanted.Resolutions, ",")))
	}
	return nil
}

func Summary(info Info) string {
	return fmt.Sprintf("device=%q architecture=%q resolution=%q firmware_raw=%q firmware=%q firmware_source=%q version_raw=%q version=%q version_source=%q",
		info.Device, info.Arch, info.Resolution, info.FirmwareRaw, info.Firmware, info.FirmwareSource, info.VersionRaw, info.Version, info.VersionSource)
}

func compatibilityError(field string, current Info, constraint string) error {
	detected := fmt.Sprintf("device=%q architecture=%q resolution=%q firmware_raw=%q firmware=%q firmware_source=%q",
		current.Device, current.Arch, current.Resolution, current.FirmwareRaw, current.Firmware, current.FirmwareSource)
	if field == "firmware_version" {
		detected += fmt.Sprintf(" version_raw=%q version=%q version_source=%q", current.VersionRaw, current.Version, current.VersionSource)
	}
	return fmt.Errorf("compatibility failed field=%s: detected %s; %s", field, detected, constraint)
}

var versionPart = regexp.MustCompile(`[0-9]+|[a-zA-Z]+`)
var resolutionPattern = regexp.MustCompile(`^[0-9]+x[0-9]+$`)
var numericVersion = regexp.MustCompile(`^[0-9]`)

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
