package appstore

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jellydn/knulli-app-store/internal/catalog"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
)

func TestServiceExposesOnlyReviewedPackagesAsActionable(t *testing.T) {
	indexPath := filepath.Join(t.TempDir(), "index.json")
	index, err := catalogForTest()
	if err != nil {
		t.Fatal(err)
	}
	if err := writeIndexForTest(index, indexPath); err != nil {
		t.Fatal(err)
	}
	service, err := Open(indexPath, installer.Manager{Root: t.TempDir()}.WithPlatform(platform.Info{
		Firmware: "knulli", Version: "scarab", Arch: "aarch64", ABI: "linux-aarch64-glibc", GLIBCVersion: "2.40", Dependencies: []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}, Device: "trimui-smart-pro", Resolution: "1280x720",
	}))
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 4 {
		t.Fatalf("expected four catalogue items, got %d", len(items))
	}
	experimental := 0
	verified := 0
	for _, item := range items {
		if item.Package.ID == "io.github.unitreign.playtime" && !item.DeviceTested {
			t.Fatalf("Smart Pro test evidence was not matched for %s", item.Package.ID)
		}
		if item.Package.Installable() {
			if item.Package.Experimental() {
				experimental++
			}
			if item.Package.Review.Status == "verified" {
				verified++
			}
			if !item.Compatible || len(item.Actions) != 1 || item.Actions[0] != Install {
				t.Fatalf("reviewed package is not installable: %#v", item)
			}
			continue
		}
		if item.Compatible || len(item.Actions) != 0 || item.Verdict.State != StateCandidate {
			t.Fatalf("candidate became actionable: %#v", item)
		}
	}
	if experimental != 2 || verified != 0 {
		t.Fatalf("expected two experimental and no universally verified packages, got %d and %d", experimental, verified)
	}
}

func TestApprovedCandidateRemainsReadOnly(t *testing.T) {
	pkg := installablePackage()
	pkg.Review = manifest.Review{Status: "candidate", Approval: &manifest.Approval{Provenance: "community"}}
	pkg.Release, pkg.Compatibility, pkg.Install = nil, nil, nil
	verdict := assess(pkg, platform.Info{}, installer.Status{}, false)
	if verdict.State != StateCandidate || verdict.Message() != "Community approved; installation is blocked by technical review" {
		t.Fatalf("unexpected approved candidate state: %#v", verdict)
	}
	if len(verdict.Actions) != 0 {
		t.Fatal("approved candidate must remain read-only")
	}
}

func TestLatestKnulliMetadataAllowsOnlyExperimentalDeviceMatrix(t *testing.T) {
	root := t.TempDir()
	for name, value := range map[string]string{
		"etc/os-release":                      "NAME=Buildroot\nID=buildroot\nVERSION_ID=2025.02\nOS_NAME=\"knulli\"\nOS_VERSION=scarab\nOS_DATE=20260510\n",
		"usr/share/knulli/knulli.version":     "scarab 2026/08/19 16:06\n",
		"boot/boot/knulli.board":              "trimui-smart-pro\n",
		"sys/class/graphics/fb0/virtual_size": "1280,13107\n",
	} {
		host := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(host), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(host, []byte(value), 0644); err != nil {
			t.Fatal(err)
		}
	}
	index, err := catalogForTest()
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(t.TempDir(), "index.json")
	if err := writeIndexForTest(index, indexPath); err != nil {
		t.Fatal(err)
	}
	detected := platform.Resolve(root)
	detected.Arch = "aarch64"
	detected.ABI = "linux-aarch64-glibc"
	detected.GLIBCVersion = "2.40"
	detected.Dependencies = []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}
	if detected.Resolution != "" {
		t.Fatalf("corrupt framebuffer virtual size became compatible: %#v", detected)
	}
	detected = detected.WithCandidates(append([]platform.ResolutionCandidate{{Source: "SDL renderer output", Width: 1280, Height: 720}}, detected.ResolutionCandidates()...))
	service, err := Open(indexPath, installer.Manager{Root: root}.WithPlatform(detected))
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Package.Experimental() {
			if !item.Compatible || len(item.Actions) != 1 || item.Verdict.State != StateAvailable || !hasReason(item.Verdict, ReasonReview, "no minimum version is claimed") || !hasReason(item.Verdict, ReasonReview, "/etc/os-release:OS_NAME") || !hasReason(item.Verdict, ReasonReview, "source=SDL renderer output") {
				t.Fatalf("latest Knulli experimental decision is wrong: %#v", item)
			}
		}
	}
}

