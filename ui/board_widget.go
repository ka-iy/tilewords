// Package ui is documented in doc.go.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"tilewords/engine"
)

// cellWidget is a single tappable board square. It displays either a premium
// label (when empty) or a committed/staged tile, and reports taps to onTap with
// its own (row, col). It satisfies fyne.Tappable so the same handler serves both
// mouse clicks and touch taps.
type cellWidget struct {
	widget.BaseWidget

	row, col int
	square   engine.SquareType

	tile      *engine.Tile // nil when the square is empty
	staged    bool         // true when tile is a tentatively-placed (staged) tile
	highlight bool         // true to outline a committed tile (the AI's most recent word)
	pickedUp  bool         // true when this staged tile is tapped to move (tap-to-move)

	onTap func(row, col int)
	// onDrag/onDragEnd power dragging a staged tile around the board: onDrag fires
	// repeatedly with the live pointer position, onDragEnd once at release.
	onDrag    func(row, col int, abs fyne.Position)
	onDragEnd func(row, col int, abs fyne.Position)

	dragging bool
	dragAbs  fyne.Position
}

// aiHighlightStrokeWidth is the border thickness used to outline the AI's most
// recently played word.
const aiHighlightStrokeWidth = 3

func newCellWidget(row, col int, square engine.SquareType, onTap func(int, int)) *cellWidget {
	c := &cellWidget{row: row, col: col, square: square, onTap: onTap}
	c.ExtendBaseWidget(c)
	return c
}

// setContent updates what the cell displays and refreshes it. highlight outlines a
// committed tile in the AI-word colour; pickedUp outlines a staged tile chosen for a
// tap-to-move. Both are ignored for an empty cell.
func (c *cellWidget) setContent(tile *engine.Tile, staged, highlight, pickedUp bool) {
	c.tile = tile
	c.staged = staged
	c.highlight = highlight
	c.pickedUp = pickedUp
	c.Refresh()
}

// Tapped reports the tap to the controller. The position within the cell is not
// needed — the cell already knows its coordinates.
func (c *cellWidget) Tapped(_ *fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap(c.row, c.col)
	}
}

// Dragged records the live pointer position and reports it. The controller starts a
// drag only for a staged tile; for any other cell this lets a slightly-moving tap
// (which Fyne classifies as a drag) still resolve as a tap on DragEnd.
func (c *cellWidget) Dragged(e *fyne.DragEvent) {
	// AbsolutePosition is (0,0) during a drag on mobile; track the pointer via the delta.
	c.dragAbs = dragAbsPosition(c, c.dragging, c.dragAbs, e)
	c.dragging = true
	if c.onDrag != nil {
		c.onDrag(c.row, c.col, c.dragAbs)
	}
}

// DragEnd reports the gesture's final pointer position; the controller moves a staged
// tile, recalls it (released off the board), or treats the gesture as a tap.
func (c *cellWidget) DragEnd() {
	if c.dragging && c.onDragEnd != nil {
		c.onDragEnd(c.row, c.col, c.dragAbs)
	}
	c.dragging = false
}

func (c *cellWidget) CreateRenderer() fyne.WidgetRenderer {
	bg, letter, points := newTileObjects()
	r := &cellRenderer{cell: c, bg: bg, letter: letter, points: points}
	r.applyState()
	return r
}

// cellRenderer draws one cellWidget.
type cellRenderer struct {
	cell   *cellWidget
	bg     *canvas.Rectangle
	letter *canvas.Text
	points *canvas.Text
}

// applyState sets colours and text from the widget's current content (no layout).
func (r *cellRenderer) applyState() {
	c := r.cell
	if c.tile != nil {
		styleAsTile(r.bg, r.letter, r.points, *c.tile, c.staged)
		// Outline the AI's most recently played word in red. A staged tile is never
		// highlighted, so this never conflicts with the staged-tile border.
		if c.highlight && !c.staged {
			r.bg.StrokeColor = colorAILastWord
			r.bg.StrokeWidth = aiHighlightStrokeWidth
		}
		// A staged tile picked up for a tap-to-move is outlined in cyan.
		if c.pickedUp {
			r.bg.StrokeColor = colorPickedUp
			r.bg.StrokeWidth = aiHighlightStrokeWidth
		}
		return
	}
	// Empty square: premium background + premium label.
	r.bg.FillColor = colorForSquare(c.square)
	r.bg.StrokeColor = colorGrid
	r.bg.StrokeWidth = 1
	r.letter.Text = premiumLabel(c.square)
	r.letter.Color = premLabelColor(c.square)
	r.points.Text = ""
}

func (r *cellRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))

	// Tile letters fill the cell; multi-character premium labels (W×2/W×3/…) use a
	// smaller glyph so they fit on one line. The ★ centre marker uses a tile-sized
	// glyph.
	factor := float32(0.5)
	if r.cell.tile == nil && len(r.letter.Text) > 1 {
		factor = 0.32
	}
	// A tile's letter is nudged right of centre (clear of the bottom-left points value);
	// premium-square labels stay centred.
	var shift float32
	if r.cell.tile != nil {
		shift = tileLetterShiftFactor
	}
	layoutTileText(r.letter, r.points, size, factor, shift)
}

func (r *cellRenderer) MinSize() fyne.Size { return fyne.NewSize(minCellPx, minCellPx) }

func (r *cellRenderer) Refresh() {
	r.applyState()
	r.Layout(r.cell.Size())
	canvas.Refresh(r.cell)
}

func (r *cellRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.letter, r.points}
}

func (r *cellRenderer) Destroy() {}
