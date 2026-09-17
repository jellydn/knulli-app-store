package sdlui

// Layout tests. The geometry is a token table, so its invariants are checked
// without cgo: the row count is derived from the column height, and a reviewer
// reading only this file can see that a change to a metric keeps the rows inside
// the panel.

import (
	"image"
	"testing"
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
