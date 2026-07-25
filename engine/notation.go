// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package engine is documented in doc.go.
package engine

import "fmt"

// AnnotatedWords returns Scrabble-notation strings for every word a committed PlayMove
// forms: the main word (along the play's axis) first, then any perpendicular cross-words
// formed where the new tiles connect against existing tiles. Each element is "<coord>
// <word>" using the conventions described on AnnotatedMainWord. Returns nil when the move
// formed no word (e.g. it placed no tiles).
//
// It must be called after the move has been committed to board.
func AnnotatedWords(board *Board, move *PlayMove) []string {
	groups := notationGroups(board, move)
	words := make([]string, 0, len(groups))
	for _, positions := range groups {
		if len(positions) < 2 {
			continue
		}
		coord, word := annotate(board, move, positions)
		words = append(words, coord+" "+word)
	}
	return words
}

// AnnotatedMainWord returns the Scrabble-notation coordinate and word for the primary word
// a committed PlayMove forms — the main word along the play's axis, or, when a single tile
// leaves that axis ambiguous, the highest-scoring of the words it forms (see
// notationGroups). It must be called after the move has been committed to board.
//
// The coordinate marks the word's starting square using the standard convention: a
// horizontal word is written row-number then column-letter (e.g. "8D"), a vertical word
// column-letter then row-number (e.g. "H8"). Columns are A–O left to right and rows 1–15
// top to bottom.
//
// Letters that were already on the board before this move — the existing word(s) the play
// was made against — are wrapped in parentheses, e.g. "(DOG)S" or "C(AT)S". Blank tiles are
// rendered as lowercase letters. ok is false when the move formed no word.
func AnnotatedMainWord(board *Board, move *PlayMove) (coord, word string, ok bool) {
	groups := notationGroups(board, move)
	if len(groups) == 0 || len(groups[0]) < 2 {
		return "", "", false
	}
	coord, word = annotate(board, move, groups[0])
	return coord, word, true
}

// notationGroups returns the words a committed move forms, as position groups ordered for
// notation: the word the play's coordinate names first, then the remaining cross-words.
//
// It differs from extractWordPositions only for a single-tile play. One tile lies on both
// axes at once, so when it forms a word in each direction there is no main axis to read off
// the placement; extractWordPositions resolves that by always treating the play as
// horizontal. Notation instead names the word the play scored most from, so the coordinate
// describes the play the way a player would read it. Equal scores keep the horizontal word
// first.
func notationGroups(board *Board, move *PlayMove) [][][2]int {
	groups := extractWordPositions(board, move)
	// Only a one-tile play can produce two groups whose order is not fixed by the
	// placement: groups[0] is then its horizontal word and groups[1] its vertical one.
	if len(move.Placed) != 1 || len(groups) != 2 {
		return groups
	}
	if wordValue(board, move.Placed, groups[1]) > wordValue(board, move.Placed, groups[0]) {
		groups[0], groups[1] = groups[1], groups[0]
	}
	return groups
}

// annotate renders one word's coordinate and text: the coordinate of its starting square,
// and the word with existing (already-on-board) tiles parenthesised in maximal runs and
// blank tiles lowercased. positions must have length >= 2, ordered from the word's start.
func annotate(board *Board, move *PlayMove, positions [][2]int) (coord, word string) {
	horiz := positions[0][0] == positions[1][0] // first two cells share a row => horizontal
	coord = notationCoord(positions[0][0], positions[0][1], horiz)

	b := make([]byte, 0, len(positions)+2)
	inExisting := false
	for _, pos := range positions {
		t := virtualTile(board, move.Placed, pos[0], pos[1])
		if t == nil {
			continue
		}
		// A cell not covered by this move's placed tiles was already on the board; group
		// maximal runs of such cells in parentheses.
		existing := !covers(move.Placed, pos[0], pos[1])
		if existing && !inExisting {
			b = append(b, '(')
			inExisting = true
		} else if !existing && inExisting {
			b = append(b, ')')
			inExisting = false
		}

		letter := t.Letter
		if t.IsBlank && t.AssignedLetter != 0 {
			letter = t.AssignedLetter
		}
		if t.IsBlank {
			letter += 'a' - 'A' // blanks are shown lowercase in Scrabble notation
		}
		b = append(b, letter)
	}
	if inExisting {
		b = append(b, ')')
	}
	return coord, string(b)
}

// notationCoord formats a 0-indexed board square as a Scrabble-notation coordinate:
// number-then-letter for a horizontal word, letter-then-number for a vertical one.
func notationCoord(row, col int, horiz bool) string {
	colLetter := byte('A' + col)
	rowNum := row + 1
	if horiz {
		return fmt.Sprintf("%d%c", rowNum, colLetter)
	}
	return fmt.Sprintf("%c%d", colLetter, rowNum)
}
