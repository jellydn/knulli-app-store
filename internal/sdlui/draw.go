//go:build sdl

package sdlui

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	imagedraw "image/draw"
	"strings"

	"github.com/jellydn/knulli-app-store/internal/appstore"
	storeinput "github.com/jellydn/knulli-app-store/internal/input"
	storeui "github.com/jellydn/knulli-app-store/internal/ui"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The released picker uses this same licensed DejaVu face at 18/26 logical px.
// Embedding it keeps the software renderer independent of firmware font paths.
//
//go:embed assets/DejaVuSans.ttf
var consoleFont []byte

var palette = struct{ background, panel, text, muted, accent, warning, error color.RGBA }{
	background: color.RGBA{12, 22, 30, 255}, panel: color.RGBA{22, 36, 47, 255},
	text: color.RGBA{229, 239, 242, 255}, muted: color.RGBA{156, 178, 186, 255},
	accent: color.RGBA{70, 220, 174, 255}, warning: color.RGBA{244, 180, 64, 255}, error: color.RGBA{239, 94, 94, 255},
}

// One responsive canvas follows the picker's 480-high logical coordinate space:
// width = max(640, outputWidth * 480 / outputHeight). No per-device layouts.
type consoleCanvas struct {
	width       int
	body, title font.Face
}

func newConsoleCanvas(width int) (*consoleCanvas, error) {
	typeface, err := opentype.Parse(consoleFont)
	if err != nil {
		return nil, fmt.Errorf("load console font: %w", err)
	}
	body, err := opentype.NewFace(typeface, &opentype.FaceOptions{Size: 18, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	title, err := opentype.NewFace(typeface, &opentype.FaceOptions{Size: 26, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		body.Close()
		return nil, err
	}
	return &consoleCanvas{width: width, body: body, title: title}, nil
}

func (c *consoleCanvas) close() { _ = c.body.Close(); _ = c.title.Close() }

func emptyCatalogueMessage(total int) string {
	if total > 0 {
		return "NO PACKAGES IN THIS FILTER"
	}
	return "CATALOGUE UNAVAILABLE"
}

func (c *consoleCanvas) label(frame *image.RGBA, value string, y, left, width int, shade color.Color, heading bool) {
	face := c.body
	if heading {
		face = c.title
	}
	value = fitText(face, strings.Join(strings.Fields(value), " "), width)
	x := left + (width-font.MeasureString(face, value).Ceil())/2
	drawer := font.Drawer{Dst: frame, Src: image.NewUniform(shade), Face: face, Dot: fixed.P(x, y+face.Metrics().Ascent.Ceil())}
	drawer.DrawString(value)
}

func fitText(face font.Face, value string, width int) string {
	if font.MeasureString(face, value).Ceil() <= width {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if font.MeasureString(face, string(runes)+"…").Ceil() <= width {
			return string(runes) + "…"
		}
	}
	return ""
}

func (c *consoleCanvas) paragraph(frame *image.RGBA, value string, top, lines int, shade color.Color) {
	words := strings.Fields(value)
	for row := 0; row < lines && len(words) > 0; row++ {
		line := words[0]
		words = words[1:]
		for len(words) > 0 && font.MeasureString(c.body, line+" "+words[0]).Ceil() <= c.width-64 {
			line += " " + words[0]
			words = words[1:]
		}
		if row == lines-1 && len(words) > 0 {
			line += " " + strings.Join(words, " ")
		}
		c.label(frame, line, top+row*23, 32, c.width-64, shade, false)
	}
}

func (c *consoleCanvas) draw(model *storeui.Model, platformName string, controls *storeinput.Session) *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, c.width, canvasHeight))
	fill(frame, frame.Bounds(), palette.background)
	c.label(frame, "KNULLI APP STORE", 16, 24, c.width-48, palette.text, true)
	badge := "EXPERIMENTAL / " + model.Tab().Label()
	if model.Focus == storeui.Actions {
		badge = "PACKAGE / ACTIONS"
	}
	if model.Focus == storeui.Health {
		badge = "PACKAGE / HEALTH"
	}
	if model.Focus == storeui.Confirm {
		badge = "REVIEW / CONFIRM"
	}
	if model.Focus == storeui.ForceConfirm {
		badge = "RECOVERY / STEP 2 OF 2"
	}
	if model.Busy {
		badge = "WORKING / PLEASE WAIT"
	}
	if controls != nil && !model.Busy && model.Focus != storeui.Confirm && model.Focus != storeui.ForceConfirm {
		badge += " / " + controls.QuitChord() + ": QUIT"
	}
	c.label(frame, badge, 52, 24, c.width-48, palette.accent, false)
	if model.Focus == storeui.Browse && !model.Busy && (controls == nil || controls.Mode == storeinput.Normal) {
		fill(frame, image.Rect(0, 49, c.width, 83), palette.background)
		for index, tab := range storeui.TabOrder {
			left := c.width/2 - 180 + index*120
			shade := palette.muted
			if tab == model.Tab() {
				shade = palette.accent
				fill(frame, image.Rect(left+22, 79, left+98, 81), shade)
			}
			c.label(frame, tab.Label(), 52, left, 120, shade, false)
		}
	}
	switch {
	case controls != nil && controls.Mode != storeinput.Normal:
		c.label(frame, "CONTROLLER UNAVAILABLE", 154, 24, c.width-48, palette.warning, true)
		c.paragraph(frame, "Reconnect a controller to continue. "+platformName, 212, 3, palette.muted)
	case len(model.Items) == 0:
		c.label(frame, emptyCatalogueMessage(model.Total()), 164, 24, c.width-48, palette.text, true)
		c.paragraph(frame, "L1 / R1 change catalogue filters.", 219, 2, palette.muted)
	case model.Focus == storeui.Browse || model.Busy:
		c.browse(frame, model)
	case model.Focus == storeui.Confirm || model.Focus == storeui.ForceConfirm:
		c.confirmation(frame, model)
	default:
		c.details(frame, model)
	}
	// Errors persist, unlike completion notices. Neither can cover the footer.
	status, shade := model.Message, palette.accent
	if model.Error != "" {
		status, shade = model.Error, palette.error
	}
	if notices := model.LiveToasts(); status == "" && len(notices) > 0 {
		status = notices[0]
	}
	if status != "" {
		fill(frame, image.Rect(16, 385, c.width-16, 427), palette.panel)
		c.paragraph(frame, status, 387, 1, shade)
	} else if model.Focus == storeui.Browse && controls != nil {
		position := ""
		if len(model.Items) > 0 {
			position = fmt.Sprintf("%d / %d  ·  ", model.Selected+1, len(model.Items))
		}
		c.label(frame, position+controls.QuitChord()+": QUIT", 397, 24, c.width-48, palette.muted, false)
	}
	c.footer(frame, model, controls)
	return frame
}

