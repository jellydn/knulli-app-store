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
}

func Run(ctx context.Context, backend appstore.Backend, options Options) error {
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
	controls := storeinput.NewSession(options.Root, options.Platform.Device)
	controller := openController()
	connectController(controls, controller, options.Diagnostics)
	if controller == nil {
		model.ExportDiagnostics(ctx)
	} else if controls.Mode == storeinput.Normal && startupOverride(controller, controls.Mapping) {
		controls.OpenSetup()
		options.Diagnostics.Event("controller_setup_override", "result", "opened", "gesture", "back+diagnostics")
	}
	defer func() {
		if controller != nil {
			C.SDL_GameControllerClose(controller.handle)
		}
	}()
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()
	for {
		var event C.SDL_Event
		for C.SDL_PollEvent(&event) != 0 {
			if handleEvent(ctx, model, &event, &controller, controls, options.Diagnostics) {
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
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

func handleEvent(ctx context.Context, model *storeui.Model, event *C.SDL_Event, controller **controllerState, controls *storeinput.Session, logger *diagnostics.Log) bool {
	switch C.event_type(event) {
	case C.SDL_QUIT:
		return true
	case C.SDL_KEYDOWN:
		if C.event_key_repeat(event) != 0 {
			return false
		}
		var action storeinput.Action
		switch C.event_key(event) {
		case C.SDLK_UP:
			action = storeinput.Up
		case C.SDLK_LEFT:
			action = storeinput.Left
		case C.SDLK_DOWN:
			action = storeinput.Down
		case C.SDLK_RIGHT:
			action = storeinput.Right
		case C.SDLK_RETURN, C.SDLK_SPACE:
			action = storeinput.Confirm
		case C.SDLK_ESCAPE:
			action = storeinput.Back
		case C.SDLK_y:
			action = storeinput.Diagnostics
		case C.SDLK_q:
			action = storeinput.Exit
		}
		if action == "" {
			return false
		}
		mapping := controls.Mapping
		if controls.FirstRun {
			mapping = storeinput.AutoMapping()
		}
		return processButton(ctx, model, controls, mapping[action], logger)
	case C.SDL_CONTROLLERBUTTONDOWN:
		if *controller == nil || C.event_controller_which(event) != (*controller).instanceID {
			return false
		}
		return processButton(ctx, model, controls, int(C.event_controller_button(event)), logger)
	case C.SDL_CONTROLLERDEVICEREMOVED:
		if *controller != nil && C.event_device_which(event) == (*controller).instanceID {
			logger.Event("controller_disconnected", "name", (*controller).identity.Name, "guid", (*controller).identity.GUID)
			C.SDL_GameControllerClose((*controller).handle)
			*controller = nil
			beforeMode, beforeSource := controls.Mode, controls.Source
			controls.Disconnect()
			logControllerTransition(logger, controls, beforeMode, beforeSource)
		}
	case C.SDL_CONTROLLERDEVICEADDED:
		if *controller == nil {
			*controller = openController()
			connectController(controls, *controller, logger)
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

func connectController(controls *storeinput.Session, controller *controllerState, logger *diagnostics.Log) {
	if controller == nil {
		logger.Event("controller_unavailable", "device", controls.Device)
		return
	}
	beforeMode, beforeSource := controls.Mode, controls.Source
	controls.Connect(controller.identity, os.Getenv("SDL_GAMECONTROLLERCONFIG") != "")
	logger.Event("controller_connected", "device", controls.Device, "name", controller.identity.Name, "guid", controller.identity.GUID, "mapping_source", controls.Source)
	logControllerTransition(logger, controls, beforeMode, beforeSource)
	if controls.ValidationError != "" {
		logger.Event("controller_mapping_validation_failed", "error", controls.ValidationError)
	}
}

func startupOverride(controller *controllerState, mapping storeinput.Mapping) bool {
	return C.controller_button(controller.handle, C.int(mapping[storeinput.Back])) != 0 &&
		C.controller_button(controller.handle, C.int(mapping[storeinput.Diagnostics])) != 0
}

func processButton(ctx context.Context, model *storeui.Model, controls *storeinput.Session, button int, logger *diagnostics.Log) bool {
	beforeMode := controls.Mode
	beforeSource := controls.Source
	beforeError := controls.ValidationError
	beforeIndex, beforeTested := setupProgress(controls)
	action, effect := controls.HandleButton(button)
	afterIndex, afterTested := setupProgress(controls)
	logControllerTransition(logger, controls, beforeMode, beforeSource)
	if afterIndex != beforeIndex && afterIndex >= 0 {
		logger.Event("controller_setup_progress", "completed", fmt.Sprint(afterIndex), "total", fmt.Sprint(len(storeinput.Actions)))
	}
	if afterTested != beforeTested && afterTested >= 0 {
		logger.Event("controller_preview_progress", "tested", fmt.Sprint(afterTested), "total", fmt.Sprint(len(storeinput.Actions)))
	}
	if controls.ValidationError != "" && controls.ValidationError != beforeError {
		logger.Event("controller_mapping_validation_failed", "error", controls.ValidationError)
	}
	if action != "" {
		logger.Event("controller_semantic_action", "action", string(action), "screen", string(beforeMode))
	}
	if effect == storeinput.ExportDiagnostics {
		logger.Event("controller_setup_action", "action", "export-diagnostics")
		model.ExportDiagnostics(ctx)
	}
	return dispatchAction(ctx, model, action)
}

func logControllerTransition(logger *diagnostics.Log, controls *storeinput.Session, beforeMode storeinput.Mode, beforeSource string) {
	if controls.Mode != beforeMode || controls.Source != beforeSource {
		logger.Event("controller_screen_transition", "from", string(beforeMode), "to", string(controls.Mode), "mapping_source", controls.Source, "message", controls.Message)
	}
}

func setupProgress(controls *storeinput.Session) (int, int) {
	if controls.Calibration == nil {
		return -1, -1
	}
	return controls.Calibration.Index, len(controls.Calibration.Tested)
}

func dispatchAction(ctx context.Context, model *storeui.Model, action storeinput.Action) bool {
	switch action {
	case storeinput.Up, storeinput.Left:
		model.Move(-1)
	case storeinput.Down, storeinput.Right:
		model.Move(1)
	case storeinput.Confirm:
		model.Select(ctx)
	case storeinput.Back:
		return model.Back()
	case storeinput.Exit:
		return true
	}
	return false
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
