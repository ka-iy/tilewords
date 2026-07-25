// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"tilewords/engine"
)

// bulletList renders each item as a bullet point on its own line, word-wrapped.
func bulletList(items []string) fyne.CanvasObject {
	var b strings.Builder
	for i, it := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("•  ")
		b.WriteString(it)
	}
	lbl := widget.NewLabel(b.String())
	lbl.Wrapping = fyne.TextWrapWord
	return lbl
}

// previewCellPx is the side length of one square in the board-layout preview.
const previewCellPx = 20

// showModeInfo pops a scrollable dialog describing a game mode: its board premium-square
// layout and its tile economy (letter counts and point values).
func (a *App) showModeInfo(mode engine.GameMode) {
	// The 15-wide board is wider than a phone dialog, so it gets its own horizontal
	// scroll (bounded to the board's height) rather than being clipped.
	board := container.NewHScroll(boardPreview(mode))
	board.SetMinSize(fyne.NewSize(0, 15*previewCellPx+16))

	items := []fyne.CanvasObject{
		widget.NewLabelWithStyle("Board layout", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		board,
	}
	if note := modeNote(mode); note != nil {
		items = append(items, note)
	}
	items = append(items,
		premiumLegend(mode),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Tile economy", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		economyView(mode),
	)
	content := container.NewVBox(items...)
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(300, 480))

	d := dialog.NewCustom(mode.String()+" Mode", "Close", scroll, a.win)
	d.Resize(fyne.NewSize(360, 620))
	d.Show()
}

// modeNote returns a short explanatory note shown under the board preview, or nil for a
// mode that needs none. Interesting mode gets a note describing how it differs from Classic:
// a pinwheel premium-square layout and a distinct tile economy.
func modeNote(mode engine.GameMode) fyne.CanvasObject {
	if mode != engine.InterestingMode {
		return nil
	}
	return bulletList([]string{
		"Premium squares are arranged in a pinwheel pattern.",
		"The tile economy differs from Classic mode: more tiles, a different letter distribution, and different per-tile points.",
	})
}

// boardPreview renders mode's 15×15 premium-square layout as a compact coloured grid,
// reusing the same colours and labels the board itself uses.
func boardPreview(mode engine.GameMode) fyne.CanvasObject {
	b := engine.NewBoardForMode(mode)
	cells := make([]fyne.CanvasObject, 0, 15*15)
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			cells = append(cells, previewCell(b.Cell(r, c).Square))
		}
	}
	return container.NewGridWithColumns(15, cells...)
}

// previewCell is one square of the board preview: a coloured background with the premium
// label (if any) centred on it.
func previewCell(sq engine.SquareType) fyne.CanvasObject {
	bg := canvas.NewRectangle(colorForSquare(sq))
	bg.StrokeColor = colorGrid
	bg.StrokeWidth = 1
	bg.SetMinSize(fyne.NewSize(previewCellPx, previewCellPx))

	label := premiumLabel(sq)
	if label == "" {
		return bg
	}
	txt := canvas.NewText(label, premLabelColor(sq))
	txt.TextSize = 9
	txt.Alignment = fyne.TextAlignCenter
	txt.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewStack(bg, container.NewCenter(txt))
}

// premiumLegend describes what each premium label means and how many of each the mode's
// board has.
func premiumLegend(mode engine.GameMode) fyne.CanvasObject {
	b := engine.NewBoardForMode(mode)
	var tw, dw, tl, dl int
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			switch b.Cell(r, c).Square {
			case engine.TripleWord:
				tw++
			case engine.DoubleWord:
				dw++
			case engine.TripleLetter:
				tl++
			case engine.DoubleLetter:
				dl++
			}
		}
	}
	return bulletList([]string{
		fmt.Sprintf("W×3 triple-word (%d)", tw),
		fmt.Sprintf("W×2 double-word (%d)", dw),
		fmt.Sprintf("L×3 triple-letter (%d)", tl),
		fmt.Sprintf("L×2 double-letter (%d)", dl),
		"★ start square",
	})
}

// economyView lists the mode's tile distribution: each letter with its count and point
// value (in parentheses), the blanks, and a summary line.
func economyView(mode engine.GameMode) fyne.CanvasObject {
	dist := engine.Distribution(mode)
	letters, blanks := 0, 0
	entries := make([]fyne.CanvasObject, 0, len(dist))
	for _, s := range dist {
		if s.Letter == 0 {
			blanks += s.Count
			entries = append(entries, widget.NewLabel(fmt.Sprintf("blank ×%d", s.Count)))
			continue
		}
		letters += s.Count
		entries = append(entries, widget.NewLabel(fmt.Sprintf("%c ×%d  (%d)", s.Letter, s.Count, s.Points)))
	}
	summary := bulletList([]string{
		fmt.Sprintf("%d tiles: %d letters + %d blanks", letters+blanks, letters, blanks),
		"each entry is  letter ×count (points)",
	})
	return container.NewVBox(summary, container.NewGridWithColumns(3, columnMajor(entries, 3)...))
}

// columnMajor reorders items so that a GridWithColumns(cols) — which fills row by row —
// displays them in column-major order (reading top-to-bottom down each column). Any short
// trailing cells are padded with blank labels to keep the columns aligned.
func columnMajor(items []fyne.CanvasObject, cols int) []fyne.CanvasObject {
	rows := (len(items) + cols - 1) / cols
	out := make([]fyne.CanvasObject, 0, rows*cols)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			if idx := c*rows + r; idx < len(items) {
				out = append(out, items[idx])
			} else {
				out = append(out, widget.NewLabel(""))
			}
		}
	}
	return out
}
