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

// prose wraps inside a bounded region; only the last visible line ellipsizes.
// Package titles and summaries share this rule at every output aspect ratio.
func (c *consoleCanvas) prose(frame *image.RGBA, value string, bounds image.Rectangle, shade color.Color, heading bool) {
	face, step := c.body, 23
	if heading {
		face, step = c.title, 31
	}
	words := strings.Fields(value)
	for top := bounds.Min.Y; top+face.Metrics().Height.Ceil() <= bounds.Max.Y && len(words) > 0; top += step {
		line := words[0]
		words = words[1:]
		for len(words) > 0 && font.MeasureString(face, line+" "+words[0]).Ceil() <= bounds.Dx() {
			line += " " + words[0]
			words = words[1:]
		}
		if top+step+face.Metrics().Height.Ceil() > bounds.Max.Y && len(words) > 0 {
			line += " " + strings.Join(words, " ")
		}
		c.text(frame, line, top, bounds.Min.X, bounds.Dx(), shade, heading)
	}
}

func (c *consoleCanvas) draw(model *storeui.Model, platformName string, controls *storeinput.Session) *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, c.width, canvasHeight))
	fill(frame, frame.Bounds(), palette.background)
	c.header(frame, model, controls)
	switch {
	case controls != nil && controls.Mode != storeinput.Normal:
		c.text(frame, "Controller unavailable", 154, 32, c.width-64, palette.warning, true)
		c.prose(frame, "Reconnect a controller to continue. "+platformName, image.Rect(32, 212, c.width-32, 310), palette.muted, false)
	case len(model.Items) == 0:
		c.text(frame, emptyCatalogueMessage(model.Total()), 164, 32, c.width-64, palette.text, true)
		c.prose(frame, "Change filters to browse other packages. Use Y to export diagnostics if the catalogue is unavailable.", image.Rect(32, 219, c.width-32, 310), palette.muted, false)
	case model.Focus == storeui.Browse || model.Busy:
		c.browse(frame, model)
	case model.Focus == storeui.Confirm || model.Focus == storeui.ForceConfirm:
		c.confirmation(frame, model)
	default:
		c.details(frame, model)
	}
	c.status(frame, model, controls)
	c.footer(frame, model, controls)
	return frame
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
	c.text(frame, item.Package.Name, 99, 32, c.width-64, palette.text, true)
	c.text(frame, "v"+item.Package.Version+" / "+trustLabel(item), 137, 32, c.width-64, palette.muted, false)
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
	c.text(frame, heading, 178, 32, c.width-64, palette.warning, true)
	c.prose(frame, notice, image.Rect(32, 222, c.width-32, 367), palette.text, false)
	c.label(frame, "Back cancels / Start never confirms", 386, 24, c.width-48, palette.muted, false)
}

func (c *consoleCanvas) details(frame *image.RGBA, model *storeui.Model) {
	item := model.Items[model.Selected]
	c.text(frame, item.Package.Name, 99, 32, c.width-64, palette.text, true)
	c.text(frame, trustLabel(item)+" / "+installState(item), 137, 32, c.width-64, statusColor(item), false)
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
	c.prose(frame, message, image.Rect(32, 178, c.width-32, 274), palette.muted, false)
	if model.Focus == storeui.Health {
		c.text(frame, "Confirm reviews available recovery actions", 302, 32, c.width-64, palette.warning, false)
		return
	}
	if len(item.Actions) == 0 {
		c.label(frame, "Read only / review required", 306, 24, c.width-48, palette.warning, false)
		return
	}
	// The model owns selection, including restoration after a cancelled review.
	fill(frame, image.Rect(32, 286, c.width-32, 330), palette.accent)
	fill(frame, image.Rect(34, 288, c.width-34, 328), palette.panel)
	c.label(frame, "‹  "+actionLabel(item.Actions[model.Action])+"  ›", 296, 48, c.width-96, palette.accent, false)
	c.label(frame, fmt.Sprintf("Action %d of %d / D-pad changes action", model.Action+1, len(item.Actions)), 343, 24, c.width-48, palette.muted, false)
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

func fill(destination imagedraw.Image, rectangle image.Rectangle, shade color.Color) {
	imagedraw.Draw(destination, rectangle, image.NewUniform(shade), image.Point{}, imagedraw.Src)
}
