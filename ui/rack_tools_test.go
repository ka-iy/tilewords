package ui

import (
	"sort"
	"testing"

	"tilewords/engine"
)

// sortedRackLetters returns the human rack's letters sorted, i.e. its multiset.
func sortedRackLetters(gs *gameScreen) string {
	tiles := gs.state.HumanRack.Tiles()
	b := make([]byte, len(tiles))
	for i, t := range tiles {
		b[i] = t.Letter
	}
	sort.Slice(b, func(i, j int) bool { return b[i] < b[j] })
	return string(b)
}

// TestMoveIndex covers the reindexing used to keep staged tiles and the selection
// aligned with engine.Rack.MoveTile.
func TestMoveIndex(t *testing.T) {
	cases := []struct{ i, f, to, want int }{
		{0, 0, 2, 2}, {1, 0, 2, 0}, {2, 0, 2, 1}, {3, 0, 2, 3}, // move 0->2
		{3, 3, 1, 1}, {0, 3, 1, 0}, {1, 3, 1, 2}, {2, 3, 1, 3}, // move 3->1
		{4, 4, 4, 4}, // no-op
	}
	for _, c := range cases {
		if got := moveIndex(c.i, c.f, c.to); got != c.want {
			t.Errorf("moveIndex(%d,%d,%d)=%d want %d", c.i, c.f, c.to, got, c.want)
		}
	}
}

// TestReorderRack_TracksDragSource: during a live drag-reorder the lifted tile's slot
// (dragRackSrc) must follow the tile through the shift, so the floating ghost — drawn from
// HumanRack.Tiles()[dragRackSrc] — keeps hovering over the same tile as the gap moves.
func TestReorderRack_TracksDragSource(t *testing.T) {
	gs := newRackHarness(t)

	const from, to = 1, 4
	lifted := gs.state.HumanRack.Tiles()[from]
	gs.dragRackSrc = from // simulate a lift in progress on slot `from`

	gs.reorderRack(from, to)

	if gs.dragRackSrc != to {
		t.Fatalf("dragRackSrc = %d, want %d (must follow the lifted tile)", gs.dragRackSrc, to)
	}
	if got := gs.state.HumanRack.Tiles()[gs.dragRackSrc]; got != lifted {
		t.Errorf("tile at dragRackSrc = %+v, want the lifted tile %+v", got, lifted)
	}
}

// TestReorderRack_RemapsStagedAndSelection: reordering the rack keeps a staged tile
// and the current selection pointing at the same tiles after the index shift.
func TestReorderRack_RemapsStagedAndSelection(t *testing.T) {
	gs := newPlacementHarness(t)
	s := firstNonBlankSlot(gs)
	if s < 0 {
		t.Skip("no non-blank tile in rack")
	}
	gs.onRackTap(s)
	gs.onBoardTap(7, 7)
	if len(gs.staged) != 1 || gs.staged[0].FromRackIdx != s {
		t.Fatalf("setup: staged=%+v", gs.staged)
	}
	stagedVal := gs.staged[0].Tile
	other := (s + 1) % engine.MaxRackSize
	gs.rackSelected = other

	const from, to = 3, 0 // shifts indices in [0,3) up by one
	gs.reorderRack(from, to)

	if want := moveIndex(s, from, to); gs.staged[0].FromRackIdx != want {
		t.Errorf("staged FromRackIdx = %d, want %d", gs.staged[0].FromRackIdx, want)
	}
	if gs.staged[0].Tile != stagedVal {
		t.Errorf("reorder altered the staged tile value")
	}
	if want := moveIndex(other, from, to); gs.rackSelected != want {
		t.Errorf("rackSelected = %d, want %d", gs.rackSelected, want)
	}
	if gs.state.HumanRack.Count() != engine.MaxRackSize {
		t.Errorf("reorder changed rack count: %d", gs.state.HumanRack.Count())
	}
}

// TestDoRecallAll returns every staged tile to the rack.
func TestDoRecallAll(t *testing.T) {
	gs := newPlacementHarness(t)
	s := firstNonBlankSlot(gs)
	if s < 0 {
		t.Skip("no non-blank tile in rack")
	}
	gs.onRackTap(s)
	gs.onBoardTap(7, 7)
	if len(gs.staged) != 1 {
		t.Fatalf("setup: staged=%d", len(gs.staged))
	}
	gs.doRecallAll()
	if len(gs.staged) != 0 {
		t.Fatalf("doRecallAll left staged=%d", len(gs.staged))
	}
}

// TestDoShuffle recalls staged tiles and preserves the rack's tile multiset.
func TestDoShuffle(t *testing.T) {
	gs := newPlacementHarness(t)
	before := sortedRackLetters(gs)
	s := firstNonBlankSlot(gs)
	if s < 0 {
		t.Skip("no non-blank tile in rack")
	}
	gs.onRackTap(s)
	gs.onBoardTap(7, 7)

	gs.doShuffle()
	if len(gs.staged) != 0 {
		t.Fatalf("doShuffle did not recall staged tiles: %d", len(gs.staged))
	}
	if gs.state.HumanRack.Count() != engine.MaxRackSize {
		t.Fatalf("doShuffle changed rack count: %d", gs.state.HumanRack.Count())
	}
	if after := sortedRackLetters(gs); after != before {
		t.Fatalf("doShuffle changed the rack multiset: %q -> %q", before, after)
	}
}

// TestPlaceFromRack stages a tile on an empty cell and refuses an occupied one.
func TestPlaceFromRack(t *testing.T) {
	gs := newPlacementHarness(t)
	tiles := gs.state.HumanRack.Tiles()
	s1, s2 := -1, -1
	for i, tile := range tiles {
		if tile.IsBlank {
			continue
		}
		if s1 < 0 {
			s1 = i
		} else {
			s2 = i
			break
		}
	}
	if s1 < 0 || s2 < 0 {
		t.Skip("need two non-blank tiles")
	}
	gs.placeFromRack(s1, 7, 7)
	if len(gs.staged) != 1 || gs.staged[0].FromRackIdx != s1 {
		t.Fatalf("placeFromRack staged=%+v", gs.staged)
	}
	gs.placeFromRack(s2, 7, 7) // (7,7) now holds a staged tile
	if len(gs.staged) != 1 {
		t.Fatalf("placeFromRack onto an occupied cell placed anyway: staged=%d", len(gs.staged))
	}
}

// TestTurnCue: on the human's turn the rack label is green and the play icon shows;
// otherwise the label is the idle colour and the play slot is blank.
func TestTurnCue(t *testing.T) {
	gs := newPlacementHarness(t) // forced to the human's turn
	gs.refresh()
	if gs.rackLabel.Color != colorTurnYou {
		t.Errorf("human turn: rack label colour = %v, want green", gs.rackLabel.Color)
	}
	if gs.playIcon.Resource != playIconResource {
		t.Error("human turn: play icon should be shown")
	}

	gs.aiThinking = true
	gs.refresh()
	if gs.rackLabel.Color != colorText {
		t.Errorf("AI turn: rack label colour = %v, want idle", gs.rackLabel.Color)
	}
	if gs.playIcon.Resource != blankIconResource {
		t.Error("AI turn: play icon should be blank")
	}
}