func (c *consoleCanvas) browse(frame *image.RGBA, model *storeui.Model) {
	// A single responsive list keeps neighbouring packages visible. Its window
	// follows model selection; drawing never changes focus or action policy.
	left := max(32, (c.width-720)/2)
	width := c.width - 2*left
	start, end := model.Window(7)
	top := 86
	for index := start; index < end; index++ {
		item := model.Items[index]
		if index != model.Selected {
			c.text(frame, item.Package.Name, top+3, left+38, width-66, palette.muted, false)
			fill(frame, image.Rect(left, top+29, left+width, top+30), palette.panel)
			top += 30
			continue
		}
		fill(frame, image.Rect(left, top, left+width, top+116), palette.panel)
		fill(frame, image.Rect(left, top, left+3, top+116), palette.accent)
		// Manifests have no reviewed artwork. Use a local mark, not invented art.
		drawBrandMark(frame, left+20, top+26, 64)
		textLeft, textWidth := left+104, width-120
		c.text(frame, item.Package.Name, top+7, textLeft, textWidth, palette.text, true)
		c.text(frame, "v"+item.Package.Version, top+39, textLeft, textWidth, palette.accent, false)
		c.text(frame, item.Package.Summary, top+63, textLeft, textWidth, palette.muted, false)
		c.text(frame, trustLabel(item)+" / "+installState(item), top+87, textLeft, textWidth, statusColor(item), false)
		top += 116
	}
}

func (c *consoleCanvas) text(frame *image.RGBA, value string, y, left, width int, shade color.Color, heading bool) {
	face := c.body
	if heading {
		face = c.title
	}
	drawer := font.Drawer{Dst: frame, Src: image.NewUniform(shade), Face: face, Dot: fixed.P(left, y+face.Metrics().Ascent.Ceil())}
	drawer.DrawString(fitText(face, strings.Join(strings.Fields(value), " "), width))
}

func (c *consoleCanvas) confirmation(frame *image.RGBA, model *storeui.Model) {
	item := model.Items[model.Selected]
	action := item.Actions[model.Action]
	c.label(frame, item.Package.Name, 94, 24, c.width-48, palette.text, true)
	c.label(frame, "v"+item.Package.Version+" / "+trustLabel(item), 132, 24, c.width-48, palette.muted, false)
	heading := "Confirm " + actionLabel(action) + "?"
	notice := ""
	if item.Package.Install != nil {
		notice = item.Package.Install.Warning
	}
	if action == appstore.ForceReinstall {
		heading = "Step 1 of 2 / Review recovery"
		if model.Focus == storeui.ForceConfirm {
			heading = "Step 2 of 2 / Confirm reinstall"
		}
		notice = item.RecoverySummary
	}
	c.label(frame, heading, 174, 24, c.width-48, palette.warning, true)
	c.paragraph(frame, notice, 220, 6, palette.text)
	c.label(frame, "Back cancels / Start never confirms", 386, 24, c.width-48, palette.muted, false)
}

