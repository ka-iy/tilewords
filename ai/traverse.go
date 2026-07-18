// Package ai is documented in doc.go.
package ai

import (
	"tilewords/dictionary"
	"tilewords/engine"
)

// extendLeft builds the reversed-prefix portion of a play by filling board cells
// from the anchor leftward (or upward), navigating the GADDAG. This is the
// "BeforeAnchor" phase of the GenerateMoves algorithm — Appel-Jacobson (1988/1998) §5.
//
// (row, col) is the cell consumed now: the anchor itself on the first call, then one
// cell further left on each recursion. Each cell is either an existing board tile
// (followed for free) or an empty cell on which a rack tile is placed. node is the
// GADDAG node reached by the letters consumed so far — because the anchor letter is
// consumed first, the prefix is built in reverse. placed holds the new tiles laid
// down at the anchor and to its left, in anchor-first order.
//
// limit caps how many more new tiles may be placed leftward; existing tiles cost
// nothing. Placing a tile (including the anchor tile on the first call) consumes one
// unit. cc is the perpendicular cross-check table for this direction.
func extendLeft(
	board *engine.Board,
	g *dictionary.GADDAG,
	dict *dictionary.Dictionary,
	cc *[15][15][26]bool,
	counts *rackCounts,
	anchor anchorSquare,
	node dictionary.NodeID,
	row, col int,
	limit int,
	placed []engine.PlacedTile,
	candidates *[]MoveCandidate,
	seen map[moveKey]bool,
) {
	if row < 0 || col < 0 {
		return
	}

	if !board.IsEmpty(row, col) {
		// Existing tile: fold it into the reversed prefix for free.
		letter := board.Cell(row, col).Tile.DisplayLetter()
		if next, ok := g.Successor(node, letter); ok {
			afterLeft(board, g, dict, cc, counts, anchor, next, row, col, limit, placed, candidates, seen)
		}
		return
	}

	if limit == 0 {
		return
	}

	// Empty cell: place each playable rack letter that passes the cross-check.
	for l := byte('A'); l <= 'Z'; l++ {
		if !cc[row][col][l-'A'] {
			continue
		}
		isBlank := false
		if counts[l-'A'] > 0 {
			counts[l-'A']--
		} else if counts[26] > 0 {
			counts[26]--
			isBlank = true
		} else {
			continue
		}
		if next, ok := g.Successor(node, l); ok {
			pt := newRackTile(board, l, isBlank, row, col)
			afterLeft(board, g, dict, cc, counts, anchor, next, row, col, limit-1, append(placed, pt), candidates, seen)
		}
		if isBlank {
			counts[26]++
		} else {
			counts[l-'A']++
		}
	}
}

// afterLeft runs immediately after the cell at (row, col) has been consumed as the
// new leftmost letter of the reversed prefix (node is the node reached). It may
// (a) record a word that is complete at the anchor with no extension on either side,
// (b) cross the arc-separator '+' to navigate the suffix starting at anchor+1, and
// always (c) continues the prefix one cell further left.
func afterLeft(
	board *engine.Board,
	g *dictionary.GADDAG,
	dict *dictionary.Dictionary,
	cc *[15][15][26]bool,
	counts *rackCounts,
	anchor anchorSquare,
	node dictionary.NodeID,
	row, col int,
	limit int,
	placed []engine.PlacedTile,
	candidates *[]MoveCandidate,
	seen map[moveKey]bool,
) {
	leftRow, leftCol := furtherLeft(anchor.dir, row, col)
	// A clean left boundary means the word does not continue further left.
	leftClean := leftRow < 0 || leftCol < 0 || board.IsEmpty(leftRow, leftCol)

	if leftClean {
		aRow, aCol := nextRightPos(anchor, anchor.row, anchor.col)
		rightClean := aRow > 14 || aCol > 14 || board.IsEmpty(aRow, aCol)

		// (a) The reversed prefix is itself a complete word ending at the anchor
		//     (nothing extends past the anchor on the right).
		if rightClean && g.IsTerminal(node) {
			recordCandidate(board, dict, mergeNewPlaced(placed, nil), candidates, seen)
		}

		// (b) Cross the separator and navigate/extend the suffix from anchor+1.
		if sepNode, ok := g.Successor(node, dictionary.ArcSep); ok {
			extendRight(board, g, dict, cc, counts, anchor, sepNode,
				aRow, aCol, placed, nil, candidates, seen)
		}
	}

	// (c) Continue building the reversed prefix further left.
	extendLeft(board, g, dict, cc, counts, anchor, node, leftRow, leftCol, limit, placed, candidates, seen)
}

