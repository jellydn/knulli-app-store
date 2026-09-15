//go:build sdl

package sdlui

import (
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
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
	drawBrandMark(frame, 16, 9, 16)
	text(frame, 38, 25, palette.text, "KNULLI APP STORE")
	text(frame, 472, 25, palette.warning, "EXPERIMENTAL")
	if platformName != "" {
		text(frame, 16, 39, palette.muted, shorten(strings.ToUpper(platformName), 58))
	}
	if controls != nil && controls.Mode != storeinput.Normal {
		drawControllerScreen(frame, model, platformName, controls)
		return frame
	}
	fill(frame, image.Rect(16, 42, 230, 322), palette.panel)
	fill(frame, image.Rect(242, 42, 624, 322), palette.panel)
	if len(model.Items) == 0 {
		text(frame, 30, 75, palette.muted, "NO PACKAGES IN CATALOGUE")
	} else {
		drawList(frame, model)
		drawDetails(frame, model, controls)
	}
	drawStatus(frame, model)
	controlHelp := "CONFIRM --  BACK --  SETTINGS --"
	if controls != nil && controls.Connected {
		controlHelp = "CONFIRM " + storeinput.ButtonLabel(controls.Mapping[storeinput.Confirm]) + "  BACK " + storeinput.ButtonLabel(controls.Mapping[storeinput.Back]) + "  SETTINGS " + storeinput.ButtonLabel(controls.Mapping[storeinput.Diagnostics])
	}
	text(frame, 16, 345, palette.muted, shorten(controlHelp, 84))
	return frame
}

func drawControllerScreen(frame *image.RGBA, model *storeui.Model, platformName string, controls *storeinput.Session) {
	fill(frame, image.Rect(16, 50, 624, 344), palette.panel)
	text(frame, 32, 76, palette.accent, "CONTROLLER SETUP")
	text(frame, 32, 96, palette.text, "DEVICE  "+shorten(strings.ToUpper(platformName), 70))
	controllerName := strings.TrimSpace(controls.Identity.Name)
	if controllerName == "" {
		controllerName = "UNAVAILABLE"
	}
	text(frame, 32, 113, palette.text, "CONTROLLER  "+shorten(strings.ToUpper(controllerName), 60))
	text(frame, 32, 130, palette.muted, "IDENTITY  "+controllerIdentityStatus(controls))
	mappingSource := controls.Source
	if controls.Mode == storeinput.Setup {
		mappingSource = controls.AutoSource
	}
	text(frame, 32, 147, palette.muted, "SOURCE  "+shorten(strings.ToUpper(mappingSource), 64))
	text(frame, 32, 160, palette.muted, controllerSetupProgress(controls))
	switch controls.Mode {
	case storeinput.Blocked:
		text(frame, 32, 178, palette.warning, "NO SDL GAMECONTROLLER")
		text(frame, 32, 201, palette.text, "RECONNECT A CONTROLLER TO CONTINUE.")
		text(frame, 32, 231, palette.muted, "LOG  /USERDATA/SYSTEM/LOGS/KNULLI-APP-STORE.LOG")
		text(frame, 32, 250, palette.muted, "DIAGNOSTICS  /USERDATA/SYSTEM/KNULLI-APP-STORE/DIAGNOSTICS/")
		diagnosticStatus := model.Message
		statusColor := palette.accent
		if model.Error != "" {
			diagnosticStatus = model.Error
			statusColor = palette.error
		}
		if diagnosticStatus != "" {
			text(frame, 32, 279, statusColor, shorten(strings.ToUpper(diagnosticStatus), 78))
		}
		text(frame, 32, 316, palette.muted, "CONFIRM EXPORTS DIAGNOSTICS  USE THE SYSTEM EXIT CONTROL TO LEAVE")
	case storeinput.Setup:
		text(frame, 32, 174, palette.warning, "SETUP IS REQUIRED FOR SAFE CONTROLS")
		for index, item := range storeinput.SetupItems {
			prefix := "  "
			shade := palette.text
			if index == controls.SetupIndex {
				prefix = "> "
				shade = palette.accent
			}
			text(frame, 48, 205+index*25, shade, prefix+item)
		}
		text(frame, 344, 205, palette.muted, "DETECTED CONTROLS")
		drawMappingSummary(frame, storeinput.AutoMapping(), nil, 344, 225, 140, 18)
		text(frame, 32, 316, palette.muted, setupHelp(controls))
	case storeinput.Settings:
		text(frame, 32, 178, palette.warning, "CONTROLLER SETTINGS")
		for index, item := range storeinput.SettingsItems {
			prefix := "  "
			shade := palette.text
			if index == controls.SettingsIndex {
				prefix = "> "
				shade = palette.accent
			}
			text(frame, 48, 207+index*25, shade, prefix+item)
		}
		text(frame, 344, 207, palette.muted, "ACTIVE CONTROLS")
		drawMappingSummary(frame, controls.Mapping, nil, 344, 227, 140, 18)
		text(frame, 32, 316, palette.muted, setupHelp(controls))
	case storeinput.Calibrating:
		action, _ := controls.Calibration.Current()
		text(frame, 32, 176, palette.warning, "PRESS A BUTTON FOR")
		text(frame, 32, 205, palette.text, storeinput.Label(action))
		text(frame, 32, 226, palette.muted, fmt.Sprintf("ACTION %d OF %d", controls.Calibration.Index+1, len(storeinput.Actions)))
		text(frame, 32, 252, palette.muted, "A BUTTON CAN HAVE ONLY ONE ACTION")
		text(frame, 344, 176, palette.muted, "COMPLETED ACTIONS")
		drawMappingSummary(frame, controls.Calibration.Mapping, nil, 344, 196, 140, 18)
	case storeinput.Review:
		lastAction := storeinput.Actions[controls.Calibration.Index-1]
		text(frame, 32, 176, palette.warning, "DETECTED  "+storeinput.ButtonLabel(controls.Calibration.Mapping[lastAction]))
		text(frame, 32, 199, palette.text, "FOR  "+storeinput.Label(lastAction))
		text(frame, 32, 216, palette.muted, fmt.Sprintf("ACTION %d OF %d", controls.Calibration.Index, len(storeinput.Actions)))
		for index, item := range storeinput.ReviewItems {
			prefix := "  "
			shade := palette.text
			if index == controls.ReviewIndex {
				prefix = "> "
				shade = palette.accent
			}
			text(frame, 48, 230+index*21, shade, prefix+item)
		}
		text(frame, 344, 176, palette.muted, "CURRENT ASSIGNMENTS")
		drawMappingSummary(frame, controls.Calibration.Mapping, nil, 344, 196, 140, 18)
	case storeinput.Preview:
		text(frame, 32, 176, palette.warning, "TEST ALL ACTIONS BEFORE SAVE")
		drawMappingSummary(frame, controls.Calibration.Mapping, controls.Calibration.Tested, 48, 205, 280, 36)
		text(frame, 32, 316, palette.muted, "PRESS EACH SHOWN CONTROL")
	}
	if controls.ValidationError != "" {
		text(frame, 32, 332, palette.error, shorten(strings.ToUpper(controls.ValidationError), 78))
	} else if controls.Message != "" {
		text(frame, 32, 332, palette.muted, shorten(strings.ToUpper(controls.Message), 78))
	}
}

