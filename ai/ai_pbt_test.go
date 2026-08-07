// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ai_test

import (
	"fmt"
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"pgregory.net/rapid"

	"tilewords/ai"
	"tilewords/dictionary"
	"tilewords/engine"
)

// boardStateGen generates a valid game state by replaying a random number of AI
// moves on an empty board. The resulting board has 0–N words placed.
func boardStateGen() *rapid.Generator[*engine.GameState] {
	return rapid.Custom(func(t *rapid.T) *engine.GameState {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewPCG(uint64(seed), 0))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		state.CurrentTurn = engine.AITurn

		// Play 0–4 AI moves to create board variety.
		moves := rapid.IntRange(0, 4).Draw(t, "moves")
		for i := 0; i < moves; i++ {
			move := ai.ChooseMove(state, testDict, 10, rng)
			switch m := move.(type) {
			case engine.PlayMove:
				cmd := &engine.PlayCommand{Move: m}
				_ = cmd.Execute(state, testDict, rng)
			default:
				// Pass or exchange — advance without modifying board significantly.
			}
			state.CurrentTurn = engine.AITurn
		}
		return state
	})
}

// simpleRackGen generates a rack with 1–7 tiles from a fixed letter set.
func simpleRackGen() *rapid.Generator[*engine.Rack] {
	return rapid.Custom(func(t *rapid.T) *engine.Rack {
		letters := []struct {
			l byte
			p int
		}{
			{'A', 1}, {'C', 3}, {'D', 2}, {'E', 1}, {'G', 2},
			{'H', 4}, {'I', 1}, {'N', 1}, {'R', 1}, {'S', 1},
			{'T', 1}, {'U', 1}, {'W', 4}, {'Y', 4},
		}
		n := rapid.IntRange(1, engine.MaxRackSize).Draw(t, "rackSize")
		rack := &engine.Rack{}
		tiles := make([]engine.Tile, n)
		for i := 0; i < n; i++ {
			idx := rapid.IntRange(0, len(letters)-1).Draw(t, "letter")
			tiles[i] = engine.Tile{Letter: letters[idx].l, Points: letters[idx].p}
		}
		_ = rack.Add(tiles)
		return rack
	})
}

// TestPBT_AI_AllCandidatesValid (PBT-AI-01): every candidate returned by
// GenerateMoves passes engine.ValidatePlacement with the same board and dictionary.
func TestPBT_AI_AllCandidatesValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewPCG(uint64(seed), 0))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		rack := simpleRackGen().Draw(t, "rack")

		candidates := ai.GenerateMoves(state.Board, rack, testDict)
		for i, c := range candidates {
			move := c.Move
			if _, err := engine.ValidatePlacement(state.Board, &move, testDict); err != nil {
				t.Fatalf("candidate %d failed ValidatePlacement: %v", i, err)
			}
		}
	})
}

// TestPBT_AI_CandidatesSorted (PBT-AI-02): result is sorted score-desc, OpponentAccess-asc.
func TestPBT_AI_CandidatesSorted(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewPCG(uint64(seed), 0))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		rack := simpleRackGen().Draw(t, "rack")

		candidates := ai.GenerateMoves(state.Board, rack, testDict)
		for i := 1; i < len(candidates); i++ {
			a, b := candidates[i-1], candidates[i]
			if a.Score < b.Score {
				t.Fatalf("not sorted: candidates[%d].Score=%d < candidates[%d].Score=%d",
					i-1, a.Score, i, b.Score)
			}
			if a.Score == b.Score && a.OpponentAccess > b.OpponentAccess {
				t.Fatalf("tie at score=%d: OpponentAccess out of order: [%d]=%d > [%d]=%d",
					a.Score, i-1, a.OpponentAccess, i, b.OpponentAccess)
			}
		}
	})
}

// TestPBT_AI_NoDuplicates (PBT-AI-03): no two candidates have the same move footprint.
func TestPBT_AI_NoDuplicates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewPCG(uint64(seed), 0))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		rack := simpleRackGen().Draw(t, "rack")

		candidates := ai.GenerateMoves(state.Board, rack, testDict)
		seen := make(map[string]bool)
		for i, c := range candidates {
			placed := c.Move.Placed
			if len(placed) == 0 {
				continue
			}
			// Build a string key from the full placed tile sequence.
			kstr := ""
			for _, pt := range placed {
				kstr += fmt.Sprintf("%d,%d,%c;", pt.Row, pt.Col, pt.Tile.Letter)
			}
			if seen[kstr] {
				t.Fatalf("duplicate candidate at index %d: %s", i, kstr)
			}
			seen[kstr] = true
		}
	})
}

