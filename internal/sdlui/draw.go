//go:build sdl
// +build sdl

package sdlui

import (
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	"github.com/jellydn/knulli-app-store/internal/manifest"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	canvasWidth  = 640
	canvasHeight = 360
)

var palette = struct {
	background color.RGBA
	panel      color.RGBA
	selected   color.RGBA
	text       color.RGBA
	muted      color.RGBA
	accent     color.RGBA
	warning    color.RGBA
	error      color.RGBA
}{
	background: color.RGBA{R: 14, G: 20, B: 31, A: 255},
	panel:      color.RGBA{R: 25, G: 34, B: 49, A: 255},
	selected:   color.RGBA{R: 43, G: 61, B: 83, A: 255},
	text:       color.RGBA{R: 239, G: 244, B: 251, A: 255},
	muted:      color.RGBA{R: 153, G: 169, B: 190, A: 255},
	accent:     color.RGBA{R: 63, G: 205, B: 168, A: 255},
	warning:    color.RGBA{R: 244, G: 180, B: 64, A: 255},
	error:      color.RGBA{R: 239, G: 94, B: 94, A: 255},
}

func draw(model *storeui.Model, platformName string, controls *storeinput.Session) *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, canvasWidth, canvasHeight))
	fill(frame, frame.Bounds(), palette.background)
	text(frame, 16, 25, palette.text, "KNULLI APP STORE")
	text(frame, 472, 25, palette.warning, "EXPERIMENTAL")
	fill(frame, image.Rect(16, 42, 230, 322), palette.panel)
	fill(frame, image.Rect(242, 42, 624, 322), palette.panel)
	if len(model.Items) == 0 {
		text(frame, 30, 75, palette.muted, "NO PACKAGES IN CATALOGUE")
	} else {
		drawList(frame, model)
		drawDetails(frame, model)
	}
	drawStatus(frame, model)
	controllerStatus := "NO CONTROLLER"
	controlHelp := "CONFIRM --  BACK --  SETTINGS --"
	if controls != nil && controls.Connected {
		controllerStatus = strings.ToUpper(shorten(controls.Source, 20))
		controlHelp = "CONFIRM " + storeinput.ButtonLabel(controls.Mapping[storeinput.Confirm]) + "  BACK " + storeinput.ButtonLabel(controls.Mapping[storeinput.Back]) + "  SETTINGS " + storeinput.ButtonLabel(controls.Mapping[storeinput.Diagnostics])
	}
	text(frame, 16, 345, palette.muted, shorten(controlHelp, 57))
	text(frame, 474, 345, palette.muted, shorten(controllerStatus, 20))
	if platformName != "" {
		text(frame, 16, 39, palette.muted, shorten(strings.ToUpper(platformName), 58))
	}
	drawControllerOverlay(frame, controls)
	return frame
}

func drawControllerOverlay(frame *image.RGBA, controls *storeinput.Session) {
	if controls == nil || controls.Mode == storeinput.Normal {
		return
	}
	fill(frame, image.Rect(242, 42, 624, 322), color.RGBA{R: 25, G: 34, B: 49, A: 255})
	text(frame, 258, 68, palette.accent, "CONTROLLER SETUP")
	text(frame, 258, 88, palette.muted, shorten(strings.ToUpper(controls.Identity.Name), 44))
	switch controls.Mode {
	case storeinput.Startup:
		text(frame, 258, 126, palette.text, "PRESS ANY CONTROLLER BUTTON TO CALIBRATE")
		text(frame, 258, 150, palette.muted, "OR WAIT 8 SECONDS TO USE KNULLI / SDL AUTO")
		text(frame, 258, 180, palette.warning, "NO KEYBOARD IS REQUIRED")
	case storeinput.Settings:
		text(frame, 258, 112, palette.text, "SETTINGS")
		for index, item := range storeinput.SettingsItems {
			prefix := "  "
			shade := palette.text
			if index == controls.SettingsIndex {
				prefix = "> "
				shade = palette.accent
			}
			text(frame, 270, 140+index*28, shade, prefix+item)
		}
	case storeinput.Calibrating:
		action, _ := controls.Calibration.Current()
		text(frame, 258, 116, palette.warning, "PRESS A PHYSICAL BUTTON FOR")
		text(frame, 258, 145, palette.text, storeinput.Label(action))
		text(frame, 258, 177, palette.muted, fmt.Sprintf("STEP %d OF %d", controls.Calibration.Index+1, len(storeinput.Actions)))
		text(frame, 258, 205, palette.muted, "CURRENT AUTO EXIT BUTTON CANCELS")
	case storeinput.Preview:
		text(frame, 258, 108, palette.warning, "TEST EVERY CONTROL BEFORE SAVE")
		for index, action := range storeinput.Actions {
			mark := "[ ]"
			if controls.Calibration.Tested[action] {
				mark = "[X]"
			}
			column := index / 4
			row := index % 4
			text(frame, 258+column*180, 138+row*28, palette.text, shorten(mark+" "+storeinput.Label(action)+" "+storeinput.ButtonLabel(controls.Calibration.Mapping[action]), 25))
		}
	}
	if controls.ValidationError != "" {
		text(frame, 258, 292, palette.error, shorten(strings.ToUpper(controls.ValidationError), 48))
	} else if controls.Message != "" {
		text(frame, 258, 292, palette.muted, shorten(strings.ToUpper(controls.Message), 48))
	}
}

