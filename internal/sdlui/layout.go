package sdlui

// Canvas geometry as tokens. Everything the GUI paints is positioned from this
// table rather than from numbers written at the drawing site, so the layout can
// be reasoned about and tested without cgo. The canvas is fixed at 640x360 and
// nearest-neighbour scaled to the display, so a token is always a whole pixel.

import "image"

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

// listRows is how many rows the list column shows. It is derived from the
// column's own geometry, so a change to a metric moves the row count with it.
const listRows = (listBottom - listTop) / rowHeight

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
