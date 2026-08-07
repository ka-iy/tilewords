// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"bytes"
	"encoding/gob"
	"math/rand/v2"
	"testing"
)

// --- Bag tests ---

func TestNewBag_Count(t *testing.T) {
	bag := NewBag(deterministicRNG())
	if bag.Count() != tileDistributionTotal {
		t.Errorf("NewBag count = %d, want %d", bag.Count(), tileDistributionTotal)
	}
}

func TestNewBag_Distribution(t *testing.T) {
	bag := NewBag(deterministicRNG())
	counts := make(map[byte]int)
	for _, tile := range bag.tiles {
		counts[tile.Letter]++
	}
	for _, entry := range tileDistribution {
		if counts[entry.letter] != entry.count {
			t.Errorf("letter %q: got %d tiles, want %d", entry.letter, counts[entry.letter], entry.count)
		}
	}
}

func TestBag_DrawAndReturn(t *testing.T) {
	rng := deterministicRNG()
	bag := NewBag(rng)
	initial := bag.Count()

	drawn := bag.Draw(7, nil) // no reshuffle
	if len(drawn) != 7 {
		t.Fatalf("Draw(7) returned %d tiles", len(drawn))
	}
	if bag.Count() != initial-7 {
		t.Errorf("after Draw(7): count = %d, want %d", bag.Count(), initial-7)
	}

	bag.Return(drawn, nil) // no reshuffle
	if bag.Count() != initial {
		t.Errorf("after Return: count = %d, want %d", bag.Count(), initial)
	}
}

func TestBag_DrawMoreThanAvailable(t *testing.T) {
	bag := newTestBag([]Tile{{Letter: 'A', Points: 1}})
	drawn := bag.Draw(5, nil)
	if len(drawn) != 1 {
		t.Errorf("Draw(5) from 1-tile bag returned %d tiles, want 1", len(drawn))
	}
}

// --- Board tests ---

func TestBoard_PremiumLayout(t *testing.T) {
	b := NewBoard()
	cases := []struct {
		row, col int
		want     SquareType
	}{
		{0, 0, TripleWord},
		{7, 7, Centre},
		{1, 1, DoubleWord},
		{1, 5, TripleLetter},
		{0, 3, DoubleLetter},
		{7, 3, DoubleLetter},
		{5, 5, TripleLetter},
		{14, 14, TripleWord},
	}
	for _, tc := range cases {
		got := b.Cell(tc.row, tc.col).Square
		if got != tc.want {
			t.Errorf("Cell(%d,%d).Square = %v, want %v", tc.row, tc.col, got, tc.want)
		}
	}
}

func TestBoard_PlaceAndRemove(t *testing.T) {
	b := NewBoard()
	tile := Tile{Letter: 'A', Points: 1}

	if err := b.Place(7, 7, tile); err != nil {
		t.Fatalf("Place: %v", err)
	}
	if b.IsEmpty(7, 7) {
		t.Error("IsEmpty(7,7) = true after Place")
	}

	b.Remove(7, 7)
	if !b.IsEmpty(7, 7) {
		t.Error("IsEmpty(7,7) = false after Remove")
	}
}

func TestBoard_Place_OccupiedError(t *testing.T) {
	b := NewBoard()
	tile := Tile{Letter: 'A', Points: 1}
	if err := b.Place(7, 7, tile); err != nil {
		t.Fatalf("first Place: %v", err)
	}
	if err := b.Place(7, 7, tile); err == nil {
		t.Error("expected error placing on occupied cell, got nil")
	}
}

