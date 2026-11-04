//go:build sdl

package sdlui

import (
	"image"
	"path"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func (c *consoleCanvas) browse(frame *image.RGBA, model *storeui.Model, top int) {
	// Equal-height rows keep neighbours stationary as focus moves. The detail
	// pane and thumbnails use reviewed, local content without network requests.
	bottom := c.statusTop() - 4
	divider := 24 + (c.width-48)*47/100
	fill(frame, image.Rect(divider, top, divider+1, bottom), palette.panel)
	start, end := model.Window(max(1, (bottom-top)/52))
	for index := start; index < end; index++ {
		rowTop := top + (index-start)*52
		row := image.Rect(24, rowTop, divider-16, rowTop+48)
		shade := palette.muted
		if index == model.Selected {
			fill(frame, row, palette.accent)
			fill(frame, row.Inset(2), palette.panel)
			shade = palette.text
		}
		item := model.Items[index]
		c.thumbnail(frame, item, image.Rect(row.Min.X+8, rowTop+4, row.Min.X+48, rowTop+44))
		c.text(frame, item.Package.Name, rowTop+2, row.Min.X+56, row.Dx()-68, shade, false)
		c.text(frame, categoryLabel(item)+" / "+installState(item), rowTop+25, row.Min.X+56, row.Dx()-68, palette.muted, false)
	}
	item := model.Items[model.Selected]
	left, right := divider+24, c.width-24
	c.prose(frame, item.Package.Name, image.Rect(left, top, right, top+63), palette.text, true)
	c.thumbnail(frame, item, image.Rect(right-80, top+68, right, top+148))
	c.text(frame, categoryLabel(item)+" / v"+item.Package.Version, top+68, left, right-left-96, palette.accent, false)
	// Test evidence and installation state are independent: an available app
	// is not necessarily tested on this device.
	c.text(frame, strings.ToUpper(item.Package.Review.Status), top+98, left, right-left-96, palette.warning, false)
	c.text(frame, trustLabel(item), top+124, left, right-left-96, palette.warning, false)
	c.text(frame, installState(item), top+150, left, right-left, statusColor(item), false)
	c.prose(frame, item.Package.Summary, image.Rect(left, top+186, right, bottom), palette.text, false)
}

func categoryLabel(item appstore.Item) string {
	if storeui.CategoryGames.Match(item) {
		// The registered ES system identifies the game format, not the host CPU.
		if install := item.Package.Install; install != nil && install.Menu != nil {
			switch path.Base(path.Dir(install.Menu.Gamelist)) {
			case "gba":
				return "GBA"
			case "arduboy":
				return "Arduboy"
			case "prboom":
				return "DOOM"
			}
		}
		return "GAME"
	}
	return "APP"
}