func drawMappingSummary(frame *image.RGBA, mapping storeinput.Mapping, tested map[storeinput.Action]bool, x, y, columnWidth, maximumCharacters int) {
	for index, action := range storeinput.Actions {
		button, assigned := mapping[action]
		if !assigned {
			continue
		}
		mark := ""
		if tested != nil {
			mark = "[X] "
			if !tested[action] {
				mark = "[ ] "
			}
		}
		column := index / 4
		row := index % 4
		label := storeinput.Label(action)
		if action == storeinput.Diagnostics {
			label = "DETAILS"
		}
		text(frame, x+column*columnWidth, y+row*22, palette.text, shorten(mark+label+": "+storeinput.ButtonLabel(button), maximumCharacters))
	}
}

func controllerIdentityStatus(controls *storeinput.Session) string {
	guid := strings.TrimSpace(controls.Identity.GUID)
	if guid == "" || strings.Trim(guid, "0") == "" {
		if controls.Identity.Name == "" {
			return "MISSING - MAPPING CANNOT BE SAVED"
		}
		return "GUID UNAVAILABLE - DEVICE + NAME FALLBACK"
	}
	return "GUID " + shorten(strings.ToUpper(guid), 32)
}

func controllerSetupProgress(controls *storeinput.Session) string {
	completed := 0
	if controls.Mode == storeinput.Normal || controls.Mode == storeinput.Settings {
		completed = len(storeinput.Actions)
	} else if controls.Calibration != nil {
		completed = controls.Calibration.Index
		if controls.Mode == storeinput.Preview {
			completed = len(controls.Calibration.Tested)
		}
	}
	return fmt.Sprintf("PROGRESS  %d OF %d ACTIONS", completed, len(storeinput.Actions))
}

