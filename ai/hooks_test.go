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

// TestGenerateMoves_BlankAndRealTileBothTried verifies that when the rack holds a letter
// both as a real tile and as a blank, BOTH physical assignments of a word using that letter
// are generated. They are different plays, not duplicates: a blank scores zero, so which
// cell it covers changes the score. Generating only one would hide the higher-scoring
// assignment from move selection, and the AI would play a weaker move while a better one
// was legal.
func TestGenerateMoves_BlankAndRealTileBothTried(t *testing.T) {
	board := engine.NewBoard()
	// FIZZ down column J: F and I are on the board, so the play supplies both Z's — one
	// from the real Z, one from the blank. (8,9) is a plain square and (9,9) a Triple
	// Letter, so the assignment that puts the real Z on the premium scores far more, and
	// the two must be told apart.
	placeTile(board, 6, 9, 'F', 4)
	placeTile(board, 7, 9, 'I', 1)
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{
		{Letter: 'Z', Points: 10},
		{Letter: 0, Points: 0, IsBlank: true},
	})

	var fizz []ai.MoveCandidate
	for _, c := range ai.GenerateMoves(board, rack, testDict) {
		for _, w := range c.Move.WordsFormed {
			if w == "FIZZ" {
				fizz = append(fizz, c)
			}
		}
	}
	if len(fizz) != 2 {
		t.Fatalf("generated %d FIZZ candidates, want 2 (real Z then blank, and blank then real Z)", len(fizz))
	}

	// The two must differ in which cell carries the blank, and therefore in score.
	blankCell := func(c ai.MoveCandidate) [2]int {
		for _, p := range c.Move.Placed {
			if p.Tile.IsBlank {
				return [2]int{p.Row, p.Col}
			}
		}
		t.Fatalf("FIZZ candidate placed no blank: %v", c.Move.Placed)
		return [2]int{}
	}
	if blankCell(fizz[0]) == blankCell(fizz[1]) {
		t.Errorf("both FIZZ candidates put the blank at %v; the two assignments were not enumerated", blankCell(fizz[0]))
	}
	if fizz[0].Score == fizz[1].Score {
		t.Errorf("both FIZZ candidates score %d; expected the blank's cell to change the score", fizz[0].Score)
	}

	// A blank must never contribute face value, whichever cell it lands on.
	for _, c := range fizz {
		for _, p := range c.Move.Placed {
			if p.Tile.IsBlank && p.Tile.Points != 0 {
				t.Errorf("blank at (%d,%d) scored %d points, want 0", p.Row, p.Col, p.Tile.Points)
			}
		}
	}
}

// TestGenerateMoves_BlankOnlyRackStillPlays verifies the blank branch is still reached when
// the rack holds no real tile for the letter, i.e. offering the real tile first did not make
// the blank fallback unreachable.
func TestGenerateMoves_BlankOnlyRackStillPlays(t *testing.T) {
	board := engine.NewBoard()
	placeTile(board, 7, 7, 'C', 3)
	placeTile(board, 7, 9, 'T', 1)
	rack := &engine.Rack{}
	_ = rack.Add([]engine.Tile{{Letter: 0, Points: 0, IsBlank: true}})
	c, ok := formedWord(ai.GenerateMoves(board, rack, testDict), "CAT")
	if !ok {
		t.Fatal("CAT not generated from a blank-only rack")
	}
	if len(c.Move.Placed) != 1 || !c.Move.Placed[0].Tile.IsBlank {
		t.Fatalf("expected a single blank tile, got %v", c.Move.Placed)
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
