// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// mobileDragEvent returns the drag event the mobile driver delivers partway through a gesture.
// Position there is the position within the receiving widget offset by the movement since the
// previous event, so a real drag reports a position outside the widget's own bounds; nothing may
// treat that as a reason to ignore the drag.
func mobileDragEvent(dx, dy float32) *fyne.DragEvent {
	return &fyne.DragEvent{
		PointEvent: fyne.PointEvent{Position: fyne.NewPos(22+dx, 22+dy)},
		Dragged:    fyne.NewDelta(dx, dy),
	}
}

// TestGestureOwnerRoutesDragBackToTheTile is the case that broke grabbing a tile: the driver
// resolves who owns a gesture at the first move it processes — a median of 75 ms after the touch
// went down — and it resolves it by hit-testing wherever the pointer is by then. A pointer heading
// for the board has left the 44 DIP tile, so the drag is delivered to whatever is under it now.
// The tile must still receive it.
func TestGestureOwnerRoutesDragBackToTheTile(t *testing.T) {
	gs := newPlacementHarness(t)
	slot := firstNonBlankSlot(gs)
	if slot < 0 {
		t.Skip("no non-blank tile in rack")
	}

	// The touch goes down on the tile, so the tile owns the gesture.
	gs.humanRack[slot].TouchDown(&mobile.TouchEvent{})

	// The driver then hands the drag to a board cell, the pointer having moved on.
	gs.cells[7*boardDim+7].Dragged(mobileDragEvent(0, -180))

	if !gs.ghost.Visible() {
		t.Error("the tile did not lift: the drag was not routed back to it")
	}
	if gs.dragRackSrc != slot {
		t.Errorf("dragRackSrc = %d, want %d", gs.dragRackSrc, slot)
	}
	if gs.dragBoardSrc != [2]int{-1, -1} {
		t.Errorf("dragBoardSrc = %v, want (-1,-1): the cell acted on a drag it did not own",
			gs.dragBoardSrc)
	}
}

// TestGestureOwnerLeavesUnclaimedDragsAlone verifies a widget that receives a drag with no claim
// outstanding handles it itself, so ordinary board and rack behaviour is untouched.
func TestGestureOwnerLeavesUnclaimedDragsAlone(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)

	gs.cells[7*boardDim+7].Dragged(mobileDragEvent(20, 20))

	if gs.dragBoardSrc != [2]int{7, 7} {
		t.Errorf("dragBoardSrc = %v, want (7,7)", gs.dragBoardSrc)
	}
}

// TestGestureOwnerReleasedOnTouchUp verifies a completed gesture cannot redirect a later one.
func TestGestureOwnerReleasedOnTouchUp(t *testing.T) {
	gs := newPlacementHarness(t)
	slot := firstNonBlankSlot(gs)
	if slot < 0 {
		t.Skip("no non-blank tile in rack")
	}

	gs.humanRack[slot].TouchDown(&mobile.TouchEvent{})
	if gs.gesture.current() == nil {
		t.Fatal("a touch on a tile did not claim the gesture")
	}

	gs.humanRack[slot].TouchUp(&mobile.TouchEvent{})

	if gs.gesture.current() != nil {
		t.Error("the claim outlived the gesture")
	}
}

// TestGestureOwnerSurvivesTouchCancel verifies the claim is kept when the pointer leaves the tile.
// The driver cancels the touch the moment that happens — measured about 100 ms into every drag,
// successful ones included — which is precisely while the gesture is still running.
func TestGestureOwnerSurvivesTouchCancel(t *testing.T) {
	gs := newPlacementHarness(t)
	slot := firstNonBlankSlot(gs)
	if slot < 0 {
		t.Skip("no non-blank tile in rack")
	}

	gs.humanRack[slot].TouchDown(&mobile.TouchEvent{})
	gs.humanRack[slot].TouchCancel(&mobile.TouchEvent{})

	if gs.gesture.current() == nil {
		t.Error("TouchCancel dropped the claim, so the rest of the gesture would be lost")
	}
}

// TestCPURackSlotMakesNoClaim verifies the CPU's rack, which has no drag handlers, does not claim a
// gesture — a touch there must leave the page free to pan.
func TestCPURackSlotMakesNoClaim(t *testing.T) {
	gs := newPlacementHarness(t)

	gs.cpuRack[3].TouchDown(&mobile.TouchEvent{})

	if gs.gesture.current() != nil {
		t.Error("the CPU rack claimed a gesture it cannot use")
	}
}

// newTestRouter returns a page router over a scrollable page, with its own ownership record.
func newTestRouter(t *testing.T) (*pageDragRouter, *container.Scroll, *gestureOwner) {
	t.Helper()
	test.NewTempApp(t)

	scroll := container.NewVScroll(widget.NewLabel(strings.Repeat("a line of page\n", 60)))
	scroll.Resize(fyne.NewSize(352, 200))
	owner := &gestureOwner{}
	return newPageDragRouter(scroll, owner), scroll, owner
}