func TestBoard_GobRoundTrip(t *testing.T) {
	b := NewBoard()
	tile := Tile{Letter: 'Z', Points: 10}
	_ = b.Place(3, 5, tile)

	data, err := b.GobEncode()
	if err != nil {
		t.Fatalf("GobEncode: %v", err)
	}

	b2 := &Board{}
	if err := b2.GobDecode(data); err != nil {
		t.Fatalf("GobDecode: %v", err)
	}

	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			orig := b.cells[r][c]
			got := b2.cells[r][c]
			if orig.Square != got.Square {
				t.Errorf("(%d,%d): Square %v != %v", r, c, orig.Square, got.Square)
			}
			origEmpty := orig.Tile == nil
			gotEmpty := got.Tile == nil
			if origEmpty != gotEmpty {
				t.Errorf("(%d,%d): Tile nil mismatch", r, c)
			}
			if !origEmpty && !gotEmpty && *orig.Tile != *got.Tile {
				t.Errorf("(%d,%d): Tile content mismatch", r, c)
			}
		}
	}
}

// Verify Board serialises correctly via standard gob (as used by SaveManager).
func TestBoard_GobViaStdlib(t *testing.T) {
	b := NewBoard()
	_ = b.Place(0, 0, Tile{Letter: 'Q', Points: 10})

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(b); err != nil {
		t.Fatalf("gob.Encode: %v", err)
	}

	b2 := &Board{}
	if err := gob.NewDecoder(&buf).Decode(b2); err != nil {
		t.Fatalf("gob.Decode: %v", err)
	}
	if b2.IsEmpty(0, 0) {
		t.Error("decoded board is empty at (0,0), expected Q tile")
	}
}

// --- Rack tests ---

func TestRack_AddRemove(t *testing.T) {
	r := &Rack{}
	tiles := []Tile{{Letter: 'A', Points: 1}, {Letter: 'B', Points: 3}}
	if err := r.Add(tiles); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if r.Count() != 2 {
		t.Errorf("Count = %d, want 2", r.Count())
	}
	if err := r.Remove([]Tile{{Letter: 'A', Points: 1}}); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if r.Count() != 1 {
		t.Errorf("Count after Remove = %d, want 1", r.Count())
	}
}

func TestRack_Remove_NotFound(t *testing.T) {
	r := &Rack{}
	_ = r.Add([]Tile{{Letter: 'A', Points: 1}})
	if err := r.Remove([]Tile{{Letter: 'Z', Points: 10}}); err == nil {
		t.Error("expected error removing absent tile, got nil")
	}
	// Rack must be unchanged after failed remove.
	if r.Count() != 1 {
		t.Errorf("rack count changed after failed remove: got %d, want 1", r.Count())
	}
}

func TestRack_OverCapacity(t *testing.T) {
	r := &Rack{}
	tiles := make([]Tile, MaxRackSize)
	for i := range tiles {
		tiles[i] = Tile{Letter: 'A', Points: 1}
	}
	_ = r.Add(tiles)
	if err := r.Add([]Tile{{Letter: 'B', Points: 3}}); err == nil {
		t.Error("expected error adding beyond capacity, got nil")
	}
}

// --- ValidatePlacement tests ---

func TestValidatePlacement_FirstMove_Centre(t *testing.T) {
	b := NewBoard()
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'C', Points: 3}, Row: 7, Col: 6},
			{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 7},
			{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 8},
		},
	}
	words, err := ValidatePlacement(b, move, testDict)
	if err != nil {
		t.Fatalf("ValidatePlacement: %v", err)
	}
	if len(words) == 0 {
		t.Error("expected at least one word formed")
	}
}

func TestValidatePlacement_FirstMove_MissesCentre(t *testing.T) {
	b := NewBoard()
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'C', Points: 3}, Row: 0, Col: 0},
			{Tile: Tile{Letter: 'A', Points: 1}, Row: 0, Col: 1},
			{Tile: Tile{Letter: 'T', Points: 1}, Row: 0, Col: 2},
		},
	}
	if _, err := ValidatePlacement(b, move, testDict); err == nil {
		t.Error("expected error for first move missing centre, got nil")
	}
}

func TestValidatePlacement_Gap(t *testing.T) {
	b := NewBoard()
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'C', Points: 3}, Row: 7, Col: 5},
			// col 6 is empty on both board and move — this is a gap
			{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 7},
		},
	}
	if _, err := ValidatePlacement(b, move, testDict); err == nil {
		t.Error("expected error for gap in placed tiles, got nil")
	}
}

