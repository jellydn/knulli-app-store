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

const experimentalLabel = "EXPERIMENTAL"

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
	text(frame, panelRight-len(experimentalLabel)*7, 25, palette.warning, experimentalLabel)
	if platformName != "" {
		text(frame, 16, 39, palette.muted, shorten(strings.ToUpper(platformName), 58))
	}
	if controls != nil && controls.Mode != storeinput.Normal {
		drawControllerScreen(frame, model, platformName, controls)
		drawFooter(frame, model, controls)
		return frame
	}
	fill(frame, image.Rect(panelLeft, panelTop, listPanelRight, panelBottom), palette.panel)
	fill(frame, image.Rect(detailPanelLeft, panelTop, panelRight, panelBottom), palette.panel)
	if len(model.Items) == 0 {
		// An empty catalogue and an empty tab are different answers, and the
		// list is the only place that can tell the user which one they are
		// looking at. The strip is painted either way, so the way out of an
		// empty tab is visible from inside it.
		drawTabs(frame, model)
		text(frame, listLeft+tabPillPaddingX, 88, palette.muted, emptyCatalogueMessage(model.Total(), len(model.Items)))
	} else {
		drawList(frame, model)
		drawDetails(frame, model)
	}
	drawStatus(frame, model)
	drawToast(frame, model)
	drawFooter(frame, model, controls)
	return frame
}

// drawFooter paints the one hint line every screen shares. The text comes from
// footerText, which reads the active mapping, so no screen writes a button
// name and no screen repeats the hint inside its panel.
func drawFooter(frame *image.RGBA, model *storeui.Model, controls *storeinput.Session) {
	if footer := footerText(model, controls); footer != "" {
		text(frame, panelLeft, footerBaseline, palette.muted, shorten(footer, footerCharacterLimit))
	}
}

