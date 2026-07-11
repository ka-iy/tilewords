package ui

import (
	"testing"

	"fyne.io/fyne/v2"

	"squabble/engine"
)

// Board cells and rack slots are tappable and draggable. Cells deliberately do NOT
// implement fyne.DoubleTappable — that would delay every tap; double-click recall is
// detected in the controller instead.
var (
	_ fyne.Tappable  = (*cellWidget)(nil)
	_ fyne.Draggable = (*cellWidget)(nil)
	_ fyne.Tappable  = (*rackSlotWidget)(nil)
	_ fyne.Draggable = (*rackSlotWidget)(nil)
)

// dragEventAt builds a drag event whose pointer is at the given absolute position.
func dragEventAt(x, y float32) *fyne.DragEvent {
	return &fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: fyne.NewPos(x, y)}}
}

// TestCellWidget_DragEndReportsPosition: a cell drag reports its coordinates and final
// pointer position to the controller, and only after a real drag.
func TestCellWidget_DragEndReportsPosition(t *testing.T) {
	gotR, gotC := -1, -1
	gotPos := fyne.Position{}
	c := newCellWidget(3, 5, engine.Normal, nil)
	c.onDragEnd = func(r, col int, abs fyne.Position) { gotR, gotC, gotPos = r, col, abs }

	c.DragEnd() // no preceding Dragged — must not report
	if gotR != -1 {
		t.Fatalf("cell DragEnd reported without a drag: (%d,%d)", gotR, gotC)
	}

	c.Dragged(dragEventAt(20, 40))
	c.DragEnd()
	if gotR != 3 || gotC != 5 || gotPos != fyne.NewPos(20, 40) {
		t.Fatalf("cell DragEnd reported (%d,%d,%v), want (3,5,(20,40))", gotR, gotC, gotPos)
	}
}

// TestRackSlot_DragEndReportsPosition: a rack drag reports its slot and final pointer
// position to the controller (only after a real drag, never on a bare DragEnd).
func TestRackSlot_DragEndReportsPosition(t *testing.T) {
	gotIdx := -1
	gotPos := fyne.Position{}
	s := newRackSlotWidget(4, nil)
	s.onDragEnd = func(idx int, abs fyne.Position) { gotIdx, gotPos = idx, abs }

	s.DragEnd() // no preceding Dragged — must not report
	if gotIdx != -1 {
		t.Fatalf("DragEnd reported without a drag in progress: idx=%d", gotIdx)
	}

	s.Dragged(dragEventAt(12, 34))
	s.DragEnd()
	if gotIdx != 4 || gotPos != fyne.NewPos(12, 34) {
		t.Fatalf("DragEnd reported (idx=%d, pos=%v), want (4, (12,34))", gotIdx, gotPos)
	}
}

// TestRepro_JitteryTapPlacesViaDrag reproduces the original bug: a press-and-release
// with slight movement arrives as a drag, not a tap. The rack drag (ending on its own
// slot) must still select the tile, and the board drag must still place it.
func TestRepro_JitteryTapPlacesViaDrag(t *testing.T) {
	gs := newPlacementHarness(t)
	slot := firstNonBlankSlot(gs)
	if slot < 0 {
		t.Skip("no non-blank tile in rack")
	}

	// Jittery tap on the rack tile (delivered as a drag ending on its own slot).
	gs.humanRack[slot].Dragged(dragEventAt(1, 1))
	gs.humanRack[slot].DragEnd()
	if gs.rackSelected != slot {
		t.Fatalf("jittery rack tap did not select: rackSelected=%d want %d", gs.rackSelected, slot)
	}

	// Jittery tap on an empty board cell (delivered as a drag) must place the tile.
	cell := gs.cells[7*boardDim+7]
	cell.Dragged(dragEventAt(1, 1))
	cell.DragEnd()
	if len(gs.staged) != 1 {
		t.Fatalf("jittery board tap did not place a tile: staged=%d want 1", len(gs.staged))
	}
}