func setupHelp(controls *storeinput.Session) string {
	mapping := controls.Mapping
	if controls.FirstRun {
		mapping = storeinput.AutoMapping()
	}
	return storeinput.Label(storeinput.Confirm) + " (" + storeinput.ButtonLabel(mapping[storeinput.Confirm]) + ") SELECT  " + storeinput.Label(storeinput.Back) + " (" + storeinput.ButtonLabel(mapping[storeinput.Back]) + ") BACK  " + storeinput.Label(storeinput.Exit) + " (" + storeinput.ButtonLabel(mapping[storeinput.Exit]) + ") EXIT"
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
		name := strings.ToUpper(item.Package.Name)
		if item.Package.Version != "" {
			name += "  " + strings.ToUpper(item.Package.Version)
		}
		text(frame, 34, y+15, palette.text, shorten(name, 24))
		text(frame, 34, y+31, statusColor(item), shorten(rowTrustLabel(item)+"  |  "+installState(item), 26))
	}
}

func drawDetails(frame *image.RGBA, model *storeui.Model, controls *storeinput.Session) {
	item := model.Items[model.Selected]
	text(frame, 258, 66, palette.text, shorten(strings.ToUpper(item.Package.Name), 38))
	text(frame, 258, 84, palette.muted, "VERSION "+strings.ToUpper(item.Package.Version)+"  |  "+strings.ToUpper(item.Package.Type))
	next := drawBadge(frame, 258, 94, trustLabel(item), trustColor(item))
	drawBadge(frame, next+8, 94, installState(item), statusColor(item))
	y := 132
	if model.Error != "" {
		drawActions(frame, model, item, controls)
		return
	}
	if model.Focus == storeui.Browse {
		drawWrapped(frame, 258, y, palette.text, item.Package.Summary, 48, 2)
		if item.HealthReason != "" {
			text(frame, 258, 184, palette.error, "ISSUE - OPEN DETAILS FOR THE HEALTH CHECK")
		} else {
			text(frame, 258, 184, palette.muted, "CONFIRM: OPEN ACTIONS AND DETAILS")
		}
		drawActions(frame, model, item, controls)
		return
	}
	text(frame, 258, y, palette.muted, "COMPATIBILITY")
	y += 18
	y = drawWrapped(frame, 258, y, compatibilityColor(item), item.Compatibility, 48, 4)
	notice := item.HealthReason
	noticeColor := palette.error
	if notice == "" && item.PreExisting {
		notice = "Lists files; no reinstall. Records safe ownership and backs up unknown files."
		noticeColor = palette.warning
	}
	if notice == "" && item.Package.Install != nil {
		notice = item.Package.Install.Warning
		noticeColor = palette.warning
	}
	if notice == "" && len(item.Package.Review.Notes) > 0 {
		notice = item.Package.Review.Notes[0]
		noticeColor = palette.text
	}
	if notice != "" && y < 235 {
		text(frame, 258, y+3, palette.muted, "NOTICE")
		drawWrapped(frame, 258, y+21, noticeColor, notice, 48, 2)
	}
	drawActions(frame, model, item, controls)
}

func trustLabel(item appstore.Item) string {
	if item.DeviceTested && item.Package.Review.Status != "verified" {
		return "DEVICE TESTED"
	}
	switch item.Package.Review.Status {
	case "verified":
		return "VERIFIED"
	case "experimental":
		return "EXPERIMENTAL"
	case "installable":
		return "REVIEWED"
	default:
		return "CANDIDATE"
	}
}

func installState(item appstore.Item) string {
	if item.Installed {
		if !item.Healthy {
			return "ISSUE"
		}
		return "INSTALLED"
	}
	if item.PreExisting {
		return "EXTERNAL"
	}
	if item.Compatible && item.Package.Installable() {
		return "AVAILABLE"
	}
	return "BLOCKED"
}

func rowTrustLabel(item appstore.Item) string {
	if item.DeviceTested && item.Package.Review.Status != "verified" {
		return "TESTED"
	}
	return trustLabel(item)
}

func trustColor(item appstore.Item) color.Color {
	if item.Package.Review.Status == "verified" || item.DeviceTested {
		return palette.accent
	}
	return palette.warning
}

func statusColor(item appstore.Item) color.Color {
	if item.Installed && !item.Healthy {
		return palette.error
	}
	if item.Compatible || item.Installed {
		return palette.accent
	}
	return palette.warning
}

