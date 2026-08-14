// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
)

// TestDragAbsPositionCompensatesTheDriverBias verifies the one positioned drag event the mobile
// driver sends is corrected the same way every other touch coordinate is.
//
// That event is reported 8 DIP above the finger, while the delta-tracked positions used for the
// rest of the drag are not (a delta is a difference, so the bias cancels). Left uncorrected, the
// drop hit-test lands a row above the cell the tile was released over whenever the finger is in
// a cell's top 8 DIP, and the ghost the player was watching — drawn from the delta-tracked
// position — visibly disagrees with where the tile ends up.
func TestDragAbsPositionCompensatesTheDriverBias(t *testing.T) {
	test.NewTempApp(t)
	obj := canvas.NewRectangle(color.White)
	obj.Resize(fyne.NewSize(40, 40))

	const reported = 300
	ev := &fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: fyne.NewPos(100, reported)}}

	got := dragAbsPosition(obj, true, fyne.NewPos(0, 0), ev)

	want := float32(reported)
	if deviceIsMobile() {
		want += touchYCompensation
	}
	if got.Y != want {
		t.Errorf("drop position Y = %.1f, want %.1f (driver reported %.1f, mobile=%v)",
			got.Y, want, float32(reported), deviceIsMobile())
	}
	if got.X != 100 {
		t.Errorf("drop position X = %.1f, want 100: only the vertical axis is biased", got.X)
	}
}
