// Package ui is documented in doc.go.
package ui

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	"squabble/engine"
)

// newTileObjects creates the three canvas objects used to render a single tile or
// board cell: a background rectangle (also drawing the border via its stroke), a
// centred letter and a small bottom-right points value.
func newTileObjects() (bg *canvas.Rectangle, letter, points *canvas.Text) {
	bg = canvas.NewRectangle(colorBoardBg)
	bg.StrokeColor = colorGrid
	bg.StrokeWidth = 1

	letter = canvas.NewText("", colorTileLetter)
	letter.Alignment = fyne.TextAlignCenter
	letter.TextStyle = fyne.TextStyle{Bold: true}

	points = canvas.NewText("", colorTilePoints)
	points.Alignment = fyne.TextAlignTrailing
	return bg, letter, points
}

// styleAsTile sets bg/letter/points to display a committed or staged tile.
func styleAsTile(bg *canvas.Rectangle, letter, points *canvas.Text, t engine.Tile, staged bool) {
	if staged {
		bg.FillColor = colorTileStagedBg
		bg.StrokeColor = colorTileStagedBorder
		bg.StrokeWidth = 2
	} else {
		bg.FillColor = colorTileBg
		bg.StrokeColor = colorTileBorder
		bg.StrokeWidth = 1
	}

	ch := t.DisplayLetter()
	if ch == 0 {
		// Unassigned blank — empty face.
		letter.Text = ""
		points.Text = ""
		return
	}
	letter.Text = string([]byte{ch})
	if t.IsBlank {
		letter.Color = colorTileBlankLetter
	} else {
		letter.Color = colorTileLetter
	}
	if !t.IsBlank && t.Points > 0 {
		points.Text = strconv.Itoa(t.Points)
		points.Color = colorTilePoints
	} else {
		points.Text = ""
	}
}

// layoutTileText positions and scales the letter and points texts within a square
// cell of the given size. letterFactor controls the glyph height relative to the
// cell (≈0.5 for tile letters, smaller for multi-character premium labels).
func layoutTileText(letter, points *canvas.Text, size fyne.Size, letterFactor float32) {
	letter.TextSize = size.Height * letterFactor
	lh := letter.MinSize().Height
	letter.Resize(fyne.NewSize(size.Width, lh))
	letter.Move(fyne.NewPos(0, (size.Height-lh)/2))

	points.TextSize = size.Height * 0.26
	ph := points.MinSize().Height
	points.Resize(fyne.NewSize(size.Width-2, ph))
	points.Move(fyne.NewPos(0, size.Height-ph-1))
}

// tileFillLayout fills its container with the tile background (first child) and lays
// out the letter and points texts (next two children) like a board cell. It is used
// by the drag ghost, whose three canvas objects come from newTileObjects.
type tileFillLayout struct{}

func (tileFillLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) < 3 {
		return
	}
	objs[0].Resize(size)
	objs[0].Move(fyne.NewPos(0, 0))
	letter, lok := objs[1].(*canvas.Text)
	points, pok := objs[2].(*canvas.Text)
	if lok && pok {
		layoutTileText(letter, points, size, 0.5)
	}
}

func (tileFillLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(minCellPx, minCellPx)
}

// premiumLabel returns the short label drawn on an unoccupied premium square.
func premiumLabel(sq engine.SquareType) string {
	switch sq {
	case engine.DoubleWord:
		return "DW"
	case engine.TripleWord:
		return "TW"
	case engine.DoubleLetter:
		return "DL"
	case engine.TripleLetter:
		return "TL"
	case engine.Centre:
		return "★"
	default:
		return ""
	}
}

// colorForSquare returns the background colour for an unoccupied square.
func colorForSquare(sq engine.SquareType) color.RGBA {
	switch sq {
	case engine.DoubleWord:
		return colorDW
	case engine.TripleWord:
		return colorTW
	case engine.DoubleLetter:
		return colorDL
	case engine.TripleLetter:
		return colorTL
	case engine.Centre:
		return colorCentre
	default:
		return colorBoardBg
	}
}
