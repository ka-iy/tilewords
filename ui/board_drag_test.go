// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"
)

// stageOneTile stages the first non-blank rack tile at (row,col) and returns its slot.
func stageOneTile(t *testing.T, gs *gameScreen, row, col int) int {
	t.Helper()
	s := firstNonBlankSlot(gs)
	if s < 0 {
		t.Skip("no non-blank tile in rack")
	}
	gs.onRackTap(s)
	gs.onBoardTap(row, col)
	if _, ok := gs.stagedAt(row, col); !ok {
		t.Fatalf("setup: tile not staged at (%d,%d)", row, col)
	}
	return s
}

// TestBoardDoubleTap_RecallsStagedOnly: double-clicking a staged tile recalls it; an
// empty cell is a no-op.
func TestBoardDoubleTap_RecallsStagedOnly(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)

	gs.onBoardDoubleTap(0, 0) // empty cell
	if len(gs.staged) != 1 {
		t.Fatalf("double-tap on empty cell changed staging: %d", len(gs.staged))
	}
	gs.onBoardDoubleTap(7, 7) // staged tile
	if len(gs.staged) != 0 {
		t.Fatalf("double-tap on staged tile did not recall it: %d", len(gs.staged))
	}
}

// TestBoardSingleTap_DoesNotRecall: a single tap on a staged tile leaves it placed
// (recall is via double-click or drag-off now).
func TestBoardSingleTap_DoesNotRecall(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)
	gs.onBoardTap(7, 7)
	if len(gs.staged) != 1 {
		t.Fatalf("single tap recalled a staged tile: staged=%d", len(gs.staged))
	}
}

// TestMoveStagedTile moves a staged tile to a free cell and refuses an occupied one.
func TestMoveStagedTile(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)

	st, _ := gs.stagedAt(7, 7)
	gs.moveStagedTile(st, 7, 8)
	if _, ok := gs.stagedAt(7, 8); !ok {
		t.Fatal("tile did not move to (7,8)")
	}
	if _, ok := gs.stagedAt(7, 7); ok {
		t.Fatal("tile still present at (7,7) after move")
	}

	// Stage a second tile and try to move the first onto it: the move must be refused.
	s2 := -1
	for i, tile := range gs.state.HumanRack.Tiles() {
		if !tile.IsBlank && !gs.isStagedSlot(i) {
			s2 = i
			break
		}
	}
	if s2 < 0 {
		return
	}
	gs.onRackTap(s2)
	gs.onBoardTap(7, 9)
	moving, _ := gs.stagedAt(7, 8)
	gs.moveStagedTile(moving, 7, 9) // (7,9) is occupied
	if _, ok := gs.stagedAt(7, 8); !ok {
		t.Fatal("refused move lost the tile from its original cell")
	}
}

// TestBoardDrag_OffBoardRecalls: dragging a staged tile to a release point that is not
// a board cell recalls it to the rack, and the ghost shows during the drag and hides
// after. The release point is above and left of the board's top-left corner, so it
// resolves to no cell whether or not the board reserves a label gutter.
func TestBoardDrag_OffBoardRecalls(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)

	cell := gs.cells[7*boardDim+7]
	cell.Dragged(dragEventAt(-50, -50))
	if !gs.ghost.Visible() {
		t.Fatal("ghost should be visible during a board-tile drag")
	}
	cell.DragEnd()
	if gs.ghost.Visible() {
		t.Fatal("ghost should be hidden after the drag ends")
	}
	if len(gs.staged) != 0 {
		t.Fatalf("dragging a staged tile off the board did not recall it: staged=%d", len(gs.staged))
	}
}

// TestBoardDrag_IgnoresUnstagedCell: a drag that begins on an empty cell does not start
// a tile drag (no ghost) and resolves as a tap.
func TestBoardDrag_IgnoresUnstagedCell(t *testing.T) {
	gs := newPlacementHarness(t)
	cell := gs.cells[3*boardDim+3]
	cell.Dragged(dragEventAt(5, 5))
	if gs.ghost.Visible() {
		t.Fatal("ghost should not show when dragging from an empty cell")
	}
	if gs.dragBoardSrc != [2]int{-1, -1} {
		t.Fatalf("empty-cell drag set a board drag source: %v", gs.dragBoardSrc)
	}
}
