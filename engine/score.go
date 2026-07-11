// Package engine is documented in doc.go.
package engine

import "fmt"

// Score calculates the total score for move on board, including all cross-word
// scores and the bingo bonus (+50 for playing all 7 tiles in one turn).
// Premium square multipliers apply only to tiles placed during this move;
// existing board tiles contribute only their face value (BR-E06).
// Score must be called after ValidatePlacement has populated move.WordsFormed.
func Score(board *Board, move *PlayMove) (int, error) {
	if len(move.WordsFormed) == 0 {
		return 0, fmt.Errorf("engine.Score: WordsFormed is empty — call ValidatePlacement first")
	}

	total := 0
	for _, positions := range wordPositions(board, move) {
		wordScore, wordMult := sumWord(board, move.Placed, positions)
		total += wordScore * wordMult
	}

	// Bingo bonus: all 7 tiles played in a single turn (BR-E07).
	if len(move.Placed) == MaxRackSize {
		total += 50
	}

	move.Score = total
	return total, nil
}

// wordPositions returns, for each word in move.WordsFormed, the board positions
// that make up that word. The order matches move.WordsFormed.
func wordPositions(board *Board, move *PlayMove) [][][2]int {
	// Re-derive positions using the same logic as extractWords so that the
	// positions correspond 1:1 with WordsFormed.
	return extractWordPositions(board, move)
}

// sumWord computes the raw word score and accumulated word multiplier for a
// sequence of board positions, respecting premium squares only for newly
// placed tiles (Pattern 2 / BR-E06).
func sumWord(board *Board, placed []PlacedTile, positions [][2]int) (wordScore, wordMult int) {
	wordMult = 1
	for _, pos := range positions {
		r, c := pos[0], pos[1]
		tile := virtualTile(board, placed, r, c)
		if tile == nil {
			continue
		}

		letterMult := 1
		if covers(placed, r, c) {
			// Premium squares only apply when a new tile is placed on them this turn.
			sq := board.Cell(r, c).Square
			switch sq {
			case DoubleLetter:
				letterMult = 2
			case TripleLetter:
				letterMult = 3
			case DoubleWord, Centre:
				// Word multiplier accumulates; letter multiplier is 1 at DW/Centre.
				wordMult *= 2
			case TripleWord:
				wordMult *= 3
			}
		}

		wordScore += tile.Points * letterMult
	}
	return wordScore, wordMult
}

// extractWords returns all words formed by placing move.Placed on board:
// the main word (along the primary axis) followed by any cross-words
// (perpendicular words of length ≥ 2 formed by the new tiles).
// Words are returned in uppercase, in left-to-right / top-to-bottom board order.
func extractWords(board *Board, move *PlayMove) []string {
	posGroups := extractWordPositions(board, move)
	words := make([]string, 0, len(posGroups))
	for _, positions := range posGroups {
		bs := make([]byte, 0, len(positions))
		for _, pos := range positions {
			t := virtualTile(board, move.Placed, pos[0], pos[1])
			if t == nil {
				continue
			}
			letter := t.Letter
			if t.IsBlank && t.AssignedLetter != 0 {
				letter = t.AssignedLetter
			}
			bs = append(bs, letter)
		}
		if len(bs) >= 2 {
			words = append(words, string(bs))
		}
	}
	return words
}

// extractWordPositions returns, for each formed word, the ordered sequence of
// board positions (row, col) that make up that word.
func extractWordPositions(board *Board, move *PlayMove) [][][2]int {
	if len(move.Placed) == 0 {
		return nil
	}

	horiz := isHorizontal(move.Placed)
	var result [][][2]int

	// Main word: extend along the primary axis from the first placed tile.
	mainPositions := mainWordPositions(board, move.Placed, horiz)
	if len(mainPositions) >= 2 {
		result = append(result, mainPositions)
	}

	// Cross-words: for each newly placed tile, extend perpendicular.
	for _, pt := range move.Placed {
		cross := crossWordPositions(board, move.Placed, pt.Row, pt.Col, !horiz)
		if len(cross) >= 2 {
			result = append(result, cross)
		}
	}

	return result
}

// mainWordPositions returns the full sequence of positions for the main word,
// extending as far as possible in the primary direction (horizontal or vertical)
// using both existing board tiles and newly placed tiles.
func mainWordPositions(board *Board, placed []PlacedTile, horiz bool) [][2]int {
	// Find the minimum position in the primary dimension.
	minRow, minCol := placed[0].Row, placed[0].Col
	for _, pt := range placed[1:] {
		if pt.Row < minRow {
			minRow = pt.Row
		}
		if pt.Col < minCol {
			minCol = pt.Col
		}
	}

	// Walk backward (up or left) past existing tiles to find the word start.
	r, c := minRow, minCol
	if horiz {
		for c > 0 && !board.IsEmpty(r, c-1) {
			c--
		}
	} else {
		for r > 0 && !board.IsEmpty(r-1, c) {
			r--
		}
	}

	// Walk forward (right or down), collecting positions until we hit empty space.
	var positions [][2]int
	for {
		if r > 14 || c > 14 {
			break
		}
		if board.IsEmpty(r, c) && !covers(placed, r, c) {
			break
		}
		positions = append(positions, [2]int{r, c})
		if horiz {
			c++
		} else {
			r++
		}
	}
	return positions
}

// crossWordPositions returns positions for the cross-word formed by the tile at
// (row, col) in the perpendicular direction. Returns nil if no cross-word exists.
func crossWordPositions(board *Board, placed []PlacedTile, row, col int, horiz bool) [][2]int {
	// Walk backward to find the start.
	r, c := row, col
	if horiz {
		for c > 0 && !board.IsEmpty(r, c-1) {
			c--
		}
	} else {
		for r > 0 && !board.IsEmpty(r-1, c) {
			r--
		}
	}

	// Walk forward collecting positions.
	var positions [][2]int
	for {
		if r > 14 || c > 14 {
			break
		}
		if board.IsEmpty(r, c) && !covers(placed, r, c) {
			break
		}
		positions = append(positions, [2]int{r, c})
		if horiz {
			c++
		} else {
			r++
		}
	}
	return positions
}

// virtualTile returns the tile at (row, col) considering both existing board
// tiles and newly placed tiles. Returns nil if the cell is empty in both.
func virtualTile(board *Board, placed []PlacedTile, row, col int) *Tile {
	for i := range placed {
		if placed[i].Row == row && placed[i].Col == col {
			t := placed[i].Tile
			return &t
		}
	}
	cell := board.Cell(row, col)
	return cell.Tile
}

// covers reports whether any PlacedTile in placed occupies (row, col).
func covers(placed []PlacedTile, row, col int) bool {
	for _, pt := range placed {
		if pt.Row == row && pt.Col == col {
			return true
		}
	}
	return false
}

// isHorizontal reports whether all placed tiles share the same row.
// For a single tile, returns true (treated as horizontal by convention).
func isHorizontal(placed []PlacedTile) bool {
	if len(placed) <= 1 {
		return true
	}
	row := placed[0].Row
	for _, pt := range placed[1:] {
		if pt.Row != row {
			return false
		}
	}
	return true
}