func TestValidatePlacement_NotAdjacent(t *testing.T) {
	b := NewBoard()
	// Place CAT at centre first.
	_ = b.Place(7, 6, Tile{Letter: 'C', Points: 3})
	_ = b.Place(7, 7, Tile{Letter: 'A', Points: 1})
	_ = b.Place(7, 8, Tile{Letter: 'T', Points: 1})

	// Try to place a word far away from any existing tile.
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'D', Points: 2}, Row: 0, Col: 0},
			{Tile: Tile{Letter: 'O', Points: 1}, Row: 0, Col: 1},
			{Tile: Tile{Letter: 'G', Points: 2}, Row: 0, Col: 2},
		},
	}
	if _, err := ValidatePlacement(b, move, testDict); err == nil {
		t.Error("expected error for non-adjacent placement, got nil")
	}
}

func TestValidatePlacement_InvalidWord(t *testing.T) {
	b := NewBoard()
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'Z', Points: 10}, Row: 7, Col: 6},
			{Tile: Tile{Letter: 'Z', Points: 10}, Row: 7, Col: 7},
			{Tile: Tile{Letter: 'Z', Points: 10}, Row: 7, Col: 8},
		},
	}
	if _, err := ValidatePlacement(b, move, testDict); err == nil {
		t.Error("expected error for invalid word ZZZ, got nil")
	}
}

func TestValidatePlacement_CrossWordFormed(t *testing.T) {
	b := NewBoard()
	// Place CAT horizontally at row 7, cols 6-8.
	_ = b.Place(7, 6, Tile{Letter: 'C', Points: 3})
	_ = b.Place(7, 7, Tile{Letter: 'A', Points: 1})
	_ = b.Place(7, 8, Tile{Letter: 'T', Points: 1})

	// Place AD vertically: A at (6,7), D at (8,7).
	// Combined with the existing A at (7,7), forms "AD" as a cross-word (A already placed).
	// Actually let's place a simpler cross: place GO at (7,8)-(8,8) vertical.
	// AT (7,8) is already T; place O at (8,8). Cross-word is TO (T at 7,8 + O at 8,8).
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'O', Points: 1}, Row: 8, Col: 8},
		},
	}
	words, err := ValidatePlacement(b, move, testDict)
	if err != nil {
		t.Fatalf("ValidatePlacement: %v", err)
	}
	// Should form the cross-word "TO" (T at 7,8 + O at 8,8)
	found := false
	for _, w := range words {
		if w == "TO" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected cross-word TO in %v", words)
	}
}

// --- Score tests ---

func TestScore_FaceValues(t *testing.T) {
	b := newFlatBoard()
	// Place CAT on the flat board; the only premium at (7,7) is irrelevant here.
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'C', Points: 3}, Row: 0, Col: 0},
			{Tile: Tile{Letter: 'A', Points: 1}, Row: 0, Col: 1},
			{Tile: Tile{Letter: 'T', Points: 1}, Row: 0, Col: 2},
		},
		WordsFormed: []string{"CAT"},
	}
	score, err := Score(b, move)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if score != 5 { // C=3 A=1 T=1
		t.Errorf("Score = %d, want 5", score)
	}
}

func TestScore_DoubleLetter(t *testing.T) {
	b := NewBoard()
	// Place a single tile on a DoubleLetter square (0,3).
	// Also need an adjacent tile to form a word of ≥2 letters.
	_ = b.Place(0, 2, Tile{Letter: 'A', Points: 1})
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'T', Points: 1}, Row: 0, Col: 3}, // DL square
		},
		WordsFormed: []string{"AT"},
	}
	score, err := Score(b, move)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	// A=1 (existing, no mult) + T=1×2 (new on DL) = 3
	if score != 3 {
		t.Errorf("Score = %d, want 3 (A=1, T×2=2)", score)
	}
}

