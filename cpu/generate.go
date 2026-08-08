// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package cpu is documented in doc.go.
package cpu

import (
	"sort"

	"tilewords/dictionary"
	"tilewords/engine"
)

// GenerateMoves enumerates every legal move for the given board, rack, and dictionary
// using the GADDAG left-extension algorithm — Appel-Jacobson (1988/1998) §5.
//
// The returned slice is sorted by Score descending, then OpponentAccess ascending
// (BR-AI-03). Every candidate has passed engine.ValidatePlacement (BR-AI-01).
//
// Returns an empty (non-nil) slice when the rack is empty or no legal play exists.
// Never panics on a valid (board, rack, dict) input (NFR-AI-R1).
func GenerateMoves(board *engine.Board, rack *engine.Rack, dict *dictionary.Dictionary) []MoveCandidate {
	candidates := make([]MoveCandidate, 0)

	if rack.Count() == 0 {
		return candidates
	}

	g := dict.GADDAG()
	counts := rackToCountArray(rack)
	seen := make(map[moveKey]bool)

	// Precompute cross-checks once per direction — NFR-AI-P4.
	// For a horizontal play, perpendicular words are vertical; vice versa.
	ccH := computeCrossChecks(board, dict, dirHorizontal)
	ccV := computeCrossChecks(board, dict, dirVertical)

	// Gather anchors for each direction and run the GADDAG traversal.
	for _, dir := range []direction{dirHorizontal, dirVertical} {
		var cc *[15][15][26]bool
		if dir == dirHorizontal {
			cc = &ccH
		} else {
			cc = &ccV
		}

		anchors := findAnchors(board, dir)
		for _, anchor := range anchors {
			// The left phase starts at the anchor itself: the anchor tile becomes the
			// first (rightmost) letter of the reversed prefix.
			extendLeft(board, g, dict, cc, &counts, anchor,
				g.Root(), anchor.row, anchor.col, anchor.limit, nil, &candidates, seen)
		}
	}

	// Sort: primary key score descending, secondary key OpponentAccess ascending (BR-AI-03).
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].OpponentAccess < candidates[j].OpponentAccess
	})

	return candidates
}
