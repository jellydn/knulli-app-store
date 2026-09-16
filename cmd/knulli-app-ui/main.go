//go:build sdl

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"github.com/jellydn/knulli-app-store/internal/diagnostics"
	"github.com/jellydn/knulli-app-store/internal/installer"
	"github.com/jellydn/knulli-app-store/internal/platform"
	"github.com/jellydn/knulli-app-store/internal/sdlui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Knulli App Store:", err)
		os.Exit(1)
	}
}

func run() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	defaultCatalogue := filepath.Join(filepath.Dir(executable), "catalog-index.json")
	catalogue := flag.String("catalog", defaultCatalogue, "catalogue index path")
	root := flag.String("root", "/", "filesystem root")
	firmware := flag.String("firmware", "", "firmware override")
	version := flag.String("firmware-version", "", "firmware version override")
	arch := flag.String("arch", "", "architecture override")
	device := flag.String("device", "", "device override")
	resolution := flag.String("resolution", "", "resolution override")
	windowed := flag.Bool("windowed", false, "use a window instead of fullscreen")
	screenshot := flag.String("screenshot", "", "save one rendered frame and exit")
	input := flag.String("input", string(sdlui.InputAuto), "input source: auto (SDL GameController) or keyboard (desktop verification)")
	keys := flag.String("keys", "", "walkthrough key sequence, for example \"down,down,enter\" (desktop verification)")
	shotDir := flag.String("shot-dir", "", "save one frame and one walk.tsv record per walkthrough step")
	walkTimeout := flag.Duration("walk-timeout", time.Minute, "bound a walkthrough run, so a stuck flow fails instead of hanging")
	flag.Parse()
	inputMode, err := sdlui.ParseInputMode(*input)
	if err != nil {
		return err
	}
	walkKeys, err := sdlui.ParseKeys(*keys)
	if err != nil {
		return err
	}
	if *shotDir != "" && len(walkKeys) == 0 {
		return fmt.Errorf("-shot-dir needs -keys: there is no walkthrough step to capture")
	}

	diagnosticLog, err := diagnostics.Open(*root)
	if err != nil {
		return fmt.Errorf("open diagnostics log: %w", err)
	}
	diagnosticLog.Event("startup", "component", "gui")
	current := platform.Resolve(*root,
		platform.WithFirmware(*firmware),
		platform.WithVersion(*version),
		platform.WithArch(*arch),
		platform.WithDevice(*device),
		platform.WithResolutionOverride(*resolution),
	)
	if current.Arch == "" {
		current.Arch = runtime.GOARCH
	}
	diagnosticLog.Event("platform_filesystem_detected", "details", platform.Summary(current))
	manager := installer.Manager{Root: *root, Diagnostics: diagnosticLog}.WithPlatform(current)
	service, err := appstore.Open(*catalogue, manager)
	if err != nil {
		diagnosticLog.Event("startup_error", "error", err.Error())
		return err
	}
	err = sdlui.Run(context.Background(), service, sdlui.Options{
		Windowed:    *windowed || *screenshot != "" || len(walkKeys) > 0,
		Screenshot:  *screenshot,
		Platform:    current,
		Diagnostics: diagnosticLog,
		Root:        *root,
		Input:       inputMode,
		Keys:        walkKeys,
		ShotDir:     *shotDir,
		WalkTimeout: *walkTimeout,
	})
	if err != nil {
		diagnosticLog.Event("final_error", "error", err.Error())
	}
	return err
}