func drawControllerScreen(frame *image.RGBA, model *storeui.Model, platformName string, controls *storeinput.Session) {
	fill(frame, image.Rect(panelLeft, panelTop, panelRight, panelBottom), palette.panel)
	text(frame, 32, 66, palette.accent, controllerScreenTitle(controls.Mode))
	text(frame, 32, 86, palette.text, "DEVICE  "+shorten(strings.ToUpper(platformName), 70))
	controllerName := strings.TrimSpace(controls.Identity.Name)
	if controllerName == "" {
		// The screen title already reports an unavailable controller, so this
		// line reports the missing identity instead of repeating the title.
		controllerName = "NONE DETECTED"
	}
	text(frame, 32, 103, palette.text, "CONTROLLER  "+shorten(strings.ToUpper(controllerName), 60))
	text(frame, 32, 120, palette.muted, "IDENTITY  "+controllerIdentityStatus(controls))
	mappingSource := controls.Source
	if controls.Mode == storeinput.Setup {
		mappingSource = controls.AutoSource
	}
	text(frame, 32, 137, palette.muted, "SOURCE  "+shorten(strings.ToUpper(mappingSource), 64))
	text(frame, 32, 150, palette.muted, controllerSetupProgress(controls))
	switch controls.Mode {
	case storeinput.Blocked:
		text(frame, 32, modeHeadingBaseline, palette.warning, "RECONNECT A CONTROLLER TO CONTINUE.")
		text(frame, 32, 207, palette.muted, "LOG  /USERDATA/SYSTEM/LOGS/KNULLI-APP-STORE.LOG")
		text(frame, 32, 232, palette.muted, "DIAGNOSTICS  /USERDATA/SYSTEM/KNULLI-APP-STORE/DIAGNOSTICS/")
		diagnosticStatus := model.Message
		statusColor := palette.accent
		if model.Error != "" {
			diagnosticStatus = model.Error
			statusColor = palette.error
		}
		if diagnosticStatus != "" {
			text(frame, 32, 259, statusColor, shorten(strings.ToUpper(diagnosticStatus), 78))
		}
	case storeinput.Setup:
		text(frame, 32, modeHeadingBaseline, palette.warning, "SETUP IS REQUIRED FOR SAFE CONTROLS")
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
		drawMappingSummary(frame, controls.DetectedMapping(), nil, 344, 225, 140, 20)
	case storeinput.Settings:
		text(frame, 32, modeHeadingBaseline, palette.warning, "MAPPING AND DIAGNOSTICS ARE MANAGED HERE")
		for index, item := range storeinput.SettingsItems {
			prefix := "  "
			shade := palette.text
			if index == controls.SettingsIndex {
				prefix = "> "
				shade = palette.accent
			}
			text(frame, 48, 205+index*25, shade, prefix+item)
		}
		text(frame, 344, 205, palette.muted, "ACTIVE CONTROLS")
		drawMappingSummary(frame, controls.Mapping, nil, 344, 225, 140, 20)
	case storeinput.Calibrating:
		action, _ := controls.Calibration.Current()
		text(frame, 32, modeHeadingBaseline, palette.warning, "ASSIGN A BUTTON TO")
		text(frame, 32, 205, palette.text, storeinput.ActionLabel(action))
		text(frame, 32, 226, palette.muted, fmt.Sprintf("ACTION %d OF %d", controls.Calibration.Index+1, len(controls.Calibration.Actions)))
		text(frame, 32, 252, palette.muted, "A BUTTON CAN HAVE ONLY ONE ACTION")
		text(frame, 344, modeHeadingBaseline, palette.muted, "COMPLETED ACTIONS")
		drawMappingSummary(frame, controls.Calibration.Mapping, nil, 344, 196, 140, 20)
	case storeinput.Review:
		if controls.Calibration.Index > 0 {
			lastAction := storeinput.Actions[controls.Calibration.Index-1]
			text(frame, 32, modeHeadingBaseline, palette.warning, "DETECTED  "+storeinput.ButtonLabel(controls.Calibration.Mapping[lastAction]))
			text(frame, 32, 199, palette.text, "ASSIGNED TO  "+storeinput.ActionLabel(lastAction))
		}
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
		text(frame, 344, modeHeadingBaseline, palette.muted, "CURRENT ASSIGNMENTS")
		drawMappingSummary(frame, controls.Calibration.Mapping, nil, 344, 196, 140, 20)
	case storeinput.Paging:
		text(frame, 32, modeHeadingBaseline, palette.warning, "PAGING IS OPTIONAL")
		text(frame, 32, 199, palette.muted, "PAGE UP AND PAGE DOWN MOVE A LONG LIST A SCREENFUL AT A TIME.")
		text(frame, 32, 214, palette.muted, "UP AND DOWN STILL MOVE ONE ROW, AND THE TABS STILL NARROW THE LIST.")
		for index, item := range storeinput.PagingItems {
			prefix := "  "
			shade := palette.text
			if index == controls.PagingIndex {
				prefix = "> "
				shade = palette.accent
			}
			text(frame, 48, 245+index*25, shade, prefix+item)
		}
	case storeinput.Preview:
		text(frame, 32, modeHeadingBaseline, palette.warning, "TEST EVERY ACTION THIS PAD HAS")
		drawMappingSummary(frame, controls.Calibration.Mapping, controls.Calibration.Tested, 48, 205, 280, 36)
		if untested := storeinput.UntestedLabels(controls.Calibration); untested != "" {
			// The preview is finished by the required actions, so this line says
			// what happens to an optional binding the user could not press.
			text(frame, 32, 308, palette.muted, shorten("NOT SAVED UNLESS TESTED  "+untested, 82))
		}
	}
	if controls.ValidationError != "" {
		text(frame, 32, 318, palette.error, shorten(strings.ToUpper(controls.ValidationError), 78))
	} else if controls.Message != "" && controls.Mode != storeinput.Blocked {
		// The blocked screen already states that no controller is available.
		text(frame, 32, 318, palette.muted, shorten(strings.ToUpper(controls.Message), 78))
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
		// The summary is two columns tall by however many the action set needs,
		// so adding actions grows the summary downwards instead of off the
		// panel's right edge.
		column := index % 2
		row := index / 2
		label := storeinput.ActionLabel(action)
		text(frame, x+column*columnWidth, y+row*22, palette.text, shorten(mark+label+": "+storeinput.ButtonLabel(button), maximumCharacters))
	}
}

