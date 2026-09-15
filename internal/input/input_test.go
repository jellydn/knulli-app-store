package input

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestFirstRunTestsDetectedMappingBeforeCatalogueAndLoadsOnRestart(t *testing.T) {
	root := t.TempDir()
	identity := Identity{GUID: "one", Name: "Pad"}
	session := NewSession(root, "magicx-zero-28")
	session.Connect(identity, true)
	if session.Mode != Setup || !session.FirstRun {
		t.Fatalf("first run skipped dedicated setup: mode=%s", session.Mode)
	}
	press(session, Down)
	press(session, Confirm)
	if session.Mode != Preview {
		t.Fatalf("detected mapping test did not open preview: %s", session.Mode)
	}
	for _, action := range Actions {
		press(session, action)
	}
	if session.Mode != Normal || session.Source != "tested detected mapping" {
		t.Fatalf("tested auto mapping was not saved: mode=%s source=%s error=%s", session.Mode, session.Source, session.ValidationError)
	}

	restarted := NewSession(root, "magicx-zero-28")
	restarted.Connect(identity, false)
	if restarted.Mode != Normal || restarted.Source != "saved controller mapping" {
		t.Fatalf("saved mapping did not bypass setup: mode=%s source=%s", restarted.Mode, restarted.Source)
	}
}

func TestFirstRunCustomMappingReviewConflictRetryStartOverAndCancel(t *testing.T) {
	session := NewSession(t.TempDir(), "trimui-smart-pro")
	session.Connect(Identity{GUID: "one", Name: "Pad"}, true)
	press(session, Down)
	press(session, Down)
	press(session, Confirm)
	if session.Mode != Calibrating {
		t.Fatalf("customize did not start calibration: %s", session.Mode)
	}

	session.HandleButton(2)
	if session.Mode != Review || session.Calibration.Mapping[Up] != 2 {
		t.Fatalf("assignment review missing: mode=%s mapping=%v", session.Mode, session.Calibration.Mapping)
	}
	press(session, Confirm)
	session.HandleButton(2)
	if session.Mode != Calibrating || !strings.Contains(session.ValidationError, "already assigned") {
		t.Fatalf("conflict did not stay on action for retry: mode=%s error=%q", session.Mode, session.ValidationError)
	}
	session.HandleButton(4)
	press(session, Down)
	press(session, Confirm)
	if session.Mode != Calibrating || session.Calibration.Index != 1 {
		t.Fatalf("retry did not remove pending assignment: mode=%s index=%d", session.Mode, session.Calibration.Index)
	}
	session.HandleButton(5)
	press(session, Down)
	press(session, Down)
	press(session, Confirm)
	if session.Mode != Calibrating || session.Calibration.Index != 0 {
		t.Fatalf("start over retained progress: mode=%s index=%d", session.Mode, session.Calibration.Index)
	}
	session.HandleButton(7)
	press(session, Back)
	if session.Mode != Setup || session.Calibration != nil {
		t.Fatalf("first-run cancel did not return to setup: mode=%s", session.Mode)
	}
}

func TestCustomMappingRequiresReviewAndPreviewBeforeAtomicSave(t *testing.T) {
	root := t.TempDir()
	identity := Identity{GUID: "custom", Name: "Pad"}
	session := NewSession(root, "magicx-zero-28")
	session.Connect(identity, false)
	press(session, Down)
	press(session, Down)
	press(session, Confirm)
	buttons := []int{2, 4, 5, 7, 8, 9, 10, 15}
	for index, button := range buttons {
		session.HandleButton(button)
		if session.Mode != Review {
			t.Fatalf("action %d skipped assignment review: %s", index, session.Mode)
		}
		if _, found, err := session.Store.Load(Identity{Device: "magicx-zero-28", GUID: "custom", Name: "Pad"}); err != nil || found {
			t.Fatalf("mapping saved before preview: found=%v err=%v", found, err)
		}
		press(session, Confirm)
	}
	if session.Mode != Preview {
		t.Fatalf("custom mapping skipped preview: %s", session.Mode)
	}
	for _, button := range buttons {
		session.HandleButton(button)
	}
	if session.Mode != Normal || session.Source != "saved custom mapping" {
		t.Fatalf("custom mapping was not saved after preview: mode=%s source=%s", session.Mode, session.Source)
	}
}

func TestNoControllerBlockedReconnectResetAndSafeExit(t *testing.T) {
	session := NewSession(t.TempDir(), "magicx-zero-28")
	if session.Mode != Blocked || !session.FirstRun {
		t.Fatalf("no-controller launch was not blocked: %#v", session)
	}
	if action, effect := session.HandleButton(0); action != "" || effect != NoEffect {
		t.Fatal("no-controller input caused an action")
	}
	session.Connect(Identity{GUID: "one", Name: "Pad"}, true)
	if session.Mode != Setup || session.Source != "Knulli SDL_GAMECONTROLLERCONFIG" {
		t.Fatalf("unexpected auto mapping state: %#v", session)
	}
	if action, _ := session.HandleButton(AutoMapping()[Back]); action != Exit {
		t.Fatalf("first-run safe exit did not exit: %q", action)
	}
	press(session, Confirm)
	press(session, Diagnostics)
	press(session, Down)
	press(session, Down)
	press(session, Confirm)
	if session.Mode != Setup || !session.FirstRun {
		t.Fatalf("reset did not require setup again: mode=%s first=%v", session.Mode, session.FirstRun)
	}
	session.Disconnect()
	if session.Connected || session.Source != "no controller" || session.Mode != Blocked {
		t.Fatal("disconnect retained controller state")
	}
	session.Connect(Identity{GUID: "two", Name: "Other Pad"}, false)
	if session.Identity.GUID != "two" || session.Source != "SDL GameController mapping" || session.Mode != Setup {
		t.Fatal("reconnect reused another controller identity")
	}
}

func TestControllerSetupRemainsAvailableAndSupportsStartupOverride(t *testing.T) {
	session := NewSession(t.TempDir(), "trimui-smart-pro")
	session.Connect(Identity{GUID: "one", Name: "Pad"}, false)
	press(session, Confirm)
	if session.Mode != Normal {
		t.Fatalf("detected mapping choice did not enter catalogue: %s", session.Mode)
	}
	press(session, Diagnostics)
	press(session, Confirm)
	if session.Mode != Setup || session.FirstRun {
		t.Fatalf("settings setup did not open: mode=%s first=%v", session.Mode, session.FirstRun)
	}
	press(session, Back)
	if session.Mode != Settings {
		t.Fatalf("settings setup cancel did not return to settings: %s", session.Mode)
	}
	session.Mode = Normal
	session.OpenSetup()
	if session.Mode != Setup || session.FirstRun {
		t.Fatalf("startup override did not open optional setup: mode=%s first=%v", session.Mode, session.FirstRun)
	}
}

func press(session *Session, action Action) {
	mapping := session.Mapping
	if session.FirstRun {
		mapping = AutoMapping()
	}
	session.HandleButton(mapping[action])
}
