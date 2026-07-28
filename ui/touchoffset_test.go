// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"tilewords/engine"
)

// onTouchDevice makes the touch-only paths active for one test. The test driver's device
// reports desktop, so without this the touch compensation would never be applied.
func onTouchDevice(t *testing.T) {
	t.Helper()
	prev := deviceIsMobile
	deviceIsMobile = func() bool { return true }
	t.Cleanup(func() { deviceIsMobile = prev })
}

// cellTapHarness returns a laid-out board square at (row,col) and the coordinates its taps
// report. The square is sized well above touchYCompensation so a press can be placed either
// side of the correction threshold.
func cellTapHarness(t *testing.T, row, col int) (*cellWidget, *[2]int) {
	t.Helper()
	_ = test.NewApp()
	got := &[2]int{-1, -1}
	c := newCellWidget(row, col, engine.Normal, func(r, cl int) { *got = [2]int{r, cl} })
	c.Resize(fyne.NewSize(24, 24))
	if c.Size().Height <= touchYCompensation {
		t.Fatalf("setup: cell height %.0f is not above the %d DIP compensation",
			c.Size().Height, touchYCompensation)
	}
	return c, got
}

// TestBoardTapCorrectsTouchCompensation verifies that a press near a square's top edge plays
// that square and not the one above it. The driver hit-tests touchYCompensation above the
// finger, so such a press is delivered to the square above with a position near ITS bottom
// edge; the cell must recognise the finger was below itself and report the next row down.
func TestBoardTapCorrectsTouchCompensation(t *testing.T) {
	c, got := cellTapHarness(t, 3, 5)
	onTouchDevice(t)

	// The press the driver delivers when the finger is on row 4's top edge.
	c.Tapped(&fyne.PointEvent{Position: fyne.NewPos(12, c.Size().Height-2)})

	if *got != [2]int{4, 5} {
		t.Errorf("tap near the square's bottom edge played %v, want [4 5] (the row the finger was on)", *got)
	}
}

// TestBoardTapUncorrectedWhenFingerIsInside verifies the correction does not fire for a press
// whose finger was inside the square that received it: the compensated position is far enough
// from the bottom edge that adding the compensation back stays within the square.
func TestBoardTapUncorrectedWhenFingerIsInside(t *testing.T) {
	c, got := cellTapHarness(t, 3, 5)
	onTouchDevice(t)

	c.Tapped(&fyne.PointEvent{Position: fyne.NewPos(12, 4)})

	if *got != [2]int{3, 5} {
		t.Errorf("tap inside the square played %v, want [3 5]", *got)
	}
}

// TestBoardTapBelowLastRowIsDropped verifies a press just under the board is not played onto
// it: the compensated position lands on the last row, and correcting it points off the board.
func TestBoardTapBelowLastRowIsDropped(t *testing.T) {
	c, got := cellTapHarness(t, boardDim-1, 5)
	onTouchDevice(t)

	c.Tapped(&fyne.PointEvent{Position: fyne.NewPos(12, c.Size().Height-2)})

	if *got != [2]int{-1, -1} {
		t.Errorf("press below the board played %v, want no play", *got)
	}
}

// TestBoardTapUncompensatedOnDesktop verifies the correction is touch-only: a mouse click
// carries the true pointer position, so the square that receives it is the square clicked.
func TestBoardTapUncompensatedOnDesktop(t *testing.T) {
	c, got := cellTapHarness(t, 3, 5)
	if deviceIsMobile() {
		t.Fatal("setup: the test device reports mobile, so the desktop path cannot be checked")
	}

	c.Tapped(&fyne.PointEvent{Position: fyne.NewPos(12, c.Size().Height-2)})

	if *got != [2]int{3, 5} {
		t.Errorf("click near the square's bottom edge played %v, want [3 5]", *got)
	}
}
