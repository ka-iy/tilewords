// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import "testing"

// place commits tiles to the board (simulating a played move) and returns the PlayMove that
// placed them, so AnnotatedMainWord can be evaluated against the committed board.
func placeAll(t *testing.T, b *Board, tiles []PlacedTile) *PlayMove {
	t.Helper()
	for _, pt := range tiles {
		if err := b.Place(pt.Row, pt.Col, pt.Tile); err != nil {
			t.Fatalf("place (%d,%d): %v", pt.Row, pt.Col, err)
		}
	}
	return &PlayMove{Placed: tiles}
}

func TestAnnotatedMainWord(t *testing.T) {
	tile := func(l byte) Tile { return Tile{Letter: l, Points: 1} }

	t.Run("horizontal is number-then-letter", func(t *testing.T) {
		b := NewBoard()
		// C A T across row 7 (row index 7 -> "8"), starting at column D (index 3).
		m := placeAll(t, b, []PlacedTile{
			{Tile: tile('C'), Row: 7, Col: 3},
			{Tile: tile('A'), Row: 7, Col: 4},
			{Tile: tile('T'), Row: 7, Col: 5},
		})
		coord, word, ok := AnnotatedMainWord(b, m)
		if !ok || coord != "8D" || word != "CAT" {
			t.Fatalf("got (%q,%q,%v), want (\"8D\",\"CAT\",true)", coord, word, ok)
		}
	})

	t.Run("vertical is letter-then-number", func(t *testing.T) {
		b := NewBoard()
		// D O G down column H (index 7 -> "H"), starting at row 7 (index 7 -> "8").
		m := placeAll(t, b, []PlacedTile{
			{Tile: tile('D'), Row: 7, Col: 7},
			{Tile: tile('O'), Row: 8, Col: 7},
			{Tile: tile('G'), Row: 9, Col: 7},
		})
		coord, word, ok := AnnotatedMainWord(b, m)
		if !ok || coord != "H8" || word != "DOG" {
			t.Fatalf("got (%q,%q,%v), want (\"H8\",\"DOG\",true)", coord, word, ok)
		}
	})

	t.Run("single tile extending a vertical word reports vertical with existing in parens", func(t *testing.T) {
		b := NewBoard()
		// Existing vertical A T at (7,7),(8,7); play S at (9,7) to make ATS down column H.
		if err := b.Place(7, 7, tile('A')); err != nil {
			t.Fatal(err)
		}
		if err := b.Place(8, 7, tile('T')); err != nil {
			t.Fatal(err)
		}
		m := placeAll(t, b, []PlacedTile{{Tile: tile('S'), Row: 9, Col: 7}})
		coord, word, ok := AnnotatedMainWord(b, m)
		if !ok || coord != "H8" || word != "(AT)S" {
			t.Fatalf("got (%q,%q,%v), want (\"H8\",\"(AT)S\",true)", coord, word, ok)
		}
	})

	t.Run("existing tiles through the middle are parenthesised", func(t *testing.T) {
		b := NewBoard()
		// Existing A T at columns E,F of row 7; play C before and S after -> C(AT)S.
		if err := b.Place(7, 4, tile('A')); err != nil {
			t.Fatal(err)
		}
		if err := b.Place(7, 5, tile('T')); err != nil {
			t.Fatal(err)
		}
		m := placeAll(t, b, []PlacedTile{
			{Tile: tile('C'), Row: 7, Col: 3},
			{Tile: tile('S'), Row: 7, Col: 6},
		})
		coord, word, ok := AnnotatedMainWord(b, m)
		if !ok || coord != "8D" || word != "C(AT)S" {
			t.Fatalf("got (%q,%q,%v), want (\"8D\",\"C(AT)S\",true)", coord, word, ok)
		}
	})

	t.Run("cross-words are listed after the main word", func(t *testing.T) {
		b := NewBoard()
		// Existing S at (6,7). Play C A T across row 7 (cols 5,6,7). The T at (7,7) sits
		// below the S, forming the vertical cross-word ST.
		if err := b.Place(6, 7, tile('S')); err != nil {
			t.Fatal(err)
		}
		m := placeAll(t, b, []PlacedTile{
			{Tile: tile('C'), Row: 7, Col: 5},
			{Tile: tile('A'), Row: 7, Col: 6},
			{Tile: tile('T'), Row: 7, Col: 7},
		})
		words := AnnotatedWords(b, m)
		want := []string{"8F CAT", "H7 (S)T"}
		if len(words) != len(want) || words[0] != want[0] || words[1] != want[1] {
			t.Fatalf("AnnotatedWords = %v, want %v", words, want)
		}
	})

	t.Run("blank tiles are lowercase", func(t *testing.T) {
		b := NewBoard()
		m := placeAll(t, b, []PlacedTile{
			{Tile: Tile{Letter: 'C', Points: 3}, Row: 7, Col: 7},
			{Tile: Tile{Letter: 'A', IsBlank: true, AssignedLetter: 'A'}, Row: 7, Col: 8},
			{Tile: Tile{Letter: 'T', Points: 1}, Row: 7, Col: 9},
		})
		coord, word, ok := AnnotatedMainWord(b, m)
		if !ok || coord != "8H" || word != "CaT" {
			t.Fatalf("got (%q,%q,%v), want (\"8H\",\"CaT\",true)", coord, word, ok)
		}
	})
}

