// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"math/rand/v2"
	"testing"
)

// bagOrder returns the bag's tiles in draw order, for comparing one bag state to another.
func bagOrder(b *Bag) []Tile {
	cp := make([]Tile, len(b.tiles))
	copy(cp, b.tiles)
	return cp
}

// sameOrder reports whether two tile sequences are identical position by position.
func sameOrder(a, b []Tile) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// letterCounts summarises a tile sequence as a per-letter multiset, so conservation can be
// checked independently of order (blanks are counted under key 0).
func letterCounts(tiles []Tile) map[byte]int {
	m := make(map[byte]int)
	for _, t := range tiles {
		k := t.Letter
		if t.IsBlank {
			k = 0
		}
		m[k]++
	}
	return m
}

func sameCounts(a, b map[byte]int) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// fullBag returns a bag holding one of each letter A-Z plus a blank, in a known order.
func fullBag() *Bag {
	tiles := make([]Tile, 0, 27)
	for l := byte('A'); l <= 'Z'; l++ {
		tiles = append(tiles, Tile{Letter: l, Points: 1})
	}
	tiles = append(tiles, Tile{IsBlank: true})
	return newTestBag(tiles)
}

// TestPlayUndo_ReshufflesBag verifies that undoing a play returns the drawn tiles and then
// reshuffles, so the draw order the move revealed is not the order the next draw follows.
// Restoring the bag exactly would let a player play, see what they drew, undo, and replay
// knowing what is coming.
func TestPlayUndo_ReshufflesBag(t *testing.T) {
	state := &GameState{
		Board:       newFlatBoard(),
		HumanRack:   &Rack{tiles: []Tile{{Letter: 'A', Points: 1}, {Letter: 'T', Points: 1}}},
		AIRack:      &Rack{},
		Bag:         fullBag(),
		CurrentTurn: HumanTurn,
	}
	before := bagOrder(state.Bag)

	cmd := &PlayCommand{Move: PlayMove{Placed: []PlacedTile{
		{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 7},
		{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 8},
	}}}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(cmd.drawnTiles) == 0 {
		t.Fatal("fixture drew no tiles, so there is no revealed order to protect")
	}

	cmd.Undo(state, rand.New(rand.NewPCG(7, 0)))

	after := bagOrder(state.Bag)
	if !sameCounts(letterCounts(before), letterCounts(after)) {
		t.Fatalf("undo did not conserve the bag's tiles: %v vs %v",
			letterCounts(before), letterCounts(after))
	}
	if sameOrder(before, after) {
		t.Error("bag order is unchanged after undo; the tiles the move revealed would be drawn again")
	}
}

// TestExchangeUndo_ReshufflesBag verifies the same protection for an exchange, whose Undo
// restores the bag from a snapshot and must reshuffle it afterwards.
func TestExchangeUndo_ReshufflesBag(t *testing.T) {
	state := &GameState{
		Board:       newFlatBoard(),
		HumanRack:   &Rack{tiles: []Tile{{Letter: 'Q', Points: 10}, {Letter: 'Z', Points: 10}}},
		AIRack:      &Rack{},
		Bag:         fullBag(),
		CurrentTurn: HumanTurn,
	}
	before := bagOrder(state.Bag)

	cmd := &ExchangeCommand{Move: ExchangeMove{Tiles: []Tile{{Letter: 'Q', Points: 10}}}}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	cmd.Undo(state, rand.New(rand.NewPCG(7, 0)))

	after := bagOrder(state.Bag)
	if !sameCounts(letterCounts(before), letterCounts(after)) {
		t.Fatalf("undo did not conserve the bag's tiles: %v vs %v",
			letterCounts(before), letterCounts(after))
	}
	if sameOrder(before, after) {
		t.Error("bag order is unchanged after undo; the replacements the exchange revealed would be drawn again")
	}
	// The rack must still be restored exactly — only the unseen bag order changes.
	if state.HumanRack.Count() != 2 {
		t.Errorf("rack count after undo = %d, want 2", state.HumanRack.Count())
	}
}

// TestUndo_NilRNGKeepsBagOrder verifies a nil rng leaves the order alone, which is what lets
// a test assert exact restoration.
//
// The baseline is the bag as it stands after Execute, not before it: a draw reshuffles the
// bag before taking each tile (see Bag.Draw), so Execute is entitled to have reordered it.
// What a nil rng promises is that Undo adds nothing of its own — the drawn tiles go back on
// the end and every other tile stays where Execute left it.
func TestUndo_NilRNGKeepsBagOrder(t *testing.T) {
	state := &GameState{
		Board:       newFlatBoard(),
		HumanRack:   &Rack{tiles: []Tile{{Letter: 'A', Points: 1}, {Letter: 'T', Points: 1}}},
		AIRack:      &Rack{},
		Bag:         fullBag(),
		CurrentTurn: HumanTurn,
	}

	cmd := &PlayCommand{Move: PlayMove{Placed: []PlacedTile{
		{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 7},
		{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 8},
	}}}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(cmd.drawnTiles) == 0 {
		t.Fatal("fixture drew no tiles, so undo would put nothing back and the check is vacuous")
	}

	want := append(bagOrder(state.Bag), cmd.drawnTiles...)
	cmd.Undo(state, nil)

	if !sameOrder(want, bagOrder(state.Bag)) {
		t.Error("undo with a nil rng changed the bag order")
	}
}

