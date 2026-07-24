package ui

import (
	"testing"

	"fyne.io/fyne/v2"
)

// TestCellAtRel maps board-relative positions to cells for a 480×480 board. The label
// gutter shifts the grid origin to (offX,offY); the top and left gutters belong to the
// labels, not any cell. Coordinates are derived from the geometry so the test holds for
// any gutter factor.
func TestCellAtRel(t *testing.T) {
	size := fyne.NewSize(480, 480)
	cell, offX, offY := boardGeometry(size.Width, size.Height)
	end := cell*boardDim - 1 // last pixel inside the grid
	cases := []struct {
		x, y   float32
		wr, wc int
		wok    bool
	}{
		{offX, offY, 0, 0, true},                                  // first cell (grid origin, past the gutter)
		{offX + cell, offY, 0, 1, true},                           // second column
		{offX, offY + cell, 1, 0, true},                           // second row
		{offX + end, offY + end, 14, 14, true},                    // last cell
		{offX + cell*boardDim, offY + cell*boardDim, 0, 0, false}, // just outside bottom-right
		{offX - 1, offY + cell, 0, 0, false},                      // left gutter (row labels), left of the grid
		{offX + cell, offY - 1, 0, 0, false},                      // top gutter (column labels), above the grid
	}
	for _, c := range cases {
		r, col, ok := cellAtRel(fyne.NewPos(c.x, c.y), size)
		if ok != c.wok || (ok && (r != c.wr || col != c.wc)) {
			t.Errorf("cellAtRel(%g,%g) = (%d,%d,%v), want (%d,%d,%v)", c.x, c.y, r, col, ok, c.wr, c.wc, c.wok)
		}
	}
}

// TestRackSlotAtRel maps rack-relative positions to slots for a 7-slot row sized so
// each slot is 44px wide with 4px gaps (stride 48), no centring offset.
func TestRackSlotAtRel(t *testing.T) {
	size := fyne.NewSize(minRackSlotPx*7+rackGapPx*6, minRackSlotPx) // 332 × 44
	cases := []struct {
		x, y float32
		widx int
		wok  bool
	}{
		{0, 0, 0, true},
		{48, 0, 1, true},
		{96, 0, 2, true},
		{331, 0, 6, true},
		{-1, 0, 0, false},  // left of the row
		{0, 44, 0, true},   // one row-height below — within the vertical drift tolerance
		{0, -44, 0, true},  // one row-height above — within the vertical drift tolerance
		{0, 88, 0, false},  // 2× row-height below — beyond the tolerance
		{400, 0, 0, false}, // right of the row
	}
	for _, c := range cases {
		idx, ok := rackSlotAtRel(fyne.NewPos(c.x, c.y), size)
		if ok != c.wok || (ok && idx != c.widx) {
			t.Errorf("rackSlotAtRel(%g,%g) = (%d,%v), want (%d,%v)", c.x, c.y, idx, ok, c.widx, c.wok)
		}
	}
}
