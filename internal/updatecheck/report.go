package updatecheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func JSON(report Report) ([]byte, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func Markdown(report Report) []byte {
	packages := append([]PackageReport(nil), report.Packages...)
	sort.Slice(packages, func(i, j int) bool { return packages[i].ID < packages[j].ID })
	var output bytes.Buffer
	fmt.Fprintf(&output, "# Catalogue update report\n\nChecked: %s\n\n", report.CheckedAt)
	for _, pkg := range packages {
		fmt.Fprintf(&output, "## %s\n\n- Status: `%s`\n- Current version: `%s`\n- Discovered stable version: `%s`\n- Repository: %s\n", pkg.ID, pkg.Status, valueOrNone(pkg.CurrentVersion), valueOrNone(pkg.DiscoveredVersion), pkg.Repository)
		if pkg.ReleaseURL != "" {
			fmt.Fprintf(&output, "- Upstream release: %s\n- GitHub immutable: `%t`\n", pkg.ReleaseURL, pkg.ReleaseImmutable)
		}
		if pkg.IgnoredPrerelease != "" {
			fmt.Fprintf(&output, "- Ignored prerelease: `%s`\n", pkg.IgnoredPrerelease)
		}
		if pkg.Error != "" {
			fmt.Fprintf(&output, "- Metadata error: `%s`\n", strings.ReplaceAll(pkg.Error, "`", "'"))
		}
		if len(pkg.Assets) > 0 {
			output.WriteString("- Version-pinned assets:\n")
			for _, asset := range pkg.Assets {
				fmt.Fprintf(&output, "  - `%s` (%d bytes, digest `%s`): %s\n", asset.Name, asset.Size, valueOrNone(asset.Digest), asset.URL)
			}
		}
		output.WriteString("\nManual review: verify release stability and immutability, asset size/SHA-256/extracted size, license, archive inventory, write and preserve paths, dependencies, compatibility evidence, and real-device results. Never copy metadata into a manifest without review.\n\n")
	}
	return output.Bytes()
}

func valueOrNone(value string) string {
	if value == "" {
		return "none"
	}
	return value
}