func (c *consoleCanvas) details(frame *image.RGBA, model *storeui.Model) {
	item := model.Items[model.Selected]
	c.label(frame, item.Package.Name, 92, 24, c.width-48, palette.text, true)
	c.label(frame, trustLabel(item)+" / "+installState(item), 128, 24, c.width-48, statusColor(item), false)
	message := item.Verdict.Message()
	if message == "" {
		message = item.Compatibility
	}
	if model.Focus == storeui.Health {
		message = item.HealthReason + " Repair restores reviewed bytes and modes."
	}
	if item.RecoveryReason != "" {
		message = item.RecoveryReason + " " + item.RecoverySummary
	}
	c.paragraph(frame, message, 166, 4, palette.muted)
	if model.Focus == storeui.Health {
		c.label(frame, "Confirm reviews available recovery actions", 302, 24, c.width-48, palette.warning, false)
		return
	}
	if len(item.Actions) == 0 {
		c.label(frame, "Read only / review required", 306, 24, c.width-48, palette.warning, false)
		return
	}
	// A focused, centered row replaces the old split-pane catalogue. The model
	// owns action selection and restores it when confirmation is cancelled.
	fill(frame, image.Rect(32, 286, c.width-32, 330), palette.panel)
	c.label(frame, "‹  "+actionLabel(item.Actions[model.Action])+"  ›", 296, 48, c.width-96, palette.accent, false)
	c.label(frame, fmt.Sprintf("Action %d of %d / D-pad changes action", model.Action+1, len(item.Actions)), 343, 24, c.width-48, palette.muted, false)
}

func (c *consoleCanvas) footer(frame *image.RGBA, model *storeui.Model, controls *storeinput.Session) {
	fill(frame, image.Rect(0, 427, c.width, 480), palette.panel)
	if model.Busy {
		c.label(frame, "Verifying and applying / controls locked", 444, 24, c.width-48, palette.muted, false)
		return
	}
	hints := footerHints(model, controls)
	// Quitting is deliberately a chord, not the picker's single Start action.
	// Show it in the upper badge, leaving four short, evenly spaced cells here.
	visible := hints[:0:0]
	for _, hint := range hints {
		if hint.Action != storeinput.Exit {
			visible = append(visible, hint)
		}
	}
	if model.Focus == storeui.Browse && len(model.Items) > 0 && (controls == nil || controls.Mode == storeinput.Normal) {
		visible = append(visible, storeinput.Hint{Action: storeinput.PageUp, With: storeinput.PageDown, Verb: "Filters"})
	}
	if len(visible) == 0 {
		return
	}
	cell := c.width / len(visible)
	mapping := footerMapping(controls)
	for i, hint := range visible {
		button := storeinput.ButtonLabel(mapping[hint.Action])
		if hint.With != "" {
			button = "↑↓"
			if hint.Action == storeinput.PageUp {
				button = "L1 R1"
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
		if len(button) == 1 {
			// Physical face-button glyphs stay distinct from action text.
			groupWidth := 34 + font.MeasureString(c.body, hint.Verb).Ceil()
			left := i*cell + (cell-groupWidth)/2
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
			c.text(frame, hint.Verb, 442, left+34, cell-34, palette.text, false)
		} else {
			c.label(frame, button+" "+hint.Verb, 442, i*cell, cell, palette.text, false)
		}
	}
}

func trustLabel(item appstore.Item) string {
	if item.Package.ID == appstore.StoreID {
		return "STORE UPDATE"
	}
	if !item.Package.Installable() {
		return "CANDIDATE"
	}
	if !item.DeviceTested {
		return "UNTESTED"
	}
	if item.Package.Review.Status == "verified" {
		return "VERIFIED"
	}
	return "DEVICE TESTED"
}

func installState(item appstore.Item) string {
	if item.Verdict.HasReason(appstore.ReasonPlatform) {
		return "INCOMPATIBLE"
	}
	if item.Verdict.State == "" {
		return "BLOCKED"
	}
	return strings.ToUpper(string(item.Verdict.State))
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

func actionLabel(action appstore.Action) string {
	if action == appstore.Adopt {
		return "Manage existing"
	}
	if action == appstore.ForceReinstall {
		return "Force reinstall"
	}
	value := string(action)
	if value == "" {
		return "Review"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func drawBrandMark(frame *image.RGBA, x, y, size int) {
	gap := size / 8
	cell := (size - gap) / 2
	for i := 0; i < 4; i++ {
		left, top := x+(i%2)*(cell+gap), y+(i/2)*(cell+gap)
		r := image.Rect(left, top, left+cell, top+cell)
		fill(frame, r, palette.accent)
		if i > 0 {
			fill(frame, r.Inset(2), palette.background)
		}
	}
}

func fill(destination imagedraw.Image, rectangle image.Rectangle, shade color.Color) {
	imagedraw.Draw(destination, rectangle, image.NewUniform(shade), image.Point{}, imagedraw.Src)
}
