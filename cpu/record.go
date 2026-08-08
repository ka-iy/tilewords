// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package cpu is documented in doc.go.
package cpu

import (
	"sort"

	"tilewords/dictionary"
	"tilewords/engine"
)

// recordCandidate validates a candidate placement, scores it, and appends it to
// candidates if it is legal. Duplicates (same physical play from different anchors)
// are discarded using the seen map — BR-AI-14.
//
// This function acts as the defensive ValidatePlacement gate described in Pattern 5:
// the GADDAG traversal is sufficient but not formally proven; any edge-case invalid
// candidate is silently discarded here rather than surfacing as a bug.
func recordCandidate(
	board *engine.Board,
	dict *dictionary.Dictionary,
	placed []engine.PlacedTile,
	candidates *[]MoveCandidate,
	seen map[moveKey]bool,
) {
	if len(placed) == 0 {
		return
	}

	// Build a move key from the footprint of newly placed tiles.
	key := makeMoveKey(placed)
	if seen[key] {
		return // duplicate: same physical play already recorded
	}

	move := engine.PlayMove{Placed: placed}

	// Defensive gate: validate the placement using the engine's authoritative rule checker.
	// ValidatePlacement also populates move.WordsFormed, which Score requires.
	if _, err := engine.ValidatePlacement(board, &move, dict); err != nil {
		return // traversal produced an edge-case invalid move; discard silently
	}

	score, err := engine.Score(board, &move)
	if err != nil {
		return
	}

	access := computeOpponentAccess(board, placed)

	*candidates = append(*candidates, MoveCandidate{
		Move:           move,
		Score:          score,
		OpponentAccess: access,
	})
	seen[key] = true
}

// computeOpponentAccess counts the number of empty premium squares that have at
// least one orthogonally adjacent tile after the proposed move is played.
// "Adjacent tile" means any existing board tile OR any of the newly placed tiles.
// This is Option A (total exposure) — BR-AI-06.
//
// Premium squares counted: DoubleLetter, TripleLetter, DoubleWord, TripleWord, Centre.
func computeOpponentAccess(board *engine.Board, placed []engine.PlacedTile) int {
	// Build a grid of all occupied cells after the move (existing + newly placed).
	// Using a fixed-size array avoids heap allocation on every candidate evaluation.
	var occupied [15][15]bool
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			if !board.IsEmpty(r, c) {
				occupied[r][c] = true
			}
		}
	}
	for _, pt := range placed {
		occupied[pt.Row][pt.Col] = true
	}

	// Count empty premium squares with at least one occupied orthogonal neighbour.
	count := 0
	deltas := [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			if occupied[r][c] {
				continue
			}
			sq := board.Cell(r, c).Square
			if sq == engine.Normal {
				continue
			}
			for _, d := range deltas {
				nr, nc := r+d[0], c+d[1]
				if nr >= 0 && nr < 15 && nc >= 0 && nc < 15 && occupied[nr][nc] {
					count++
					break
				}
			}
		}
	}
	return count
}

// makeMoveKey builds a canonical key from the placed tiles — each tile's row,
// col, letter, and blank flag, ordered by position. Two different words that
// share a footprint (e.g. CAT vs COT), and a real-letter play vs the blank-letter
// play of the same word at the same cells, therefore produce distinct keys, so a
// legal (possibly higher-scoring) alternative is never dropped by the seen map.
func makeMoveKey(placed []engine.PlacedTile) moveKey {
	sorted := make([]engine.PlacedTile, len(placed))
	copy(sorted, placed)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Row != sorted[j].Row {
			return sorted[i].Row < sorted[j].Row
		}
		return sorted[i].Col < sorted[j].Col
	})
	buf := make([]byte, 0, len(sorted)*4)
	for _, pt := range sorted {
		blank := byte(0)
		if pt.Tile.IsBlank {
			blank = 1
		}
		buf = append(buf, byte(pt.Row), byte(pt.Col), pt.Tile.Letter, blank)
	}
	return moveKey(buf)
}