// TestPageDragRouterPansWhenUnclaimed verifies the router still pans the page for a gesture that
// began on page background, which is how the page is scrolled by finger.
func TestPageDragRouterPansWhenUnclaimed(t *testing.T) {
	router, scroll, _ := newTestRouter(t)

	router.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -40)})

	if scroll.Offset.Y != 40 {
		t.Errorf("page offset = %.1f, want 40: the page did not pan", scroll.Offset.Y)
	}
}

// TestPageDragRouterDefersWhenClaimed verifies a claimed gesture goes to its owner and the page
// stays put, which is what stops a grab turning into a pan.
func TestPageDragRouterDefersWhenClaimed(t *testing.T) {
	router, scroll, owner := newTestRouter(t)
	claimant := &recordingDraggable{}
	owner.claim(claimant)

	router.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -40)})
	router.DragEnd()

	if claimant.drags != 1 || claimant.ends != 1 {
		t.Errorf("owner got %d drags and %d ends, want 1 and 1", claimant.drags, claimant.ends)
	}
	if scroll.Offset.Y != 0 {
		t.Errorf("page offset = %.1f, want 0: the page panned on a claimed gesture", scroll.Offset.Y)
	}
	if owner.current() != nil {
		t.Error("DragEnd left the claim in place")
	}
}

// TestPageDragRouterTouchDownClearsStaleClaim verifies a touch landing on page background clears an
// earlier claim, since anything wanting the gesture would have received the touch first.
func TestPageDragRouterTouchDownClearsStaleClaim(t *testing.T) {
	router, scroll, owner := newTestRouter(t)
	owner.claim(&recordingDraggable{})

	router.TouchDown(&mobile.TouchEvent{})
	router.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -40)})

	if scroll.Offset.Y != 40 {
		t.Errorf("page offset = %.1f, want 40: a stale claim captured a page gesture",
			scroll.Offset.Y)
	}
}

// recordingDraggable counts the drag events routed to it.
type recordingDraggable struct {
	drags int
	ends  int
}

func (r *recordingDraggable) Dragged(*fyne.DragEvent) { r.drags++ }
func (r *recordingDraggable) DragEnd()                { r.ends++ }

// TestCPURackDragPansThePage verifies the CPU's rack row is not a dead zone for scrolling. Its slots
// are draggable as a type but have nothing to move, so a drag the driver hands them must reach the
// page rather than be absorbed.
func TestCPURackDragPansThePage(t *testing.T) {
	gs := newPlacementHarness(t)
	if gs.page == nil {
		t.Skip("arrangement has no page scroll")
	}
	page := gs.pageScrollForTest()
	if page == nil {
		t.Skip("page fits its viewport, so there is nothing to scroll")
	}
	before := page.Offset.Y

	gs.cpuRack[3].Dragged(mobileDragEvent(0, -30))

	if page.Offset.Y <= before {
		t.Errorf("page offset = %.1f, want more than %.1f: a drag on the CPU rack was swallowed",
			page.Offset.Y, before)
	}
	if gs.ghost.Visible() {
		t.Error("a drag on the CPU rack lifted a tile")
	}
}

// TestPaneWithNoTravelPansThePage verifies a panel too short to scroll hands the drag on. Measured
// on a device the history pane was 280 tall around 81 of content — no travel at all — yet it
// absorbed every swipe that began on it, so the page behind it could not be moved.
func TestPaneWithNoTravelPansThePage(t *testing.T) {
	gs := newPlacementHarness(t)
	if gs.page == nil {
		t.Skip("arrangement has no page scroll")
	}
	page := gs.pageScrollForTest()
	if page == nil {
		t.Skip("page fits its viewport, so there is nothing to scroll")
	}
	// A pane whose content is far shorter than its viewport.
	gs.historyScroll.Resize(fyne.NewSize(352, 280))
	if gs.historyScroll.Content.MinSize().Height >= gs.historyScroll.Size().Height {
		t.Skip("history pane has travel; this test needs one with none")
	}
	panner := newDragScroller(gs.historyScroll, nil, nil, gs.panPage)
	before := page.Offset.Y

	panner.Dragged(mobileDragEvent(0, -30))

	if page.Offset.Y <= before {
		t.Errorf("page offset = %.1f, want more than %.1f: a pane with no travel swallowed the drag",
			page.Offset.Y, before)
	}
}

// pageScrollForTest returns the scroll that pans the page, or nil when the arrangement has none or
// the page currently fits its viewport.
func (gs *gameScreen) pageScrollForTest() *container.Scroll {
	if gs.page == nil || gs.page.router == nil {
		return nil
	}
	return gs.page.router.target
}
