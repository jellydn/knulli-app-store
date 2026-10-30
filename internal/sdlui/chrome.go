//go:build sdl

package sdlui

import (
	"fmt"
	"image"

	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
	"golang.org/x/image/font"
)

func (c *consoleCanvas) header(frame *image.RGBA, model *storeui.Model, controls *storeinput.Session) {
	c.text(frame, "KNULLI APP STORE", 16, 24, c.width-48, palette.text, true)
	fill(frame, image.Rect(24, 84, c.width-24, 85), palette.panel)
	context := "Library"
	switch model.Focus {
	case storeui.Actions:
		context = "Package actions"
	case storeui.Health:
		context = "Package health"
	case storeui.Confirm:
		context = "Review before changes"
	case storeui.ForceConfirm:
		context = "Recovery / step 2 of 2"
	}
	if model.Busy {
		context = "Working / please wait"
	}
	if model.Focus != storeui.Browse || model.Busy || (controls != nil && controls.Mode != storeinput.Normal) {
		c.text(frame, context, 56, 24, c.width-48, palette.accent, false)
		return
	}
	for index, tab := range storeui.TabOrder {
		left := 24 + index*120
		shade := palette.muted
		if tab == model.Tab() {
			shade = palette.accent
			fill(frame, image.Rect(left, 80, left+108, 83), shade)
		}
		c.label(frame, tab.Label(), 53, left, 108, shade, false)
	}
}

func (c *consoleCanvas) status(frame *image.RGBA, model *storeui.Model, controls *storeinput.Session) {
	// Errors persist; notices expire. Neither may cover the controls below.
	status, shade := model.Message, palette.accent
	if model.Error != "" {
		status, shade = model.Error, palette.error
	}
	if notices := model.LiveToasts(); status == "" && len(notices) > 0 {
		status = notices[0]
	}
	if status != "" {
		fill(frame, image.Rect(16, 385, c.width-16, 427), palette.panel)
		c.text(frame, status, 394, 24, c.width-48, shade, false)
		return
	}
	if model.Focus == storeui.Browse && len(model.Items) > 0 {
		c.text(frame, fmt.Sprintf("%d / %d packages", model.Selected+1, len(model.Items)), 397, 24, c.width/2-24, palette.muted, false)
	}
	if controls != nil && !model.Busy && model.Focus != storeui.Confirm && model.Focus != storeui.ForceConfirm {
		c.label(frame, controls.QuitChord()+": Quit", 397, c.width/2, c.width/2-24, palette.muted, false)
	}
}

func (c *consoleCanvas) footer(frame *image.RGBA, model *storeui.Model, controls *storeinput.Session) {
	fill(frame, image.Rect(0, 427, c.width, 480), palette.panel)
	if model.Busy {
		c.label(frame, "Please wait / controls locked", 444, 24, c.width-48, palette.muted, false)
		return
	}
	hints := footerHints(model, controls)
	// Quit stays above the footer, leaving four short cells for browsing.
	visible := hints[:0:0]
	for _, hint := range hints {
		if hint.Action != storeinput.Exit {
			visible = append(visible, hint)
		}
	}
	if model.Focus == storeui.Browse && (controls == nil || controls.Mode == storeinput.Normal) {
		visible = append(visible, storeinput.Hint{Action: storeinput.PageUp, With: storeinput.PageDown, Verb: "Filters"})
	}
	if len(visible) == 0 {
		return
	}
	mapping := footerMapping(controls)
	buttons, widths := make([]string, len(visible)), make([]int, len(visible))
	total := 0
	for i, hint := range visible {
		button := storeinput.ButtonLabel(mapping[hint.Action])
		if hint.With != "" {
			button = "↑↓"
			if hint.Action == storeinput.PageUp {
				button = "L1 R1"
				if controls != nil && controls.Source == storeinput.KeyboardSourceName {
					button = "PgUp/PgDn"
				}
			}
		}
		if hint.Action == storeinput.Confirm && model.Focus == storeui.Browse && len(model.Items) > 0 {
			item := model.Items[model.Selected]
			if len(item.Actions) == 0 {
				hint.Verb = "Review"
			} else {
				hint.Verb = actionLabel(item.Actions[0])
			}
			if item.HealthReason != "" {
				hint.Verb = "Health"
			}
		}
		visible[i], buttons[i] = hint, button
		widths[i] = font.MeasureString(c.body, button+" "+hint.Verb).Ceil()
		if len(button) == 1 {
			widths[i] = 34 + font.MeasureString(c.body, hint.Verb).Ceil()
		}
		total += widths[i]
	}
	// Measure groups before spacing them: keyboard names and Manage existing
	// must not be truncated by equal-width footer cells at 640x480.
	gap := max(0, (c.width-48-total)/(len(visible)+1))
	left := 24 + gap
	for i, hint := range visible {
		button := buttons[i]
		if len(button) == 1 {
			shade := palette.text
			if hint.Action == storeinput.Confirm {
				shade = palette.accent
			}
			for y := -13; y <= 13; y++ {
				for x := -13; x <= 13; x++ {
					if distance := x*x + y*y; distance <= 169 && distance >= 132 {
						frame.Set(left+13+x, 453+y, shade)
					}
				}
			}
			c.label(frame, button, 442, left, 26, shade, false)
			c.text(frame, hint.Verb, 442, left+34, widths[i]-34, palette.text, false)
		} else {
			c.text(frame, button+" "+hint.Verb, 442, left, widths[i], palette.text, false)
		}
		left += widths[i] + gap
	}
}
