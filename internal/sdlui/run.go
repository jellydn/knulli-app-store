//go:build sdl
// +build sdl

package sdlui

/*
#cgo pkg-config: sdl2
#include <stdlib.h>
#include <SDL2/SDL.h>

static Uint32 event_type(SDL_Event *event) { return event->type; }
static SDL_Keycode event_key(SDL_Event *event) { return event->key.keysym.sym; }
static Uint8 event_key_repeat(SDL_Event *event) { return event->key.repeat; }
static Uint8 event_controller_button(SDL_Event *event) { return event->cbutton.button; }
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
	if options.Platform.ResolutionSource == "command-line override" {
		allCandidates = append(append([]platform.ResolutionCandidate(nil), options.Platform.ResolutionCandidates...), runtimeCandidates...)
	} else {
		allCandidates = append(allCandidates, options.Platform.ResolutionCandidates...)
	}
	options.Platform = platform.WithResolutionCandidates(options.Platform, allCandidates)
	logResolutionCandidates(options.Diagnostics, platform.AssessResolutions(allCandidates), options.Platform)
	backend.SetPlatform(options.Platform)
	platformHeader := platform.DisplayHeader(options.Platform, 0, 0)
	texture := C.SDL_CreateTexture(renderer, C.SDL_PIXELFORMAT_ABGR8888, C.SDL_TEXTUREACCESS_STREAMING, canvasWidth, canvasHeight)
	if texture == nil {
		return sdlError("create SDL2 texture")
	}
	defer C.SDL_DestroyTexture(texture)

	model := storeui.New(backend)
	if err := model.Load(ctx); err != nil {
		return err
	}
	controller := openController()
	defer func() {
		if controller != nil {
			C.SDL_GameControllerClose(controller)
		}
	}()
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()
	for {
		var event C.SDL_Event
		for C.SDL_PollEvent(&event) != 0 {
			if handleEvent(ctx, model, &event, &controller) {
				return nil
			}
		}
		model.Poll()
		frame := draw(model, platformHeader, controller != nil)
		if C.SDL_UpdateTexture(texture, nil, unsafe.Pointer(&frame.Pix[0]), C.int(frame.Stride)) != 0 {
			return sdlError("upload UI frame")
		}
		if C.SDL_RenderCopy(renderer, texture, nil, nil) != 0 {
			return sdlError("render UI frame")
		}
		C.SDL_RenderPresent(renderer)
		if options.Screenshot != "" {
			return saveScreenshot(options.Screenshot, frame)
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
	logger.Event("resolution_selected", "resolution", selected.Resolution, "source", selected.ResolutionSource)
	logger.Event("platform_detected", "details", platform.Summary(selected))
}

func handleEvent(ctx context.Context, model *storeui.Model, event *C.SDL_Event, controller **C.SDL_GameController) bool {
	switch C.event_type(event) {
	case C.SDL_QUIT:
		return true
	case C.SDL_KEYDOWN:
		if C.event_key_repeat(event) != 0 {
			return false
		}
		switch C.event_key(event) {
		case C.SDLK_UP, C.SDLK_LEFT:
			model.Move(-1)
		case C.SDLK_DOWN, C.SDLK_RIGHT:
			model.Move(1)
		case C.SDLK_RETURN, C.SDLK_SPACE:
			model.Select(ctx)
		case C.SDLK_ESCAPE:
			return model.Back()
		case C.SDLK_y:
			model.ExportDiagnostics(ctx)
		}
	case C.SDL_CONTROLLERBUTTONDOWN:
		switch C.event_controller_button(event) {
		case C.SDL_CONTROLLER_BUTTON_DPAD_UP, C.SDL_CONTROLLER_BUTTON_DPAD_LEFT:
			model.Move(-1)
		case C.SDL_CONTROLLER_BUTTON_DPAD_DOWN, C.SDL_CONTROLLER_BUTTON_DPAD_RIGHT:
			model.Move(1)
		case C.SDL_CONTROLLER_BUTTON_A:
			model.Select(ctx)
		case C.SDL_CONTROLLER_BUTTON_B:
			return model.Back()
		case C.SDL_CONTROLLER_BUTTON_Y:
			model.ExportDiagnostics(ctx)
		}
	case C.SDL_CONTROLLERDEVICEREMOVED:
		if *controller != nil {
			C.SDL_GameControllerClose(*controller)
			*controller = nil
		}
	case C.SDL_CONTROLLERDEVICEADDED:
		if *controller == nil {
			*controller = openController()
		}
	}
	return false
}

func openController() *C.SDL_GameController {
	for index := C.int(0); index < C.SDL_NumJoysticks(); index++ {
		if C.SDL_IsGameController(index) == C.SDL_TRUE {
			return C.SDL_GameControllerOpen(index)
		}
	}
	return nil
}

func sdlError(operation string) error {
	return fmt.Errorf("%s: %s", operation, C.GoString(C.SDL_GetError()))
}

func saveScreenshot(path string, frame *image.RGBA) error {
	scaled := image.NewRGBA(image.Rect(0, 0, windowWidth, windowHeight))
	xdraw.NearestNeighbor.Scale(scaled, scaled.Bounds(), frame, frame.Bounds(), xdraw.Src, nil)
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