// TestExchangeUndo_NilRNGRestoresBagExactly verifies the other half of the nil-rng contract:
// an exchange undone with a nil rng restores the bag to exactly its pre-Execute order, since
// its Undo replaces the bag from a snapshot taken before the draw reshuffled anything.
func TestExchangeUndo_NilRNGRestoresBagExactly(t *testing.T) {
	state := &GameState{
		Board:       newFlatBoard(),
		HumanRack:   &Rack{tiles: []Tile{{Letter: 'Q', Points: 10}, {Letter: 'Z', Points: 10}}},
		AIRack:      &Rack{},
		Bag:         fullBag(),
		CurrentTurn: HumanTurn,
	}
	before := bagOrder(state.Bag)

	cmd := &ExchangeCommand{Move: ExchangeMove{Tiles: []Tile{{Letter: 'Q', Points: 10}}}}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	cmd.Undo(state, nil)

	if !sameOrder(before, bagOrder(state.Bag)) {
		t.Error("undoing an exchange with a nil rng did not restore the bag order exactly")
	}
}

// TestUndo_RestoresEverythingButBagOrder verifies the rest of the state is still restored
// exactly, so reshuffling the bag has not weakened undo anywhere else.
func TestUndo_RestoresEverythingButBagOrder(t *testing.T) {
	state := &GameState{
		Board:             newFlatBoard(),
		HumanRack:         &Rack{tiles: []Tile{{Letter: 'A', Points: 1}, {Letter: 'T', Points: 1}}},
		AIRack:            &Rack{tiles: []Tile{{Letter: 'E', Points: 1}}},
		Bag:               fullBag(),
		CurrentTurn:       HumanTurn,
		HumanScore:        11,
		AIScore:           7,
		ConsecutivePasses: 3,
		MoveNumber:        4,
	}
	wantHuman, wantAI := state.HumanScore, state.AIScore
	wantPasses, wantMove, wantTurn := state.ConsecutivePasses, state.MoveNumber, state.CurrentTurn
	wantRack := letterCounts(state.HumanRack.Tiles())
	wantBagCount := state.Bag.Count()

	cmd := &PlayCommand{Move: PlayMove{Placed: []PlacedTile{
		{Tile: Tile{Letter: 'A', Points: 1}, Row: 7, Col: 7},
		{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 8},
	}}}
	if err := cmd.Execute(state, testDict, deterministicRNG()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	cmd.Undo(state, rand.New(rand.NewPCG(3, 0)))

	if state.HumanScore != wantHuman || state.AIScore != wantAI {
		t.Errorf("scores = %d/%d, want %d/%d", state.HumanScore, state.AIScore, wantHuman, wantAI)
	}
	if state.ConsecutivePasses != wantPasses {
		t.Errorf("ConsecutivePasses = %d, want %d", state.ConsecutivePasses, wantPasses)
	}
	if state.MoveNumber != wantMove || state.CurrentTurn != wantTurn {
		t.Errorf("MoveNumber/CurrentTurn = %d/%v, want %d/%v",
			state.MoveNumber, state.CurrentTurn, wantMove, wantTurn)
	}
	if !sameCounts(letterCounts(state.HumanRack.Tiles()), wantRack) {
		t.Errorf("rack = %v, want %v", letterCounts(state.HumanRack.Tiles()), wantRack)
	}
	if state.Bag.Count() != wantBagCount {
		t.Errorf("bag count = %d, want %d", state.Bag.Count(), wantBagCount)
	}
	if !state.Board.IsEmpty(7, 7) || !state.Board.IsEmpty(7, 8) {
		t.Error("board was not cleared by undo")
	}
}

// TestIsGameOver_GoingOutBeatsSixScorelessTurns verifies that a zero-scoring play which empties
// the last rack is reported as going out, not as the sixth scoreless turn. Both conditions
// become true on that turn; reporting the scoreless-turn ending would deduct both racks and
// deny the going-out bonus to a player who did use their last letter.
func TestIsGameOver_GoingOutBeatsSixScorelessTurns(t *testing.T) {
	state := &GameState{
		Board:             newFlatBoard(),
		HumanRack:         &Rack{}, // played out
		AIRack:            &Rack{tiles: []Tile{{Letter: 'Q', Points: 10}}},
		Bag:               newTestBag(nil), // empty, so no replenish
		ConsecutivePasses: 6,               // the same turn also reached the scoreless limit
		HumanScore:        50,
		AIScore:           40,
	}

	over, reason := IsGameOver(state)
	if !over {
		t.Fatal("game should be over")
	}
	if reason != RackExhausted {
		t.Fatalf("reason = %v, want RackExhausted (going out takes precedence)", reason)
	}

	ApplyEndgameScoring(state, reason)
	// Going out: the human gains the AI's remaining tiles, the AI loses them.
	if state.HumanScore != 60 || state.AIScore != 30 {
		t.Errorf("scores = %d/%d, want 60/30 (going-out bonus applied)", state.HumanScore, state.AIScore)
	}
}

// TestIsGameOver_SixScorelessTurnsWhenNobodyPlayedOut verifies the scoreless-turn ending still
// fires when it is the only condition met — the case it exists for.
func TestIsGameOver_SixScorelessTurnsWhenNobodyPlayedOut(t *testing.T) {
	state := &GameState{
		Board:             newFlatBoard(),
		HumanRack:         &Rack{tiles: []Tile{{Letter: 'A', Points: 1}}},
		AIRack:            &Rack{tiles: []Tile{{Letter: 'Q', Points: 10}}},
		Bag:               newTestBag(nil),
		ConsecutivePasses: 6,
		HumanScore:        50,
		AIScore:           40,
	}

	over, reason := IsGameOver(state)
	if !over || reason != SixConsecutivePasses {
		t.Fatalf("IsGameOver = %v,%v; want true,SixConsecutivePasses", over, reason)
	}

	ApplyEndgameScoring(state, reason)
	// Each player loses their own remaining tiles, with no redistribution.
	if state.HumanScore != 49 || state.AIScore != 30 {
		t.Errorf("scores = %d/%d, want 49/30", state.HumanScore, state.AIScore)
	}
}
