// Package ai is documented in doc.go.
package ai

import (
	"math"
	"math/rand"
)

// SelectMove chooses one candidate from a sorted MoveCandidate slice according to
// the difficulty level — FR-05 / BR-AI-04 / BR-AI-05.
//
// candidates must be sorted by Score descending then OpponentAccess ascending
// (as returned by GenerateMoves). SelectMove does not re-sort.
//
// Level 10 — deterministic: always returns candidates[0], the highest-scoring move
// with the lowest opponent access as a tiebreaker. No RNG is used (BR-AI-04).
//
// Levels 1–9 — randomised: compute k = max(1, round(total × (1 − (level−1)/9))),
// then sample uniformly from candidates[:k] using rng (BR-AI-05).
//
//	Example level→k mapping for total=100:
//	  level 1  → k=100 (full set)
//	  level 5  → k=56  (round(100 × 0.556))
//	  level 9  → k=11  (round(100 × 0.111))
//	  level 10 → candidates[0] (deterministic)
//
// Panics if candidates is empty — callers must check len(candidates) > 0 (NFR-AI-R2).
func SelectMove(candidates []MoveCandidate, level int, rng *rand.Rand) MoveCandidate {
	if len(candidates) == 0 {
		panic("ai.SelectMove: empty candidates slice")
	}

	// Clamp out-of-range levels (e.g. from a tampered or corrupt save) into [1, 10].
	if level < 1 {
		level = 1
	}
	if level > 10 {
		level = 10
	}

	// Level 10: deterministic best move — BR-AI-04.
	if level == 10 {
		return candidates[0]
	}

	// Levels 1–9: k-formula — FR-05 / BR-AI-05.
	// fraction approaches 0 at level 9 (few candidates) and is 1.0 at level 1 (all candidates).
	fraction := 1.0 - float64(level-1)/9.0
	k := int(math.Round(float64(len(candidates)) * fraction))
	if k < 1 {
		k = 1
	}
	if k > len(candidates) {
		k = len(candidates)
	}
	return candidates[rng.Intn(k)]
}
