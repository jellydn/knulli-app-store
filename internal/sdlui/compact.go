//go:build sdl

package sdlui

import (
	"fmt"
	"image"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
	"golang.org/x/image/font"
)

func (c *consoleCanvas) footerTop() int {
	return c.footerStart
}

func (c *consoleCanvas) compactScreen(frame *image.RGBA, model *storeui.Model, platformName string, controls *storeinput.Session) {
	content := frame.SubImage(image.Rect(0, 0, c.width, c.statusTop())).(*image.RGBA)
	left, width := 8, c.width-16
	top := c.header(content, model)
	statusTop := c.statusTop()
	switch {
	case controls != nil && controls.Mode != storeinput.Normal:
		c.text(content, "Controller unavailable", top, left, width, palette.warning, false)
		c.prose(content, "Reconnect a controller. "+platformName, image.Rect(left, top+c.step, c.width-left, statusTop), palette.muted, false)
	case len(model.Items) == 0:
		c.prose(content, emptyCatalogueMessage(model.Total()), image.Rect(left, top, c.width-left, statusTop), palette.text, false)
	case model.Busy:
		c.text(content, model.Items[model.Selected].Package.Name, top, left, width, palette.text, false)
		for i, line := range wrapReview(c.body, model.Message, width) {
			if y := top + (i+2)*c.step; y+c.step <= statusTop {
				c.text(content, line, y, left, width, palette.accent, false)
			}
		}
	case model.Focus != storeui.Browse:
		c.review(content, model, image.Rect(left, top, c.width-left, statusTop))
	default:
		item := model.Items[model.Selected]
		summaryTop := statusTop - 2*c.step - 4
		rowHeight := c.step + 2
		if summaryTop < top+rowHeight {
			summaryTop = statusTop
		}
		start, end := model.Window(max(1, (summaryTop-top)/rowHeight))
		for i := start; i < end; i++ {
			y := top + (i-start)*rowHeight
			row := image.Rect(left, y, c.width-left, y+rowHeight-2)
			shade := palette.muted
			if i == model.Selected {
				fill(content, row, palette.accent)
				fill(content, row.Inset(1), palette.panel)
				shade = palette.text
			}
			c.text(content, model.Items[i].Package.Name, y, left+4, width-8, shade, false)
		}
		c.text(content, trustLabel(item)+" / "+installState(item), summaryTop+2, left, width, statusColor(item), false)
		c.text(content, item.Package.Summary, summaryTop+c.step+2, left, width, palette.muted, false)
	}
	message, shade := model.Message, palette.accent
	if notices := model.LiveToasts(); message == "" && len(notices) > 0 {
		message = notices[0]
	}
	if model.Error != "" {
		message, shade = model.Error, palette.error
	}
	if message != "" {
		fill(frame, image.Rect(0, statusTop, c.width, c.footerTop()), palette.panel)
		c.text(frame, message, statusTop+2, left, width, shade, false)
	}
	c.footer(frame, model)
}

// wrapReview splits long words as well as prose, so every rune of a warning is
// reachable. Paragraph breaks remain visible instead of being flattened away.
func wrapReview(face font.Face, value string, width int) []string {
	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		line := ""
		for _, word := range strings.Fields(paragraph) {
			if line != "" && font.MeasureString(face, line+" "+word).Ceil() <= width {
				line += " " + word
				continue
			}
			if line != "" {
				lines = append(lines, line)
			}
			line = ""
			for _, r := range word {
				if line != "" && font.MeasureString(face, line+string(r)).Ceil() > width {
					lines = append(lines, line)
					line = ""
				}
				line += string(r)
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func (c *consoleCanvas) review(frame *image.RGBA, model *storeui.Model, bounds image.Rectangle) {
	item := model.Items[model.Selected]
	heading := ""
	text := item.Package.Name + " / v" + item.Package.Version + "\n" + trustLabel(item) + " / " + installState(item)
	if item.Package.Install != nil && item.Package.Install.Warning != "" {
		text += "\n" + item.Package.Install.Warning
	}
	for _, value := range []string{item.Verdict.Message(), item.Compatibility, item.HealthReason, item.RecoveryReason, item.RecoverySummary, model.Error} {
		if value != "" {
			text += "\n\n" + value
		}
	}
	confirming := model.Focus == storeui.Confirm || model.Focus == storeui.ForceConfirm
	if confirming && len(item.Actions) > 0 {
		heading = "Confirm " + actionLabel(item.Actions[model.Action]) + "?"
		if item.Actions[model.Action] == appstore.ForceReinstall {
			heading = "Recovery / step 1 of 2"
			if model.Focus == storeui.ForceConfirm {
				heading = "Recovery / step 2 of 2"
			}
		}
	}
	text += "\n\n" + item.Package.Summary
	if len(item.Actions) == 0 {
		text += "\nRead only / review required"
	}
	top := bounds.Min.Y
	// Below the target minimum, prefer a readable text window and page range
	// over optional review chrome. Confirmation headings still fit at 320×240.
	if heading != "" && bounds.Dy() >= 3*c.step+4 {
		c.text(frame, heading, top, bounds.Min.X, bounds.Dx(), palette.warning, false)
		top += c.step + 4
	}
	bottom := bounds.Max.Y - c.step
	if model.Focus == storeui.Actions && len(item.Actions) > 0 && bottom-top >= 2*c.step {
		bottom -= c.step
		c.text(frame, "←→ "+actionLabel(item.Actions[model.Action])+fmt.Sprintf(" (%d/%d)", model.Action+1, len(item.Actions)), bottom, bounds.Min.X, bounds.Dx(), palette.accent, false)
	}
	lines := wrapReview(c.body, text, bounds.Dx())
	start, end := model.ReviewWindow(len(lines), max(1, (bottom-top)/c.step))
	for i := start; i < end; i++ {
		c.text(frame, lines[i], top+(i-start)*c.step, bounds.Min.X, bounds.Dx(), palette.text, false)
	}
	c.text(frame, fmt.Sprintf("Text %d–%d / %d", start+1, end, len(lines)), bounds.Max.Y-c.step, bounds.Min.X, bounds.Dx(), palette.muted, false)
}