// TestPBT_AI_Level10Deterministic (PBT-AI-04): two calls with the same inputs return
// identical MoveCandidate at level 10.
// TestPBT_AI_Level10PlaysNearBest (PBT-AI-04): whatever seed it is given, a level-10 play
// always scores within topPlayMargin of the best available play. Level 10 varies its choice
// among comparable plays rather than always taking the optimum, so the invariant worth
// holding is the bound on what it gives up, not that the move is identical every time.
func TestPBT_AI_Level10PlaysNearBest(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewPCG(uint64(seed), 0))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		rack := simpleRackGen().Draw(t, "rack")

		candidates := ai.GenerateMoves(state.Board, rack, testDict)
		if len(candidates) == 0 {
			return // no candidates: nothing to check
		}
		best := candidates[0].Score

		// Any seed must land inside the window; the margin is a fraction of the best score,
		// and a best score of zero admits only the first candidate.
		for _, s := range []int64{1, 42, 99, seed} {
			got := ai.SelectMove(candidates, 10, rand.New(rand.NewPCG(uint64(s), 0)))
			if best <= 0 {
				if got.Score != best {
					t.Fatalf("all plays score %d but level 10 returned %d", best, got.Score)
				}
				continue
			}
			if float64(got.Score) < float64(best)*0.9 {
				t.Fatalf("level 10 played %d, more than 10%% below the best available %d", got.Score, best)
			}
		}
	})
}

// TestPBT_AI_SelectMove_RangeCorrect (PBT-AI-05): the selected candidate for level L
// is always within candidates[:k] where k = round(total × (1-(L-1)/9)).
func TestPBT_AI_SelectMove_RangeCorrect(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(1, 50).Draw(t, "n")
		level := rapid.IntRange(1, 9).Draw(t, "level")
		seed := rapid.Int64().Draw(t, "seed")

		candidates := make([]ai.MoveCandidate, n)
		for i := range candidates {
			candidates[i] = ai.MoveCandidate{Score: n - i}
		}

		rng := rand.New(rand.NewPCG(uint64(seed), 0))
		selected := ai.SelectMove(candidates, level, rng)

		// Compute expected k using the same formula as SelectMove (BR-AI-05).
		fraction := 1.0 - float64(level-1)/9.0
		k := int(math.Round(float64(n) * fraction))
		if k < 1 {
			k = 1
		}
		if k > n {
			k = n
		}

		// Verify selected is within candidates[:k].
		found := false
		for i := 0; i < k; i++ {
			if candidates[i].Score == selected.Score {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("selected score %d not within candidates[:%d] (n=%d, level=%d)",
				selected.Score, k, n, level)
		}
	})
}

// TestPBT_AI_ChooseMove_NonNil (PBT-AI-06): ChooseMove always returns a non-nil move.
func TestPBT_AI_ChooseMove_NonNil(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewPCG(uint64(seed), 0))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		state.CurrentTurn = engine.AITurn

		level := rapid.IntRange(1, 10).Draw(t, "level")
		rng2 := rand.New(rand.NewPCG(uint64(rapid.Int64().Draw(t, "seed2")), 0))

		move := ai.ChooseMove(state, testDict, level, rng2)
		if move == nil {
			t.Fatal("ChooseMove returned nil")
		}
	})
}

// TestPBT_AI_OffGoroutine_NoRace (PBT-AI-07): choosing a move on a background goroutine,
// the way a UI must so its own thread never blocks, produces a valid move and no data race.
// The state handed over is a Clone, which is what makes the caller's live state safe to keep
// reading meanwhile. Run with go test -race.
func TestPBT_AI_OffGoroutine_NoRace(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewPCG(uint64(seed), 0))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		state.CurrentTurn = engine.AITurn

		// Snapshot and hand off exactly as ui does, with the goroutine owning its own rng.
		snapshot := state.Clone()
		result := make(chan engine.Move, 1)
		go func() {
			result <- ai.ChooseMove(snapshot, testDict, 10, rand.New(rand.NewPCG(uint64(seed), 0)))
		}()

		// Keep reading the live state while the AI works: a clone that shared anything
		// mutable with it would show up here under -race.
		for i := 0; i < 50; i++ {
			_ = state.Bag.Count()
			_ = state.AIRack.Count()
			_ = state.Board.HasAnyTile()
		}

		select {
		case gotMove := <-result:
			if gotMove == nil {
				t.Fatal("ChooseMove returned nil")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("ChooseMove did not produce a move within 5 seconds")
		}
	})
}