// extendRight extends the play to the right of (or below) the anchor, navigating
// GADDAG forward-suffix arcs. This is the "AfterAnchor" phase — Appel-Jacobson §5.
//
// It is entered at the cell just past the anchor once the reversed prefix has
// crossed the '+' separator. leftTiles holds the prefix tiles already placed (the
// anchor tile and any tiles left of it, anchor-first); newRight accumulates new
// tiles placed at and beyond anchor+1. Existing board tiles are followed for
// navigation but never added to newRight, so recordCandidate sees only new tiles.
func extendRight(
	board *engine.Board,
	g *dictionary.GADDAG,
	dict *dictionary.Dictionary,
	cc *[15][15][26]bool,
	counts *rackCounts,
	anchor anchorSquare,
	node dictionary.NodeID,
	row, col int,
	leftTiles, newRight []engine.PlacedTile,
	candidates *[]MoveCandidate,
	seen map[moveKey]bool,
) {
	outOfBounds := (anchor.dir == dirHorizontal && col > 14) ||
		(anchor.dir == dirVertical && row > 14)

	if outOfBounds {
		// Off the board edge: record if the node is terminal and a tile was placed.
		if g.IsTerminal(node) && len(leftTiles)+len(newRight) >= 1 {
			recordCandidate(board, dict, mergeNewPlaced(leftTiles, newRight), candidates, seen)
		}
		return
	}

	if !board.IsEmpty(row, col) {
		// Occupied cell: follow the existing tile's arc — Appel-Jacobson §5.
		traverseExistingRight(board, g, dict, cc, counts, anchor, node,
			row, col, leftTiles, newRight, candidates, seen)
		return
	}

	// Empty cell: if node is terminal the word ends here naturally.
	if g.IsTerminal(node) && len(leftTiles)+len(newRight) >= 1 {
		recordCandidate(board, dict, mergeNewPlaced(leftTiles, newRight), candidates, seen)
	}

	// Right-extension phase: try each rack letter that passes the cross-check.
	for l := byte('A'); l <= 'Z'; l++ {
		if !cc[row][col][l-'A'] {
			continue
		}
		isBlank := false
		if counts[l-'A'] > 0 {
			counts[l-'A']--
		} else if counts[26] > 0 {
			counts[26]--
			isBlank = true
		} else {
			continue
		}
		if next, ok := g.Successor(node, l); ok {
			pt := newRackTile(board, l, isBlank, row, col)
			nr, nc := nextRightPos(anchor, row, col)
			extendRight(board, g, dict, cc, counts, anchor, next,
				nr, nc, leftTiles, append(newRight, pt), candidates, seen)
		}
		if isBlank {
			counts[26]++
		} else {
			counts[l-'A']++
		}
	}
}

// traverseExistingRight follows the arc for the letter already on the board at
// (row, col), then continues extendRight without adding the existing tile to
// newRight. Appel-Jacobson §5: "if the square is occupied, navigate its arc."
func traverseExistingRight(
	board *engine.Board,
	g *dictionary.GADDAG,
	dict *dictionary.Dictionary,
	cc *[15][15][26]bool,
	counts *rackCounts,
	anchor anchorSquare,
	node dictionary.NodeID,
	row, col int,
	leftTiles, newRight []engine.PlacedTile,
	candidates *[]MoveCandidate,
	seen map[moveKey]bool,
) {
	letter := board.Cell(row, col).Tile.DisplayLetter()

	next, ok := g.Successor(node, letter)
	if !ok {
		return // no arc for this board letter: no valid extension through this cell
	}

	nr, nc := nextRightPos(anchor, row, col)
	extendRight(board, g, dict, cc, counts, anchor, next,
		nr, nc, leftTiles, newRight, candidates, seen)
}

// newRackTile builds the PlacedTile for a rack letter placed at (row, col). Blank
// tiles carry their assigned letter but always score zero. The face value is taken
// from the board's mode so Interesting-mode plays score with the right economy.
func newRackTile(board *engine.Board, l byte, isBlank bool, row, col int) engine.PlacedTile {
	pts := board.LetterPoints(l)
	if isBlank {
		pts = 0
	}
	return engine.PlacedTile{
		Tile: engine.Tile{Letter: l, Points: pts, IsBlank: isBlank, AssignedLetter: l},
		Row:  row,
		Col:  col,
	}
}

// furtherLeft returns the coordinates of the cell one step to the left of (or
// above) (row, col) for the given direction. The result may be off-board
// (negative), which callers treat as the board edge.
func furtherLeft(dir direction, row, col int) (int, int) {
	if dir == dirHorizontal {
		return row, col - 1
	}
	return row - 1, col
}

// nextRightPos returns the coordinates of the next cell to the right of (or below)
// (row, col) for the given direction.
func nextRightPos(anchor anchorSquare, row, col int) (int, int) {
	if anchor.dir == dirHorizontal {
		return row, col + 1
	}
	return row + 1, col
}

// mergeNewPlaced produces the final placed-tile slice for ValidatePlacement:
// the prefix tiles reversed (they are stored anchor-first, i.e. rightmost-first)
// followed by the suffix tiles. Only newly placed rack tiles are included; existing
// board tiles navigated during traversal are not.
func mergeNewPlaced(leftTiles, newRight []engine.PlacedTile) []engine.PlacedTile {
	result := make([]engine.PlacedTile, 0, len(leftTiles)+len(newRight))
	for i := len(leftTiles) - 1; i >= 0; i-- {
		result = append(result, leftTiles[i])
	}
	result = append(result, newRight...)
	return result
}
