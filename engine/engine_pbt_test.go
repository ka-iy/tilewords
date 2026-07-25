// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"math/rand"
	"testing"

	"pgregory.net/rapid"
)

// tileGen generates a random non-blank lettered tile using values from tileDistribution.
func tileGen() *rapid.Generator[Tile] {
	entries := tileDistribution[1:] // skip blank (index 0)
	return rapid.Custom(func(t *rapid.T) Tile {
		idx := rapid.IntRange(0, len(entries)-1).Draw(t, "distIdx")
		e := entries[idx]
		return Tile{Letter: e.letter, Points: e.points}
	})
}

// TestPBT_BagCount: NewBag always contains exactly 100 tiles (PBT-E01).
func TestPBT_BagCount(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewSource(seed))
		bag := NewBag(rng)
		if bag.Count() != tileDistributionTotal {
			t.Fatalf("NewBag count = %d, want %d", bag.Count(), tileDistributionTotal)
		}
	})
}

// TestPBT_TileConservation: total tiles (bag + racks + board) always equals 100 (PBT-E07).
func TestPBT_TileConservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rng := deterministicRNG()
		state := New(testDict.Name(), 5, rng)

		boardCount := countBoardTiles(state.Board)
		total := state.Bag.Count() + state.HumanRack.Count() + state.AIRack.Count() + boardCount
		if total != tileDistributionTotal {
			t.Fatalf("tile conservation violated: bag=%d humanRack=%d aiRack=%d board=%d total=%d want=%d",
				state.Bag.Count(), state.HumanRack.Count(), state.AIRack.Count(), boardCount,
				total, tileDistributionTotal)
		}
	})
}

// TestPBT_ScoreNonNegative: Score(board, move) ≥ 0 for any valid move (PBT-E03).
func TestPBT_ScoreNonNegative(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		b := newFlatBoard()
		n := rapid.IntRange(2, MaxRackSize).Draw(t, "numTiles")
		placed := make([]PlacedTile, n)
		for i := 0; i < n; i++ {
			placed[i] = PlacedTile{
				Tile: tileGen().Draw(t, "tile"),
				Row:  0,
				Col:  i,
			}
		}
		// Construct a word string from the placed tiles.
		word := make([]byte, n)
		for i, pt := range placed {
			word[i] = pt.Tile.Letter
		}
		move := &PlayMove{
			Placed:      placed,
			WordsFormed: []string{string(word)},
		}
		score, err := Score(b, move)
		if err != nil {
			t.Fatalf("Score error: %v", err)
		}
		if score < 0 {
			t.Fatalf("Score = %d for move, want ≥ 0", score)
		}
	})
}

// TestPBT_BingoBonus: any 7-tile move yields score ≥ face sum + 50 (PBT-E04).
func TestPBT_BingoBonus(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		b := newFlatBoard()
		placed := make([]PlacedTile, MaxRackSize)
		faceSum := 0
		word := make([]byte, MaxRackSize)
		for i := 0; i < MaxRackSize; i++ {
			tile := tileGen().Draw(t, "tile")
			placed[i] = PlacedTile{Tile: tile, Row: 0, Col: i}
			faceSum += tile.Points
			word[i] = tile.Letter
		}
		move := &PlayMove{
			Placed:      placed,
			WordsFormed: []string{string(word)},
		}
		score, err := Score(b, move)
		if err != nil {
			t.Fatalf("Score error: %v", err)
		}
		if score < faceSum+50 {
			t.Fatalf("7-tile move: score=%d faceSum=%d; expected score ≥ %d", score, faceSum, faceSum+50)
		}
	})
}