// TestAnnotatedWordsSingleTileAxis covers the one placement whose axis the tiles do not
// determine: a single tile forming a word in both directions. Notation names the
// higher-scoring word, falling back to the horizontal one when the scores are equal.
// Every square these subtests use is premium-free, so each word's score is the plain sum
// of its tile values.
func TestAnnotatedWordsSingleTileAxis(t *testing.T) {
	tile := func(l byte, pts int) Tile { return Tile{Letter: l, Points: pts} }

	// existing commits tiles that were on the board before the play under test.
	existing := func(t *testing.T, b *Board, tiles []PlacedTile) {
		t.Helper()
		for _, pt := range tiles {
			if err := b.Place(pt.Row, pt.Col, pt.Tile); err != nil {
				t.Fatalf("place (%d,%d): %v", pt.Row, pt.Col, err)
			}
		}
	}

	t.Run("higher-scoring vertical word is named first", func(t *testing.T) {
		b := NewBoard()
		// Vertical C A down column H rows 9-10, and A at G11; playing T at H11 forms
		// vertical (CA)T for 3 and horizontal (A)T for 2.
		existing(t, b, []PlacedTile{
			{Tile: tile('C', 1), Row: 8, Col: 7},
			{Tile: tile('A', 1), Row: 9, Col: 7},
			{Tile: tile('A', 1), Row: 10, Col: 6},
		})
		m := placeAll(t, b, []PlacedTile{{Tile: tile('T', 1), Row: 10, Col: 7}})
		words := AnnotatedWords(b, m)
		want := []string{"H9 (CA)T", "11G (A)T"}
		if len(words) != len(want) || words[0] != want[0] || words[1] != want[1] {
			t.Fatalf("AnnotatedWords = %v, want %v", words, want)
		}
	})

	t.Run("higher-scoring horizontal word is named first", func(t *testing.T) {
		b := NewBoard()
		// Horizontal Q I across row 11 cols F-G, and A at H10; playing S at H11 forms
		// horizontal (QI)S for 12 and vertical (A)S for 2.
		existing(t, b, []PlacedTile{
			{Tile: tile('Q', 10), Row: 10, Col: 5},
			{Tile: tile('I', 1), Row: 10, Col: 6},
			{Tile: tile('A', 1), Row: 9, Col: 7},
		})
		m := placeAll(t, b, []PlacedTile{{Tile: tile('S', 1), Row: 10, Col: 7}})
		words := AnnotatedWords(b, m)
		want := []string{"11F (QI)S", "H10 (A)S"}
		if len(words) != len(want) || words[0] != want[0] || words[1] != want[1] {
			t.Fatalf("AnnotatedWords = %v, want %v", words, want)
		}
	})

	t.Run("equal scores name the horizontal word first", func(t *testing.T) {
		b := NewBoard()
		// A at G11 and A at H10; playing T at H11 forms (A)T horizontally and (A)T
		// vertically, both for 2.
		existing(t, b, []PlacedTile{
			{Tile: tile('A', 1), Row: 10, Col: 6},
			{Tile: tile('A', 1), Row: 9, Col: 7},
		})
		m := placeAll(t, b, []PlacedTile{{Tile: tile('T', 1), Row: 10, Col: 7}})
		words := AnnotatedWords(b, m)
		want := []string{"11G (A)T", "H10 (A)T"}
		if len(words) != len(want) || words[0] != want[0] || words[1] != want[1] {
			t.Fatalf("AnnotatedWords = %v, want %v", words, want)
		}
	})

	t.Run("main word follows the score", func(t *testing.T) {
		b := NewBoard()
		// Same board as the higher-scoring-vertical case: the vertical word wins, so it is
		// the one AnnotatedMainWord reports.
		existing(t, b, []PlacedTile{
			{Tile: tile('C', 1), Row: 8, Col: 7},
			{Tile: tile('A', 1), Row: 9, Col: 7},
			{Tile: tile('A', 1), Row: 10, Col: 6},
		})
		m := placeAll(t, b, []PlacedTile{{Tile: tile('T', 1), Row: 10, Col: 7}})
		coord, word, ok := AnnotatedMainWord(b, m)
		if !ok || coord != "H9" || word != "(CA)T" {
			t.Fatalf("got (%q,%q,%v), want (\"H9\",\"(CA)T\",true)", coord, word, ok)
		}
	})

	t.Run("a tile forming one word keeps that word's axis", func(t *testing.T) {
		b := NewBoard()
		// A T down column H rows 9-10 with no horizontal neighbour: playing S at H11
		// forms only the vertical word, so there is no tie to break.
		existing(t, b, []PlacedTile{
			{Tile: tile('A', 1), Row: 8, Col: 7},
			{Tile: tile('T', 1), Row: 9, Col: 7},
		})
		m := placeAll(t, b, []PlacedTile{{Tile: tile('S', 1), Row: 10, Col: 7}})
		words := AnnotatedWords(b, m)
		if len(words) != 1 || words[0] != "H9 (AT)S" {
			t.Fatalf("AnnotatedWords = %v, want [\"H9 (AT)S\"]", words)
		}
	})
}
