// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package engine is documented in doc.go.
package engine

import (
	"fmt"

	"tilewords/dictionary"
)

// ValidatePlacement checks that move is legally playable on board given dict.
// On success it populates move.WordsFormed and returns the list of words formed.
// Returns a descriptive error for any rule violation (BR-E01 through BR-E05).
func ValidatePlacement(board *Board, move *PlayMove, dict *dictionary.Dictionary) ([]string, error) {
	// Step 1 — Sanity checks (BR-E01)
	if len(move.Placed) == 0 {
		return nil, fmt.Errorf("engine.ValidatePlacement: no tiles placed")
	}
	if len(move.Placed) > MaxRackSize {
		return nil, fmt.Errorf("engine.ValidatePlacement: too many tiles placed (%d > %d)",
			len(move.Placed), MaxRackSize)
	}

	// Step 2 — Orientation: all tiles must be in one row or one column (BR-E03)
	allSameRow := true
	allSameCol := true
	r0, c0 := move.Placed[0].Row, move.Placed[0].Col
	for _, pt := range move.Placed[1:] {
		if pt.Row != r0 {
			allSameRow = false
		}
		if pt.Col != c0 {
			allSameCol = false
		}
	}
	if !allSameRow && !allSameCol {
		return nil, fmt.Errorf("engine.ValidatePlacement: tiles must all be in one row or one column")
	}

	// Step 3 — Occupancy: no placed tile may overwrite an existing tile (BR-E01), and no
	// two placed tiles may claim the same cell. Checking each tile only against the board
	// would let a move stack tiles on one square: the extra entries inflate len(Placed) into
	// an unearned bingo bonus, and Board.Place rejects the repeat only after the rack has
	// already been debited, which PlayCommand.Execute cannot recover from.
	var seen [15][15]bool
	for _, pt := range move.Placed {
		if pt.Row < 0 || pt.Row > 14 || pt.Col < 0 || pt.Col > 14 {
			return nil, fmt.Errorf("engine.ValidatePlacement: position (%d,%d) is out of bounds",
				pt.Row, pt.Col)
		}
		if seen[pt.Row][pt.Col] {
			return nil, fmt.Errorf("engine.ValidatePlacement: position (%d,%d) is used by more than one tile",
				pt.Row, pt.Col)
		}
		seen[pt.Row][pt.Col] = true
		if !board.IsEmpty(pt.Row, pt.Col) {
			return nil, fmt.Errorf("engine.ValidatePlacement: cell (%d,%d) is already occupied",
				pt.Row, pt.Col)
		}
	}

	// Step 4 — Contiguity: no gaps between first and last tile (BR-E03)
	horiz := isHorizontal(move.Placed)
	if err := checkContiguity(board, move.Placed, horiz); err != nil {
		return nil, fmt.Errorf("engine.ValidatePlacement: %w", err)
	}

	// Step 5 — First-move and adjacency rules
	if !board.HasAnyTile() {
		// First move must cover the centre square (BR-E02).
		if !covers(move.Placed, 7, 7) {
			return nil, fmt.Errorf("engine.ValidatePlacement: first move must cover the centre square (7,7)")
		}
	} else {
		// Subsequent moves must be adjacent to at least one existing tile (BR-E04).
		if !hasAdjacent(board, move.Placed) {
			return nil, fmt.Errorf("engine.ValidatePlacement: move must connect to an existing tile on the board")
		}
	}

	// Step 6 — Word extraction
	words := extractWords(board, move)

	// Step 7 — At least one word of length ≥ 2 must be formed (BR-E05).
	// A single-tile play with no adjacent tiles (e.g. a lone tile on an empty board)
	// produces an empty word list and is not a legal move.
	if len(words) == 0 {
		return nil, fmt.Errorf("engine.ValidatePlacement: placement forms no valid word")
	}

	// Step 8 — Dictionary validation of all formed words (BR-E05)
	for _, w := range words {
		if !dict.Validate(w) {
			return nil, fmt.Errorf("engine.ValidatePlacement: %q is not a valid word", w)
		}
	}

	// Step 9 — Cache formed words on the move for use by Score and the UI
	move.WordsFormed = words
	return words, nil
}