func TestMagicXAllowsOnlyPlayTimeExperimentalPackage(t *testing.T) {
	index, err := catalogForTest()
	if err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(t.TempDir(), "index.json")
	if err := writeIndexForTest(index, indexPath); err != nil {
		t.Fatal(err)
	}
	detected := platform.Resolve(t.TempDir(), platform.WithFirmware("knulli"), platform.WithVersion("scarab 2026/08/19 16:06"), platform.WithArch("aarch64"), platform.WithDevice("magicx-zero-28"), platform.WithResolutionOverride("640x480"))
	detected.ABI = "linux-aarch64-glibc"
	detected.GLIBCVersion = "2.40"
	detected.Dependencies = []string{"sdl2", "sdl2-image", "sdl2-ttf", "libc", "libresolv", "libpthread"}
	service, err := Open(indexPath, installer.Manager{Root: t.TempDir()}.WithPlatform(detected))
	if err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		switch item.Package.ID {
		case "io.github.unitreign.playtime":
			if !item.Compatible || item.DeviceTested || !item.Package.Experimental() || len(item.Actions) != 1 || item.Actions[0] != Install || !strings.Contains(strings.ToLower(item.Package.Install.Warning), "unverified") {
				t.Fatalf("PlayTime was not offered as an unverified MagicX experiment: %#v", item)
			}
		case "app.romm.grout":
			if !item.Compatible || item.DeviceTested || len(item.Actions) != 1 || item.Actions[0] != Install || item.Verdict.State != StateAvailable {
				t.Fatalf("Grout was not offered as an unverified MagicX experiment: %#v", item)
			}
		}
	}
}

func TestVerdictActionsReflectInstallStateAndHealth(t *testing.T) {
	pkg := installablePackage()
	current := platform.Info{Firmware: "knulli", Version: "2026.05", Arch: "aarch64", ABI: "linux-aarch64-glibc", Dependencies: []string{"sdl2"}, Device: "trimui-smart-pro", Resolution: "1280x720"}
	assertActions(t, assess(pkg, current, installer.Status{}, false).Actions, Install)
	assertActions(t, assess(pkg, current, installer.Status{}, true).Actions, Adopt)
	status := installer.Status{Installed: true, Version: pkg.Version, Healthy: true}
	assertActions(t, assess(pkg, current, status, false).Actions, Uninstall, Repair)
	status.Healthy = false
	assertActions(t, assess(pkg, current, status, false).Actions, Uninstall, Repair)
	status.Version = "0.9.0"
	assertActions(t, assess(pkg, current, status, false).Actions, Update, Uninstall, Repair)
	status.Version = pkg.Version
	incompatible := platform.Info{Firmware: "other", Arch: "aarch64", Device: "trimui-smart-pro", Resolution: "1280x720"}
	verdict := assess(pkg, incompatible, status, false)
	assertActions(t, verdict.Actions, Uninstall)
	if verdict.Compatible(pkg) {
		t.Fatal("installed package became compatible after the platform rejected it")
	}
}

func TestRecoveryStateRefinesVerdictActions(t *testing.T) {
	pkg := installablePackage()
	current := platform.Info{Firmware: "knulli", Version: "2026.05", Arch: "aarch64", ABI: "linux-aarch64-glibc", Dependencies: []string{"sdl2"}, Device: "trimui-smart-pro", Resolution: "1280x720"}
	item := Item{Package: pkg, Compatible: true, PreExisting: true, Verdict: assess(pkg, current, installer.Status{}, true)}
	assertActions(t, actions(item), Adopt)
	item.RecoveryAllowed = true
	assertActions(t, actions(item), Adopt, ForceReinstall)
	item.RecoveryActive = true
	assertActions(t, actions(item))
}

func hasReason(verdict Verdict, kind ReasonKind, fragment string) bool {
	for _, reason := range verdict.Reasons {
		if reason.Kind == kind && strings.Contains(reason.Detail, fragment) {
			return true
		}
	}
	return false
}

