//go:build sdl

package sdlui

import (
	"image"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func (c *consoleCanvas) browse(frame *image.RGBA, model *storeui.Model) {
	// Equal-height rows keep neighbours stationary as focus moves. The detail
	// pane uses catalogue text, not unreviewed cover art or network images.
	divider := 24 + (c.width-48)*2/5
	fill(frame, image.Rect(divider, 120, divider+1, 381), palette.panel)
	start, end := model.Window(5)
	for index := start; index < end; index++ {
		top := 120 + (index-start)*52
		row := image.Rect(24, top, divider-16, top+48)
		shade := palette.muted
		if index == model.Selected {
			fill(frame, row, palette.accent)
			fill(frame, row.Inset(2), palette.panel)
			shade = palette.text
		}
		item := model.Items[index]
		c.text(frame, item.Package.Name, top+2, row.Min.X+12, row.Dx()-24, shade, false)
		c.text(frame, categoryLabel(item)+" / "+installState(item), top+25, row.Min.X+12, row.Dx()-24, palette.muted, false)
	}
	item := model.Items[model.Selected]
	left, right := divider+24, c.width-24
	c.prose(frame, item.Package.Name, image.Rect(left, 120, right, 183), palette.text, true)
	c.text(frame, categoryLabel(item)+" / v"+item.Package.Version, 188, left, right-left, palette.accent, false)
	// Test evidence and installation state are independent: an available app
	// is not necessarily tested on this device.
	c.text(frame, strings.ToUpper(item.Package.Review.Status), 218, left, right-left, palette.warning, false)
	c.text(frame, trustLabel(item), 244, left, right-left, palette.warning, false)
	c.text(frame, installState(item), 270, left, right-left, statusColor(item), false)
	c.prose(frame, item.Package.Summary, image.Rect(left, 306, right, 381), palette.text, false)
}

func categoryLabel(item appstore.Item) string {
	if storeui.CategoryGames.Match(item) {
		return "GAME"
	}
	return "APP"
}