func drawList(frame *image.RGBA, model *storeui.Model) {
	text(frame, 28, 62, palette.muted, "CATALOGUE")
	start := model.Selected - 2
	if start < 0 {
		start = 0
	}
	if start+5 > len(model.Items) {
		start = len(model.Items) - 5
		if start < 0 {
			start = 0
		}
	}
	end := start + 5
	if end > len(model.Items) {
		end = len(model.Items)
	}
	for index := start; index < end; index++ {
		item := model.Items[index]
		y := 72 + (index-start)*44
		if index == model.Selected {
			fill(frame, image.Rect(24, y, 222, y+38), palette.selected)
			fill(frame, image.Rect(24, y, 28, y+38), palette.accent)
		}
		text(frame, 34, y+15, palette.text, shorten(strings.ToUpper(item.Package.Name), 24))
		status := reviewLabel(item.Package)
		statusColor := palette.warning
		if item.Package.Review.Status == "verified" {
			statusColor = palette.accent
		}
		text(frame, 34, y+31, statusColor, status)
	}
}

func drawDetails(frame *image.RGBA, model *storeui.Model) {
	item := model.Items[model.Selected]
	text(frame, 258, 66, palette.text, shorten(strings.ToUpper(item.Package.Name), 38))
	text(frame, 258, 84, palette.muted, strings.ToUpper(item.Package.Type)+"  /  "+reviewLabel(item.Package))
	y := 101
	installed := "NOT INSTALLED"
	installedColor := palette.muted
	if item.PreExisting {
		installed = "EXISTING COPY / NOT MANAGED"
		installedColor = palette.warning
	}
	if item.Installed {
		installed = "INSTALLED " + strings.ToUpper(item.InstalledVersion)
		installedColor = palette.accent
		if !item.Healthy {
			installed += " / REPAIR NEEDED"
			installedColor = palette.error
		}
	}
	text(frame, 258, y, installedColor, installed)
	y += 18
	for _, line := range wrapText(strings.ToUpper(item.Package.Summary), 48) {
		text(frame, 258, y, palette.text, line)
		y += 15
	}
	y += 12
	text(frame, 258, y, palette.muted, "TRUST AND COMPATIBILITY")
	y += 18
	statusColor := palette.warning
	if item.Compatible {
		statusColor = palette.accent
	}
	for _, line := range wrapText(strings.ToUpper(item.Compatibility), 48) {
		text(frame, 258, y, statusColor, line)
		y += 15
	}
	if len(item.Package.Review.Notes) > 0 {
		y += 10
		text(frame, 258, y, palette.muted, "REVIEW NOTE")
		y += 18
		for _, line := range wrapText(strings.ToUpper(item.Package.Review.Notes[0]), 48) {
			text(frame, 258, y, palette.text, line)
			y += 15
			if y > 245 {
				break
			}
		}
	}
	drawActions(frame, model, item)
}

func reviewLabel(pkg manifest.Package) string {
	if pkg.Review.Approval != nil {
		return "APPROVED / " + strings.ToUpper(pkg.Review.Status)
	}
	return strings.ToUpper(pkg.Review.Status)
}

