// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package engine is documented in doc.go.
package engine

import (
	"bytes"
	"encoding/gob"
	"fmt"
)

// SquareType identifies the premium-square multiplier for a board cell.
// Multipliers apply only when a new tile is placed on the square this turn.
type SquareType int

const (
	// Normal squares carry no multiplier.
	Normal SquareType = iota
	// DoubleLetter multiplies the placed tile's point value by 2.
	DoubleLetter
	// TripleLetter multiplies the placed tile's point value by 3.
	TripleLetter
	// DoubleWord multiplies the total word score by 2.
	DoubleWord
	// TripleWord multiplies the total word score by 3.
	TripleWord
	// Centre is the starting square (7,7). It acts as a DoubleWord square and
	// must be covered by the first move of the game.
	Centre
)

// Cell is a single cell on the 15×15 board.
type Cell struct {
	// Tile is nil when the cell is unoccupied.
	Tile *Tile
	// Square identifies the premium-square type. Fixed at board construction.
	Square SquareType
}

// Board is the 15×15 playing grid. All coordinates are (row, col) with
// row 0 at the top and col 0 at the left.
type Board struct {
	cells [15][15]Cell // unexported; accessed via methods
	// mode records the game mode this board was built for. It selects both the
	// premium-square layout (baked into cells) and the letter point values used by
	// LetterPoints, so the CPU (which is handed only the board) scores in the right mode.
	mode GameMode
}

// boardWire is the exported wire struct used by GobEncode/GobDecode.
// All fields must be exported for encoding/gob to process them.
type boardWire struct {
	Cells [15][15]Cell
	// Mode persists the board's game mode. Absent in older saves, where it decodes
	// as the zero value (ClassicMode) — matching the layout those saves stored.
	Mode GameMode
}

// premiumSquares defines the ClassicMode premium-square layout for a 15×15 board.
// The layout is symmetric about both axes. Coordinates are (row, col).
// Source: standard crossword board game tournament layout.
var premiumSquares = []struct {
	row, col int
	sq       SquareType
}{
	// Triple Word Score — corners and edge midpoints
	{0, 0, TripleWord}, {0, 7, TripleWord}, {0, 14, TripleWord},
	{7, 0, TripleWord}, {7, 14, TripleWord},
	{14, 0, TripleWord}, {14, 7, TripleWord}, {14, 14, TripleWord},

	// Centre — also acts as Double Word for the first move
	{7, 7, Centre},

	// Double Word Score — two diagonals from each corner toward centre
	{1, 1, DoubleWord}, {2, 2, DoubleWord}, {3, 3, DoubleWord}, {4, 4, DoubleWord},
	{1, 13, DoubleWord}, {2, 12, DoubleWord}, {3, 11, DoubleWord}, {4, 10, DoubleWord},
	{10, 4, DoubleWord}, {11, 3, DoubleWord}, {12, 2, DoubleWord}, {13, 1, DoubleWord},
	{10, 10, DoubleWord}, {11, 11, DoubleWord}, {12, 12, DoubleWord}, {13, 13, DoubleWord},

	// Triple Letter Score
	{1, 5, TripleLetter}, {1, 9, TripleLetter},
	{5, 1, TripleLetter}, {5, 5, TripleLetter}, {5, 9, TripleLetter}, {5, 13, TripleLetter},
	{9, 1, TripleLetter}, {9, 5, TripleLetter}, {9, 9, TripleLetter}, {9, 13, TripleLetter},
	{13, 5, TripleLetter}, {13, 9, TripleLetter},

	// Double Letter Score
	{0, 3, DoubleLetter}, {0, 11, DoubleLetter},
	{2, 6, DoubleLetter}, {2, 8, DoubleLetter},
	{3, 0, DoubleLetter}, {3, 7, DoubleLetter}, {3, 14, DoubleLetter},
	{6, 2, DoubleLetter}, {6, 6, DoubleLetter}, {6, 8, DoubleLetter}, {6, 12, DoubleLetter},
	{7, 3, DoubleLetter}, {7, 11, DoubleLetter},
	{8, 2, DoubleLetter}, {8, 6, DoubleLetter}, {8, 8, DoubleLetter}, {8, 12, DoubleLetter},
	{11, 0, DoubleLetter}, {11, 7, DoubleLetter}, {11, 14, DoubleLetter},
	{12, 6, DoubleLetter}, {12, 8, DoubleLetter},
	{14, 3, DoubleLetter}, {14, 11, DoubleLetter},
}

