package sdlui

// Layout tests. The geometry is a token table, so its invariants are checked
// without cgo: the row count is derived from the column height, and a reviewer
// reading only this file can see that a change to a metric keeps the rows inside
// the panel.

import (
	"image"
	"testing"

	storeui "github.com/jellydn/knulli-app-store/internal/ui"
)

func TestListGeometryKeepsEveryRowInsideTheListColumn(t *testing.T) {
	if listLeft < panelLeft || listRight > listPanelRight {
		t.Fatalf("list column %d-%d escapes the catalogue panel %d-%d", listLeft, listRight, panelLeft, listPanelRight)
	}
	if listRows < 2 {
		t.Fatalf("the list shows %d rows", listRows)
	}
	if last := listTop + listRows*rowHeight; last > panelBottom {
		t.Fatalf("the rows end at %d, past the panel bottom %d", last, panelBottom)
	}
	previous := image.Rectangle{}
	for index := 0; index < listRows; index++ {
		row := listRowRectangle(index)
		if row.Min.Y < listTop || row.Max.Y > listBottom {
			t.Fatalf("row %d = %v is outside the list window %d-%d", index, row, listTop, listBottom)
		}
		if index > 0 && row.Min.Y < previous.Max.Y {
			t.Fatalf("row %d %v overlaps row %d %v", index, row, index-1, previous)
		}
		previous = row
	}
}

// The tab strip shares the list column's own edges and sits above the rows, so
// the bar and the list it filters cannot drift apart.
func TestTabStripFitsTheListColumnAndClearsTheRows(t *testing.T) {
	labels := tabLabels()
	if len(labels) != len(storeui.TabOrder) {
		t.Fatalf("the bar draws %d tabs for %d in order", len(labels), len(storeui.TabOrder))
	}
	previous := image.Rectangle{}
	for index, pill := range tabPillRectangles(labels) {
		if pill.Min.X < listLeft || pill.Max.X > listRight {
			t.Fatalf("tab %q spans %v, outside the list column %d-%d", labels[index], pill, listLeft, listRight)
		}
		if pill.Min.Y < panelTop || pill.Max.Y > listTop {
			t.Fatalf("tab %q spans %v, outside the strip above the rows", labels[index], pill)
		}
		if pill.Dx() < len(labels[index])*tabAdvance {
			t.Fatalf("tab %q is narrower than its label", labels[index])
		}
		if tabStripBaseline-glyphHeight < pill.Min.Y || tabStripBaseline > pill.Max.Y {
			t.Fatalf("tab %q does not keep its baseline %d inside the pill", labels[index], tabStripBaseline)
		}
		if index > 0 && pill.Min.X < previous.Max.X {
			t.Fatalf("tab %q overlaps the tab before it: %v then %v", labels[index], previous, pill)
		}
		previous = pill
	}
}

// The range caption sits under the rows it describes, inside the panel, so the
// strip at the top of the column holds only tabs.
func TestRangeCaptionSitsBelowTheRows(t *testing.T) {
	if lastRow := listRowRectangle(listRows - 1); listRangeBaseline-glyphHeight < lastRow.Max.Y {
		t.Fatalf("the range caption (ink from %d) reaches the last row box (bottom %d)", listRangeBaseline-glyphHeight, lastRow.Max.Y)
	}
	if listRangeBaseline > panelBottom {
		t.Fatalf("the range caption baseline %d is outside the panel bottom %d", listRangeBaseline, panelBottom)
	}
	for _, caption := range []string{"1-5 OF 12", "1-5 OF 108"} {
		left := listRangePosition(caption)
		if left < listLeft || left+len(caption)*tabAdvance > listRight {
			t.Fatalf("caption %q spans %d-%d, outside the list column", caption, left, left+len(caption)*tabAdvance)
		}
	}
}

// A notice is drawn over the panel foot: it starts below the action row, so it
// cannot cover a button, and stops at the panel's own edge, so it cannot reach
// the footer's hints.
func TestNoticeBarClearsTheActionRowAndTheFooter(t *testing.T) {
	bar := toastRectangle()
	if bar.Min.Y < actionRowBaseline+22 {
		t.Fatalf("the notice bar (top %d) covers the action row (bottom %d)", bar.Min.Y, actionRowBaseline+22)
	}
	if bar.Max.Y > panelBottom || bar.Min.X < panelLeft || bar.Max.X > panelRight {
		t.Fatalf("the notice bar %v escapes the panel", bar)
	}
	if toastFirstLine-glyphHeight < bar.Min.Y {
		t.Fatalf("the first notice line (ink from %d) starts above the bar (top %d)", toastFirstLine-glyphHeight, bar.Min.Y)
	}
	last := toastFirstLine + (toastMaxLines-1)*toastLineHeight
	if last > bar.Max.Y {
		t.Fatalf("the last notice line (baseline %d) is below the bar (bottom %d)", last, bar.Max.Y)
	}
	if toastCharacterLimit*tabAdvance > bar.Max.X-(bar.Min.X+toastInsetX) {
		t.Fatal("a notice line is wider than the bar")
	}
}

// Row text is set inside its own row, so a wider pitch cannot push a detail
// line onto the next package's title.
func TestListRowTextStaysInsideItsOwnRow(t *testing.T) {
	for index := 0; index < listRows; index++ {
		row := listRowRectangle(index)
		if title, detail := listRowTitleBaseline(row), listRowDetailBaseline(row); detail <= title {
			t.Fatalf("row %d puts the detail line on the title: %d then %d", index, title, detail)
		}
		if top := listRowTitleBaseline(row) - glyphHeight; top < row.Min.Y {
			t.Fatalf("row %d title starts at %d, above the row top %d", index, top, row.Min.Y)
		}
		if listRowDetailBaseline(row) > row.Max.Y {
			t.Fatalf("row %d detail baseline %d is below the row bottom %d", index, listRowDetailBaseline(row), row.Max.Y)
		}
		if listRight-(listLeft+rowTextInset) < rowTitleCharacters*7 {
			t.Fatal("a truncated row title does not fit the list column")
		}
	}
}