func TestRetryTargetsRemainBoundToOriginatingOperation(t *testing.T) {
	pkg := installablePackage()
	tests := []struct {
		name  string
		item  Item
		retry Action
		want  Action
	}{
		{name: "fresh install absent", item: Item{Package: pkg, Compatible: true}, retry: Install, want: Install},
		{name: "manage external", item: Item{Package: pkg, Compatible: true, PreExisting: true}, retry: Adopt, want: Adopt},
		{name: "manage cannot target absent", item: Item{Package: pkg, Compatible: true}, retry: Adopt},
		{name: "update managed old version", item: Item{Package: pkg, Compatible: true, Installed: true, InstalledVersion: "0.9.0"}, retry: Update, want: Update},
		{name: "repair managed", item: Item{Package: pkg, Compatible: true, Installed: true, InstalledVersion: pkg.Version}, retry: Repair, want: Repair},
		{name: "force external", item: Item{Package: pkg, Compatible: true, PreExisting: true}, retry: ForceReinstall, want: ForceReinstall},
		{name: "uninstall managed", item: Item{Package: pkg, Installed: true}, retry: Uninstall, want: Uninstall},
		{name: "compatibility still blocks retry", item: Item{Package: pkg}, retry: Install},
		{name: "stale fresh transaction retries install", item: Item{Package: pkg, Compatible: true, PreExisting: true, RecoveryReason: "Interrupted package transaction"}, retry: Install, want: Install},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validRetryAction(test.item, test.retry); got != test.want {
				t.Fatalf("retry target = %q, want %q", got, test.want)
			}
		})
	}
}

func TestPersistedRetryRevalidatesAbsentAndExternalInstallTypes(t *testing.T) {
	root := t.TempDir()
	log, err := diagnostics.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	pkg := installablePackage()
	manager := installer.Manager{Root: root, Diagnostics: log}.WithPlatform(platform.Info{
		Firmware: "knulli", Version: "2026.05", Arch: "aarch64", ABI: "linux-aarch64-glibc", Dependencies: []string{"sdl2"}, Device: "trimui-smart-pro", Resolution: "1280x720",
	})
	service := &Service{index: catalog.Index{Packages: []catalog.Entry{{ID: pkg.ID, Package: pkg}}}, manager: manager}
	failedInstall := installer.LifecycleState{PackageID: pkg.ID, RequestedOperation: string(Install), DetectedInstallType: "absent", RetryTarget: string(Install), Failure: "download release: SHA-256 mismatch"}
	if err := manager.RecordLifecycle(failedInstall); err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(context.Background())
	if err != nil || len(items) != 1 || items[0].PreExisting || items[0].RetryAction != Install || len(items[0].Actions) == 0 || items[0].Actions[0] != Install {
		t.Fatalf("failed fresh install did not retry install: %#v, %v", items, err)
	}

	destination := filepath.Join(root, "userdata/roms/ports/test")
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destination, "run.sh"), []byte("external"), 0755); err != nil {
		t.Fatal(err)
	}
	failedManage := installer.LifecycleState{PackageID: pkg.ID, RequestedOperation: string(Adopt), DetectedInstallType: "external", RetryTarget: string(Adopt), Failure: "unexpected adoption error"}
	if err := manager.RecordLifecycle(failedManage); err != nil {
		t.Fatal(err)
	}
	restarted := &Service{index: service.index, manager: manager}
	items, err = restarted.Items(context.Background())
	if err != nil || !items[0].PreExisting || items[0].RetryAction != Adopt || items[0].Actions[0] != Adopt {
		t.Fatalf("failed management did not survive restart: %#v, %v", items, err)
	}
	if err := os.RemoveAll(destination); err != nil {
		t.Fatal(err)
	}
	items, err = restarted.Items(context.Background())
	if err != nil || items[0].RetryAction != "" || len(items[0].Actions) != 1 || items[0].Actions[0] != Install {
		t.Fatalf("absent copy retained adoption retry: %#v, %v", items, err)
	}
	data, err := os.ReadFile(filepath.Join(root, "userdata/system/logs/knulli-app-store.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`event=lifecycle_state`, `package="org.example.test"`, `requested_operation="install"`, `detected_install_type="absent"`, `selected_retry_target="install"`, `health_reason=""`} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("structured lifecycle log lacks %s: %s", field, data)
		}
	}
}

