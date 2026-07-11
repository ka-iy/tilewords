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
func dragAbsPosition(obj fyne.CanvasObject, dragging bool, prev fyne.Position, e *fyne.DragEvent) fyne.Position {
	if e.AbsolutePosition.X != 0 || e.AbsolutePosition.Y != 0 {
		return e.AbsolutePosition
	}
	if !dragging {
		origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(obj)
		sz := obj.Size()
		return origin.Add(fyne.NewPos(sz.Width/2, sz.Height/2))
	}
	return prev.Add(fyne.NewPos(e.Dragged.DX, e.Dragged.DY))
}
