//go:build sdl

package sdlui

/*
#cgo pkg-config: sdl2
#include <stdlib.h>
#include <SDL2/SDL.h>

static Uint32 event_type(SDL_Event *event) { return event->type; }
static SDL_Keycode event_key(SDL_Event *event) { return event->key.keysym.sym; }
static Uint8 event_key_repeat(SDL_Event *event) { return event->key.repeat; }
static Uint8 event_controller_button(SDL_Event *event) { return event->cbutton.button; }
static SDL_JoystickID event_controller_which(SDL_Event *event) { return event->cbutton.which; }
static SDL_JoystickID event_device_which(SDL_Event *event) { return event->cdevice.which; }
static void controller_guid(SDL_GameController *controller, char *output, int size) {
    SDL_JoystickGetGUIDString(SDL_JoystickGetGUID(SDL_GameControllerGetJoystick(controller)), output, size);
}
static Uint8 controller_button(SDL_GameController *controller, int button) {
    return SDL_GameControllerGetButton(controller, (SDL_GameControllerButton)button);
}
*/
import "C"

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

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	"github.com/jellydn/knulli-app-store/internal/platform"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if C.SDL_Init(C.SDL_INIT_VIDEO|C.SDL_INIT_GAMECONTROLLER|C.SDL_INIT_EVENTS) != 0 {
		return sdlError("initialize SDL2")
	}
	defer C.SDL_Quit()
	flags := C.Uint32(C.SDL_WINDOW_SHOWN)
	if !options.Windowed {
		flags |= C.SDL_WINDOW_FULLSCREEN_DESKTOP
	}
	title := C.CString("Knulli App Store")
	defer C.free(unsafe.Pointer(title))
	window := C.SDL_CreateWindow(title, C.SDL_WINDOWPOS_UNDEFINED, C.SDL_WINDOWPOS_UNDEFINED, windowWidth, windowHeight, flags)
	if window == nil {
		return sdlError("create SDL2 window")
	}
	defer C.SDL_DestroyWindow(window)
	renderer := C.SDL_CreateRenderer(window, -1, C.SDL_RENDERER_ACCELERATED|C.SDL_RENDERER_PRESENTVSYNC)
	if renderer == nil {
		renderer = C.SDL_CreateRenderer(window, -1, C.SDL_RENDERER_SOFTWARE)
	}
	if renderer == nil {
		return sdlError("create SDL2 renderer")
	}
	defer C.SDL_DestroyRenderer(renderer)
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
	sdlDestination := C.SDL_Rect{x: C.int(destination.Min.X), y: C.int(destination.Min.Y), w: C.int(destination.Dx()), h: C.int(destination.Dy())}
	texture := C.SDL_CreateTexture(renderer, C.SDL_PIXELFORMAT_ABGR8888, C.SDL_TEXTUREACCESS_STREAMING, canvasWidth, canvasHeight)
	if texture == nil {
		return sdlError("create SDL2 texture")
	}
	defer C.SDL_DestroyTexture(texture)

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
		} else if controls.Mode == storeinput.Normal && startupOverride(controller, controls.Mapping) {
			controls.OpenSetup()
			options.Diagnostics.Event("controller_setup_override", "result", "opened", "gesture", "back+diagnostics")
		}
	}
	defer func() {
		if controller != nil {
			C.SDL_GameControllerClose(controller.handle)
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
	for {
		var event C.SDL_Event
		for C.SDL_PollEvent(&event) != 0 {
			if handleEvent(ctx, model, &event, &controller, controls, options.Diagnostics, options.Input) {
				return nil
			}
		}
		model.Poll()
		frame := draw(model, platformHeader, controls)
		if C.SDL_UpdateTexture(texture, nil, unsafe.Pointer(&frame.Pix[0]), C.int(frame.Stride)) != 0 {
			return sdlError("upload UI frame")
		}
		C.SDL_SetRenderDrawColor(renderer, 0, 0, 0, 255)
		C.SDL_RenderClear(renderer)
		if C.SDL_RenderCopy(renderer, texture, nil, &sdlDestination) != 0 {
			return sdlError("render UI frame")
		}
		C.SDL_RenderPresent(renderer)
		if options.Screenshot != "" {
			return saveOutputScreenshot(options.Screenshot, frame, outputWidth, outputHeight)
		}
		if walk != nil {
			shot, capture := walk.Capture(WalkState(model, controls), model.Busy)
			if capture && walkLog != nil {
				if err := saveOutputScreenshot(filepath.Join(options.ShotDir, shot.File), frame, outputWidth, outputHeight); err != nil {
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

func runtimeResolutionCandidates(window *C.SDL_Window, renderer *C.SDL_Renderer) []platform.ResolutionCandidate {
	var candidates []platform.ResolutionCandidate
	var width, height C.int
	result := C.SDL_GetRendererOutputSize(renderer, &width, &height)
	rendererCandidate := platform.ResolutionCandidate{Source: "SDL renderer output", Width: int(width), Height: int(height)}
	if result != 0 {
		rendererCandidate.Error = "SDL_GetRendererOutputSize failed: " + C.GoString(C.SDL_GetError())
	}
	candidates = append(candidates, rendererCandidate)
	width, height = 0, 0
	C.SDL_GetWindowSize(window, &width, &height)
	candidates = append(candidates, platform.ResolutionCandidate{Source: "SDL window size", Width: int(width), Height: int(height)})
	displayIndex := C.SDL_GetWindowDisplayIndex(window)
	var mode C.SDL_DisplayMode
	if displayIndex >= 0 && C.SDL_GetCurrentDisplayMode(displayIndex, &mode) == 0 {
		candidates = append(candidates, platform.ResolutionCandidate{Source: "SDL current display mode", Width: int(mode.w), Height: int(mode.h)})
	} else {
		candidates = append(candidates, platform.ResolutionCandidate{Source: "SDL current display mode", Error: "SDL display mode query failed: " + C.GoString(C.SDL_GetError())})
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

func handleEvent(ctx context.Context, model *storeui.Model, event *C.SDL_Event, controller **controllerState, controls *storeinput.Session, logger *diagnostics.Log, input InputMode) bool {
	eventType := C.event_type(event)
	if input.IsKeyboard() && (eventType == C.SDL_CONTROLLERBUTTONDOWN || eventType == C.SDL_CONTROLLERDEVICEREMOVED || eventType == C.SDL_CONTROLLERDEVICEADDED) {
		// A desktop run keeps the keyboard as its only source, so attaching a
		// controller cannot replace it mid-flow.
		return false
	}
	switch eventType {
	case C.SDL_QUIT:
		return Handle(ctx, model, controls, logger, Event{Kind: EventQuit}).Exit
	case C.SDL_KEYDOWN:
		if C.event_key_repeat(event) != 0 {
			return false
		}
		var key Key
		switch C.event_key(event) {
		case C.SDLK_UP:
			key = KeyUp
		case C.SDLK_LEFT:
			key = KeyLeft
		case C.SDLK_DOWN:
			key = KeyDown
		case C.SDLK_RIGHT:
			key = KeyRight
		case C.SDLK_RETURN, C.SDLK_SPACE:
			key = KeyConfirm
		case C.SDLK_ESCAPE:
			key = KeyBack
		case C.SDLK_y:
			key = KeyDiagnostics
		case C.SDLK_q:
			key = KeyExit
		}
		if key == KeyNone {
			return false
		}
		return Handle(ctx, model, controls, logger, Event{Kind: EventKey, Key: key}).Exit
	case C.SDL_CONTROLLERBUTTONDOWN:
		if *controller == nil || C.event_controller_which(event) != (*controller).instanceID {
			return false
		}
		return Handle(ctx, model, controls, logger, Event{Kind: EventButton, Button: int(C.event_controller_button(event))}).Exit
	case C.SDL_CONTROLLERDEVICEREMOVED:
		if *controller != nil && C.event_device_which(event) == (*controller).instanceID {
			logger.Event("controller_disconnected", "name", (*controller).identity.Name, "guid", (*controller).identity.GUID)
			C.SDL_GameControllerClose((*controller).handle)
			*controller = nil
			Handle(ctx, model, controls, logger, Event{Kind: EventControllerRemoved})
		}
	case C.SDL_CONTROLLERDEVICEADDED:
		if *controller == nil {
			*controller = openController()
			connectController(ctx, model, controls, *controller, logger)
		}
	}
	return false
}

type controllerState struct {
	handle     *C.SDL_GameController
	identity   storeinput.Identity
	instanceID C.SDL_JoystickID
}

func openController() *controllerState {
	for index := C.int(0); index < C.SDL_NumJoysticks(); index++ {
		if C.SDL_IsGameController(index) == C.SDL_TRUE {
			handle := C.SDL_GameControllerOpen(index)
			if handle == nil {
				continue
			}
			var guid [33]C.char
			C.controller_guid(handle, &guid[0], C.int(len(guid)))
			joystick := C.SDL_GameControllerGetJoystick(handle)
			return &controllerState{handle: handle, identity: storeinput.Identity{GUID: C.GoString(&guid[0]), Name: C.GoString(C.SDL_GameControllerName(handle))}, instanceID: C.SDL_JoystickInstanceID(joystick)}
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

func startupOverride(controller *controllerState, mapping storeinput.Mapping) bool {
	return C.controller_button(controller.handle, C.int(mapping[storeinput.Back])) != 0 &&
		C.controller_button(controller.handle, C.int(mapping[storeinput.Diagnostics])) != 0
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

func sdlError(operation string) error {
	return fmt.Errorf("%s: %s", operation, C.GoString(C.SDL_GetError()))
}

func saveScreenshot(path string, frame *image.RGBA) error {
	return saveOutputScreenshot(path, frame, windowWidth, windowHeight)
}

func saveOutputScreenshot(path string, frame *image.RGBA, width, height int) error {
	scaled := renderOutput(frame, width, height)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, scaled); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
