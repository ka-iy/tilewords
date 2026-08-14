// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/mobile"

	"tilewords/engine"
)

// pressNearBottomOf returns a touch event positioned inside cell, low enough that the driver's
// upward compensation means the finger was really on the row below.
func pressNearBottomOf(cell *cellWidget) *mobile.TouchEvent {
	h := cell.Size().Height
	ev := &mobile.TouchEvent{}
	ev.Position = fyne.NewPos(1, h-1)
	return ev
}

// TestJitteryPressOnBoardResolvesToTheRowTouched guards the placement row for a press that the
// driver classifies as a drag.
//
// The driver reports every touch touchYCompensation DIP above the finger, so a press in a row's
// top band is delivered to the cell above it. Tapped corrects that, but the driver sends no
// Tapped once a gesture moves more than its 4 DIP threshold — normal finger jitter — and the
// gesture arrives as Dragged + DragEnd instead. Those handlers must apply the same correction,
// or the identical press stages a tile one row higher purely because the finger wobbled.
func TestJitteryPressOnBoardResolvesToTheRowTouched(t *testing.T) {
	gotRow, gotCol := -1, -1
	cell := newCellWidget(3, 5, engine.Normal, nil)
	cell.Resize(fyne.NewSize(22, 22)) // a phone-sized board cell
	cell.onDragEnd = func(r, c int, _ fyne.Position) { gotRow, gotCol = r, c }

	cell.TouchDown(pressNearBottomOf(cell))
	cell.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, 5)}) // jitter past the drag threshold
	cell.DragEnd()

	wantRow := 3
	if deviceIsMobile() {
		wantRow = 4 // the finger was on the row below the cell the driver picked
	}
	if gotRow != wantRow || gotCol != 5 {
		t.Errorf("a jittery press resolved to (%d,%d), want (%d,5); the tile is staged on the "+
			"wrong row when the finger moves, while an identical still press is correct",
			gotRow, gotCol, wantRow)
	}
}

// TestBoardGestureRowIsStableAcrossTheGesture verifies every handler for one gesture is given the
// same row.
//
// onBoardDragEnd decides whether this is still the drag onBoardDrag recorded by comparing the
// row and column it is handed against the stored source. A row recomputed per event rather than
// latched at TouchDown could differ between the two, and the drag would be rejected as a
// different gesture — silently dropping the move the player just made.
func TestBoardGestureRowIsStableAcrossTheGesture(t *testing.T) {
	var dragRows []int
	cell := newCellWidget(3, 5, engine.Normal, nil)
	cell.Resize(fyne.NewSize(22, 22))
	cell.onDrag = func(r, _ int, _ fyne.Position) { dragRows = append(dragRows, r) }
	cell.onDragEnd = func(r, _ int, _ fyne.Position) { dragRows = append(dragRows, r) }

	cell.TouchDown(pressNearBottomOf(cell))
	cell.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, 5)})
	cell.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, 5)})
	cell.DragEnd()

	if len(dragRows) != 3 {
		t.Fatalf("handlers fired %d times, want 3", len(dragRows))
	}
	for _, r := range dragRows {
		if r != dragRows[0] {
			t.Fatalf("the row changed during one gesture: %v; onBoardDragEnd would not recognise "+
				"it as the drag onBoardDrag started", dragRows)
		}
	}
}

// TestBottomRowPressIsNotCreditedOffTheBoard verifies the correction never reports a row past the
// last one. A press below the board hit-tests into the bottom row, and shifting that down again
// would hand the controller row boardDim, which indexes nothing.
func TestBottomRowPressIsNotCreditedOffTheBoard(t *testing.T) {
	gotRow := -1
	cell := newCellWidget(boardDim-1, 5, engine.Normal, nil)
	cell.Resize(fyne.NewSize(22, 22))
	cell.onDragEnd = func(r, _ int, _ fyne.Position) { gotRow = r }

	cell.TouchDown(pressNearBottomOf(cell))
	cell.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, 5)})
	cell.DragEnd()

	if gotRow != boardDim-1 {
		t.Errorf("bottom-row press resolved to row %d, want %d (the last row)", gotRow, boardDim-1)
	}
}
