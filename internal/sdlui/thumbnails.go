//go:build sdl

package sdlui

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	imagedraw "image/draw"
	"image/png"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	"golang.org/x/image/draw"
)

// Reviewed, bundled images keep browsing offline and independent of installed files.
//
//go:embed assets/thumbnails/*.png
var thumbnailFiles embed.FS

func loadThumbnails() (map[string]image.Image, error) {
	files, err := thumbnailFiles.ReadDir("assets/thumbnails")
	if err != nil {
		return nil, err
	}
	thumbnails := make(map[string]image.Image, len(files))
	for _, file := range files {
		data, err := thumbnailFiles.ReadFile("assets/thumbnails/" + file.Name())
		if err != nil {
			return nil, err
		}
		thumbnail, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, fmt.Errorf("load thumbnail %s: %w", file.Name(), err)
		}
		thumbnails[strings.TrimSuffix(file.Name(), ".png")] = thumbnail
	}
	return thumbnails, nil
}

func (c *consoleCanvas) thumbnail(frame *image.RGBA, item appstore.Item, bounds image.Rectangle) {
	fill(frame, bounds, palette.panel)
	if thumbnail := c.thumbnails[item.Package.ID]; thumbnail != nil {
		width, height := thumbnail.Bounds().Dx(), thumbnail.Bounds().Dy()
		fittedWidth, fittedHeight := bounds.Dx(), bounds.Dx()*height/width
		if fittedHeight > bounds.Dy() {
			fittedHeight, fittedWidth = bounds.Dy(), bounds.Dy()*width/height
		}
		left := bounds.Min.X + (bounds.Dx()-fittedWidth)/2
		top := bounds.Min.Y + (bounds.Dy()-fittedHeight)/2
		draw.CatmullRom.Scale(frame, image.Rect(left, top, left+fittedWidth, top+fittedHeight), thumbnail, thumbnail.Bounds(), imagedraw.Over, nil)
		return
	}
	// Initials are a fallback, not a claim to be the publisher's logo.
	initials := []rune(strings.ToUpper(item.Package.Name))
	if len(initials) > 2 {
		initials = initials[:2]
	}
	c.label(frame, string(initials), bounds.Min.Y+(bounds.Dy()-c.body.Metrics().Height.Ceil())/2, bounds.Min.X, bounds.Dx(), palette.accent, false)
}
