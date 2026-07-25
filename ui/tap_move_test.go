// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"
	"time"
)

// freeNonBlankSlot returns a non-blank rack slot whose tile is not staged.
func freeNonBlankSlot(gs *gameScreen) int {
	for i, tile := range gs.state.HumanRack.Tiles() {
		if !tile.IsBlank && !gs.isStagedSlot(i) {
			return i
		}
	}
	return -1
}

// TestTapToMove_PickUpAndPutDown: tapping a staged tile picks it up; a later tap on it
// puts it down (the tile stays staged). Picking up clears any rack selection.
func TestTapToMove_PickUpAndPutDown(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)

	gs.onBoardTap(7, 7)
	if gs.pickedUp != [2]int{7, 7} {
		t.Fatalf("tap did not pick up the staged tile: pickedUp=%v", gs.pickedUp)
	}
	if gs.rackSelected != -1 {
		t.Fatalf("pick-up did not clear the rack selection: %d", gs.rackSelected)
	}

	// A second tap well after the double-press window just puts the tile down.
	gs.lastPressAt = time.Now().Add(-time.Hour)
	gs.onBoardTap(7, 7)
	if gs.pickedUp != [2]int{-1, -1} {
		t.Fatalf("tapping the picked-up tile again did not put it down: %v", gs.pickedUp)
	}
	if _, ok := gs.stagedAt(7, 7); !ok {
		t.Fatal("putting a tile down recalled it instead of leaving it staged")
	}
}

// TestTapToMove_DoubleTapRecalls: a quick second tap on the just-picked-up tile recalls
// it (double-click), instead of putting it down.
func TestTapToMove_DoubleTapRecalls(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)

	gs.onBoardTap(7, 7) // pick up (records the time)
	gs.onBoardTap(7, 7) // immediate second tap → within the window → recall
	if _, ok := gs.stagedAt(7, 7); ok {
		t.Fatal("a quick double-tap did not recall the staged tile")
	}
	if gs.pickedUp != [2]int{-1, -1} {
		t.Fatalf("pickedUp not cleared after a double-tap recall: %v", gs.pickedUp)
	}
}

// TestTapToMove_MovesToEmptyCell: with a tile picked up, tapping an empty cell moves it.
func TestTapToMove_MovesToEmptyCell(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)

	gs.onBoardTap(7, 7) // pick up
	gs.onBoardTap(7, 9) // drop on an empty cell
	if _, ok := gs.stagedAt(7, 9); !ok {
		t.Fatal("tap-to-move did not relocate the tile to (7,9)")
	}
	if _, ok := gs.stagedAt(7, 7); ok {
		t.Fatal("tile still present at the original cell after a move")
	}
	if gs.pickedUp != [2]int{-1, -1} {
		t.Fatalf("pickedUp not cleared after a move: %v", gs.pickedUp)
	}
}

// TestTapToMove_SwitchPickup: tapping a second staged tile switches the pickup to it
// (neither tile moves).
func TestTapToMove_SwitchPickup(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)
	s2 := freeNonBlankSlot(gs)
	if s2 < 0 {
		t.Skip("need a second non-blank tile")
	}
	gs.onRackTap(s2)
	gs.onBoardTap(7, 8)

	gs.onBoardTap(7, 7)
	if gs.pickedUp != [2]int{7, 7} {
		t.Fatalf("did not pick up (7,7): %v", gs.pickedUp)
	}
	gs.onBoardTap(7, 8)
	if gs.pickedUp != [2]int{7, 8} {
		t.Fatalf("tapping another staged tile did not switch the pickup: %v", gs.pickedUp)
	}
	if _, ok := gs.stagedAt(7, 7); !ok {
		t.Fatal("(7,7) lost after switching pickup")
	}
	if _, ok := gs.stagedAt(7, 8); !ok {
		t.Fatal("(7,8) lost after switching pickup")
	}
}

// TestTapToMove_CancelledBySelectionAndDragAndRecall: a pickup is cancelled by selecting
// a rack tile, by starting a drag, and by recalling the tile.
func TestTapToMove_CancelledBySelectionAndDragAndRecall(t *testing.T) {
	t.Run("rack selection", func(t *testing.T) {
		gs := newPlacementHarness(t)
		stageOneTile(t, gs, 7, 7)
		gs.onBoardTap(7, 7)
		s := freeNonBlankSlot(gs)
		if s < 0 {
			t.Skip("need a free rack tile")
		}
		gs.onRackTap(s)
		if gs.pickedUp != [2]int{-1, -1} {
			t.Fatalf("rack selection did not cancel the pickup: %v", gs.pickedUp)
		}
	})

	t.Run("drag start", func(t *testing.T) {
		gs := newPlacementHarness(t)
		stageOneTile(t, gs, 7, 7)
		gs.onBoardTap(7, 7)
		gs.cells[7*boardDim+7].Dragged(dragEventAt(5, 5))
		if gs.pickedUp != [2]int{-1, -1} {
			t.Fatalf("starting a drag did not cancel the pickup: %v", gs.pickedUp)
		}
		gs.cells[7*boardDim+7].DragEnd()
	})

	t.Run("recall", func(t *testing.T) {
		gs := newPlacementHarness(t)
		stageOneTile(t, gs, 7, 7)
		gs.onBoardTap(7, 7)
		gs.onBoardDoubleTap(7, 7) // recall to rack
		if gs.pickedUp != [2]int{-1, -1} {
			t.Fatalf("recall did not clear the pickup: %v", gs.pickedUp)
		}
	})
}