func TestScore_TripleWord(t *testing.T) {
	b := NewBoard()
	// Place CA at (0,0)-(0,1). (0,0) is TripleWord.
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'C', Points: 3}, Row: 0, Col: 0}, // TW square
			{Tile: Tile{Letter: 'A', Points: 1}, Row: 0, Col: 1},
		},
		WordsFormed: []string{"CA"},
	}
	// Note: CA may not be in testDict; we're testing scoring math, not validation.
	score, err := Score(b, move)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	// (C=3 + A=1) × 3 (TW) = 12
	if score != 12 {
		t.Errorf("Score = %d, want 12", score)
	}
}

func TestScore_BingoBonus(t *testing.T) {
	b := newFlatBoard()
	placed := []PlacedTile{
		{Tile: Tile{Letter: 'A', Points: 1}, Row: 0, Col: 0},
		{Tile: Tile{Letter: 'B', Points: 3}, Row: 0, Col: 1},
		{Tile: Tile{Letter: 'C', Points: 3}, Row: 0, Col: 2},
		{Tile: Tile{Letter: 'D', Points: 2}, Row: 0, Col: 3},
		{Tile: Tile{Letter: 'E', Points: 1}, Row: 0, Col: 4},
		{Tile: Tile{Letter: 'F', Points: 4}, Row: 0, Col: 5},
		{Tile: Tile{Letter: 'G', Points: 2}, Row: 0, Col: 6},
	}
	faceSum := 1 + 3 + 3 + 2 + 1 + 4 + 2 // 16
	move := &PlayMove{
		Placed:      placed,
		WordsFormed: []string{"ABCDEFG"},
	}
	score, err := Score(b, move)
	if err != nil {
		t.Fatalf("Score: %v", err)
	}
	if score != faceSum+50 {
		t.Errorf("Score = %d, want %d (face=%d + bingo=50)", score, faceSum+50, faceSum)
	}
}

func TestScore_BeforeValidate_Error(t *testing.T) {
	b := newFlatBoard()
	move := &PlayMove{
		Placed: []PlacedTile{{Tile: Tile{Letter: 'A'}, Row: 0, Col: 0}},
		// WordsFormed intentionally left nil
	}
	if _, err := Score(b, move); err == nil {
		t.Error("expected error from Score when WordsFormed is empty, got nil")
	}
}

// --- Command tests ---

func TestPlayCommand_ExecuteUndo(t *testing.T) {
	state := newGameState()
	state.CurrentTurn = HumanTurn

	// Manually give the human rack a known set of tiles and set the board
	// so we can place a valid word.
	state.HumanRack = &Rack{tiles: []Tile{
		{Letter: 'C', Points: 3},
		{Letter: 'A', Points: 1},
		{Letter: 'T', Points: 1},
		{Letter: 'S', Points: 1},
		{Letter: 'E', Points: 1},
		{Letter: 'D', Points: 2},
		{Letter: 'R', Points: 1},
	}}

	move := PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'C', Points: 3}, Row: 7, Col: 6},
			{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 7},
			{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 8},
		},
	}

	rng := deterministicRNG()
	cmd := &PlayCommand{Move: move}
	if err := cmd.Execute(state, testDict, rng); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if state.CurrentTurn != AITurn {
		t.Error("CurrentTurn should be AITurn after human play")
	}
	if state.HumanScore <= 0 {
		t.Errorf("HumanScore = %d, expected > 0", state.HumanScore)
	}
	if state.ConsecutivePasses != 0 {
		t.Errorf("ConsecutivePasses = %d, want 0 after play", state.ConsecutivePasses)
	}

	scoreBefore := state.HumanScore
	moveNumBefore := state.MoveNumber

	cmd.Undo(state, nil)

	if state.CurrentTurn != HumanTurn {
		t.Error("CurrentTurn should be HumanTurn after undo")
	}
	if state.HumanScore != 0 {
		t.Errorf("HumanScore after undo = %d, want 0 (score was %d)", state.HumanScore, scoreBefore)
	}
	if state.MoveNumber != moveNumBefore-1 {
		t.Errorf("MoveNumber after undo = %d, want %d", state.MoveNumber, moveNumBefore-1)
	}
	if !state.Board.IsEmpty(7, 6) || !state.Board.IsEmpty(7, 7) || !state.Board.IsEmpty(7, 8) {
		t.Error("board should be empty at play positions after undo")
	}
}

