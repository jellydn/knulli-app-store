package sdlui

// Canvas geometry as tokens. Everything the GUI paints is positioned from this
// table rather than from numbers written at the drawing site, so the layout can
// be reasoned about and tested without cgo. The canvas is fixed at 640x360 and
// nearest-neighbour scaled to the display, so a token is always a whole pixel.

import (
	"image"

	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

const (
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	canvasWidth = 640
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	canvasHeight = 360
	// One panel and one footer line are shared by every screen, so a hint can
	// only ever appear in the footer.
	panelTop       = 42
	panelLeft      = 16
	panelRight     = 624
	panelBottom    = 330
	footerBaseline = 348
	// The catalogue panel is split into the list column and the details column.
	// The gap between them keeps the two from reading as one surface.
	listPanelRight  = 230
	detailPanelLeft = 242
	// glyphHeight is the built-in face's ascent: a baseline this far from the
	// top of a row still leaves its ink inside the row.
	glyphHeight = 11
	// panelInset is the left margin every panel body line shares.
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	panelInset = 258
	// modeHeadingBaseline is where every controller screen starts its content,
	// so switching screens never moves the heading.
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	modeHeadingBaseline = 176
	// One status block serves the catalogue, details, health and error screens.
	// It sits above the action buttons, so a status never covers an action and
	// never lands on the notice.
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	statusBoxLeft = 248
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	statusBoxTop = 190
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	statusBoxRight = 618
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	statusBoxBottom = 246
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	statusBaseline = 205
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	statusLines = 3
	// The action row is the lowest interactive element in the catalogue panel.
	// The status block above it can never hide a button while it stays clear.
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	actionLabelBaseline = 264
	//lint:ignore U1000 used by the sdl-tagged draw code in this package
	actionRowBaseline = 278
)

// The list column is the left panel: a heading, then one row per package. Row
// metrics are tokens like everything else, so the number of rows a screen shows
// is derived from the list height instead of written down beside the loop that
// fills it.
const (
	listLeft  = 24
	listRight = 222
	listTop   = 72
	// listBottom leaves the hairline of panel background below the last row.
	listBottom = 316
	// rowHeight is the pitch from one row to the next; rowBoxHeight is the row
	// itself, which is smaller so consecutive rows never touch.
	rowHeight    = 44
	rowBoxHeight = 38
	// Row text is set inside the row, so the pitch can change without the
	// baselines drifting out of their box.
	rowTitleBaseline  = 15
	rowDetailBaseline = 31
	// rowTextInset is the left margin row text shares, inside the selection
	// ring, so the chrome never sits on a glyph.
	rowTextInset = 10
	// Row text is truncated to what the column can show: the title at the full
	// column width, the detail line slightly wider because it is set in the
	// same face and the rows are the only content there.
	rowTitleCharacters  = 24
	rowDetailCharacters = 26
)

// The list column carries a tab strip where a column heading would sit: one
// pill per tab, sized to its own label so a tab's chrome matches its word, and
// laid left to right with a fixed gap. The strip shares the list's own left
// edge, so the bar and the rows below it start in the same place.
const (
	tabStripTop      = 52
	tabStripBottom   = 68
	tabStripBaseline = 64
	// tabPillPaddingX is the gap between a label and its pill's edge.
	tabPillPaddingX = 6
	// tabGap is the gap between two pills, wide enough that neighbouring tabs
	// never touch.
	tabGap = 6
	// tabAdvance is one character of the built-in face, which is what a pill's
	// width is measured in.
	tabAdvance = 7
)

// listRangeBaseline is where the list reports the window it is showing. The
// caption sits under the last row rather than beside a heading, so the strip at
// the top of the column holds only tabs.
const listRangeBaseline = 326

// The notice bar is bottom-anchored over the panel foot: it starts below the
// action row, so a notice never covers a button, and stops at the panel's own
// bottom edge, so it never reaches the footer.
const (
	toastHeight = 30
	toastTop    = panelBottom - toastHeight
	toastInsetX = 10
	// toastFirstLine is the first line's baseline: one pixel of clearance below
	// the bar's own top edge, then the face's ascent.
	toastFirstLine  = toastTop + 13
	toastLineHeight = 15
	toastMaxLines   = 2
)

// toastCharacterLimit is the widest notice line that fits the bar.
const toastCharacterLimit = (panelRight - panelLeft - toastInsetX) / tabAdvance

// listRows is how many rows the list column shows. It is derived from the
// column's own geometry, so a change to a metric moves the row count with it.
const listRows = (listBottom - listTop) / rowHeight

// tabLabels names the bar in tab order. The strip is drawn from this and the
// active tab is marked by identity, so the bar and the cycle share one order.
func tabLabels() []string {
	labels := make([]string, 0, len(storeui.TabOrder))
	for _, tab := range storeui.TabOrder {
		labels = append(labels, tab.Label())
	}
	return labels
}

// tabPillRectangle is one tab's pill: as wide as its label plus the padding,
// offset from the list's left edge.
func tabPillRectangle(offset int, label string) image.Rectangle {
	left := listLeft + offset
	return image.Rect(left, tabStripTop, left+len(label)*tabAdvance+2*tabPillPaddingX, tabStripBottom)
}

// tabPillRectangles lays the whole bar out in tab order. A pill may be wider
// than an even share of the column and the bar still fits, because the labels
// are short and the gaps between them stay fixed.
func tabPillRectangles(labels []string) []image.Rectangle {
	rectangles := make([]image.Rectangle, 0, len(labels))
	offset := 0
	for _, label := range labels {
		rectangle := tabPillRectangle(offset, label)
		rectangles = append(rectangles, rectangle)
		offset += rectangle.Dx() + tabGap
	}
	return rectangles
}

// listRangePosition is the left edge of the range caption, right-aligned to the
// column so it reads as the list's own footer.
func listRangePosition(caption string) int {
	return listRight - len(caption)*tabAdvance
}

// toastRectangle is the notice bar's band across the panel foot.
func toastRectangle() image.Rectangle {
	return image.Rect(panelLeft, toastTop, panelRight, panelBottom)
}

// emptyCatalogueMessage names why the list has no rows. An index that listed
// nothing and a tab that hides everything it listed are different answers, and
// the second one has a way out the first does not.
//
//lint:ignore U1000 used by the sdl-tagged draw code in this package
func emptyCatalogueMessage(total, visible int) string {
	if total > 0 && visible == 0 {
		return "NO PACKAGES IN THIS TAB"
	}
	return "NO PACKAGES IN CATALOGUE"
}

// listRowRectangle is where one visible row sits. A row's position depends on
// its index in the window and nothing else, so every row is the same pitch.
func listRowRectangle(index int) image.Rectangle {
	top := listTop + index*rowHeight
	return image.Rect(listLeft, top, listRight, top+rowBoxHeight)
}

// listRowTitleBaseline and listRowDetailBaseline are the text baselines inside
// a row, one line each, in that fixed order.
func listRowTitleBaseline(rectangle image.Rectangle) int {
	return rectangle.Min.Y + rowTitleBaseline
}

func listRowDetailBaseline(rectangle image.Rectangle) int {
	return rectangle.Min.Y + rowDetailBaseline
}
