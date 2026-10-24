//go:build sdl

package sdlui

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	gaba "github.com/BrandonKowalski/gabagool/v2/pkg/gabagool"
	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	"github.com/jellydn/knulli-app-store/internal/platform"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
	"github.com/veandco/go-sdl2/sdl"
	xdraw "golang.org/x/image/draw"
)

const (
	windowWidth  = 1280
	windowHeight = 720
)

type Options struct {
	Windowed    bool
	Screenshot  string
	Platform    platform.Info
	Diagnostics *diagnostics.Log
	Root        string
	// Input selects the source. The zero value is the device default: an SDL
	// GameController, or a blocked screen when none is attached.
	Input InputMode
	// Keys is a scripted key sequence for a walkthrough run. An empty sequence
	// is an interactive run.
	Keys []Key
	// ShotDir collects one frame per walkthrough step, plus the walk.tsv record
	// that names the screen each key reached.
	ShotDir string
	// WalkTimeout bounds a walkthrough run, so a stuck flow fails instead of
	// hanging. Zero means no bound.
	WalkTimeout time.Duration
}

// walkSettleFrames is how many idle frames a walkthrough renders after its last
// key before it ends, so the final screen is settled when it is captured.
const walkSettleFrames = 3

func Run(ctx context.Context, backend appstore.Backend, options Options) error {
	// The toolkit's developer wizard writes mappings and exits the process.
	// It is not part of the Store's automatic, non-persistent input contract.
	if os.Getenv("INPUT_CAPTURE") != "" {
		return fmt.Errorf("INPUT_CAPTURE is not supported; unset it to use automatic KNULLI controls")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := prepareRenderer(); err != nil {
		return err
	}
	// Use gabagool's advanced renderer API rather than its blocking screens:
	// those screens add Start-to-confirm and a second input/transition policy.
	// The Store's card, confirmations and asynchronous operations share one
	// event loop and retain their existing semantic controls.
	frameworkOptions := gaba.Options{
		WindowTitle: "Knulli App Store",
		LogPath:     "/dev/stderr",
		DisabledInputSources: gaba.DisabledInputSources{
			Keyboard: true, Controller: true, Joystick: true,
		},
	}
	frameworkOptions.WindowOptions.FullscreenDesktop = !options.Windowed
	// Do not load a firmware-specific raw joystick map or a saved assignment.
	// KNULLI's SDL GameController mapping is resolved by internal/input once.
	gaba.SetInputMappingBytes([]byte(`{}`))
	gaba.Init(frameworkOptions)
	defer gaba.Close()
	window := gaba.GetWindow().Window
	renderer := gaba.GetWindow().Renderer
	if options.Windowed {
		width, height := selectedOutputSize(options.Platform.Resolution)
		window.SetSize(int32(width), int32(height))
	}
	// The framework initializes a display-sized logical viewport. The Store
	// handles letterboxing explicitly so window and framebuffer sizes agree.
	if err := renderer.SetLogicalSize(0, 0); err != nil {
		return fmt.Errorf("reset renderer viewport: %w", err)
	}
	runtimeCandidates := runtimeResolutionCandidates(window, renderer)
	allCandidates := append([]platform.ResolutionCandidate(nil), runtimeCandidates...)
	if options.Platform.Evidence(platform.FieldResolution).Location == "command-line override" {
		allCandidates = append(options.Platform.ResolutionCandidates(), runtimeCandidates...)
	} else {
		allCandidates = append(allCandidates, options.Platform.ResolutionCandidates()...)
	}
	options.Platform = options.Platform.WithCandidates(allCandidates)
	logResolutionCandidates(options.Diagnostics, platform.AssessResolutions(allCandidates), options.Platform)
	backend.SetPlatform(options.Platform)
	platformHeader := platform.DisplayHeader(options.Platform, 0, 0)
	outputWidth, outputHeight := selectedOutputSize(options.Platform.Resolution)
	destination := outputRectangle(outputWidth, outputHeight)
	sdlDestination := sdl.Rect{X: int32(destination.Min.X), Y: int32(destination.Min.Y), W: int32(destination.Dx()), H: int32(destination.Dy())}
	texture, err := renderer.CreateTexture(sdl.PIXELFORMAT_ABGR8888, sdl.TEXTUREACCESS_STREAMING, canvasWidth, canvasHeight)
	if err != nil {
		return fmt.Errorf("create card texture: %w", err)
	}
	defer texture.Destroy()

	model := storeui.New(backend)
	if err := model.Load(ctx); err != nil {
		return err
	}
	var controls *storeinput.Session
	var controller *controllerState
	if options.Input.IsKeyboard() {
		// A desktop run has no device controller. The keyboard is the source,
		// so the catalogue opens and every flow stays reachable.
		controls = storeinput.NewDesktopSession(options.Root, options.Platform.Device)
		options.Diagnostics.Event("controller_source", "source", storeinput.KeyboardSourceName, "device", options.Platform.Device, "screen", string(controls.Mode))
	} else {
		controls = storeinput.NewSession(options.Root, options.Platform.Device)
		controller = openController()
		connectController(ctx, model, controls, controller, options.Diagnostics)
		if controller == nil {
			model.ExportDiagnostics(ctx)
		}
	}
	defer func() {
		if controller != nil {
			controller.handle.Close()
		}
	}()
	walk := NewWalkRunner(options.Keys, walkSettleFrames)
	walkLog, err := openWalkLog(walk, options.ShotDir)
	if err != nil {
		return err
	}
	defer closeWalkLog(walkLog)
	var deadline time.Time
	if walk != nil && options.WalkTimeout > 0 {
		deadline = time.Now().Add(options.WalkTimeout)
	}
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()
	exitRequested := false
	var repeat NavRepeat
	for {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			if handleEvent(ctx, model, event, &controller, controls, options.Diagnostics, options.Input) {
				return nil
			}
		}
		if !options.Input.IsKeyboard() {
			repeatHeldDirection(ctx, model, controls, controller, options.Diagnostics, &repeat)
		}
		model.Poll()
		frame := draw(model, platformHeader, controls)
		if err := texture.Update(nil, unsafe.Pointer(&frame.Pix[0]), frame.Stride); err != nil {
			return fmt.Errorf("upload card frame: %w", err)
		}
		renderer.SetDrawColor(0, 0, 0, 255)
		renderer.Clear()
		if err := renderer.Copy(texture, nil, &sdlDestination); err != nil {
			return fmt.Errorf("render card frame: %w", err)
		}
		// Capture the actual renderer before Present invalidates its backbuffer,
		// not just the source bitmap: viewport or texture regressions must show
		// up in the desktop walkthrough evidence.
		var rendered *image.RGBA
		if options.Screenshot != "" || walkLog != nil {
			width, height, err := renderer.GetOutputSize()
			if err != nil {
				return err
			}
			rendered = image.NewRGBA(image.Rect(0, 0, int(width), int(height)))
			if err := renderer.ReadPixels(nil, sdl.PIXELFORMAT_ABGR8888, unsafe.Pointer(&rendered.Pix[0]), rendered.Stride); err != nil {
				return fmt.Errorf("capture rendered frame: %w", err)
			}
		}
		gaba.GetWindow().Present()
		if options.Screenshot != "" {
			return savePNG(options.Screenshot, rendered)
		}
		if walk != nil {
			shot, capture := walk.Capture(WalkState(model, controls), model.Busy)
			if capture && walkLog != nil {
				if err := savePNG(filepath.Join(options.ShotDir, shot.File), rendered); err != nil {
					return err
				}
				if _, err := walkLog.WriteString(WalkRecord(shot, WalkDetail(model, controls))); err != nil {
					return err
				}
			}
			if exitRequested {
				if walk.Sent() != len(options.Keys) {
					return fmt.Errorf("walkthrough exited after %d of %d keys", walk.Sent(), len(options.Keys))
				}
				walkFinished(options, walk)
				return nil
			}
			if key, ok := walk.Step(model.Busy); ok {
				// The key goes through the same handler as a real keystroke, so a
				// walkthrough reaches the same screens a user would.
				if Handle(ctx, model, controls, options.Diagnostics, Event{Kind: EventKey, Key: key}).Exit {
					exitRequested = true
				}
			}
			if !exitRequested && walk.Done(model.Busy) {
				walkFinished(options, walk)
				return nil
			}
			if !deadline.IsZero() && time.Now().After(deadline) {
				cancel()
				return fmt.Errorf("walkthrough timed out after %s: sent %d of %d keys, captured %d frames", options.WalkTimeout, walk.Sent(), len(options.Keys), walk.Steps())
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// Gabagool requires an accelerated renderer unless SDL's driver hint selects
// software. Probe first to retain the Store's software fallback (also used by
// dummy-video walkthroughs), rather than letting the toolkit exit the process.
func prepareRenderer() error {
	if err := sdl.InitSubSystem(sdl.INIT_VIDEO); err != nil {
		return fmt.Errorf("initialize video: %w", err)
	}
	defer sdl.QuitSubSystem(sdl.INIT_VIDEO)
	window, err := sdl.CreateWindow("Renderer check", 0, 0, windowWidth, windowHeight, sdl.WINDOW_HIDDEN)
	if err != nil {
		return fmt.Errorf("check video window: %w", err)
	}
	defer window.Destroy()
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC|sdl.RENDERER_TARGETTEXTURE)
	if err != nil {
		renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_SOFTWARE|sdl.RENDERER_TARGETTEXTURE)
		if err != nil {
			return fmt.Errorf("check software renderer: %w", err)
		}
		sdl.SetHint(sdl.HINT_RENDER_DRIVER, "software")
	}
	return renderer.Destroy()
}

// openWalkLog creates the walkthrough evidence record. A run without a scripted
// sequence, or without an evidence directory, writes no record.
func openWalkLog(walk *WalkRunner, directory string) (*os.File, error) {
	if walk == nil || directory == "" {
		return nil, nil
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}
	log, err := os.Create(filepath.Join(directory, "walk.tsv"))
	if err != nil {
		return nil, err
	}
	if _, err := log.WriteString(WalkHeader); err != nil {
		closeWalkLog(log)
		return nil, err
	}
	return log, nil
}

func closeWalkLog(log *os.File) {
	if log != nil {
		log.Close()
	}
}

// walkFinished records what the walkthrough actually did, so the evidence is
// self-describing.
func walkFinished(options Options, walk *WalkRunner) {
	if walk != nil {
		options.Diagnostics.Event("walkthrough_complete", "keys", fmt.Sprint(walk.Sent()), "frames", fmt.Sprint(walk.Steps()))
	}
}

func runtimeResolutionCandidates(window *sdl.Window, renderer *sdl.Renderer) []platform.ResolutionCandidate {
	var candidates []platform.ResolutionCandidate
	width, height, err := renderer.GetOutputSize()
	rendererCandidate := platform.ResolutionCandidate{Source: "SDL renderer output", Width: int(width), Height: int(height)}
	if err != nil {
		rendererCandidate.Error = "SDL_GetRendererOutputSize failed: " + err.Error()
	}
	candidates = append(candidates, rendererCandidate)
	width, height = window.GetSize()
	candidates = append(candidates, platform.ResolutionCandidate{Source: "SDL window size", Width: int(width), Height: int(height)})
	displayIndex, err := window.GetDisplayIndex()
	if err == nil {
		var mode sdl.DisplayMode
		mode, err = sdl.GetCurrentDisplayMode(displayIndex)
		if err == nil {
			candidates = append(candidates, platform.ResolutionCandidate{Source: "SDL current display mode", Width: int(mode.W), Height: int(mode.H)})
		}
	}
	if err != nil {
		candidates = append(candidates, platform.ResolutionCandidate{Source: "SDL current display mode", Error: "SDL display mode query failed: " + err.Error()})
	}
	return candidates
}

func logResolutionCandidates(logger *diagnostics.Log, assessments []platform.ResolutionAssessment, selected platform.Info) {
	for _, candidate := range assessments {
		logger.Event("resolution_candidate", "source", candidate.Source, "width", fmt.Sprint(candidate.Width), "height", fmt.Sprint(candidate.Height), "valid", fmt.Sprint(candidate.Valid), "reason", candidate.Reason)
	}
	logger.Event("resolution_selected", "resolution", selected.Resolution, "source", selected.Evidence(platform.FieldResolution).Location)
	logger.Event("platform_detected", "details", platform.Summary(selected))
}

func handleEvent(ctx context.Context, model *storeui.Model, event sdl.Event, controller **controllerState, controls *storeinput.Session, logger *diagnostics.Log, input InputMode) bool {
	eventType := event.GetType()
	if input.IsKeyboard() && (eventType == sdl.CONTROLLERBUTTONDOWN || eventType == sdl.CONTROLLERDEVICEREMOVED || eventType == sdl.CONTROLLERDEVICEADDED) {
		// A desktop run keeps the keyboard as its only source, so attaching a
		// controller cannot replace it mid-flow.
		return false
	}
	switch event := event.(type) {
	case *sdl.QuitEvent:
		return Handle(ctx, model, controls, logger, Event{Kind: EventQuit}).Exit
	case *sdl.KeyboardEvent:
		if event.Type == sdl.KEYUP {
			switch event.Keysym.Sym {
			case sdl.K_TAB, sdl.K_F5:
				return Handle(ctx, model, controls, logger, Event{Kind: EventChordAnchor, Held: false}).Exit
			}
			return false
		}
		// The quit chord's anchor is held state, not an action: Tab (or F5, the
		// key a desktop hand reaches for) turns the next Y into the quit, the
		// way holding Select does on a pad. It is handled before the translation
		// below, which drops a repeated anchor.
		if event.Repeat == 0 {
			switch event.Keysym.Sym {
			case sdl.K_TAB, sdl.K_F5:
				return Handle(ctx, model, controls, logger, Event{Kind: EventChordAnchor, Held: true}).Exit
			}
		}
		var key Key
		switch event.Keysym.Sym {
		case sdl.K_UP:
			key = KeyUp
		case sdl.K_LEFT:
			key = KeyLeft
		case sdl.K_DOWN:
			key = KeyDown
		case sdl.K_RIGHT:
			key = KeyRight
		case sdl.K_PAGEUP:
			key = KeyPageUp
		case sdl.K_PAGEDOWN:
			key = KeyPageDown
		case sdl.K_RETURN, sdl.K_SPACE:
			key = KeyConfirm
		case sdl.K_ESCAPE:
			key = KeyBack
		case sdl.K_y:
			key = KeyDiagnostics
		}
		if key == KeyNone {
			return false
		}
		// A key repeat reaches Handle marked as one, which lets only navigation
		// answer it: a held Enter must not confirm twice.
		return Handle(ctx, model, controls, logger, Event{Kind: EventKey, Key: key, Repeat: event.Repeat != 0}).Exit
	case *sdl.ControllerButtonEvent:
		if *controller == nil || event.Which != (*controller).instanceID {
			return false
		}
		button := int(event.Button)
		if button == chordAnchorButton {
			return Handle(ctx, model, controls, logger, Event{Kind: EventChordAnchor, Held: event.Type == sdl.CONTROLLERBUTTONDOWN}).Exit
		}
		if event.Type == sdl.CONTROLLERBUTTONDOWN {
			return Handle(ctx, model, controls, logger, Event{Kind: EventButton, Button: button}).Exit
		}
	case *sdl.ControllerDeviceEvent:
		if event.Type == sdl.CONTROLLERDEVICEREMOVED && *controller != nil && event.Which == (*controller).instanceID {
			logger.Event("controller_disconnected", "name", (*controller).identity.Name, "guid", (*controller).identity.GUID)
			(*controller).handle.Close()
			*controller = nil
			Handle(ctx, model, controls, logger, Event{Kind: EventControllerRemoved})
		}
		if event.Type == sdl.CONTROLLERDEVICEADDED && *controller == nil {
			*controller = openController()
			connectController(ctx, model, controls, *controller, logger)
		}
	}
	return false
}

type controllerState struct {
	handle     *sdl.GameController
	identity   storeinput.Identity
	instanceID sdl.JoystickID
}

func openController() *controllerState {
	for index := 0; index < sdl.NumJoysticks(); index++ {
		if sdl.IsGameController(index) {
			handle := sdl.GameControllerOpen(index)
			if handle == nil {
				continue
			}
			joystick := handle.Joystick()
			return &controllerState{handle: handle, identity: storeinput.Identity{GUID: sdl.JoystickGetGUIDString(joystick.GUID()), Name: handle.Name()}, instanceID: joystick.InstanceID()}
		}
	}
	return nil
}

func connectController(ctx context.Context, model *storeui.Model, controls *storeinput.Session, controller *controllerState, logger *diagnostics.Log) {
	if controller == nil {
		logger.Event("controller_unavailable", "device", controls.Device)
		return
	}
	Handle(ctx, model, controls, logger, Event{Kind: EventControllerConnected, Identity: controller.identity, KnulliMapping: os.Getenv("SDL_GAMECONTROLLERCONFIG") != ""})
}

// chordAnchorButton is SDL's SELECT/BACK button, the pad's own Select key. It
// is deliberately outside the semantic mapping: quitting is the app's escape
// hatch, not an action a user can bind, and holding it alone does nothing.
const chordAnchorButton = 4

// heldDirection reports the direction a pad is holding right now, if any. The
// adapter polls this instead of tracking button releases, which is what lets a
// dropped release event leave nothing stuck.
func heldDirection(controller *controllerState, mapping storeinput.Mapping) storeinput.Action {
	for _, action := range []storeinput.Action{storeinput.Up, storeinput.Down, storeinput.Left, storeinput.Right} {
		button, ok := mapping[action]
		if !ok {
			continue
		}
		if controller.handle.Button(sdl.GameControllerButton(button)) != 0 {
			return action
		}
	}
	return ""
}

// repeatHeldDirection feeds the pad's held direction to the repeat schedule and
// re-dispatches the direction through the same handler a press uses, so a
// repeat reaches the screen the press did.
func repeatHeldDirection(ctx context.Context, model *storeui.Model, controls *storeinput.Session, controller *controllerState, logger *diagnostics.Log, repeat *NavRepeat) {
	var direction storeinput.Action
	if controller != nil {
		direction = heldDirection(controller, controls.ScreenMapping())
	}
	action, due := repeat.Held(direction, time.Now())
	if !due {
		return
	}
	button, ok := controls.ScreenMapping()[action]
	if !ok {
		return
	}
	Handle(ctx, model, controls, logger, Event{Kind: EventButton, Button: button})
}

func selectedOutputSize(resolution string) (int, int) {
	width, height := windowWidth, windowHeight
	if _, err := fmt.Sscanf(resolution, "%dx%d", &width, &height); err != nil || width <= 0 || height <= 0 {
		return windowWidth, windowHeight
	}
	return width, height
}

func outputRectangle(width, height int) image.Rectangle {
	scaleWidth := width
	scaleHeight := scaleWidth * canvasHeight / canvasWidth
	if scaleHeight > height {
		scaleHeight = height
		scaleWidth = scaleHeight * canvasWidth / canvasHeight
	}
	x := (width - scaleWidth) / 2
	y := (height - scaleHeight) / 2
	return image.Rect(x, y, x+scaleWidth, y+scaleHeight)
}

func renderOutput(frame *image.RGBA, width, height int) *image.RGBA {
	output := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.NearestNeighbor.Scale(output, outputRectangle(width, height), frame, frame.Bounds(), xdraw.Src, nil)
	return output
}

func saveOutputScreenshot(path string, frame *image.RGBA, width, height int) error {
	return savePNG(path, renderOutput(frame, width, height))
}

func savePNG(path string, frame image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, frame); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
