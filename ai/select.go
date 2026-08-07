// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ai is documented in doc.go.
package ai

import (
	"math"
	"math/rand/v2"
)

// MinLevel and MaxLevel bound the difficulty levels SelectMove accepts. They are exported so
// the UI's slider and its persisted-settings validation use the same range the AI does,
// rather than each restating it.
const (
	// MinLevel is the easiest difficulty.
	MinLevel = 1
	// MaxLevel is the hardest difficulty: GodModeLevel.
	MaxLevel = GodModeLevel
)

// GodModeLevel is the difficulty at which the AI always plays the single best move available,
// with no randomness at all. It sits above NearBestLevel so a player who wants a perfect
// opponent can ask for one explicitly, while the top ordinary level still plays like a strong
// human rather than a solver.
const GodModeLevel = 11

// NearBestLevel is the highest ordinary difficulty. It plays one of the near-best moves rather
// than always the optimum — see topPlayMargin.
const NearBestLevel = 10

// topPlayMargin is how far below the best available score a NearBestLevel play may fall, as a
// fraction of that best score.
//
// The window is defined by score rather than by rank because a rank says nothing about how
// much is given up. Where the three best plays score 40, 12 and 11, sampling the top three
// would surrender most of the turn's value two times out of three — that is not an expert
// playing imperfectly, it is a weak player. A score window gives up a bounded amount by
// construction, at most this fraction of the best play, so level 10 stays strong while no
// longer always finding the single best move.
const topPlayMargin = 0.10

// SelectMove chooses one candidate from a sorted MoveCandidate slice according to
// the difficulty level — FR-05 / BR-AI-04 / BR-AI-05.
//
// candidates must be sorted by Score descending then OpponentAccess ascending
// (as returned by GenerateMoves). SelectMove does not re-sort.
//
// GodModeLevel — perfect: always returns candidates[0], the highest-scoring move with the
// lowest opponent access as a tiebreaker. No randomness is used, so the same board and rack
// always produce the same move (BR-AI-04).
//
// NearBestLevel — near-best: sample uniformly from the plays scoring within topPlayMargin of
// the best one (BR-AI-04). The same board and rack can therefore yield different moves, which
// is deliberate: an opponent that unfailingly finds the single optimum is both predictable and
// unlike any human expert. What it never does is play a weak move.
//
// Levels 1–9 — randomised: compute k = max(1, round(total × (1 − (level−1)/9))),
// then sample uniformly from candidates[:k] using rng (BR-AI-05).
//
//	Example level→k mapping for total=100:
//	  level 1  → k=100 (full set)
//	  level 5  → k=56  (round(100 × 0.556))
//	  level 9  → k=11  (round(100 × 0.111))
//	  level 10 → the plays within topPlayMargin of the best score
//	  level 11 → candidates[0] (perfect, deterministic)
//
// rng must be non-nil for every level below GodModeLevel. Panics if candidates is empty —
// callers must check len(candidates) > 0 (NFR-AI-R2).
func SelectMove(candidates []MoveCandidate, level int, rng *rand.Rand) MoveCandidate {
	if len(candidates) == 0 {
		panic("ai.SelectMove: empty candidates slice")
	}

	// Clamp out-of-range levels (e.g. from a tampered or corrupt save) into [MinLevel, MaxLevel].
	if level < MinLevel {
		level = MinLevel
	}
	if level > MaxLevel {
		level = MaxLevel
	}

	// God mode: the single best play, every time — BR-AI-04.
	if level == GodModeLevel {
		return candidates[0]
	}

	// Near-best: one of the plays comparable to the best — BR-AI-04.
	if level == NearBestLevel {
		return candidates[rng.IntN(nearBestCount(candidates))]
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
	return candidates[rng.IntN(k)]
}

// nearBestCount returns how many leading candidates score within topPlayMargin of the best.
// Used by NearBestLevel; GodModeLevel bypasses it and takes the first candidate outright.
// candidates must be non-empty and sorted by score descending, so the qualifying candidates
// are a prefix and the result is always at least 1.
func nearBestCount(candidates []MoveCandidate) int {
	best := candidates[0].Score
	if best <= 0 {
		// Every play is worth the same, so there is nothing for a score window to choose
		// between; fall back to the first, which the sort gives the lowest OpponentAccess.
		return 1
	}
	cutoff := float64(best) * (1 - topPlayMargin)
	n := 1
	for n < len(candidates) && float64(candidates[n].Score) >= cutoff {
		n++
	}
	return n
}
