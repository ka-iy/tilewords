// Package ui is documented in doc.go.
package ui

import (
	"fmt"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// boardDim is the number of cells along each edge of the board.
const boardDim = 15

// labelGutterFactor is the width of the row/column label strip, expressed as a
// fraction of one cell. A gutter is reserved along the top (column letters A–O) and
// left (row numbers 1–15) edges. Being a fraction of the cell, the labels scale with
// the board: they shrink as the board shrinks and never claim a fixed slice that
// would starve the cells on a narrow phone.
const labelGutterFactor = 0.6

// minCellPx is the smallest a board cell may render. At 24px/cell a cell stays
// comfortably tappable on an average phone (~360–412dp wide) where the board fills
// the screen width.
const minCellPx = 24

// minBoardPx is the board's minimum edge length: the floor the phone layout scales
// the board up from to fill the screen width. It includes the label gutter so cells
// stay at their tappable minimum once the labels' share is set aside.
const minBoardPx = float32(minCellPx * (boardDim + labelGutterFactor))

// boardLabelTextFactor is the row/column label glyph size, as a fraction of one cell.
// It is small enough that a two-digit row number (e.g. "15") fits within the gutter
// (labelGutterFactor cells) wide.
const boardLabelTextFactor = 0.4

// newBoardLabels builds the column labels (A–O, left to right) and row labels (1–15,
// top to bottom) for the board, matching the Scrabble notation convention (columns are
// letters, rows are numbers; see engine notation). The two runs are returned in the
// order boardLayout expects them appended after the cells.
func newBoardLabels() (colLabels, rowLabels []fyne.CanvasObject) {
	colLabels = make([]fyne.CanvasObject, boardDim)
	rowLabels = make([]fyne.CanvasObject, boardDim)
	for i := 0; i < boardDim; i++ {
		colLabels[i] = newBoardLabelText(string(rune('A' + i)))
		rowLabels[i] = newBoardLabelText(fmt.Sprintf("%d", i+1))
	}
	return colLabels, rowLabels
}

// newBoardLabelText creates a single centred board-edge label. Its size is set later by
// the layout to scale with the cell; the colour follows the current theme so the label
// reads against the screen background behind the gutter.
func newBoardLabelText(s string) *canvas.Text {
	t := canvas.NewText(s, bodyTextColor())
	t.Alignment = fyne.TextAlignCenter
	t.TextStyle = fyne.TextStyle{Bold: true}
	return t
}

// rackGapPx is the gap between adjacent rack slots, in logical pixels.
const rackGapPx = 4

// minRackSlotPx is the smallest a rack slot may render. ≥44px keeps the touch
// target within Android/iOS accessibility guidance.
const minRackSlotPx = 44

// boardGeometry computes the square cell size and the top-left origin offset of the
// cell grid for a board rendered inside an area of the given width and height. A label
// gutter of labelGutterFactor cells is reserved along the top and left edges, and the
// grid is the largest boardDim×boardDim square of integer-sized cells that fits in the
// remaining space; the gutter-plus-grid block is centred. The returned offsets are the
// grid's top-left corner (already past the gutter), so the row/column labels occupy
// [offX-gutter, offX] and [offY-gutter, offY] where gutter is cell*labelGutterFactor.
//
// It is a pure function (no Fyne state) so it can be unit-tested headlessly.
func boardGeometry(w, h float32) (cell, offX, offY float32) {
	side := w
	if h < side {
		side = h
	}
	cell = float32(math.Floor(float64(side) / (boardDim + labelGutterFactor)))
	if cell < 0 {
		cell = 0
	}
	gutter := cell * labelGutterFactor
	block := gutter + cell*boardDim
	offX = gutter + (w-block)/2
	offY = gutter + (h-block)/2
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

// boardLayout lays out the board grid together with its row/column labels as a
// centred square. The children are expected in three consecutive runs:
//   - boardDim*boardDim cell widgets in row-major order (index = row*boardDim+col);
//   - boardDim column labels (A–O), left to right, in the top gutter;
//   - boardDim row labels (1–15), top to bottom, in the left gutter.
//
// Any run may be absent (e.g. an unlabelled board passes only the cells); labels are
// positioned only for the children actually present.
type boardLayout struct{}

func (boardLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	cell, offX, offY := boardGeometry(size.Width, size.Height)
	cellSize := fyne.NewSize(cell, cell)
	nCells := boardDim * boardDim
	for i, o := range objects {
		if i >= nCells {
			break
		}
		row := i / boardDim
		col := i % boardDim
		o.Resize(cellSize)
		o.Move(fyne.NewPos(offX+float32(col)*cell, offY+float32(row)*cell))
	}
	layoutBoardLabels(objects, cell, offX, offY)
}

// layoutBoardLabels positions the column and row label runs (following the cells) in
// the gutters reserved by boardGeometry. Labels are centred over their column / beside
// their row, and their glyph size scales with the cell so they fit the gutter.
func layoutBoardLabels(objects []fyne.CanvasObject, cell, offX, offY float32) {
	nCells := boardDim * boardDim
	if cell <= 0 || len(objects) <= nCells {
		return
	}
	gutter := cell * labelGutterFactor
	textSize := cell * boardLabelTextFactor

	// Column labels: one cell wide each, centred horizontally over the column and
	// vertically within the top gutter.
	for c := 0; c < boardDim; c++ {
		idx := nCells + c
		if idx >= len(objects) {
			break
		}
		t, ok := objects[idx].(*canvas.Text)
		if !ok {
			continue
		}
		t.TextSize = textSize
		t.Resize(fyne.NewSize(cell, gutter))
		th := t.MinSize().Height
		t.Move(fyne.NewPos(offX+float32(c)*cell, offY-gutter+(gutter-th)/2))
	}

	// Row labels: one gutter wide each, centred horizontally within the left gutter
	// and vertically beside the row.
	for r := 0; r < boardDim; r++ {
		idx := nCells + boardDim + r
		if idx >= len(objects) {
			break
		}
		t, ok := objects[idx].(*canvas.Text)
		if !ok {
			continue
		}
		t.TextSize = textSize
		t.Resize(fyne.NewSize(gutter, cell))
		th := t.MinSize().Height
		t.Move(fyne.NewPos(offX-gutter, offY+float32(r)*cell+(cell-th)/2))
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
