package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAutoMappingUsesSemanticGameControllerButtons(t *testing.T) {
	mapping := AutoMapping()
	if err := mapping.Validate(); err != nil {
		t.Fatal(err)
	}
	if action, ok := mapping.Action(0); !ok || action != Confirm {
		t.Fatalf("SDL A did not map to Confirm: %q %v", action, ok)
	}
}

func TestDeviceProfilesDoNotInventMagicXFallback(t *testing.T) {
	trimui := Profile("trimui-smart-pro")
	if trimui.Resolution != "1280x720" || trimui.Fallback == nil {
		t.Fatalf("reviewed TrimUI fallback missing: %#v", trimui)
	}
	magicx := Profile("magicx-zero-28")
	if magicx.Resolution != "640x480" || magicx.Fallback != nil || !strings.Contains(magicx.Evidence, "raw controls unavailable") {
		t.Fatalf("MagicX profile invented controls or lacks evidence: %#v", magicx)
	}
}

func TestCalibrationRejectsConflictsAndRequiresPreview(t *testing.T) {
	calibration := NewCalibration()
	if err := calibration.Assign(11); err != nil {
		t.Fatal(err)
	}
	if err := calibration.Assign(11); err == nil || calibration.Index != 1 {
		t.Fatalf("conflict advanced calibration: %#v", calibration)
	}
	for _, button := range []int{12, 13, 14, 0, 1, 3, 6} {
		if err := calibration.Assign(button); err != nil {
			t.Fatal(err)
		}
	}
	if !calibration.Preview {
		t.Fatal("mapping skipped preview")
	}
	for index, button := range []int{11, 12, 13, 14, 0, 1, 3, 6} {
		done := calibration.Test(button)
		if done != (index == 7) {
			t.Fatalf("preview completed at input %d", index)
		}
	}
}

func TestStoreSeparatesDeviceAndControllerIdentities(t *testing.T) {
	store := NewStore(t.TempDir())
	trimui := Identity{Device: "trimui-smart-pro", GUID: "AAAA", Name: "Pad"}
	magicx := Identity{Device: "magicx-zero-28", GUID: "AAAA", Name: "Pad"}
	custom := AutoMapping()
	custom[Confirm], custom[Back] = custom[Back], custom[Confirm]
	if err := store.Save(trimui, custom); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.Load(magicx); err != nil || found {
		t.Fatalf("mapping leaked to another device: found=%v err=%v", found, err)
	}
	loaded, found, err := store.Load(trimui)
	if err != nil || !found || loaded[Confirm] != 1 {
		t.Fatalf("saved mapping did not load: %#v %v %v", loaded, found, err)
	}
}

func TestIdentityUsesDeviceScopedNameWhenGUIDIsMissing(t *testing.T) {
	one, err := (Identity{Device: "magicx-zero-28", GUID: "00000000000000000000000000000000", Name: " Internal Pad "}).Key()
	if err != nil {
		t.Fatal(err)
	}
	two, _ := (Identity{Device: "trimui-smart-pro", Name: "Internal Pad"}).Key()
	if one == two || !strings.Contains(one, "name:internal pad") {
		t.Fatalf("unsafe fallback identities: %q %q", one, two)
	}
	if _, err := (Identity{Device: "magicx-zero-28"}).Key(); err == nil {
		t.Fatal("empty controller identity was accepted")
	}
}