func drawActions(frame *image.RGBA, model *storeui.Model, item appstore.Item) {
	y := 278
	if len(item.Actions) == 0 {
		text(frame, 258, y, palette.warning, "READ ONLY - REVIEW REQUIRED")
		return
	}
	text(frame, 258, y-14, palette.muted, "AVAILABLE ACTIONS")
	x := 258
	for index, action := range item.Actions {
		label := " " + strings.ToUpper(string(action)) + " "
		background := palette.selected
		if model.Focus != storeui.Browse && index == model.Action {
			background = palette.accent
		}
		width := len(label)*7 + 8
		fill(frame, image.Rect(x, y, x+width, y+22), background)
		text(frame, x+4, y+15, palette.text, label)
		x += width + 8
	}
	if model.Focus == storeui.Confirm {
		action := item.Actions[model.Action]
		if item.Package.Experimental() && (action == appstore.Install || action == appstore.Adopt) {
			fill(frame, image.Rect(250, 108, 616, 318), color.RGBA{R: 35, G: 46, B: 64, A: 255})
			text(frame, 266, 132, palette.warning, "EXPERIMENTAL PACKAGE TEST")
			text(frame, 266, 151, palette.text, strings.ToUpper(string(action))+" "+shorten(strings.ToUpper(item.Package.Name), 24)+"?")
			warning := strings.ToUpper(item.Package.Install.Warning)
			for index, line := range wrapText(warning, 40) {
				if index == 2 {
					break
				}
				text(frame, 266, 171+index*15, palette.warning, line)
			}
			text(frame, 266, 220, palette.muted, "B CONFIRM   A CANCEL")
		} else {
			fill(frame, image.Rect(286, 126, 580, 214), color.RGBA{R: 35, G: 46, B: 64, A: 255})
			text(frame, 304, 153, palette.warning, "CONFIRM PACKAGE CHANGE")
			text(frame, 304, 176, palette.text, strings.ToUpper(string(action))+" "+shorten(strings.ToUpper(item.Package.Name), 24)+"?")
			text(frame, 304, 199, palette.muted, "B CONFIRM   A CANCEL")
		}
	}
}

func drawStatus(frame *image.RGBA, model *storeui.Model) {
	if model.Error != "" {
		fill(frame, image.Rect(242, 206, 624, 246), color.RGBA{R: 74, G: 31, B: 39, A: 255})
		for index, line := range wrapText(strings.ToUpper(model.Error), 50) {
			if index == 2 {
				break
			}
			text(frame, 252, 223+index*15, palette.error, line)
		}
		return
	}
	if model.Message != "" {
		message := strings.ToUpper(model.Message)
		if strings.HasPrefix(message, "DIAGNOSTICS SAVED TO ") {
			fill(frame, image.Rect(248, 244, 618, 318), color.RGBA{R: 28, G: 55, B: 57, A: 255})
			for index, line := range wrapText(message, 48) {
				if index == 4 {
					break
				}
				text(frame, 256, 261+index*15, palette.accent, line)
			}
			return
		}
		for index, line := range wrapText(message, 50) {
			if index == 1 {
				break
			}
			text(frame, 252, 312+index*15, palette.accent, line)
		}
	}
}

func text(destination imagedraw.Image, x, baseline int, colour color.Color, value string) {
	drawer := font.Drawer{Dst: destination, Src: image.NewUniform(colour), Face: basicfont.Face7x13, Dot: fixed.P(x, baseline)}
	drawer.DrawString(value)
}

func fill(destination imagedraw.Image, rectangle image.Rectangle, colour color.Color) {
	imagedraw.Draw(destination, rectangle, image.NewUniform(colour), image.Point{}, imagedraw.Src)
}

func wrapText(value string, width int) []string {
	words := strings.Fields(value)
	var lines []string
	current := ""
	for _, word := range words {
		if len(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			for len(word) > width {
				lines = append(lines, word[:width])
				word = word[width:]
			}
			current = word
			continue
		}
		if current == "" {
			current = word
			continue
		}
		if len(current)+1+len(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func shorten(value string, width int) string {
	if len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}
