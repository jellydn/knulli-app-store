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
	Firmware   string
	Version    string
	Arch       string
	Device     string
	Resolution string
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
	for _, release := range []string{"etc/knulli-release", "etc/os-release"} {
		values := readKeyValues(filepath.Join(root, release))
		if info.Firmware == "" {
			id := strings.ToLower(values["ID"])
			if strings.Contains(id, "knulli") {
				info.Firmware = "knulli"
			}
		}
		if info.Version == "" {
			info.Version = values["VERSION_ID"]
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
		return fmt.Errorf("package has no compatibility metadata")
	}
	if current.Firmware != wanted.Firmware {
		return fmt.Errorf("firmware %s is not supported; need %s", current.Firmware, wanted.Firmware)
	}
	if wanted.MinimumVersion != "" && compareVersions(current.Version, wanted.MinimumVersion) < 0 {
		return fmt.Errorf("firmware version %s is older than required %s", current.Version, wanted.MinimumVersion)
	}
	if !includes(wanted.Architectures, current.Arch) {
		return fmt.Errorf("architecture %s is not supported", current.Arch)
	}
	if !includes(wanted.Devices, current.Device) {
		return fmt.Errorf("device %s is not supported", current.Device)
	}
	if !includes(wanted.Resolutions, current.Resolution) {
		return fmt.Errorf("resolution %s is not supported", current.Resolution)
	}
	return nil
}

var versionPart = regexp.MustCompile(`[0-9]+|[a-zA-Z]+`)
var resolutionPattern = regexp.MustCompile(`^[0-9]+x[0-9]+$`)

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
			values[key] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	return values
}