func TestStoreRejectsCorruptConfigAndResetKeepsOtherControllers(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	one := Identity{Device: "magicx-zero-28", GUID: "one"}
	two := Identity{Device: "magicx-zero-28", GUID: "two"}
	if err := store.Save(one, AutoMapping()); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(two, AutoMapping()); err != nil {
		t.Fatal(err)
	}
	if err := store.Reset(one); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := store.Load(one); found {
		t.Fatal("reset mapping remains")
	}
	if _, found, _ := store.Load(two); !found {
		t.Fatal("reset removed another controller")
	}
	path := filepath.Join(root, strings.TrimPrefix(Path, "/"))
	if err := os.WriteFile(path, []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Load(two); err == nil {
		t.Fatal("corrupt config was accepted")
	}
	if err := store.Save(two, AutoMapping()); err != nil {
		t.Fatal(err)
	}
	corrupt, err := os.ReadFile(path + ".corrupt")
	if err != nil || string(corrupt) != "not json" {
		t.Fatalf("corrupt config was not preserved: %q %v", corrupt, err)
	}
	if _, found, err := store.Load(two); err != nil || !found {
		t.Fatalf("replacement mapping did not load: found=%v err=%v", found, err)
	}
}

func TestStoreValidatesEverySavedRecord(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	path := filepath.Join(root, strings.TrimPrefix(Path, "/"))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	encoded := `{"schema":"org.knulli.app-store/controller-mappings/v1","records":[{"identity":{"device":"magicx-zero-28","guid":"bad"},"mapping":{"up":11}}]}`
	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		t.Fatal(err)
	}
	other := Identity{Device: "magicx-zero-28", GUID: "other"}
	if _, _, err := store.Load(other); err == nil || !strings.Contains(err.Error(), "invalid saved mapping") {
		t.Fatalf("invalid unrelated record was ignored: %v", err)
	}
}

func TestStoreDoesNotTreatReadFailureAsCorruption(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read files regardless of mode")
	}
	root := t.TempDir()
	store := NewStore(root)
	identity := Identity{Device: "magicx-zero-28", GUID: "one"}
	if err := store.Save(identity, AutoMapping()); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, strings.TrimPrefix(Path, "/"))
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0600)
	if err := store.Save(identity, AutoMapping()); err == nil {
		t.Fatal("unreadable mapping was replaced")
	}
	if _, err := os.Stat(path + ".corrupt"); !os.IsNotExist(err) {
		t.Fatalf("read failure was renamed as corruption: %v", err)
	}
}

func TestSessionCompletesPreviewAndLoadsSavedMapping(t *testing.T) {
	root := t.TempDir()
	now := time.Unix(100, 0)
	identity := Identity{GUID: "one", Name: "Pad"}
	session := NewSession(root, "magicx-zero-28")
	session.Connect(identity, true, now)
	session.HandleButton(2, now.Add(time.Second))
	buttons := []int{0, 1, 2, 3, 4, 5, 7, 6}
	for _, button := range buttons {
		session.HandleButton(button, now.Add(2*time.Second))
	}
	if session.Mode != Preview {
		t.Fatalf("setup did not reach preview: %s (%s)", session.Mode, session.ValidationError)
	}
	for _, button := range buttons {
		session.HandleButton(button, now.Add(3*time.Second))
	}
	if session.Mode != Normal || session.Source != "saved calibration" {
		t.Fatalf("tested mapping was not saved: mode=%s source=%s error=%s", session.Mode, session.Source, session.ValidationError)
	}

	restarted := NewSession(root, "magicx-zero-28")
	restarted.Connect(identity, false, now)
	if restarted.Source != "saved calibration" || restarted.Mapping[Confirm] != 4 {
		t.Fatalf("saved mapping did not survive restart: source=%s mapping=%v", restarted.Source, restarted.Mapping)
	}
}

func TestSessionReconnectTimeoutAndNoController(t *testing.T) {
	now := time.Unix(100, 0)
	session := NewSession(t.TempDir(), "magicx-zero-28")
	if action, effect := session.HandleButton(0, now); action != "" || effect != NoEffect {
		t.Fatal("no-controller input caused an action")
	}
	session.Connect(Identity{GUID: "one", Name: "Pad"}, true, now)
	if session.Mode != Startup || session.Source != "Knulli SDL_GAMECONTROLLERCONFIG" {
		t.Fatalf("unexpected auto mapping state: %#v", session)
	}
	session.Tick(now.Add(9 * time.Second))
	if session.Mode != Normal {
		t.Fatal("startup setup prompt did not safely time out")
	}
	session.Disconnect()
	if session.Connected || session.Source != "no controller" {
		t.Fatal("disconnect retained controller state")
	}
	session.Connect(Identity{GUID: "two", Name: "Other Pad"}, false, now)
	if session.Identity.GUID != "two" || session.Source != "SDL GameController auto mapping" {
		t.Fatal("reconnect reused another controller identity")
	}
}