// TestPBT_ScoreFlat_FaceValues: on a flat board (all Normal), Score == sum of face values (PBT-E08).
func TestPBT_ScoreFlat_FaceValues(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		b := newFlatBoard()
		// Use 2–6 tiles: ≥2 so the word is valid length; <7 to avoid the bingo bonus.
		n := rapid.IntRange(2, MaxRackSize-1).Draw(t, "numTiles")
		placed := make([]PlacedTile, n)
		faceSum := 0
		word := make([]byte, n)
		for i := 0; i < n; i++ {
			tile := tileGen().Draw(t, "tile")
			placed[i] = PlacedTile{Tile: tile, Row: 0, Col: i}
			faceSum += tile.Points
			word[i] = tile.Letter
		}
		move := &PlayMove{
			Placed:      placed,
			WordsFormed: []string{string(word)},
		}
		score, err := Score(b, move)
		if err != nil {
			t.Fatalf("Score error: %v", err)
		}
		if score != faceSum {
			t.Fatalf("flat board score = %d, want face sum %d", score, faceSum)
		}
	})
}

// TestPBT_ExecuteUndo_RoundTrip: Execute then Undo restores GameState (PBT-E06).
func TestPBT_ExecuteUndo_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rng := deterministicRNG()
		state := New(testDict.Name(), 5, rng)
		state.CurrentTurn = HumanTurn

		// Take a snapshot of key state values before.
		prevScore := state.HumanScore
		prevPasses := state.ConsecutivePasses
		prevTurn := state.CurrentTurn
		prevMoveNum := state.MoveNumber
		prevBagCount := state.Bag.Count()
		prevRackCount := state.HumanRack.Count()

		cmd := &PassCommand{}
		rng2 := deterministicRNG()
		if err := cmd.Execute(state, testDict, rng2); err != nil {
			t.Fatalf("Execute: %v", err)
		}
		cmd.Undo(state, nil)

		if state.HumanScore != prevScore {
			t.Fatalf("HumanScore: got %d, want %d", state.HumanScore, prevScore)
		}
		if state.ConsecutivePasses != prevPasses {
			t.Fatalf("ConsecutivePasses: got %d, want %d", state.ConsecutivePasses, prevPasses)
		}
		if state.CurrentTurn != prevTurn {
			t.Fatalf("CurrentTurn: got %v, want %v", state.CurrentTurn, prevTurn)
		}
		if state.MoveNumber != prevMoveNum {
			t.Fatalf("MoveNumber: got %d, want %d", state.MoveNumber, prevMoveNum)
		}
		if state.Bag.Count() != prevBagCount {
			t.Fatalf("Bag.Count: got %d, want %d", state.Bag.Count(), prevBagCount)
		}
		if state.HumanRack.Count() != prevRackCount {
			t.Fatalf("HumanRack.Count: got %d, want %d", state.HumanRack.Count(), prevRackCount)
		}
	})
}

// TestPBT_ExistingTilesNoMultiplier: existing board tiles never receive premium
// multipliers (PBT-E05). Place a tile on a DL square first, then score a move
// using that cell as an existing tile — it should contribute only face value.
func TestPBT_ExistingTilesNoMultiplier(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		b := NewBoard()
		// (0,3) is a DoubleLetter square.
		existingTile := Tile{Letter: 'A', Points: 1}
		_ = b.Place(0, 3, existingTile)

		// Place T at (0,4): forms "AT" where A is an existing tile on DL.
		// Score should be: A=1 (no mult, pre-existing) + T=1 = 2 (no word mult).
		move := &PlayMove{
			Placed: []PlacedTile{
				{Tile: Tile{Letter: 'T', Points: 1}, Row: 0, Col: 4},
			},
			WordsFormed: []string{"AT"},
		}
		score, err := Score(b, move)
		if err != nil {
			t.Fatalf("Score: %v", err)
		}
		// A (existing, on DL) contributes 1×1=1; T (new, Normal) contributes 1.
		if score != 2 {
			t.Fatalf("score = %d, want 2 (existing tile on DL must not get multiplier)", score)
		}
	})
}

// countBoardTiles counts non-nil tiles on the board.
func countBoardTiles(b *Board) int {
	count := 0
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			if b.cells[r][c].Tile != nil {
				count++
			}
		}
	}
	return count
}
