// Package ui is documented in doc.go.
package ui

import (
	"math"

	"fyne.io/fyne/v2"
)

// boardDim is the number of cells along each edge of the board.
const boardDim = 15

// minCellPx is the smallest a board cell may render. At 24px/cell the board's
// minimum edge is minBoardPx (360), keeping a cell comfortably tappable on an
// average phone (~360–412dp wide) where the board fills the screen width.
const minCellPx = 24

// minBoardPx is the board's minimum edge length: the floor the phone layout scales
// the board up from to fill the screen width.
const minBoardPx = minCellPx * boardDim

// rackGapPx is the gap between adjacent rack slots, in logical pixels.
const rackGapPx = 4

// minRackSlotPx is the smallest a rack slot may render. ≥44px keeps the touch
// target within Android/iOS accessibility guidance.
const minRackSlotPx = 44

// boardGeometry computes the square cell size and the top-left origin offset for
// a board rendered inside an area of the given width and height. The board is
// the largest centred boardDim×boardDim square of integer-sized cells that fits.
//
// It is a pure function (no Fyne state) so it can be unit-tested headlessly.
func boardGeometry(w, h float32) (cell, offX, offY float32) {
	side := w
	if h < side {
		side = h
	}
	cell = float32(math.Floor(float64(side) / boardDim))
	if cell < 0 {
		cell = 0
	}
	boardSide := cell * boardDim
	offX = (w - boardSide) / 2
	offY = (h - boardSide) / 2
	return cell, offX, offY
}

// rackGeometry computes the square slot size and the left origin offset for a row
// of n rack slots (separated by rackGapPx) rendered inside an area w×h. Slots are
// square, sized to the smaller of the height and the per-slot width, and the row
// is centred horizontally.
//
// It is a pure function so it can be unit-tested headlessly.
func rackGeometry(w, h float32, n int) (slot, offX float32) {
	if n <= 0 {
		return 0, 0
	}
	gaps := rackGapPx * float32(n-1)
	perSlot := (w - gaps) / float32(n)
	slot = perSlot
	if h < slot {
		slot = h
	}
	if slot < 0 {
		slot = 0
	}
	total := slot*float32(n) + gaps
	offX = (w - total) / 2
	if offX < 0 {
		// The area is narrower than the fixed gaps allow; left-align rather than
		// positioning slots at a negative offset.
		offX = 0
	}
	return slot, offX
}

// boardLayout lays out boardDim*boardDim children as a centred square grid of
// equal cells. Children are expected in row-major order (index = row*boardDim+col).
type boardLayout struct{}

func (boardLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	cell, offX, offY := boardGeometry(size.Width, size.Height)
	cellSize := fyne.NewSize(cell, cell)
	for i, o := range objects {
		row := i / boardDim
		col := i % boardDim
		o.Resize(cellSize)
		o.Move(fyne.NewPos(offX+float32(col)*cell, offY+float32(row)*cell))
	}
}

func (boardLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(minBoardPx, minBoardPx)
}

// rackLayout lays out its children as a centred horizontal row of equal squares.
type rackLayout struct{}

func (rackLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	n := len(objects)
	slot, offX := rackGeometry(size.Width, size.Height, n)
	slotSize := fyne.NewSize(slot, slot)
	y := (size.Height - slot) / 2
	stride := slot + rackGapPx
	for i, o := range objects {
		o.Resize(slotSize)
		o.Move(fyne.NewPos(offX+float32(i)*stride, y))
	}
}

func (rackLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	n := len(objects)
	if n == 0 {
		return fyne.NewSize(0, 0)
	}
	w := minRackSlotPx*float32(n) + rackGapPx*float32(n-1)
	return fyne.NewSize(w, minRackSlotPx)
}
