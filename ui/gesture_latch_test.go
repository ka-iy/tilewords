// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/mobile"
)

// TestRouterKeepsGestureWithItsOwnStarter guards the boundary between one gesture and the next.
//
// The driver keeps replaying a fast release as decaying drag events after the finger is gone, so a
// second touch can claim while the first gesture is still delivering. Those replayed events belong
// to the gesture that started them: a flick on the page must go on panning the page even though a
// rack slot has since claimed the next gesture. Otherwise the slot is handed a drag the player
// never made, at the coordinates where the flick was released.
func TestRouterKeepsGestureWithItsOwnStarter(t *testing.T) {
	router, scroll, owner := newTestRouter(t)

	// A flick starts on page background: nothing claims it, so the router pans.
	router.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -40)})
	if scroll.Offset.Y != 40 {
		t.Fatalf("setup: page offset = %.1f, want 40", scroll.Offset.Y)
	}

	// The finger lifts and a second touch claims a rack slot while the replay is still running.
	slot := &recordingDraggable{}
	owner.claim(slot)

	// The replay's remaining events, and its terminal DragEnd, still belong to the flick.
	router.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -10)})
	router.DragEnd()

	if slot.drags != 0 || slot.ends != 0 {
		t.Errorf("the newly claimed slot received %d drags and %d ends from the previous gesture; "+
			"a replayed flick would lift and drop a tile the player never dragged",
			slot.drags, slot.ends)
	}
	if scroll.Offset.Y != 50 {
		t.Errorf("page offset = %.1f, want 50: the flick stopped panning once another widget claimed",
			scroll.Offset.Y)
	}
	if owner.current() != slot {
		t.Error("the finished gesture cleared a claim belonging to the gesture after it")
	}
}

// TestRouterEndsTheGestureItLatched verifies a routed gesture's DragEnd reaches the widget that
// owned it, and that the claim is cleared afterwards so it cannot redirect a later gesture.
func TestRouterEndsTheGestureItLatched(t *testing.T) {
	router, _, owner := newTestRouter(t)

	slot := &recordingDraggable{}
	owner.claim(slot)

	router.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -10)})
	router.DragEnd()

	if slot.drags != 1 || slot.ends != 1 {
		t.Errorf("owner received %d drags and %d ends, want 1 and 1", slot.drags, slot.ends)
	}
	if owner.current() != nil {
		t.Error("the claim outlived its gesture and would redirect the next one")
	}
}

// TestRackSlotDragEndReleasesItsClaim guards the only end-of-gesture a dragging widget sees.
//
// The driver delivers no TouchUp for a gesture that became a drag, so TouchUp cannot be what ends
// the claim. A leaked claim redirects the next gesture to a widget the player is no longer
// touching, which moves or recalls a tile they never grabbed.
func TestRackSlotDragEndReleasesItsClaim(t *testing.T) {
	gs := newPlacementHarness(t)
	slot := gs.humanRack[0]

	slot.TouchDown(&mobile.TouchEvent{})
	if gs.gesture.current() != slot {
		t.Fatal("setup: the slot did not claim the gesture")
	}

	slot.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, 20)})
	slot.DragEnd()

	if gs.gesture.current() != nil {
		t.Error("the rack slot kept its claim after the drag ended; the next gesture would be " +
			"delivered to it instead of to the widget the player touched")
	}
}

// TestBoardCellDragEndReleasesItsClaim is TestRackSlotDragEndReleasesItsClaim for a board cell,
// which claims gestures on the same terms and leaks them the same way.
func TestBoardCellDragEndReleasesItsClaim(t *testing.T) {
	gs := newPlacementHarness(t)
	cell := gs.cells[7*boardDim+7]

	cell.TouchDown(&mobile.TouchEvent{})
	if gs.gesture.current() != cell {
		t.Fatal("setup: the cell did not claim the gesture")
	}

	cell.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, 20)})
	cell.DragEnd()

	if gs.gesture.current() != nil {
		t.Error("the board cell kept its claim after the drag ended; the next gesture would be " +
			"delivered to it instead of to the widget the player touched")
	}
}
