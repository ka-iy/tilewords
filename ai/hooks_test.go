package ai_test

import (
	"testing"

	"tilewords/ai"
	"tilewords/engine"
)

// formedWord returns the first candidate that forms word w, if any.
func formedWord(cands []ai.MoveCandidate, w string) (ai.MoveCandidate, bool) {
	for _, c := range cands {
		for _, fw := range c.Move.WordsFormed {
			if fw == w {
				return c, true
			}
		}
	}
	return ai.MoveCandidate{}, false
}

// TestGenerateMoves_FirstMoveScored verifies that newly placed tiles contribute
// their face value to a generated move's score. Regression: the generator left
// Tile.Points unset on every tile it placed, so all plays scored 0 and the board
// was committed with 0-point tiles.
func TestGenerateMoves_FirstMoveScored(t *testing.T) {
	board := engine.NewBoard()
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{
		{Letter: 'C', Points: 3}, {Letter: 'A', Points: 1}, {Letter: 'T', Points: 1},
	})
	cands := ai.GenerateMoves(board, rack, testDict)
	c, ok := formedWord(cands, "CAT")
	if !ok {
		t.Fatal("CAT not generated on empty board")
	}
	// CAT across the centre (Double Word): (3+1+1) * 2 = 10.
	if c.Score != 10 {
		t.Fatalf("CAT first-move score = %d, want 10", c.Score)
	}
}

// TestGenerateMoves_FrontHook verifies the AI can place a tile to the LEFT of an
// existing word. Regression: left-extension could not navigate existing board
// tiles, so the AI missed every prefix/front-hook play.
func TestGenerateMoves_FrontHook(t *testing.T) {
	board := engine.NewBoard()
	placeTile(board, 7, 8, 'A', 1)
	placeTile(board, 7, 9, 'T', 1)
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{{Letter: 'C', Points: 3}})
	if _, ok := formedWord(ai.GenerateMoves(board, rack, testDict), "CAT"); !ok {
		t.Fatal("front-hook CAT (C left of existing AT) not generated")
	}
}

// TestGenerateMoves_VerticalFrontHook verifies front-hooks work vertically too.
func TestGenerateMoves_VerticalFrontHook(t *testing.T) {
	board := engine.NewBoard()
	placeTile(board, 8, 7, 'A', 1)
	placeTile(board, 9, 7, 'T', 1)
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{{Letter: 'C', Points: 3}})
	if _, ok := formedWord(ai.GenerateMoves(board, rack, testDict), "CAT"); !ok {
		t.Fatal("vertical front-hook CAT (C above existing AT) not generated")
	}
}

// TestGenerateMoves_Append verifies the AI can extend an existing word to the right.
func TestGenerateMoves_Append(t *testing.T) {
	board := engine.NewBoard()
	placeTile(board, 7, 7, 'C', 3)
	placeTile(board, 7, 8, 'A', 1)
	placeTile(board, 7, 9, 'R', 1)
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{{Letter: 'T', Points: 1}})
	if _, ok := formedWord(ai.GenerateMoves(board, rack, testDict), "CART"); !ok {
		t.Fatal("append CART (T after existing CAR) not generated")
	}
}

// TestGenerateMoves_PlayThrough verifies a single tile placed between two existing
// tiles to complete a word is generated.
func TestGenerateMoves_PlayThrough(t *testing.T) {
	board := engine.NewBoard()
	placeTile(board, 7, 7, 'C', 3)
	placeTile(board, 7, 9, 'T', 1)
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{{Letter: 'A', Points: 1}})
	if _, ok := formedWord(ai.GenerateMoves(board, rack, testDict), "CAT"); !ok {
		t.Fatal("play-through CAT (A between existing C and T) not generated")
	}
}

// TestGenerateMoves_DistinctWordsSameFootprint verifies that multiple legal words
// occupying the SAME cells are all generated. Regression: the dedup key was the
// bounding box only, so all but one same-footprint word were silently dropped,
// hiding legal (often higher-scoring) moves from move selection.
func TestGenerateMoves_DistinctWordsSameFootprint(t *testing.T) {
	board := engine.NewBoard()
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{
		{Letter: 'C', Points: 3}, {Letter: 'A', Points: 1}, {Letter: 'B', Points: 3},
		{Letter: 'P', Points: 3}, {Letter: 'R', Points: 1}, {Letter: 'T', Points: 1},
	})
	cands := ai.GenerateMoves(board, rack, testDict)
	for _, w := range []string{"CAB", "CAP", "CAR", "CAT"} {
		if _, ok := formedWord(cands, w); !ok {
			t.Errorf("%s (shares footprint (7,7)-(7,9)) not generated — dedup key too coarse", w)
		}
	}
}

// TestGenerateMoves_HookStillValidated verifies that even with the broadened
// generator, every candidate produced for a populated board passes the engine's
// authoritative ValidatePlacement (no illegal moves leak through).
func TestGenerateMoves_HookStillValidated(t *testing.T) {
	board := engine.NewBoard()
	placeTile(board, 7, 7, 'C', 3)
	placeTile(board, 7, 8, 'A', 1)
	placeTile(board, 7, 9, 'R', 1)
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{
		{Letter: 'T', Points: 1}, {Letter: 'S', Points: 1}, {Letter: 'E', Points: 1},
		{Letter: 'D', Points: 2}, {Letter: 'O', Points: 1},
	})
	cands := ai.GenerateMoves(board, rack, testDict)
	for i, c := range cands {
		m := c.Move
		if _, err := engine.ValidatePlacement(board, &m, testDict); err != nil {
			t.Errorf("candidate %d (%v) failed ValidatePlacement: %v", i, m.WordsFormed, err)
		}
	}
}
