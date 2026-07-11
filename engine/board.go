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
}

// boardWire is the exported wire struct used by GobEncode/GobDecode.
// All fields must be exported for encoding/gob to process them.
type boardWire struct {
	Cells [15][15]Cell
}

// premiumSquares defines the standard premium-square layout for a 15×15 board.
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

// NewBoard returns a 15×15 board initialised with the standard premium-square layout.
// All cells start unoccupied (Tile == nil).
func NewBoard() *Board {
	b := &Board{}
	for _, ps := range premiumSquares {
		b.cells[ps.row][ps.col].Square = ps.sq
	}
	return b
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
// read-only AI use.
func (b *Board) Clone() *Board {
	clone := &Board{}
	clone.cells = b.cells // array copy — all 225 Cell values copied by value
	// Each Cell.Tile is a pointer. The pointed-to Tile is not copied, but since
	// placed tiles are never mutated (only placed and removed), the shared pointer
	// is safe for read-only access by the AI goroutine.
	return clone
}

// GobEncode serialises the board for save/load. Required because cells is unexported.
func (b *Board) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(boardWire{Cells: b.cells}); err != nil {
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
	return nil
}
