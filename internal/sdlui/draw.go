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

type consoleCanvas struct {
	width, height int
	compact       bool
	step          int
	body, title   font.Face
	thumbnails    map[string]image.Image
	groups        []footerGroup
	footerStart   int
}

func newConsoleCanvas(width, height int) (*consoleCanvas, error) {
	thumbnails, err := loadThumbnails()
	if err != nil {
		return nil, err
	}
	typeface, err := opentype.Parse(consoleFont)
	if err != nil {
		return nil, fmt.Errorf("load console font: %w", err)
	}
	bodySize, titleSize, step := 18.0, 26.0, 23
	if height < canvasHeight || width < 400 {
		bodySize, titleSize, step = 14, 18, 18
	}
	body, err := opentype.NewFace(typeface, &opentype.FaceOptions{Size: bodySize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	title, err := opentype.NewFace(typeface, &opentype.FaceOptions{Size: titleSize, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		body.Close()
		return nil, err
	}
	return &consoleCanvas{width: width, height: height, compact: width < 640 || height < canvasHeight, step: step, body: body, title: title, thumbnails: thumbnails}, nil
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
	c.layoutFooter(model, controls)
	frame := image.NewRGBA(image.Rect(0, 0, c.width, c.height))
	fill(frame, frame.Bounds(), palette.background)
	if c.compact {
		c.compactScreen(frame, model, platformName, controls)
		return frame
	}
	content := frame.SubImage(image.Rect(0, 0, c.width, c.statusTop()-4)).(*image.RGBA)
	top := c.header(content, model)
	switch {
	case controls != nil && controls.Mode != storeinput.Normal:
		c.text(content, "Controller unavailable", 154, 32, c.width-64, palette.warning, true)
		c.prose(content, "Reconnect a controller to continue. "+platformName, image.Rect(32, 212, c.width-32, 310), palette.muted, false)
	case len(model.Items) == 0:
		c.text(content, emptyCatalogueMessage(model.Total()), 164, 32, c.width-64, palette.text, true)
	case model.Focus == storeui.Browse || model.Busy:
		c.browse(content, model, top)
	default:
		c.review(content, model, image.Rect(32, top, c.width-32, c.statusTop()-4))
	}
	c.status(frame, model)
	c.footer(frame, model)
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
