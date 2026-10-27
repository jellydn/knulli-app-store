//go:build sdl

package sdlui

import (
	"image"

	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func (c *consoleCanvas) browse(frame *image.RGBA, model *storeui.Model) {
	// Equal-height rows keep neighbours stationary as focus moves. The detail
	// pane uses catalogue text, not unreviewed cover art or network images.
	divider := 24 + (c.width-48)*2/5
	fill(frame, image.Rect(divider, 98, divider+1, 369), palette.panel)
	start, end := model.Window(7)
	for index := start; index < end; index++ {
		top := 99 + (index-start)*38
		row := image.Rect(24, top, divider-16, top+34)
		shade := palette.muted
		if index == model.Selected {
			fill(frame, row, palette.accent)
			fill(frame, row.Inset(2), palette.panel)
			shade = palette.text
		}
		c.text(frame, model.Items[index].Package.Name, top+5, row.Min.X+12, row.Dx()-24, shade, false)
	}
	item := model.Items[model.Selected]
	left, right := divider+24, c.width-24
	c.prose(frame, item.Package.Name, image.Rect(left, 99, right, 162), palette.text, true)
	c.text(frame, "v"+item.Package.Version, 167, left, right-left, palette.muted, false)
	// Test evidence and installation state are independent: an available app
	// is not necessarily tested on this device.
	c.text(frame, trustLabel(item), 200, left, right-left, palette.warning, false)
	c.text(frame, installState(item), 226, left, right-left, statusColor(item), false)
	c.prose(frame, item.Package.Summary, image.Rect(left, 265, right, 365), palette.text, false)
}
