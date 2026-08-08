// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package cpu is documented in doc.go.
package cpu

import "tilewords/engine"

// MoveCandidate is a fully validated, scored play candidate returned by GenerateMoves.
type MoveCandidate struct {
	Move           engine.PlayMove
	Score          int
	OpponentAccess int // count of premium squares newly accessible to the opponent
}

// direction indicates the axis along which tiles are placed.
type direction int

const (
	dirHorizontal direction = iota
	dirVertical
)

// anchorSquare is an empty cell that anchors a GADDAG left-extension search.
// limit is the maximum number of rack tiles that may be placed to the left of
// (row, col) in this direction (Appel-Jacobson §5 "BeforeAnchor" bound). It is
// set to the rack capacity: existing board tiles to the left are navigated for
// free, and plays reachable from more than one anchor are deduplicated by the
// seen map, so no tighter per-anchor bound is needed for correctness.
type anchorSquare struct {
	row, col, limit int
	dir             direction
}

// rackCounts is a compact representation of the tiles available in a rack.
// Indices 0–25 correspond to letters A–Z; index 26 holds the blank count.
// Operations are O(1) increment/decrement — no allocation during traversal.
type rackCounts [27]int

// moveKey uniquely identifies a physical play by the exact tiles it places —
// each tile's position, letter, and blank flag. Two traversal paths that produce
// an identical placement share a key and are deduplicated, but two different words
// that merely share a bounding box (e.g. CAT vs COT, or a real letter vs a blank
// at the same cells) produce distinct keys and are both kept.
type moveKey string

// rackToCountArray converts a rack into a rackCounts array for traversal.
//
// A tile that is neither a blank nor an A-Z letter is skipped rather than counted: its
// letter cannot be played, and indexing by it would run off the end of the array. Racks
// decoded from a save file are not guaranteed to hold only well-formed tiles (a single
// flipped bit in the encoded IsBlank flag turns a blank into Letter 0), and byte
// subtraction wraps, so an unchecked index here would abort the process from the CPU
// goroutine — where there is no recover — leaving a save that crashes on every load.
func rackToCountArray(rack *engine.Rack) rackCounts {
	var counts rackCounts
	for _, t := range rack.Tiles() {
		switch {
		case t.IsBlank:
			counts[26]++
		case t.Letter >= 'A' && t.Letter <= 'Z':
			counts[t.Letter-'A']++
		}
	}
	return counts
}

// findAnchors returns all anchor squares for the given board and direction.
//
// An anchor is any empty cell that is orthogonally adjacent to an occupied cell.
// On an empty board the single anchor is the centre cell (7, 7).
//
// Every anchor is given a left-extension budget equal to the rack capacity. A
// play can never place more than a full rack of tiles, so this never under-counts;
// over-generation from multiple anchors reaching the same play is deduplicated by
// the seen map in recordCandidate.
func findAnchors(board *engine.Board, dir direction) []anchorSquare {
	var anchors []anchorSquare

	if !board.HasAnyTile() {
		// Empty board: single anchor at the centre.
		anchors = append(anchors, anchorSquare{row: 7, col: 7, limit: engine.MaxRackSize, dir: dir})
		return anchors
	}

	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			if !board.IsEmpty(r, c) {
				continue
			}
			if !hasOccupiedNeighbour(board, r, c) {
				continue
			}
			anchors = append(anchors, anchorSquare{row: r, col: c, limit: engine.MaxRackSize, dir: dir})
		}
	}
	return anchors
}

// hasOccupiedNeighbour reports whether any of the four orthogonal neighbours of
// (r, c) contains a tile. Out-of-bounds neighbours are treated as empty.
func hasOccupiedNeighbour(board *engine.Board, r, c int) bool {
	dirs := [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
	for _, d := range dirs {
		nr, nc := r+d[0], c+d[1]
		if nr >= 0 && nr < 15 && nc >= 0 && nc < 15 && !board.IsEmpty(nr, nc) {
			return true
		}
	}
	return false
}
