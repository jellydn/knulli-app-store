//go:build sdl

package sdlui

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
	"golang.org/x/image/font"
)

func (c *consoleCanvas) header(frame *image.RGBA, model *storeui.Model, controls *storeinput.Session) {
	c.text(frame, "KNULLI APP STORE", 16, 24, c.width-48, palette.text, true)
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
		fill(frame, image.Rect(24, 84, c.width-24, 85), palette.panel)
		c.text(frame, context, 56, 24, c.width-48, palette.accent, false)
		return
	}
	fill(frame, image.Rect(24, 109, c.width-24, 110), palette.panel)
	cell := (c.width - 152) / len(storeui.CategoryOrder)
	c.text(frame, "←→ Type", 48, 24, 100, palette.muted, false)
	for index, category := range storeui.CategoryOrder {
		left := 128 + index*cell
		shade := palette.muted
		if category == model.Tabs.Category() {
			shade = palette.accent
			fill(frame, image.Rect(left+4, 73, left+cell-4, 76), shade)
		}
		c.label(frame, category.Label(), 48, left, cell, shade, false)
	}
	statusKeys := "L1/R1"
	if controls != nil && controls.Source == storeinput.KeyboardSourceName {
		statusKeys = "PgUp/Dn"
	}
	c.text(frame, statusKeys, 80, 24, 100, palette.muted, false)
	for index, tab := range storeui.TabOrder {
		left := 128 + index*cell
		shade := palette.muted
		if tab == model.Tab() {
			shade = palette.accent
			fill(frame, image.Rect(left+4, 105, left+cell-4, 108), shade)
		}
		label := tab.Label()
		if tab == storeui.TabAll {
			label = "ALL STATES"
		}
		c.label(frame, label, 80, left, cell, shade, false)
	}
}

func (c *consoleCanvas) status(frame *image.RGBA, model *storeui.Model) {
	// Errors persist; notices expire. Neither may cover the controls below.
	status, shade := model.Message, palette.accent
	if model.Error != "" {
		status, shade = model.Error, palette.error
	}
	if notices := model.LiveToasts(); status == "" && len(notices) > 0 {
		status = notices[0]
	}
	if status != "" {
		fill(frame, image.Rect(16, c.statusTop(), c.width-16, c.footerTop()), palette.panel)
		c.text(frame, status, c.statusTop()+8, 24, c.width-48, shade, false)
		return
	}
	if model.Focus == storeui.Browse && len(model.Items) > 0 {
		c.text(frame, fmt.Sprintf("%s / %d of %d", model.Tab().Label(), model.Selected+1, len(model.Items)), c.statusTop()+8, 24, c.width-48, palette.muted, false)
	}
}

type footerGroup struct {
	key, label string
	bounds     image.Rectangle
	keyWidth   int
}