func TestExchangeCommand_ExecuteUndo(t *testing.T) {
	rng := deterministicRNG()
	state := newGameState()
	state.CurrentTurn = HumanTurn

	// Ensure bag has ≥7 tiles (it will after New).
	initialBagCount := state.Bag.Count()
	initialRackCount := state.HumanRack.Count()

	tilesToExchange := []Tile{state.HumanRack.tiles[0], state.HumanRack.tiles[1]}
	cmd := &ExchangeCommand{Move: ExchangeMove{Tiles: tilesToExchange}}

	if err := cmd.Execute(state, testDict, rng); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if state.ConsecutivePasses != 1 {
		t.Errorf("ConsecutivePasses = %d, want 1 after exchange", state.ConsecutivePasses)
	}
	if state.HumanRack.Count() != initialRackCount {
		t.Errorf("rack count after exchange = %d, want %d", state.HumanRack.Count(), initialRackCount)
	}

	cmd.Undo(state, nil)

	if state.CurrentTurn != HumanTurn {
		t.Error("CurrentTurn should be HumanTurn after undo")
	}
	if state.ConsecutivePasses != 0 {
		t.Errorf("ConsecutivePasses after undo = %d, want 0", state.ConsecutivePasses)
	}
	if state.Bag.Count() != initialBagCount {
		t.Errorf("bag count after undo = %d, want %d", state.Bag.Count(), initialBagCount)
	}
}

func TestPassCommand_ExecuteUndo(t *testing.T) {
	state := newGameState()
	state.CurrentTurn = HumanTurn

	cmd := &PassCommand{}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if state.ConsecutivePasses != 1 {
		t.Errorf("ConsecutivePasses = %d, want 1", state.ConsecutivePasses)
	}
	if state.CurrentTurn != AITurn {
		t.Error("expected AITurn after pass")
	}

	cmd.Undo(state, nil)

	if state.ConsecutivePasses != 0 {
		t.Errorf("ConsecutivePasses after undo = %d, want 0", state.ConsecutivePasses)
	}
	if state.CurrentTurn != HumanTurn {
		t.Error("expected HumanTurn after undo of pass")
	}
}

// --- IsGameOver tests ---

func TestIsGameOver_SixPasses(t *testing.T) {
	state := newGameState()
	state.ConsecutivePasses = 6
	over, reason := IsGameOver(state)
	if !over {
		t.Error("expected game over with 6 consecutive passes")
	}
	if reason != SixConsecutivePasses {
		t.Errorf("reason = %v, want SixConsecutivePasses", reason)
	}
}

func TestIsGameOver_RackExhausted(t *testing.T) {
	state := newGameState()
	state.Bag = newTestBag(nil) // empty bag
	state.HumanRack = &Rack{}   // empty rack
	over, reason := IsGameOver(state)
	if !over {
		t.Error("expected game over with empty rack and empty bag")
	}
	if reason != RackExhausted {
		t.Errorf("reason = %v, want RackExhausted", reason)
	}
}

func TestIsGameOver_NotOver(t *testing.T) {
	state := newGameState()
	over, _ := IsGameOver(state)
	if over {
		t.Error("fresh game should not be over")
	}
}

// --- ApplyEndgameScoring tests ---

func TestApplyEndgameScoring_RackExhausted_HumanWins(t *testing.T) {
	state := newGameState()
	state.HumanRack = &Rack{}
	state.AIRack = &Rack{tiles: []Tile{
		{Letter: 'Q', Points: 10},
		{Letter: 'Z', Points: 10},
	}}
	state.Bag = newTestBag(nil)

	ApplyEndgameScoring(state, RackExhausted)

	if state.HumanScore != 20 {
		t.Errorf("HumanScore = %d, want 20", state.HumanScore)
	}
	if state.AIScore != -20 {
		t.Errorf("AIScore = %d, want -20", state.AIScore)
	}
}

