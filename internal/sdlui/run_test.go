//go:build sdl

package sdlui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	"github.com/jellydn/knulli-app-store/internal/platform"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
	"github.com/veandco/go-sdl2/sdl"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func TestRunRejectsToolkitMappingWizard(t *testing.T) {
	t.Setenv("INPUT_CAPTURE", "1")
	// No display is required for this guard. It must run before SDL or the
	// toolkit can create a window, write assignments or exit the process.
	t.Setenv("SDL_VIDEODRIVER", "not-a-video-driver")
	err := Run(context.Background(), &routerBackend{}, Options{})
	if err == nil || !strings.Contains(err.Error(), "INPUT_CAPTURE is not supported") {
		t.Fatalf("expected automatic-controls guard before SDL startup, got %v", err)
	}
}

func TestResolutionCandidateLoggingIncludesRejectionAndSelection(t *testing.T) {
	root := t.TempDir()
	logger, err := diagnostics.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	candidates := []platform.ResolutionCandidate{
		{Source: "SDL current display mode", Width: 1280, Height: 0x3333},
		{Source: "SDL renderer output", Width: 1280, Height: 720},
	}
	selected := platform.Info{Device: "trimui-smart-pro"}.WithCandidates(candidates)
	logResolutionCandidates(logger, platform.AssessResolutions(candidates), selected)
	data, err := os.ReadFile(filepath.Join(root, "userdata/system/logs/knulli-app-store.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, wanted := range []string{`source="SDL current display mode"`, `height="13107"`, `valid="false"`, `reason="above maximum 7680x4320"`, `event=resolution_selected`, `resolution="1280x720"`, `source="SDL renderer output"`} {
		if !strings.Contains(string(data), wanted) {
			t.Fatalf("resolution log lacks %q: %s", wanted, data)
		}
	}
}

func TestControllerLogsActionsWithoutRawButtonSpam(t *testing.T) {
	root := t.TempDir()
	logger, err := diagnostics.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	controls := storeinput.NewSession(root, "magicx-zero-28")
	controls.Connect(storeinput.Identity{GUID: "03000000", Name: "Runtime controller"}, true)
	model := &storeui.Model{}
	processButton(context.Background(), model, controls, storeinput.AutoMapping()[storeinput.Down], logger)
	processButton(context.Background(), model, controls, storeinput.AutoMapping()[storeinput.Confirm], logger)
	data, err := os.ReadFile(filepath.Join(root, "userdata/system/logs/knulli-app-store.log"))
	if err != nil {
		t.Fatal(err)
	}
	logText := string(data)
	for _, wanted := range []string{`event=controller_semantic_action`, `action="down"`, `action="confirm"`} {
		if !strings.Contains(logText, wanted) {
			t.Fatalf("missing %s: %s", wanted, logText)
		}
	}
	if strings.Contains(logText, "controller_input") || strings.Contains(logText, "button=") {
		t.Fatal("raw button spam remains")
	}
}

// These are raw SDL events, not semantic test keys: the renderer migration
// must not introduce gabagool's face swaps or Start-to-confirm behavior.
func TestSDLControlsPreserveConfirmationContract(t *testing.T) {
	for _, layout := range []struct {
		name          string
		confirm, back uint8
		more          uint8
	}{
		{"knulli", 1, 0, 2},
		{"xbox", 0, 1, 3},
		{"", 0, 1, 3},
	} {
		t.Run(layout.name, func(t *testing.T) {
			t.Setenv("STORE_CONTROLLER_LAYOUT", layout.name)
			model, controls, logger, _ := newRouterHarness(t)
			model.Items = []appstore.Item{{
				Package: manifest.Package{ID: "org.example.utility", Name: "Utility"},
				Actions: []appstore.Action{appstore.Install},
			}}
			controller := &controllerState{instanceID: 17}
			button := func(kind uint32, which sdl.JoystickID, code uint8) bool {
				return handleEvent(context.Background(), model, &sdl.ControllerButtonEvent{
					Type: kind, Which: which, Button: code,
				}, &controller, controls, logger, InputAuto)
			}
			button(sdl.CONTROLLERBUTTONDOWN, 18, layout.confirm)
			button(sdl.CONTROLLERBUTTONUP, 17, layout.confirm)
			button(sdl.CONTROLLERBUTTONDOWN, 17, uint8(sdl.CONTROLLER_BUTTON_START))
			if model.Focus != storeui.Browse || model.Busy {
				t.Fatal("another controller, button release or Start accepted an action")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, layout.confirm)
			if model.Focus != storeui.Confirm || model.Busy {
				t.Fatal("first confirm must only open confirmation")
			}
			handleEvent(context.Background(), model, &sdl.KeyboardEvent{
				Type: sdl.KEYDOWN, Repeat: 1, Keysym: sdl.Keysym{Sym: sdl.K_RETURN},
			}, &controller, controls, logger, InputAuto)
			if model.Focus != storeui.Confirm || model.Busy {
				t.Fatal("held Enter must not execute the operation")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, layout.back)
			if model.Focus != storeui.Browse {
				t.Fatal("back did not cancel confirmation")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, layout.more)
			if model.Focus != storeui.Actions {
				t.Fatal("More did not open secondary actions")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, chordAnchorButton)
			button(sdl.CONTROLLERBUTTONUP, 17, chordAnchorButton)
			if button(sdl.CONTROLLERBUTTONDOWN, 17, layout.more) {
				t.Fatal("released Select must not leave the quit chord active")
			}
			button(sdl.CONTROLLERBUTTONDOWN, 17, chordAnchorButton)
			if !button(sdl.CONTROLLERBUTTONDOWN, 17, layout.more) {
				t.Fatal("held Select+Y did not quit")
			}
		})
	}
}

func TestFilterHeaderPreservesRawInputAndSelectionMemory(t *testing.T) {
	for _, policy := range []string{"knulli", "xbox", "keyboard"} {
		t.Run(policy, func(t *testing.T) {
			t.Setenv("STORE_CONTROLLER_LAYOUT", policy)
			model, controls, logger, backend := newRouterHarness(t)
			mode := InputAuto
			if policy == "keyboard" {
				controls, mode = storeinput.NewDesktopSession("", ""), InputKeyboard
			}
			for _, spec := range []struct {
				id, kind string
				state    appstore.State
			}{
				{"first-game", "game", appstore.StateAvailable},
				{"last-game", "game", appstore.StateAvailable},
				{"installed-app", "utility", appstore.StateInstalled},
			} {
				backend.items = append(backend.items, appstore.Item{Package: manifest.Package{ID: spec.id, Name: spec.id, Type: spec.kind}, Verdict: appstore.Verdict{State: spec.state}, Actions: []appstore.Action{appstore.Install}})
			}
			if err := model.Load(context.Background()); err != nil {
				t.Fatal(err)
			}
			controller := &controllerState{instanceID: 17}
			press := func(action storeinput.Action, key sdl.Keycode) {
				t.Helper()
				var event sdl.Event = &sdl.ControllerButtonEvent{Type: sdl.CONTROLLERBUTTONDOWN, Which: 17, Button: uint8(controls.Mapping[action])}
				if mode.IsKeyboard() {
					event = &sdl.KeyboardEvent{Type: sdl.KEYDOWN, Keysym: sdl.Keysym{Sym: key}}
				}
				if handleEvent(context.Background(), model, event, &controller, controls, logger, mode) {
					t.Fatal("filter input exited")
				}
			}
			press(storeinput.Right, sdl.K_RIGHT)
			press(storeinput.PageDown, sdl.K_PAGEDOWN)
			press(storeinput.Down, sdl.K_DOWN)
			for _, size := range [][2]int{{320, 240}, {320, 200}, {480, 320}, {639, 480}, {640, 480}, {641, 480}, {640, 479}, {640, 481}, {720, 480}, {720, 720}, {480, 640}, {640, 960}, {320, 640}, {1280, 720}, {1920, 1080}, {1920, 1200}, {2560, 1080}, {320, 240}} {
				w, h := canvasSize(size[0], size[1])
				canvas, err := newConsoleCanvas(w, h)
				if err != nil {
					t.Fatal(err)
				}
				canvas.draw(model, "unknown board", controls)
				if model.Tabs.Category() != storeui.CategoryGames || model.Tab() != storeui.TabReady || len(model.Items) != 2 || model.Items[model.Selected].Package.ID != "last-game" {
					t.Fatal("render/resize lost intersection or remembered row")
				}
				press(storeinput.PageDown, sdl.K_PAGEDOWN)
				empty := canvas.draw(model, "", controls)
				if len(model.Items) != 0 || model.Tab() != storeui.TabInstalled || model.Tabs.Category() != storeui.CategoryGames || !renderedText(empty, canvas, "GAMES / INSTALLED", image.Rect(0, 0, w, 96), palette.background, palette.accent) {
					t.Fatal("empty intersection lost visible filter state")
				}
				press(storeinput.Right, sdl.K_RIGHT)
				if len(model.Items) != 1 || model.Items[0].Package.ID != "installed-app" {
					t.Fatal("empty view could not change type within status")
				}
				press(storeinput.Left, sdl.K_LEFT)
				press(storeinput.PageUp, sdl.K_PAGEUP)
				if model.Items[model.Selected].Package.ID != "last-game" {
					t.Fatal("intersection forgot selection after empty view")
				}
				press(storeinput.Diagnostics, sdl.K_y)
				press(storeinput.Right, sdl.K_RIGHT)
				canvas.draw(model, "", controls)
				if model.Focus != storeui.Actions || model.Tab() != storeui.TabReady || model.Tabs.Category() != storeui.CategoryGames {
					t.Fatal("review accepted filter input")
				}
				press(storeinput.Back, sdl.K_ESCAPE)
				model.Busy = true
				press(storeinput.Right, sdl.K_RIGHT)
				press(storeinput.PageDown, sdl.K_PAGEDOWN)
				canvas.draw(model, "", controls)
				if model.Tab() != storeui.TabReady || model.Tabs.Category() != storeui.CategoryGames || !model.Busy {
					t.Fatal("busy rendering/input changed filters")
				}
				model.Busy = false
				canvas.close()
			}
		})
	}
}

func TestFooterReflowPreservesRawNavigation(t *testing.T) {
	for _, policy := range []string{"knulli", "xbox", "keyboard"} {
		t.Run(policy, func(t *testing.T) {
			t.Setenv("STORE_CONTROLLER_LAYOUT", policy)
			model, controls, logger, _ := newRouterHarness(t)
			mode := InputAuto
			if policy == "keyboard" {
				controls, mode = storeinput.NewDesktopSession("", ""), InputKeyboard
			}
			item := appstore.Item{Package: manifest.Package{ID: "org.example.hints", Name: "Hint test", Install: &manifest.Install{Warning: strings.Repeat("Read before changing files. ", 80)}}, Actions: []appstore.Action{appstore.Adopt, appstore.ForceReinstall}}
			model.Items = []appstore.Item{item, item}
			controller := &controllerState{instanceID: 17}
			press := func(action storeinput.Action, key sdl.Keycode) {
				t.Helper()
				var event sdl.Event = &sdl.ControllerButtonEvent{Type: sdl.CONTROLLERBUTTONDOWN, Which: 17, Button: uint8(controls.Mapping[action])}
				if mode.IsKeyboard() {
					event = &sdl.KeyboardEvent{Type: sdl.KEYDOWN, Keysym: sdl.Keysym{Sym: key}}
				}
				if handleEvent(context.Background(), model, event, &controller, controls, logger, mode) {
					t.Fatal("footer navigation requested exit")
				}
			}
			for _, size := range [][2]int{{320, 240}, {639, 480}, {640, 480}, {641, 480}, {480, 640}, {720, 720}, {1280, 720}, {2560, 1080}, {320, 240}} {
				w, h := canvasSize(size[0], size[1])
				canvas, err := newConsoleCanvas(w, h)
				if err != nil {
					t.Fatal(err)
				}
				model.Selected = 0
				canvas.draw(model, "unknown board", controls)
				press(storeinput.Down, sdl.K_DOWN)
				press(storeinput.Diagnostics, sdl.K_y)
				press(storeinput.Right, sdl.K_RIGHT)
				canvas.draw(model, "unknown board", controls)
				if model.Selected != 1 || model.Action != 1 || model.Focus != storeui.Actions || model.Busy {
					t.Fatal("reflow changed selection/action or More navigation")
				}
				press(storeinput.PageDown, sdl.K_PAGEDOWN)
				canvas.draw(model, "unknown board", controls)
				if model.ReviewOffset == 0 || model.Tab() != storeui.TabAll || model.Tabs.Category() != storeui.CategoryAll || model.Action != 1 || model.Busy {
					t.Fatal("reading changed filters/action or failed to page")
				}
				press(storeinput.Back, sdl.K_ESCAPE)
				press(storeinput.Confirm, sdl.K_RETURN)
				if model.Focus != storeui.Confirm || model.Busy {
					t.Fatal("confirm did not open review without executing")
				}
				canvas.draw(model, "unknown board", controls)
				press(storeinput.Back, sdl.K_ESCAPE)
				model.Busy = true
				canvas.draw(model, "unknown board", controls)
				press(storeinput.Down, sdl.K_DOWN)
				press(storeinput.PageDown, sdl.K_PAGEDOWN)
				if !model.Busy || model.Selected != 1 || len(canvas.groups) != 0 {
					t.Fatal("busy footer promised or accepted a control")
				}
				model.Busy = false
				canvas.close()
			}
		})
	}
}

func TestSDLPagingRendersCompleteWarningsWithoutChangingOperation(t *testing.T) {
	const first = "REVIEW-START-5D8E"
	const warningEnd = "WARNING-END-7F3A"
	const recoveryEnd = "RECOVERY-END-9B2C"
	for _, size := range [][2]int{{320, 200}, {320, 240}, {480, 320}, {639, 480}, {640, 480}, {480, 640}, {720, 720}} {
		for _, input := range []InputMode{InputKeyboard, InputAuto} {
			for _, focus := range []storeui.Focus{storeui.Health, storeui.Actions, storeui.Confirm, storeui.ForceConfirm} {
				t.Run(fmt.Sprintf("%dx%d/%s/focus-%d", size[0], size[1], input, focus), func(t *testing.T) {
					w, h := canvasSize(size[0], size[1])
					canvas, err := newConsoleCanvas(w, h)
					if err != nil {
						t.Fatal(err)
					}
					defer canvas.close()
					model, controls, logger, _ := newRouterHarness(t)
					if input.IsKeyboard() {
						controls = storeinput.NewDesktopSession(t.TempDir(), "")
					}
					item := appstore.Item{
						Package: manifest.Package{ID: "org.example.review", Name: first, Version: "1", Type: "game", Install: &manifest.Install{
							Warning: strings.Repeat("Read before replacing existing files. ", 45) + "\n" + warningEnd,
						}},
						Actions:         []appstore.Action{appstore.Adopt, appstore.ForceReinstall},
						RecoverySummary: strings.Repeat("Preserve user data and retain the backup for manual restore. ", 37) + "\n" + recoveryEnd,
					}
					model.Items = []appstore.Item{{Package: manifest.Package{ID: "org.example.other"}}, item}
					model.Selected, model.Action, model.Focus = 1, 1, focus
					model.Tabs.Cycle(1)
					model.Tabs.CycleCategory(1)
					controller := &controllerState{instanceID: 17}
					page := func(down bool) {
						var event sdl.Event
						if input.IsKeyboard() {
							key := sdl.Keycode(sdl.K_PAGEUP)
							if down {
								key = sdl.K_PAGEDOWN
							}
							event = &sdl.KeyboardEvent{Type: sdl.KEYDOWN, Keysym: sdl.Keysym{Sym: key}}
						} else {
							button := sdl.CONTROLLER_BUTTON_LEFTSHOULDER
							if down {
								button = sdl.CONTROLLER_BUTTON_RIGHTSHOULDER
							}
							event = &sdl.ControllerButtonEvent{Type: sdl.CONTROLLERBUTTONDOWN, Which: 17, Button: uint8(button)}
						}
						if handleEvent(context.Background(), model, event, &controller, controls, logger, input) {
							t.Fatal("paging requested exit")
						}
						if model.Selected != 1 || model.Action != 1 || model.Focus != focus || model.Tab() != storeui.TabReady || model.Tabs.Category() != storeui.CategoryGames || model.Busy {
							t.Fatal("paging changed selection, action, focus, filter or busy state")
						}
					}
					// Match independently rasterized glyphs in the content, not an
					// offset or the wrapping helper's own account of visible text.
					visible := func(frame *image.RGBA, marker string) bool {
						expected := image.NewRGBA(image.Rect(0, 0, font.MeasureString(canvas.body, marker).Ceil(), canvas.body.Metrics().Height.Ceil()))
						fill(expected, expected.Bounds(), palette.background)
						drawer := font.Drawer{Dst: expected, Src: image.NewUniform(palette.text), Face: canvas.body, Dot: fixed.P(0, canvas.body.Metrics().Ascent.Ceil())}
						drawer.DrawString(marker)
						middle := expected.Bounds().Dy() / 2
						signature := expected.Pix[middle*expected.Stride : (middle+1)*expected.Stride]
						for y := 0; y+expected.Bounds().Dy() < canvas.footerTop(); y++ {
							strip := frame.Pix[(y+middle)*frame.Stride : (y+middle+1)*frame.Stride]
							for offset := 0; offset+expected.Stride <= len(strip); {
								index := bytes.Index(strip[offset:], signature)
								if index < 0 {
									break
								}
								left := offset + index
								offset = left + 4
								matches := true
								for row := 0; row < expected.Bounds().Dy(); row++ {
									start := (y+row)*frame.Stride + left
									if !bytes.Equal(frame.Pix[start:start+expected.Stride], expected.Pix[row*expected.Stride:(row+1)*expected.Stride]) {
										matches = false
										break
									}
								}
								if matches {
									return true
								}
							}
						}
						return false
					}
					frame := canvas.draw(model, "", controls)
					if !visible(frame, first) || visible(frame, warningEnd) || visible(frame, recoveryEnd) {
						t.Fatal("fixture must start with the warning ends offscreen")
					}
					seenWarning, seenRecovery := false, false
					for range 128 {
						before := model.ReviewOffset
						page(true)
						frame = canvas.draw(model, "", controls)
						seenWarning = seenWarning || visible(frame, warningEnd)
						seenRecovery = seenRecovery || visible(frame, recoveryEnd)
						if model.ReviewOffset == before {
							break
						}
					}
					if !seenWarning || !seenRecovery {
						t.Fatalf("final text unreachable: warning=%v recovery=%v", seenWarning, seenRecovery)
					}
					for range 128 {
						page(false)
						frame = canvas.draw(model, "", controls)
						if visible(frame, first) {
							break
						}
					}
					if !visible(frame, first) {
						t.Fatal("cannot read back to the first warning")
					}
				})
			}
		}
	}
}

func TestOutputResizeRendersWithoutResettingReviewOrBusyState(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("SDL_VIDEODRIVER", "dummy")
	if err := sdl.Init(sdl.INIT_VIDEO); err != nil {
		t.Fatal(err)
	}
	defer sdl.Quit()
	window, err := sdl.CreateWindow("Resize test", 0, 0, 640, 480, sdl.WINDOW_HIDDEN|sdl.WINDOW_RESIZABLE)
	if err != nil {
		t.Fatal(err)
	}
	defer window.Destroy()
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_SOFTWARE)
	if err != nil {
		t.Fatal(err)
	}
	defer renderer.Destroy()
	var surface outputSurface
	defer surface.close()
	model, controls, logger, _ := newRouterHarness(t)
	model.Items = []appstore.Item{{}, {Package: manifest.Package{Name: "Asymmetric review", Summary: "Keep this selected", Install: &manifest.Install{Warning: strings.Repeat("Read every warning. ", 40) + "FINAL WARNING"}}, Actions: []appstore.Action{appstore.Adopt, appstore.ForceReinstall}}}
	model.Selected, model.Action, model.Focus = 1, 1, storeui.Confirm
	for _, size := range [][2]int{{320, 240}, {720, 720}, {1280, 720}, {639, 480}, {640, 480}, {641, 480}, {480, 640}, {320, 240}} {
		window.SetSize(int32(size[0]), int32(size[1]))
		w, h, err := renderer.GetOutputSize()
		if err != nil {
			t.Fatal(err)
		}
		if changed, err := surface.resize(renderer, int(w), int(h)); err != nil || !changed {
			t.Fatalf("resize: %v %v", changed, err)
		}
		frame := surface.canvas.draw(model, "Unknown device", controls)
		handleEvent(context.Background(), model, &sdl.KeyboardEvent{Type: sdl.KEYDOWN, Keysym: sdl.Keysym{Sym: sdl.K_PAGEDOWN}}, new(*controllerState), controls, logger, InputKeyboard)
		if model.Selected != 1 || model.Action != 1 || model.Focus != storeui.Confirm || model.Busy || model.ReviewOffset == 0 {
			t.Fatal("resize/paging reset review or executed operation")
		}
		if err := surface.texture.Update(nil, unsafe.Pointer(&frame.Pix[0]), frame.Stride); err != nil {
			t.Fatal(err)
		}
		renderer.SetDrawColor(palette.background.R, palette.background.G, palette.background.B, 255)
		renderer.Clear()
		if err := renderer.Copy(surface.texture, nil, &surface.destination); err != nil {
			t.Fatal(err)
		}
		capture := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
		if err := renderer.ReadPixels(nil, sdl.PIXELFORMAT_ABGR8888, unsafe.Pointer(&capture.Pix[0]), capture.Stride); err != nil {
			t.Fatal(err)
		}
		if capture.RGBAAt(int(w)/2, int(h)-1) != palette.panel {
			t.Fatal("renderer did not fill output height")
		}
		if directory := os.Getenv("KNULLI_UI_SCREENSHOT_DIR"); directory != "" {
			if err := os.MkdirAll(directory, 0755); err != nil {
				t.Fatal(err)
			}
			if err := savePNG(filepath.Join(directory, fmt.Sprintf("resize-%dx%d.png", w, h)), capture); err != nil {
				t.Fatal(err)
			}
		}
		before := surface.canvas
		for _, invalid := range [][2]int{{0, 0}, {319, 240}, {8000, 480}, {320, 1000}} {
			if changed, err := surface.resize(renderer, invalid[0], invalid[1]); err != nil || changed || surface.canvas != before {
				t.Fatal("invalid output replaced valid surface")
			}
		}
	}
	model.Busy, model.Message = true, "Installing files: 66% (2/3)"
	if _, err := surface.resize(renderer, 640, 480); err != nil {
		t.Fatal(err)
	}
	surface.canvas.draw(model, "", controls)
	if !model.Busy || model.Message != "Installing files: 66% (2/3)" || model.Selected != 1 {
		t.Fatal("resize reset operation state")
	}
}

func TestSDLKeyboardModeIgnoresControllerInput(t *testing.T) {
	model, _, logger, _ := newRouterHarness(t)
	controls := storeinput.NewDesktopSession(t.TempDir(), "desktop")
	controller := &controllerState{instanceID: 17}
	for _, event := range []sdl.Event{
		&sdl.ControllerButtonEvent{Type: sdl.CONTROLLERBUTTONDOWN, Which: 17, Button: chordAnchorButton},
		&sdl.ControllerDeviceEvent{Type: sdl.CONTROLLERDEVICEREMOVED, Which: 17},
	} {
		if handleEvent(context.Background(), model, event, &controller, controls, logger, InputKeyboard) {
			t.Fatal("controller input exited a keyboard session")
		}
	}
	if controller == nil || controls.Source != storeinput.KeyboardSourceName {
		t.Fatal("controller input replaced the keyboard session")
	}
}