func compatibilityColor(item appstore.Item) color.Color {
	if item.Compatible {
		return palette.accent
	}
	return palette.warning
}

// drawBrandMark paints the 2x2 catalogue mark used in docs/assets/icon.svg.
// The GUI is nearest-neighbour scaled, so the mark stays on whole pixels.
func drawBrandMark(frame *image.RGBA, x, y, size int) {
	if size < 8 {
		fill(frame, image.Rect(x, y, x+size, y+size), palette.accent)
		return
	}
	gap := size / 8
	if gap < 1 {
		gap = 1
	}
	cell := (size - gap) / 2
	if cell < 2 {
		fill(frame, image.Rect(x, y, x+size, y+size), palette.accent)
		return
	}
	used := cell*2 + gap
	offsetX := x + (size-used)/2
	offsetY := y + (size-used)/2
	cells := []image.Point{
		{X: offsetX, Y: offsetY},
		{X: offsetX + cell + gap, Y: offsetY},
		{X: offsetX, Y: offsetY + cell + gap},
		{X: offsetX + cell + gap, Y: offsetY + cell + gap},
	}
	for index, cellOrigin := range cells {
		outer := image.Rect(cellOrigin.X, cellOrigin.Y, cellOrigin.X+cell, cellOrigin.Y+cell)
		fill(frame, outer, palette.accent)
		if index == 0 || cell < 4 {
			continue
		}
		inner := image.Rect(cellOrigin.X+1, cellOrigin.Y+1, cellOrigin.X+cell-1, cellOrigin.Y+cell-1)
		fill(frame, inner, palette.background)
	}
}

func drawBadge(frame *image.RGBA, x, y int, label string, shade color.Color) int {
	label = " " + strings.ToUpper(label) + " "
	width := len(label)*7 + 6
	fill(frame, image.Rect(x, y, x+width, y+21), palette.selected)
	text(frame, x+3, y+15, shade, label)
	return x + width
}

func drawWrapped(frame *image.RGBA, x, y int, shade color.Color, value string, width, limit int) int {
	for index, line := range wrapText(strings.ToUpper(value), width) {
		if index == limit {
			break
		}
		text(frame, x, y, shade, line)
		y += 15
	}
	return y
}

func drawActions(frame *image.RGBA, model *storeui.Model, item appstore.Item, controls *storeinput.Session) {
	y := 278
	if len(item.Actions) == 0 {
		text(frame, 258, y, palette.warning, "READ ONLY - REVIEW REQUIRED")
		return
	}
	text(frame, 258, y-14, palette.muted, "ACTIONS")
	x := 258
	for index, action := range item.Actions {
		label := " " + strings.ToUpper(actionLabel(action)) + " "
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
		if item.Package.Install != nil && item.Package.Install.Warning != "" && (action == appstore.Install || action == appstore.Adopt) {
			fill(frame, image.Rect(250, 108, 616, 318), color.RGBA{R: 35, G: 46, B: 64, A: 255})
			text(frame, 266, 132, trustColor(item), "PACKAGE NOTICE  |  "+trustLabel(item))
			text(frame, 266, 151, palette.text, strings.ToUpper(actionLabel(action))+" "+shorten(strings.ToUpper(item.Package.Name), 24)+"?")
			warning := strings.ToUpper(item.Package.Install.Warning)
			for index, line := range wrapText(warning, 40) {
				if index == 2 {
					break
				}
				text(frame, 266, 171+index*15, palette.warning, line)
			}
			text(frame, 266, 220, palette.muted, confirmationHelp(controls))
		} else {
			fill(frame, image.Rect(286, 126, 580, 214), color.RGBA{R: 35, G: 46, B: 64, A: 255})
			text(frame, 304, 153, palette.warning, "CONFIRM PACKAGE CHANGE")
			text(frame, 304, 176, palette.text, strings.ToUpper(actionLabel(action))+" "+shorten(strings.ToUpper(item.Package.Name), 24)+"?")
			text(frame, 304, 199, palette.muted, confirmationHelp(controls))
		}
	}
}

func actionLabel(action appstore.Action) string {
	if action == appstore.Adopt {
		return "Manage existing"
	}
	return string(action)
}

func confirmationHelp(controls *storeinput.Session) string {
	mapping := storeinput.AutoMapping()
	if controls != nil {
		mapping = controls.Mapping
	}
	return "CONFIRM (" + storeinput.ButtonLabel(mapping[storeinput.Confirm]) + ") ACCEPT  BACK (" + storeinput.ButtonLabel(mapping[storeinput.Back]) + ") CANCEL"
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