func TestApplyEndgameScoring_SixPasses(t *testing.T) {
	state := newGameState()
	state.HumanRack = &Rack{tiles: []Tile{{Letter: 'A', Points: 1}}}
	state.AIRack = &Rack{tiles: []Tile{{Letter: 'Z', Points: 10}}}
	state.HumanScore = 50
	state.AIScore = 60

	ApplyEndgameScoring(state, SixConsecutivePasses)

	if state.HumanScore != 49 {
		t.Errorf("HumanScore = %d, want 49", state.HumanScore)
	}
	if state.AIScore != 50 {
		t.Errorf("AIScore = %d, want 50", state.AIScore)
	}
}

// TestApplyEndgameScoring_BranchesOnReason verifies the scoring keys off the
// authoritative EndReason, not a reconstruction from bag/rack counts. Here the bag
// is empty and one rack is empty, yet the game ended on six passes — so each player
// must lose only their own remaining tiles, with no going-out redistribution.
func TestApplyEndgameScoring_BranchesOnReason(t *testing.T) {
	state := newGameState()
	state.HumanRack = &Rack{} // empty
	state.AIRack = &Rack{tiles: []Tile{{Letter: 'Z', Points: 10}}}
	state.Bag = newTestBag(nil) // empty
	state.HumanScore = 50
	state.AIScore = 60

	ApplyEndgameScoring(state, SixConsecutivePasses)

	if state.HumanScore != 50 {
		t.Errorf("HumanScore = %d, want 50 (six-pass: no redistribution)", state.HumanScore)
	}
	if state.AIScore != 50 {
		t.Errorf("AIScore = %d, want 50 (six-pass: loses own 10)", state.AIScore)
	}
}

// TestApplyEndgameScoring_Idempotent verifies a stray second call does not adjust
// the scores a second time.
func TestApplyEndgameScoring_Idempotent(t *testing.T) {
	state := newGameState()
	state.HumanRack = &Rack{}
	state.AIRack = &Rack{tiles: []Tile{{Letter: 'Q', Points: 10}}}
	state.Bag = newTestBag(nil)

	ApplyEndgameScoring(state, RackExhausted)
	h, a := state.HumanScore, state.AIScore
	ApplyEndgameScoring(state, RackExhausted) // must be a no-op

	if state.HumanScore != h || state.AIScore != a {
		t.Errorf("second call changed scores: (%d,%d) -> (%d,%d)", h, a, state.HumanScore, state.AIScore)
	}
}

// TestPlayCommand_ZeroScoringPlayIsScorelessTurn verifies official-rule conformance:
// a play that scores zero (an all-blank word on plain squares) counts toward the six
// consecutive scoreless turns rather than resetting the counter.
func TestPlayCommand_ZeroScoringPlayIsScorelessTurn(t *testing.T) {
	state := &GameState{
		Board:             newFlatBoard(), // all-Normal squares: no multipliers
		HumanRack:         &Rack{tiles: []Tile{{IsBlank: true}, {IsBlank: true}}},
		AIRack:            &Rack{},
		Bag:               newTestBag(nil),
		CurrentTurn:       HumanTurn,
		ConsecutivePasses: 3,
	}
	cmd := &PlayCommand{Move: PlayMove{Placed: []PlacedTile{
		{Tile: Tile{Letter: 'A', IsBlank: true, AssignedLetter: 'A'}, Row: 7, Col: 7},
		{Tile: Tile{Letter: 'A', IsBlank: true, AssignedLetter: 'A'}, Row: 7, Col: 8},
	}}}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if cmd.Move.Score != 0 {
		t.Fatalf("expected a zero-scoring play, got score %d", cmd.Move.Score)
	}
	if state.ConsecutivePasses != 4 {
		t.Fatalf("zero-scoring play must count as a scoreless turn: ConsecutivePasses = %d, want 4", state.ConsecutivePasses)
	}
}