// premiumSquaresInteresting defines the InterestingMode premium-square layout: an
// independently-designed 4-fold rotational ("pinwheel") pattern with even coverage
// (every cell is within two steps of a premium) and no orthogonally adjacent word
// multipliers. Corners are empty. Distinct from the ClassicMode layout.
var premiumSquaresInteresting = []struct {
	row, col int
	sq       SquareType
}{
	// Triple Word Score
	{0, 10, TripleWord}, {4, 0, TripleWord}, {10, 14, TripleWord}, {14, 4, TripleWord},

	// Centre — also acts as Double Word for the first move
	{7, 7, Centre},

	// Double Word Score
	{2, 3, DoubleWord}, {3, 6, DoubleWord}, {3, 12, DoubleWord}, {5, 8, DoubleWord},
	{6, 5, DoubleWord}, {6, 11, DoubleWord}, {8, 3, DoubleWord}, {8, 9, DoubleWord},
	{9, 6, DoubleWord}, {11, 2, DoubleWord}, {11, 8, DoubleWord}, {12, 11, DoubleWord},

	// Triple Letter Score
	{0, 2, TripleLetter}, {0, 4, TripleLetter}, {2, 14, TripleLetter}, {3, 10, TripleLetter},
	{4, 3, TripleLetter}, {4, 14, TripleLetter}, {10, 0, TripleLetter}, {10, 11, TripleLetter},
	{11, 4, TripleLetter}, {12, 0, TripleLetter}, {14, 10, TripleLetter}, {14, 12, TripleLetter},

	// Double Letter Score
	{0, 13, DoubleLetter}, {1, 0, DoubleLetter}, {1, 6, DoubleLetter}, {1, 7, DoubleLetter},
	{1, 8, DoubleLetter}, {1, 11, DoubleLetter}, {3, 1, DoubleLetter}, {3, 5, DoubleLetter},
	{3, 9, DoubleLetter}, {5, 3, DoubleLetter}, {5, 11, DoubleLetter}, {6, 1, DoubleLetter},
	{6, 13, DoubleLetter}, {7, 1, DoubleLetter}, {7, 13, DoubleLetter}, {8, 1, DoubleLetter},
	{8, 13, DoubleLetter}, {9, 3, DoubleLetter}, {9, 11, DoubleLetter}, {11, 5, DoubleLetter},
	{11, 9, DoubleLetter}, {11, 13, DoubleLetter}, {13, 3, DoubleLetter}, {13, 6, DoubleLetter},
	{13, 7, DoubleLetter}, {13, 8, DoubleLetter}, {13, 14, DoubleLetter}, {14, 1, DoubleLetter},
}

// NewBoard returns a 15×15 ClassicMode board. All cells start unoccupied (Tile == nil).
func NewBoard() *Board { return NewBoardForMode(ClassicMode) }

// NewBoardForMode returns a 15×15 board initialised with mode's premium-square layout.
// All cells start unoccupied (Tile == nil).
func NewBoardForMode(mode GameMode) *Board {
	b := &Board{mode: mode}
	squares := premiumSquares
	if mode == InterestingMode {
		squares = premiumSquaresInteresting
	}
	for _, ps := range squares {
		b.cells[ps.row][ps.col].Square = ps.sq
	}
	return b
}

// Mode returns the game mode this board was built for.
func (b *Board) Mode() GameMode { return b.mode }

// malformedTile returns the first tile on the board that the game could not have produced,
// and whether one was found. See Tile.wellFormed; used by ValidateDecodedState.
func (b *Board) malformedTile() (Tile, bool) {
	for r := range b.cells {
		for c := range b.cells[r] {
			if t := b.cells[r][c].Tile; t != nil && !t.wellFormed() {
				return *t, true
			}
		}
	}
	return Tile{}, false
}

// LetterPoints returns the face value of an uppercase letter A–Z in this board's mode
// (0 for any other byte, including the blank sentinel). The CPU move generator uses it to
// stamp the correct per-mode face value onto tiles it places.
func (b *Board) LetterPoints(letter byte) int {
	if letter < 'A' || letter > 'Z' {
		return 0
	}
	return letterPointsForMode(b.mode)[letter-'A']
}

// Cell returns the cell at (row, col). Panics if coordinates are out of bounds.
func (b *Board) Cell(row, col int) Cell {
	if row < 0 || row > 14 || col < 0 || col > 14 {
		panic(fmt.Sprintf("engine.Board.Cell: coordinates (%d,%d) out of bounds", row, col))
	}
	return b.cells[row][col]
}

// Place sets the tile at (row, col). Returns an error if the cell is already occupied.
func (b *Board) Place(row, col int, tile Tile) error {
	if row < 0 || row > 14 || col < 0 || col > 14 {
		return fmt.Errorf("engine.Board.Place: coordinates (%d,%d) out of bounds", row, col)
	}
	if b.cells[row][col].Tile != nil {
		return fmt.Errorf("engine.Board.Place: cell (%d,%d) is already occupied", row, col)
	}
	t := tile // copy so the board owns an independent value
	b.cells[row][col].Tile = &t
	return nil
}

// Remove clears the tile at (row, col). No-op if the cell is already empty.
func (b *Board) Remove(row, col int) {
	if row < 0 || row > 14 || col < 0 || col > 14 {
		return
	}
	b.cells[row][col].Tile = nil
}

// IsEmpty reports whether (row, col) contains no tile.
func (b *Board) IsEmpty(row, col int) bool {
	if row < 0 || row > 14 || col < 0 || col > 14 {
		return true
	}
	return b.cells[row][col].Tile == nil
}

// HasAnyTile reports whether the board has at least one placed tile.
func (b *Board) HasAnyTile() bool {
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			if b.cells[r][c].Tile != nil {
				return true
			}
		}
	}
	return false
}

// Clone returns a deep copy of the board. The returned board is safe for
// independent modification. Tile pointers within cells are shallow-copied:
// tiles on the board are immutable after placement, so this is safe for
// read-only CPU use.
func (b *Board) Clone() *Board {
	clone := &Board{mode: b.mode}
	clone.cells = b.cells // array copy — all 225 Cell values copied by value
	// Each Cell.Tile is a pointer. The pointed-to Tile is not copied, but since
	// placed tiles are never mutated (only placed and removed), the shared pointer
	// is safe for read-only access by the CPU goroutine.
	return clone
}

// GobEncode serialises the board for save/load. Required because cells is unexported.
func (b *Board) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(boardWire{Cells: b.cells, Mode: b.mode}); err != nil {
		return nil, fmt.Errorf("engine.Board.GobEncode: %w", err)
	}
	return buf.Bytes(), nil
}

// GobDecode deserialises the board from save/load data.
func (b *Board) GobDecode(data []byte) error {
	var w boardWire
	dec := gob.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&w); err != nil {
		return fmt.Errorf("engine.Board.GobDecode: %w", err)
	}
	b.cells = w.Cells
	b.mode = w.Mode
	return nil
}
