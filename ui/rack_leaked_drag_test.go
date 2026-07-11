package ui

import (
	"testing"

	"squabble/engine"
)

// TestRackDisplay_LeakedDragSourceRecovers reproduces the reported "rack is short one
// tile" bug: a rack drag whose DragEnd never fired (an interrupted touch gesture)
// leaves dragRackSrc pointing at a slot, which refresh() then renders as a permanent
// empty slot even though the engine rack is full. A subsequent tap must clear it.
func TestRackDisplay_LeakedDragSourceRecovers(t *testing.T) {
	gs := newRackHarness(t)
	if gs.state.HumanRack.Count() != engine.MaxRackSize {
		t.Fatalf("precondition: rack=%d want %d", gs.state.HumanRack.Count(), engine.MaxRackSize)
	}

	// Simulate the leaked drag source.
	gs.dragRackSrc = 3
	if got := displayedRackCount(gs); got != engine.MaxRackSize-1 {
		t.Fatalf("precondition: leaked dragRackSrc should hide one tile; displayed=%d", got)
	}

	// Any subsequent tap must clear the leaked source and restore the full rack.
	gs.onRackTap(0)
	if gs.dragRackSrc != -1 {
		t.Errorf("onRackTap left leaked dragRackSrc=%d", gs.dragRackSrc)
	}
	if got := displayedRackCount(gs); got != engine.MaxRackSize {
		t.Errorf("after tap, displayed rack=%d want %d (phantom empty slot remains)", got, engine.MaxRackSize)
	}
}

// TestRackDisplay_RecallAllClearsDragState verifies recallAll (used by pass/undo/
// shuffle/recall) also clears any leaked drag state so no phantom slot survives.
func TestRackDisplay_RecallAllClearsDragState(t *testing.T) {
	gs := newRackHarness(t)
	gs.dragRackSrc = 4
	gs.dragBoardSrc = [2]int{7, 7}

	gs.recallAll()

	if gs.dragRackSrc != -1 {
		t.Errorf("recallAll left dragRackSrc=%d, want -1", gs.dragRackSrc)
	}
	if gs.dragBoardSrc != [2]int{-1, -1} {
		t.Errorf("recallAll left dragBoardSrc=%v, want {-1,-1}", gs.dragBoardSrc)
	}
	if got := displayedRackCount(gs); got != engine.MaxRackSize {
		t.Errorf("after recallAll, displayed rack=%d want %d", got, engine.MaxRackSize)
	}
}
