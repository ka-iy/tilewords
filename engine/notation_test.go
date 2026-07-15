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
