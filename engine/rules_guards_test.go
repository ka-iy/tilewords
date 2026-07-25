package engine

import (
	"strings"
	"testing"
)

// TestValidatePlacement_RejectsDuplicateCells verifies a move claiming one cell with more
// than one tile is refused. Accepting it would let len(Placed) reach MaxRackSize while the
// tiles cover fewer squares, awarding the bingo bonus for a short word, and Board.Place
// would then reject the repeat only after the rack had been debited.
func TestValidatePlacement_RejectsDuplicateCells(t *testing.T) {
	board := NewBoard()
	move := &PlayMove{Placed: []PlacedTile{
		{Tile: Tile{Letter: 'C', Points: 3}, Row: 7, Col: 7},
		{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 8},
		{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 9},
		{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 9},
	}}
	_, err := ValidatePlacement(board, move, testDict)
	if err == nil {
		t.Fatal("accepted a move with two tiles on cell (7,9)")
	}
	if !strings.Contains(err.Error(), "more than one tile") {
		t.Errorf("error = %q, want it to report the duplicated position", err)
	}
}

// TestValidatePlacement_DuplicateCellsCannotForgeBingo verifies the specific consequence:
// padding a short word with repeats of one cell must not reach the 7-tile bingo bonus.
func TestValidatePlacement_DuplicateCellsCannotForgeBingo(t *testing.T) {
	board := NewBoard()
	placed := []PlacedTile{
		{Tile: Tile{Letter: 'C', Points: 3}, Row: 7, Col: 7},
		{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 8},
	}
	// Pad to exactly MaxRackSize entries, all stacked on the last cell.
	for len(placed) < MaxRackSize {
		placed = append(placed, PlacedTile{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 9})
	}
	move := &PlayMove{Placed: placed}
	if len(move.Placed) != MaxRackSize {
		t.Fatalf("fixture has %d tiles, want %d so the bingo test would fire", len(move.Placed), MaxRackSize)
	}
	if _, err := ValidatePlacement(board, move, testDict); err == nil {
		t.Fatal("accepted a 7-entry move covering 3 cells; the bingo bonus would be unearned")
	}
}

// TestApplyEndgameScoring_NotOverIsNoOp verifies that scoring a game that has not ended
// changes nothing and does not latch EndgameScored. Latching it would permanently suppress
// the real adjustment when the game does end, leaving the final scores wrong for good.
func TestApplyEndgameScoring_NotOverIsNoOp(t *testing.T) {
	state := &GameState{
		Board:      NewBoard(),
		HumanRack:  &Rack{tiles: []Tile{{Letter: 'Q', Points: 10}}},
		AIRack:     &Rack{tiles: []Tile{{Letter: 'Z', Points: 10}}},
		Bag:        newTestBag([]Tile{{Letter: 'E', Points: 1}}),
		HumanScore: 100,
		AIScore:    100,
	}
	over, reason := IsGameOver(state)
	if over {
		t.Fatalf("fixture should not be over, got reason %v", reason)
	}

	ApplyEndgameScoring(state, reason)

	if state.HumanScore != 100 || state.AIScore != 100 {
		t.Errorf("scores changed on a live game: human=%d ai=%d, want 100/100", state.HumanScore, state.AIScore)
	}
	if state.EndgameScored {
		t.Fatal("EndgameScored latched on a live game; the real endgame adjustment would never apply")
	}

	// The real ending must still score normally afterwards.
	state.HumanRack = &Rack{}
	state.Bag = newTestBag(nil)
	over, reason = IsGameOver(state)
	if !over || reason != RackExhausted {
		t.Fatalf("IsGameOver = %v,%v; want true,RackExhausted", over, reason)
	}
	ApplyEndgameScoring(state, reason)
	if state.HumanScore != 110 || state.AIScore != 90 {
		t.Errorf("going-out scoring = human %d / ai %d, want 110/90", state.HumanScore, state.AIScore)
	}
}
