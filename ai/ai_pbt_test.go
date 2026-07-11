package ai_test

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"

	"pgregory.net/rapid"

	"squabble/ai"
	"squabble/dictionary"
	"squabble/engine"
)

// boardStateGen generates a valid game state by replaying a random number of AI
// moves on an empty board. The resulting board has 0–N words placed.
func boardStateGen() *rapid.Generator[*engine.GameState] {
	return rapid.Custom(func(t *rapid.T) *engine.GameState {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewSource(seed))
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
		rng := rand.New(rand.NewSource(seed))
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
		rng := rand.New(rand.NewSource(seed))
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
		rng := rand.New(rand.NewSource(seed))
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
func TestPBT_AI_Level10Deterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewSource(seed))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		rack := simpleRackGen().Draw(t, "rack")

		c1 := ai.GenerateMoves(state.Board, rack, testDict)
		c2 := ai.GenerateMoves(state.Board, rack, testDict)

		if len(c1) == 0 {
			return // no candidates: nothing to check
		}

		rng1 := rand.New(rand.NewSource(42))
		rng2 := rand.New(rand.NewSource(99)) // different seed — must not affect level 10
		m1 := ai.SelectMove(c1, 10, rng1)
		m2 := ai.SelectMove(c2, 10, rng2)

		if m1.Score != m2.Score {
			t.Fatalf("level 10 non-deterministic: scores %d vs %d", m1.Score, m2.Score)
		}
		if m1.OpponentAccess != m2.OpponentAccess {
			t.Fatalf("level 10 non-deterministic: OpponentAccess %d vs %d",
				m1.OpponentAccess, m2.OpponentAccess)
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

		rng := rand.New(rand.NewSource(seed))
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
		rng := rand.New(rand.NewSource(seed))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		state.CurrentTurn = engine.AITurn

		level := rapid.IntRange(1, 10).Draw(t, "level")
		rng2 := rand.New(rand.NewSource(rapid.Int64().Draw(t, "seed2")))

		move := ai.ChooseMove(state, testDict, level, rng2)
		if move == nil {
			t.Fatal("ChooseMove returned nil")
		}
	})
}

// TestPBT_AI_Worker_NoRace (PBT-AI-07): AIWorker produces valid moves under race
// detector without data races. Run with go test -race.
func TestPBT_AI_Worker_NoRace(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seed := rapid.Int64().Draw(t, "seed")
		rng := rand.New(rand.NewSource(seed))
		state := engine.New(dictionary.DictENABLE, 10, rng)
		state.CurrentTurn = engine.AITurn

		worker := ai.NewAIWorker(ai.ChooseMove)
		worker.Request(state, testDict, 10)

		deadline := time.Now().Add(5 * time.Second)
		var gotMove engine.Move
		for time.Now().Before(deadline) {
			if move, ok := worker.Poll(); ok {
				gotMove = move
				break
			}
			time.Sleep(time.Millisecond)
		}
		if gotMove == nil {
			t.Fatal("AIWorker did not produce a move within 5 seconds")
		}
	})
}
