//go:build sdl
// +build sdl

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jellydn/knulli-app-store/internal/appstore"
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
	flag.Parse()

	current := platform.Detect(*root)
	override(&current.Firmware, *firmware)
	override(&current.Version, *version)
	override(&current.Arch, *arch)
	override(&current.Device, *device)
	override(&current.Resolution, *resolution)
	if current.Arch == "" {
		current.Arch = runtime.GOARCH
	}
	manager := installer.Manager{Root: *root, Platform: current}
	service, err := appstore.Open(*catalogue, manager)
	if err != nil {
		return err
	}
	return sdlui.Run(context.Background(), service, sdlui.Options{Windowed: *windowed || *screenshot != "", Screenshot: *screenshot, Platform: current})
}

func override(target *string, value string) {
	if value != "" {
		*target = value
	}
}