// checkContiguity verifies that there are no gaps between the first and last
// placed tile along the primary axis, considering existing board tiles.
func checkContiguity(board *Board, placed []PlacedTile, horiz bool) error {
	if len(placed) <= 1 {
		return nil
	}
	minIdx, maxIdx := 0, 0
	if horiz {
		minIdx, maxIdx = placed[0].Col, placed[0].Col
		for _, pt := range placed[1:] {
			if pt.Col < minIdx {
				minIdx = pt.Col
			}
			if pt.Col > maxIdx {
				maxIdx = pt.Col
			}
		}
		row := placed[0].Row
		for i := minIdx; i <= maxIdx; i++ {
			if board.IsEmpty(row, i) && !covers(placed, row, i) {
				return fmt.Errorf("gap at column %d in row %d", i, row)
			}
		}
	} else {
		minIdx, maxIdx = placed[0].Row, placed[0].Row
		for _, pt := range placed[1:] {
			if pt.Row < minIdx {
				minIdx = pt.Row
			}
			if pt.Row > maxIdx {
				maxIdx = pt.Row
			}
		}
		col := placed[0].Col
		for i := minIdx; i <= maxIdx; i++ {
			if board.IsEmpty(i, col) && !covers(placed, i, col) {
				return fmt.Errorf("gap at row %d in column %d", i, col)
			}
		}
	}
	return nil
}

// hasAdjacent reports whether any placed tile is orthogonally adjacent to an
// existing board tile (BR-E04).
func hasAdjacent(board *Board, placed []PlacedTile) bool {
	dirs := [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, pt := range placed {
		for _, d := range dirs {
			nr, nc := pt.Row+d[0], pt.Col+d[1]
			if nr < 0 || nr > 14 || nc < 0 || nc > 14 {
				continue
			}
			if !board.IsEmpty(nr, nc) {
				return true
			}
		}
	}
	return false
}

// IsGameOver reports whether state meets any end condition (BR-E10).
//
// Rack exhaustion is tested first, because both conditions can become true on the same turn:
// a zero-scoring play that empties the last rack is a scoreless turn, so it can push the
// counter to six while also playing the player out. Going out is the stronger claim — that
// player used their last letter, which is what earns the going-out bonus, whereas the
// scoreless-turn rule exists for a game where nobody can play at all. Testing passes first
// would report SixConsecutivePasses and deduct both racks, denying the bonus to a player who
// had in fact gone out.
func IsGameOver(state *GameState) (bool, EndReason) {
	// One player exhausted their rack while the bag is empty (BR-E11).
	if state.Bag.Count() == 0 {
		if state.HumanRack.Count() == 0 || state.AIRack.Count() == 0 {
			return true, RackExhausted
		}
	}
	// Six consecutive non-play moves across both players (BR-E10).
	if state.ConsecutivePasses >= 6 {
		return true, SixConsecutivePasses
	}
	return false, NotOver
}

// ApplyEndgameScoring adjusts scores when the game ends (BR-E11, BR-E12). reason
// must be the EndReason returned by IsGameOver, which is the authoritative record
// of how the game ended. The function is idempotent: repeated calls are no-ops, so
// a stray second invocation cannot double-adjust the scores.
func ApplyEndgameScoring(state *GameState, reason EndReason) {
	// A game that has not ended has nothing to adjust. This is checked before the
	// EndgameScored latch is set, so a caller that forwards IsGameOver's reason without
	// checking its bool cannot both deduct the racks from a live game and suppress the real
	// adjustment when the game does end.
	if reason == NotOver {
		return
	}
	if state.EndgameScored {
		return
	}
	state.EndgameScored = true

	humanRemaining := sumRackPoints(state.HumanRack)
	aiRemaining := sumRackPoints(state.AIRack)

	switch reason {
	case RackExhausted:
		// The player who emptied their rack gains the opponent's remaining tile
		// values; the opponent loses them (the going-out bonus, BR-E11). At least
		// one rack is empty here by IsGameOver's definition of RackExhausted.
		if state.HumanRack.Count() == 0 {
			state.HumanScore += aiRemaining
			state.AIScore -= aiRemaining
		} else {
			state.AIScore += humanRemaining
			state.HumanScore -= humanRemaining
		}
	case SixConsecutivePasses:
		// Six consecutive scoreless turns (BR-E12): each player loses their own remaining
		// tile values, with no redistribution.
		state.HumanScore -= humanRemaining
		state.AIScore -= aiRemaining
	}
}

// sumRackPoints returns the sum of point values of all tiles in r.
func sumRackPoints(r *Rack) int {
	total := 0
	for _, t := range r.tiles {
		total += t.Points
	}
	return total
}
