package engine

import "testing"

func mustPlaceTile(t *testing.T, b *Board, row, col int, tile Tile) {
	t.Helper()
	if err := b.Place(row, col, tile); err != nil {
		t.Fatalf("Place(%d,%d): %v", row, col, err)
	}
}

// TestPlayCommand_RefillsFullyWhenWordReusesBoardTile reproduces the reported bug:
// playing a word that reuses an existing board tile places fewer tiles than the word
// length, and the rack must still refill back to MaxRackSize.
func TestPlayCommand_RefillsFullyWhenWordReusesBoardTile(t *testing.T) {
	board := newFlatBoard()
	// Existing word CAT across row 7, cols 6-8 (as if from a prior turn).
	mustPlaceTile(t, board, 7, 6, Tile{Letter: 'C', Points: 3})
	mustPlaceTile(t, board, 7, 7, Tile{Letter: 'A', Points: 1})
	mustPlaceTile(t, board, 7, 8, Tile{Letter: 'T', Points: 1})

	rack := &Rack{tiles: []Tile{
		{Letter: 'O', Points: 1}, {Letter: 'R', Points: 1}, {Letter: 'D', Points: 2},
		{Letter: 'B', Points: 3}, {Letter: 'G', Points: 2}, {Letter: 'M', Points: 3},
		{Letter: 'P', Points: 3},
	}}
	if rack.Count() != MaxRackSize {
		t.Fatalf("precondition: rack has %d tiles, want %d", rack.Count(), MaxRackSize)
	}

	bag := newTestBag([]Tile{
		{Letter: 'E', Points: 1}, {Letter: 'I', Points: 1}, {Letter: 'A', Points: 1},
		{Letter: 'N', Points: 1}, {Letter: 'S', Points: 1},
	})

	state := &GameState{
		Board:       board,
		HumanRack:   rack,
		AIRack:      &Rack{},
		Bag:         bag,
		CurrentTurn: HumanTurn,
		DictName:    testDict.Name(),
	}

	// Play CORD vertically, reusing the existing C at (7,6): 3 NEW tiles placed.
	cmd := &PlayCommand{Move: PlayMove{Placed: []PlacedTile{
		{Row: 8, Col: 6, Tile: Tile{Letter: 'O', Points: 1}},
		{Row: 9, Col: 6, Tile: Tile{Letter: 'R', Points: 1}},
		{Row: 10, Col: 6, Tile: Tile{Letter: 'D', Points: 2}},
	}}}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if got := state.HumanRack.Count(); got != MaxRackSize {
		t.Errorf("rack count after playing 3 tiles (CORD reusing C) = %d, want %d", got, MaxRackSize)
	}
	if got := len(cmd.drawnTiles); got != 3 {
		t.Errorf("drew %d tiles, want 3 (one per placed tile)", got)
	}
}