// Measure the same key chips and labels that will be painted. Unlike the
// toolkit's fixed left/right groups, wrap every whole group instead of dropping
// actions or letting groups consume each other's space.
func (c *consoleCanvas) layoutFooter(model *storeui.Model, controls *storeinput.Session) {
	mapping := footerMapping(controls)
	confirm, back := storeinput.ButtonLabel(mapping[storeinput.Confirm]), storeinput.ButtonLabel(mapping[storeinput.Back])
	more, pages, chord := storeinput.ButtonLabel(mapping[storeinput.Diagnostics]), "L1/R1", storeinput.QuitChord(mapping, "")
	if controls != nil {
		chord = controls.QuitChord()
		if controls.Source == storeinput.KeyboardSourceName {
			confirm, back, pages = "Enter", "Esc", "PgUp/Dn"
		}
	}
	chord = strings.ReplaceAll(chord, " ", "")
	c.groups = nil
	add := func(key, label string) { c.groups = append(c.groups, footerGroup{key: key, label: label}) }
	switch {
	case model.Busy:
	case controls != nil && controls.Mode != storeinput.Normal:
		add(confirm, "Diagnostics")
		add(chord, "Quit")
	case len(model.Items) == 0:
		add("←→", "Type")
		add(pages, "Status")
		add(more, "Diagnostics")
		add(chord, "Quit")
	case model.Focus == storeui.Browse:
		verb := "Review"
		item := model.Items[model.Selected]
		if len(item.Actions) > 0 {
			verb = actionLabel(item.Actions[0])
		}
		if item.HealthReason != "" {
			verb = "Health"
		}
		add("↑↓", "Browse")
		add("←→", "Type")
		add(confirm, verb)
		add(more, "More")
		add(pages, "Status")
		add(chord, "Quit")
	case model.Focus == storeui.Confirm || model.Focus == storeui.ForceConfirm:
		add(confirm, "Confirm")
		add(back, "Back")
		add(pages, "Read")
	default:
		if model.Focus == storeui.Actions && len(model.Items[model.Selected].Actions) > 1 {
			add("←→", "Action")
		}
		add(confirm, "Review")
		add(back, "Back")
		add(pages, "Read")
		add(more, "Diagnostics")
		add(chord, "Quit")
	}
	margin, gap := 8, 8
	if !c.compact {
		margin = 16
	}
	height := c.body.Metrics().Height.Ceil() + 10
	x, y := margin, 8
	for i := range c.groups {
		group := &c.groups[i]
		keyInk, keyAdvance := font.BoundString(c.body, group.key)
		labelInk, labelAdvance := font.BoundString(c.body, group.label)
		group.keyWidth = max(height-8, max(keyAdvance.Ceil(), keyInk.Max.X.Ceil())-min(0, keyInk.Min.X.Floor())+8)
		width := group.keyWidth + 12 + max(labelAdvance.Ceil(), labelInk.Max.X.Ceil()) - min(0, labelInk.Min.X.Floor())
		if x > margin && x+width > c.width-margin {
			x, y = margin, y+height+4
		}
		group.bounds = image.Rect(x, y, x+width, y+height)
		x += width + gap
	}
	c.footerStart = c.height - (y + height + 8)
	for i := range c.groups {
		c.groups[i].bounds = c.groups[i].bounds.Add(image.Pt(0, c.footerStart))
	}
}

func (c *consoleCanvas) statusTop() int {
	if c.compact {
		return c.footerTop() - c.step - 6
	}
	return c.footerTop() - 42
}

func (c *consoleCanvas) footer(frame *image.RGBA, model *storeui.Model) {
	fill(frame, image.Rect(0, c.footerTop(), c.width, c.height), palette.panel)
	if model.Busy {
		left := 8
		if !c.compact {
			left = 16
		}
		c.text(frame, "Please wait / controls locked", c.footerTop()+8, left, c.width-2*left, palette.muted, false)
	}
	for _, group := range c.groups {
		chip := image.Rect(group.bounds.Min.X, group.bounds.Min.Y+4, group.bounds.Min.X+group.keyWidth, group.bounds.Max.Y-4)
		roundedHint(frame, chip, palette.background)
		c.label(frame, group.key, group.bounds.Min.Y+5, chip.Min.X, chip.Dx(), palette.accent, false)
		c.text(frame, group.label, group.bounds.Min.Y+5, chip.Max.X+6, group.bounds.Max.X-chip.Max.X-6, palette.text, false)
	}
}

func roundedHint(frame *image.RGBA, bounds image.Rectangle, shade color.Color) {
	radius := min(6, bounds.Dy()/2)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		inset := 0
		if edge := min(y-bounds.Min.Y, bounds.Max.Y-1-y); edge < radius {
			for inset < radius && (radius-inset)*(radius-inset)+(radius-edge)*(radius-edge) > radius*radius {
				inset++
			}
		}
		fill(frame, image.Rect(bounds.Min.X+inset, y, bounds.Max.X-inset, y+1), shade)
	}
}
