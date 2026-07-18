// Package ai is documented in doc.go.
package ai

import (
	"tilewords/dictionary"
	"tilewords/engine"
)

// computeCrossChecks builds the valid-letter set for every empty cell in dir's
// perpendicular direction. The result is a [15][15][26]bool where index [r][c][l]
// is true if letter ('A'+l) may be placed at (r,c) without forming an invalid
// perpendicular word — Appel-Jacobson §5 cross-check precomputation (Pattern 3).
//
// For a horizontal play, cross-checks constrain vertical words; for a vertical
// play, they constrain horizontal words.
//
// Cells with no perpendicular neighbours receive an all-true set: any letter is
// valid because no perpendicular word is formed at all.
// Occupied cells also receive an all-true set: the traversal never places a tile
// on an occupied cell, so their cross-check set is never consulted.
func computeCrossChecks(board *engine.Board, dict *dictionary.Dictionary, dir direction) [15][15][26]bool {
	var cc [15][15][26]bool
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			if !board.IsEmpty(r, c) {
				// Occupied — set all-true; cross-check is not consulted here.
				setAllTrue(&cc[r][c])
				continue
			}
			// Collect the perpendicular prefix (tiles before this cell) and
			// suffix (tiles after this cell) in the direction perpendicular to dir.
			prefix := collectPerp(board, r, c, dir, true)
			suffix := collectPerp(board, r, c, dir, false)
			if len(prefix) == 0 && len(suffix) == 0 {
				// No perpendicular neighbours: any letter is valid.
				setAllTrue(&cc[r][c])
				continue
			}
			// For each candidate letter, check whether prefix+letter+suffix is a valid word.
			for l := byte('A'); l <= 'Z'; l++ {
				word := string(prefix) + string(l) + string(suffix)
				cc[r][c][l-'A'] = dict.Validate(word)
			}
		}
	}
	return cc
}

// collectPerp gathers the contiguous run of letters on the board in the direction
// perpendicular to dir, starting adjacent to (r, c).
//
// If backward is true, tiles are collected in the "before" direction (left for
// vertical dir, above for horizontal dir) and returned in reading order (reversed).
// If backward is false, tiles are collected in the "after" direction and returned
// as-is.
//
// For a horizontal play (dir == dirHorizontal), perpendicular is vertical:
//
//	backward=true  → scan upward   from (r-1, c)
//	backward=false → scan downward from (r+1, c)
//
// For a vertical play (dir == dirVertical), perpendicular is horizontal:
//
//	backward=true  → scan leftward  from (r, c-1)
//	backward=false → scan rightward from (r, c+1)
func collectPerp(board *engine.Board, r, c int, dir direction, backward bool) []byte {
	var dr, dc int
	if dir == dirHorizontal {
		// Perpendicular to horizontal play is vertical.
		if backward {
			dr, dc = -1, 0 // scan upward
		} else {
			dr, dc = 1, 0 // scan downward
		}
	} else {
		// Perpendicular to vertical play is horizontal.
		if backward {
			dr, dc = 0, -1 // scan leftward
		} else {
			dr, dc = 0, 1 // scan rightward
		}
	}

	var letters []byte
	nr, nc := r+dr, c+dc
	for nr >= 0 && nr < 15 && nc >= 0 && nc < 15 && !board.IsEmpty(nr, nc) {
		// Blanks on the board carry their assigned letter (DisplayLetter handles this).
		letters = append(letters, board.Cell(nr, nc).Tile.DisplayLetter())
		nr += dr
		nc += dc
	}

	if backward {
		// Reverse so the result reads in the natural word direction (left→right or top→bottom).
		for i, j := 0, len(letters)-1; i < j; i, j = i+1, j-1 {
			letters[i], letters[j] = letters[j], letters[i]
		}
	}
	return letters
}

// setAllTrue fills all 26 entries of a letter-set with true.
func setAllTrue(cc *[26]bool) {
	for i := range cc {
		cc[i] = true
	}
}
