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
// centred letter and a small upper-right points value.
func newTileObjects() (bg *canvas.Rectangle, letter, points *canvas.Text) {
	bg = canvas.NewRectangle(colorBoardBg)
	bg.StrokeColor = colorGrid
	bg.StrokeWidth = 1

	letter = canvas.NewText("", colorTileLetter)
	letter.Alignment = fyne.TextAlignCenter
	letter.TextStyle = fyne.TextStyle{Bold: true}

	points = canvas.NewText("", colorTilePoints)
	points.Alignment = fyne.TextAlignTrailing
	points.TextStyle = fyne.TextStyle{Bold: true}
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
		bg.StrokeWidth = 2
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

	// The points value is drawn bold and larger relative to the letter, inset from the
	// top-right corner towards the centre so it stays legible on small mobile cells.
	points.TextSize = size.Height * 0.30
	ph := points.MinSize().Height
	inset := size.Width * 0.05
	points.Resize(fyne.NewSize(size.Width-inset, ph))
	points.Move(fyne.NewPos(0, inset))
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
		return "W×2"
	case engine.TripleWord:
		return "W×3"
	case engine.DoubleLetter:
		return "L×2"
	case engine.TripleLetter:
		return "L×3"
	case engine.Centre:
		return "★"
	default:
		return ""
	}
}

// isLightColor reports whether c is light enough that a white label would read poorly on
// it, so a dark label should be used instead. Uses the ITU-R BT.601 luma approximation.
func isLightColor(c color.RGBA) bool {
	luma := 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	return luma > 140
}

// premLabelColor returns the label colour for an unoccupied premium square: a dark glyph
// on light-toned fills (light orange, lavender centre) and white on dark ones, so the
// short label stays legible regardless of the square's fill.
func premLabelColor(sq engine.SquareType) color.Color {
	if isLightColor(colorForSquare(sq)) {
		return colorPremTextDark
	}
	return colorPremText
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
