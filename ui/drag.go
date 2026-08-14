// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import "fyne.io/fyne/v2"

// dragAbsPosition returns the absolute pointer position for a drag event, tracking it
// reliably across platforms.
//
// The desktop driver fills in DragEvent.AbsolutePosition on every drag event. The mobile
// driver, however, leaves AbsolutePosition at (0,0) for every drag event except the last
// (the release), supplying only the per-event movement delta. Relying on AbsolutePosition
// there makes the drag ghost jump to the top-left corner and, if the final positioned
// event is missed (as with the emulator's mouse), leaves the drop hit-test at (0,0) so a
// rack reorder or board move silently fails.
//
// So: use AbsolutePosition when the driver provides it; otherwise seed the position at the
// dragged object's centre on the first event and advance it by the delta thereafter.
//
// The one positioned event the mobile driver does send carries the same upward bias as every
// other touch coordinate it reports (see touchYCompensation), while the delta-tracked positions
// do not — deltas are differences, so the bias cancels. Taking that event at face value would
// therefore move the drop hit-test 8 DIP above the finger for the whole of the last event,
// landing a dragged tile one row above the cell it was released over whenever the finger is in a
// cell's top 8 DIP. It is compensated back here so both paths report the same point.
func dragAbsPosition(obj fyne.CanvasObject, dragging bool, prev fyne.Position, e *fyne.DragEvent) fyne.Position {
	if e.AbsolutePosition.X != 0 || e.AbsolutePosition.Y != 0 {
		pos := e.AbsolutePosition
		if deviceIsMobile() {
			pos.Y += touchYCompensation
		}
		return pos
	}
	if !dragging {
		origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(obj)
		sz := obj.Size()
		return origin.Add(fyne.NewPos(sz.Width/2, sz.Height/2))
	}
	return prev.Add(fyne.NewPos(e.Dragged.DX, e.Dragged.DY))
}