// controllerScreenTitle names each controller screen from its mode, so a
// blocked or settings screen does not claim to be setting up a controller.
func controllerScreenTitle(mode storeinput.Mode) string {
	switch mode {
	case storeinput.Blocked:
		return "CONTROLLER UNAVAILABLE"
	case storeinput.Settings:
		return "CONTROLLER SETTINGS"
	default:
		return "CONTROLLER SETUP"
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

// controllerSetupProgress reports how far a setup has come. The count follows
// the screen: a calibration walks the plan it was given, which is shorter when
// the user skipped paging, while a preview only has to see the required
// actions, because an optional button this pad does not carry cannot be
// pressed.
func controllerSetupProgress(controls *storeinput.Session) string {
	total := len(storeinput.Required)
	completed := 0
	if controls.Mode == storeinput.Normal || controls.Mode == storeinput.Settings {
		completed = total
	}
	if controls.Calibration != nil {
		total = len(controls.Calibration.Actions)
		completed = controls.Calibration.Index
		if controls.Mode == storeinput.Preview {
			total = len(storeinput.Required)
			completed = controls.Calibration.RequiredTested()
		}
	}
	return fmt.Sprintf("PROGRESS  %d OF %d ACTIONS", completed, total)
}

// drawList paints the visible window of catalogue rows. The model owns which
// rows that window shows, so the list and the paging keys can never disagree
// about how far a screenful is.
func drawList(frame *image.RGBA, model *storeui.Model) {
	drawTabs(frame, model)
	start, end := model.Window(listRows)
	for index := start; index < end; index++ {
		item := model.Items[index]
		row := listRowRectangle(index - start)
		if index == model.Selected {
			drawRowSelection(frame, row)
		}
		name := strings.ToUpper(item.Package.Name)
		if item.Package.Version != "" {
			name += "  " + strings.ToUpper(item.Package.Version)
		}
		text(frame, row.Min.X+rowTextInset, listRowTitleBaseline(row), palette.text, shorten(name, rowTitleCharacters))
		text(frame, row.Min.X+rowTextInset, listRowDetailBaseline(row), statusColor(item), shorten(rowTrustLabel(item)+"  |  "+rowInstallState(item), rowDetailCharacters))
	}
	if paged(model) {
		// The range is worth showing exactly when the list is longer than the
		// window — the same condition the footer uses to offer paging — because
		// a list that fits would only repeat its own count. It sits under the
		// rows it describes, right-aligned, so the strip above them holds only
		// tabs.
		caption := listRangeCaption(start, end, len(model.Items))
		text(frame, listRangePosition(caption), listRangeBaseline, palette.muted, caption)
	}
}

// listRangeCaption names the window a paged list is showing.
func listRangeCaption(start, end, total int) string {
	return fmt.Sprintf("%d-%d OF %d", start+1, end, total)
}

// drawTabs paints the catalogue's view selector. The active tab carries the
// same accent wash and accent ring as a selected row, so "this is the view you
// are in" reads the same way wherever it appears, and the inactive labels stay
// visible rather than hidden behind a menu.
func drawTabs(frame *image.RGBA, model *storeui.Model) {
	labels := tabLabels()
	for index, rectangle := range tabPillRectangles(labels) {
		shade := palette.muted
		if storeui.TabOrder[index] == model.Tab() {
			fill(frame, rectangle, blend(palette.accent, palette.panel, selectionWash))
			ring(frame, rectangle, palette.accent)
			shade = palette.text
		}
		text(frame, rectangle.Min.X+tabPillPaddingX, tabStripBaseline, shade, labels[index])
	}
}

// drawToast paints the bottom-anchored notice bar. It is drawn over the panel's
// foot so it never reflows the screen it lands on, and it keeps away from the
// footer, because the footer is the one line a screen must not lose: a notice
// is transient, an instruction is not.
func drawToast(frame *image.RGBA, model *storeui.Model) {
	notices := model.LiveToasts()
	if len(notices) == 0 {
		return
	}
	bar := toastRectangle()
	fill(frame, bar, blend(palette.accent, palette.panel, toastWash))
	ring(frame, bar, palette.accent)
	// Oldest first, capped at the bar's height: the queue holds one notice in
	// practice, since operations are serialised, and a burst shows its first
	// lines rather than overrunning the panel.
	lines := 0
	for _, notice := range notices {
		for _, line := range wrapText(strings.ToUpper(notice), toastCharacterLimit) {
			if lines == toastMaxLines {
				return
			}
			text(frame, bar.Min.X+toastInsetX, toastFirstLine+lines*toastLineHeight, palette.text, line)
			lines++
		}
	}
}

// drawRowSelection marks the selected row: a translucent accent wash, so the
// row's own text stays legible, inside a solid accent ring, so the selection is
// unmistakable on a small panel where a wash alone is easy to miss.
func drawRowSelection(frame *image.RGBA, row image.Rectangle) {
	fill(frame, row, blend(palette.accent, palette.panel, selectionWash))
	ring(frame, row, palette.accent)
}

// selectionWash is how much accent a selected row carries. The rest of the row
// stays panel, so the wash reads as a tint rather than as a filled button.
const selectionWash = 0.22

// toastWash is the notice bar's tint. It is weaker than a selection, so a
// notice never reads as something the user has selected or should act on.
const toastWash = 0.14

// ring strokes a one pixel outline just inside a rectangle, so the outline
// never grows the row, covers the row beside it, or touches the next one.
func ring(frame *image.RGBA, row image.Rectangle, shade color.Color) {
	fill(frame, image.Rect(row.Min.X, row.Min.Y, row.Max.X, row.Min.Y+1), shade)
	fill(frame, image.Rect(row.Min.X, row.Max.Y-1, row.Max.X, row.Max.Y), shade)
	fill(frame, image.Rect(row.Min.X, row.Min.Y, row.Min.X+1, row.Max.Y), shade)
	fill(frame, image.Rect(row.Max.X-1, row.Min.Y, row.Max.X, row.Max.Y), shade)
}

// blend mixes foreground into background. amount is how much of the foreground
// shows: 0 keeps the background, 1 replaces it.
func blend(foreground, background color.RGBA, amount float64) color.RGBA {
	mix := func(front, back uint8) uint8 {
		return uint8(float64(front)*amount + float64(back)*(1-amount))
	}
	return color.RGBA{R: mix(foreground.R, background.R), G: mix(foreground.G, background.G), B: mix(foreground.B, background.B), A: 255}
}

func drawDetails(frame *image.RGBA, model *storeui.Model) {
	item := model.Items[model.Selected]
	text(frame, panelInset, 66, palette.text, shorten(strings.ToUpper(item.Package.Name), 38))
	// A candidate declares no version, so the line lists only the fields it has.
	metadata := make([]string, 0, 2)
	if item.Package.Version != "" {
		metadata = append(metadata, "VERSION "+strings.ToUpper(item.Package.Version))
	}
	if item.Package.Type != "" {
		metadata = append(metadata, strings.ToUpper(item.Package.Type))
	}
	if len(metadata) > 0 {
		text(frame, panelInset, 84, palette.muted, strings.Join(metadata, "  |  "))
	}
	next := drawBadge(frame, panelInset, 94, trustLabel(item), trustColor(item))
	drawBadge(frame, next+8, 94, installState(item), statusColor(item))
	y := 132
	if model.Error != "" && item.RecoveryReason == "" {
		drawActions(frame, model, item)
		return
	}
	if model.Focus == storeui.Health {
		text(frame, panelInset, y, palette.error, "HEALTH CHECK")
		// The remediation guidance follows the reason, so a short reason does
		// not leave a gap and a long one does not overrun it.
		y = drawWrapped(frame, panelInset, y+20, palette.error, item.HealthReason, 48, 8)
		text(frame, panelInset, y+18, palette.muted, "REMEDIATION")
		text(frame, panelInset, y+36, palette.text, "REPAIR RESTORES REVIEWED BYTES AND MODES")
		return
	}
	if model.Focus == storeui.Browse {
		y = drawWrapped(frame, panelInset, y, palette.text, item.Package.Summary, 48, 2)
		if item.HealthReason != "" {
			text(frame, panelInset, y+18, palette.error, "ISSUE - CONFIRM OPENS THE HEALTH CHECK")
		}
		drawActions(frame, model, item)
		return
	}
	text(frame, panelInset, y, palette.muted, "COMPATIBILITY")
	y += 18
	y = drawWrapped(frame, panelInset, y, compatibilityColor(item), item.Verdict.Message(), 48, 4)
	notice := item.RecoveryReason
	noticeColor := palette.error
	if notice != "" && item.RecoverySummary != "" {
		notice += " " + item.RecoverySummary
	}
	if notice == "" {
		notice = item.HealthReason
	}
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
	// The status block owns its band, so the notice yields to it instead of
	// printing the same reason twice in the same place.
	if notice != "" && model.Error == "" && model.Message == "" && y < 205 {
		text(frame, panelInset, y+18, palette.muted, "NOTICE")
		drawWrapped(frame, panelInset, y+36, noticeColor, notice, 48, 2)
	}
	drawActions(frame, model, item)
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
	if item.Verdict.HasReason(appstore.ReasonPlatform) {
		return "INCOMPATIBLE"
	}
	switch item.Verdict.State {
	case appstore.StateIssue:
		return "ISSUE"
	case appstore.StateInstalled:
		return "INSTALLED"
	case appstore.StateExternal:
		return "EXTERNAL"
	case appstore.StateAvailable:
		return "AVAILABLE"
	case appstore.StateCandidate:
		return "CANDIDATE"
	case appstore.StateIncompatible:
		return "INCOMPATIBLE"
	default:
		return "BLOCKED"
	}
}

func rowTrustLabel(item appstore.Item) string {
	if item.DeviceTested && item.Package.Review.Status != "verified" {
		return "TESTED"
	}
	return trustLabel(item)
}

func rowInstallState(item appstore.Item) string {
	if installState(item) == "INCOMPATIBLE" {
		return "INCOMPAT."
	}
	return installState(item)
}

func trustColor(item appstore.Item) color.Color {
	if item.Package.Review.Status == "verified" || item.DeviceTested {
		return palette.accent
	}
	return palette.warning
}

func statusColor(item appstore.Item) color.Color {
	if item.Verdict.State == appstore.StateIssue || item.Verdict.HasReason(appstore.ReasonPlatform) {
		return palette.error
	}
	if item.Verdict.State == appstore.StateAvailable || item.Verdict.State == appstore.StateInstalled {
		return palette.accent
	}
	return palette.warning
}

func compatibilityColor(item appstore.Item) color.Color {
	if item.Verdict.Compatible(item.Package) {
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

func drawActions(frame *image.RGBA, model *storeui.Model, item appstore.Item) {
	y := actionRowBaseline
	if len(item.Actions) == 0 {
		text(frame, panelInset, y, palette.warning, "READ ONLY - REVIEW REQUIRED")
		return
	}
	text(frame, panelInset, actionLabelBaseline, palette.muted, "ACTIONS")
	x := panelInset
	for index, action := range item.Actions {
		labelText := actionLabel(action)
		if item.RetryAction == action {
			labelText = "Retry " + strings.ToLower(labelText)
		}
		if action == appstore.Adopt && item.RecoveryReason != "" {
			labelText = "Retry manage"
		}
		label := " " + strings.ToUpper(labelText) + " "
		background := palette.selected
		if model.Focus != storeui.Browse && index == model.Action {
			background = palette.accent
		}
		width := len(label)*7 + 8
		fill(frame, image.Rect(x, y, x+width, y+22), background)
		text(frame, x+4, y+15, palette.text, label)
		x += width + 8
	}
	if model.Focus == storeui.Confirm || model.Focus == storeui.ForceConfirm {
		action := item.Actions[model.Action]
		if action == appstore.ForceReinstall {
			fill(frame, image.Rect(250, 94, 616, 318), color.RGBA{R: 74, G: 31, B: 39, A: 255})
			step := "STEP 1 OF 2 - REVIEW RECOVERY"
			instruction := "CONFIRM AGAIN TO CONTINUE"
			if model.Focus == storeui.ForceConfirm {
				step = "STEP 2 OF 2 - CONFIRM REINSTALL"
				instruction = "CONFIRM TO REPLACE REVIEWED APP FILES"
			}
			text(frame, 266, 120, palette.error, step)
			text(frame, 266, 143, palette.text, instruction)
			drawWrapped(frame, 266, 165, palette.warning, item.RecoverySummary, 44, 4)
			text(frame, 266, 258, palette.muted, "BACKUP: /USERDATA/SYSTEM/KNULLI-APP-STORE/")
			text(frame, 266, 273, palette.muted, "RECOVERY-BACKUPS/<PACKAGE>/<TIMESTAMP>/")
			return
		}
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
		} else {
			fill(frame, image.Rect(286, 126, 580, 214), color.RGBA{R: 35, G: 46, B: 64, A: 255})
			text(frame, 304, 153, palette.warning, "CONFIRM PACKAGE CHANGE")
			text(frame, 304, 176, palette.text, strings.ToUpper(actionLabel(action))+" "+shorten(strings.ToUpper(item.Package.Name), 24)+"?")
		}
	}
}

func actionLabel(action appstore.Action) string {
	if action == appstore.Adopt {
		return "Manage existing"
	}
	if action == appstore.ForceReinstall {
		return "Force reinstall"
	}
	return string(action)
}

// drawStatus paints the one status block a screen can show. An error or a
// message replaces it in the same place with its own colour, so a status never
// moves and never covers the action buttons.
func drawStatus(frame *image.RGBA, model *storeui.Model) {
	if model.Focus == storeui.Health || model.Focus == storeui.Confirm || model.Focus == storeui.ForceConfirm {
		return
	}
	if model.Error != "" {
		drawStatusBlock(frame, color.RGBA{R: 74, G: 31, B: 39, A: 255}, palette.error, model.Error)
		return
	}
	if model.Message != "" {
		drawStatusBlock(frame, color.RGBA{R: 28, G: 55, B: 57, A: 255}, palette.accent, model.Message)
	}
}

func drawStatusBlock(frame *image.RGBA, background, shade color.Color, value string) {
	fill(frame, image.Rect(statusBoxLeft, statusBoxTop, statusBoxRight, statusBoxBottom), background)
	for index, line := range wrapText(strings.ToUpper(value), 50) {
		if index == statusLines {
			break
		}
		text(frame, panelInset, statusBaseline+index*15, shade, line)
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