func TestOnlyRecoverableAdoptionFailuresAllowForceReinstall(t *testing.T) {
	for _, test := range []struct {
		err     error
		allowed bool
	}{
		{err: &installer.AdoptionConflictError{Path: "/userdata/roms/tools/demo/run.sh"}, allowed: true},
		{err: errors.New("download release: SHA-256 mismatch"), allowed: false},
		{err: errors.New("compatibility failed field=architecture"), allowed: false},
		{err: errors.New("not enough free space"), allowed: false},
		{err: errors.New("archive path escapes destination"), allowed: false},
		{err: errors.New("catalogue signature is invalid"), allowed: false},
	} {
		if got := recoverableAdoptionFailure(test.err); got != test.allowed {
			t.Fatalf("recoverableAdoptionFailure(%q) = %v, want %v", test.err, got, test.allowed)
		}
	}
}

func TestExecuteInstallFailureAndUninstall(t *testing.T) {
	root := t.TempDir()
	asset := testZip(t, map[string]string{"run.sh": "#!/bin/sh\n"})
	digest := sha256.Sum256(asset)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Length", strconv.Itoa(len(asset)))
		_, _ = writer.Write(asset)
	}))
	defer server.Close()
	client := server.Client()
	client.Transport = testRewriteTransport{base: client.Transport, host: strings.TrimPrefix(server.URL, "https://")}
	log, err := diagnostics.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	pkg := installablePackage()
	pkg.Release.SHA256 = hex.EncodeToString(digest[:])
	pkg.Release.Size = int64(len(asset))
	pkg.Release.InstalledSize = int64(len("#!/bin/sh\n"))
	manager := installer.Manager{Root: root, Client: client, Diagnostics: log}.WithPlatform(platform.Info{
		Firmware: "knulli", Version: "2026.05", Arch: "aarch64", ABI: "linux-aarch64-glibc", Dependencies: []string{"sdl2"}, Device: "trimui-smart-pro", Resolution: "1280x720",
	})
	service := &Service{index: catalog.Index{Packages: []catalog.Entry{{ID: pkg.ID, Package: pkg}}}, manager: manager}
	var progress []string
	if err := service.Execute(context.Background(), pkg.ID, Install, func(message string) { progress = append(progress, message) }); err != nil {
		t.Fatalf("install through Execute: %v", err)
	}
	if len(progress) != 2 || !strings.Contains(progress[1], "completed") {
		t.Fatalf("install progress did not consume returned outcome: %v", progress)
	}

	broken := pkg
	broken.Release = &manifest.Release{}
	*broken.Release = *pkg.Release
	broken.Version = "2.0.0"
	broken.Release.SHA256 = strings.Repeat("0", 64)
	service.index.Packages[0].Package = broken
	if err := service.Execute(context.Background(), pkg.ID, Update, func(string) {}); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("update failure through Execute = %v", err)
	}
	service.index.Packages[0].Package = pkg
	if err := service.Execute(context.Background(), pkg.ID, Uninstall, func(string) {}); err != nil {
		t.Fatalf("uninstall through Execute: %v", err)
	}
}

type testRewriteTransport struct {
	base http.RoundTripper
	host string
}

func (transport testRewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.URL.Scheme = "https"
	clone.URL.Host = transport.host
	clone.Host = transport.host
	return transport.base.RoundTrip(clone)
}

func testZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, body := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetMode(0755)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestCompletionMessageReportsRefreshOrRestartPrecisely(t *testing.T) {
	if got := completionMessage(Uninstall, installer.OperationOutcome{GameListRefreshAccepted: true}); got != "Uninstall completed; game list refresh requested" {
		t.Fatalf("refresh message = %q", got)
	}
	if got := completionMessage(Adopt, installer.OperationOutcome{RestartRequired: true}); got != "Manage existing install completed; restart required to update game list" {
		t.Fatalf("restart message = %q", got)
	}
}

func assertActions(t *testing.T, got []Action, wanted ...Action) {
	t.Helper()
	if len(got) != len(wanted) {
		t.Fatalf("actions = %v, wanted %v", got, wanted)
	}
	for index := range got {
		if got[index] != wanted[index] {
			t.Fatalf("actions = %v, wanted %v", got, wanted)
		}
	}
}