// TestPlayCommand_ScoringPlayResetsScorelessCounter verifies a scoring play resets
// the consecutive-scoreless-turn counter.
func TestPlayCommand_ScoringPlayResetsScorelessCounter(t *testing.T) {
	state := &GameState{
		Board:             newFlatBoard(),
		HumanRack:         &Rack{tiles: []Tile{{Letter: 'A', Points: 1}, {Letter: 'T', Points: 1}}},
		AIRack:            &Rack{},
		Bag:               newTestBag(nil),
		CurrentTurn:       HumanTurn,
		ConsecutivePasses: 3,
	}
	cmd := &PlayCommand{Move: PlayMove{Placed: []PlacedTile{
		{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 7},
		{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 8},
	}}}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if cmd.Move.Score == 0 {
		t.Fatal("expected a scoring play")
	}
	if state.ConsecutivePasses != 0 {
		t.Fatalf("scoring play must reset counter: got %d, want 0", state.ConsecutivePasses)
	}
}

// --- GameState.Clone tests ---

func TestGameState_Clone_Independent(t *testing.T) {
	rng := rand.New(rand.NewPCG(99, 0))
	state := New(testDict.Name(), 5, rng)
	clone := state.Clone()

	// Mutate the clone; original must be unaffected.
	clone.HumanScore = 9999
	clone.Board.cells[0][0].Square = TripleWord // already TW, but change tile
	_ = clone.Board.Place(0, 1, Tile{Letter: 'X', Points: 8})

	if state.HumanScore == 9999 {
		t.Error("mutating clone.HumanScore affected original")
	}
	if !state.Board.IsEmpty(0, 1) {
		t.Error("placing tile on clone board affected original board")
	}
}

// --- Regression tests ---

// TestPlayCommand_Undo_BlankTileReset is a regression test for R15-BUG-01:
// a blank tile returned to the rack after undo must have AssignedLetter and
// Letter cleared so it renders as a blank, not as its previously-played letter.
func TestPlayCommand_Undo_BlankTileReset(t *testing.T) {
	state := newGameState()
	state.CurrentTurn = HumanTurn

	// Give the human a blank tile and two regular tiles to form "CAT"
	// where the blank is played as 'A'.
	state.HumanRack = &Rack{tiles: []Tile{
		{Letter: 'C', Points: 3},
		{IsBlank: true, Points: 0},
		{Letter: 'T', Points: 1},
		{Letter: 'S', Points: 1},
		{Letter: 'E', Points: 1},
		{Letter: 'D', Points: 2},
		{Letter: 'R', Points: 1},
	}}

	// Blank assigned to 'A', played as the middle tile of CAT.
	blankTile := Tile{IsBlank: true, Letter: 'A', AssignedLetter: 'A', Points: 0}
	move := PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'C', Points: 3}, Row: 7, Col: 6},
			{Tile: blankTile, Row: 7, Col: 7},
			{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 8},
		},
	}

	rng := deterministicRNG()
	cmd := &PlayCommand{Move: move}
	if err := cmd.Execute(state, testDict, rng); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	cmd.Undo(state, nil)

	// After undo, find the blank in the rack; it must have Letter=0 and AssignedLetter=0.
	found := false
	for _, tile := range state.HumanRack.Tiles() {
		if tile.IsBlank {
			found = true
			if tile.Letter != 0 {
				t.Errorf("blank tile Letter after undo = %q, want 0", tile.Letter)
			}
			if tile.AssignedLetter != 0 {
				t.Errorf("blank tile AssignedLetter after undo = %q, want 0", tile.AssignedLetter)
			}
		}
	}
	if !found {
		t.Error("blank tile not found in rack after undo")
	}
}

// TestValidatePlacement_SingleTileNoWord is a regression test for R16-BUG-01:
// a single-tile first move that forms no word of length ≥ 2 must be rejected
// with a clear error message, not passed to Score where it would fail confusingly.
func TestValidatePlacement_SingleTileNoWord(t *testing.T) {
	board := NewBoard()
	move := &PlayMove{
		Placed: []PlacedTile{
			{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 7},
		},
	}
	_, err := ValidatePlacement(board, move, testDict)
	if err == nil {
		t.Error("expected error for single-tile first move forming no word, got nil")
	}
}
