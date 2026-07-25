package ui

import (
	"testing"

	"fyne.io/fyne/v2"
)

// TestClearDragState_ResetsWidgetTracking verifies clearDragState resets the widgets' own drag
// tracking as well as the controller's drag sources. On touch a DragEvent carries no absolute
// position, so the pointer is followed by accumulating deltas from where the gesture began — a
// widget left believing it is mid-drag seeds the next gesture from the abandoned one's end
// point, putting the ghost away from the finger and the drop on the wrong cell.
func TestClearDragState_ResetsWidgetTracking(t *testing.T) {
	gs := &gameScreen{}
	for i := range gs.cells {
		gs.cells[i] = newCellWidget(i/boardDim, i%boardDim, 0, nil)
	}
	for i := range gs.humanRack {
		gs.humanRack[i] = newRackSlotWidget(i, nil)
	}

	// Simulate a gesture interrupted after Dragged but before DragEnd.
	cell := gs.cells[3*boardDim+4]
	cell.dragging = true
	cell.dragAbs = fyne.NewPos(120, 240)
	slot := gs.humanRack[2]
	slot.dragging = true
	slot.dragAbs = fyne.NewPos(80, 400)

	gs.clearDragState()

	if gs.dragRackSrc != -1 || gs.dragBoardSrc != [2]int{-1, -1} {
		t.Errorf("controller drag sources not cleared: rack=%d board=%v", gs.dragRackSrc, gs.dragBoardSrc)
	}
	if cell.dragging {
		t.Error("cell still believes it is mid-drag; the next gesture would resume from the abandoned one")
	}
	if cell.dragAbs != (fyne.Position{}) {
		t.Errorf("cell dragAbs = %v, want zero", cell.dragAbs)
	}
	if slot.dragging {
		t.Error("rack slot still believes it is mid-drag")
	}
	if slot.dragAbs != (fyne.Position{}) {
		t.Errorf("rack slot dragAbs = %v, want zero", slot.dragAbs)
	}
}

// TestClearDragState_BeforeBuild verifies clearDragState is safe on a screen whose widgets have
// not been created yet, since it is called from turn boundaries that a test or an early tap can
// reach before build().
func TestClearDragState_BeforeBuild(t *testing.T) {
	gs := &gameScreen{}
	gs.clearDragState() // must not panic on the nil widget arrays
	if gs.dragRackSrc != -1 {
		t.Errorf("dragRackSrc = %d, want -1", gs.dragRackSrc)
	}
}
