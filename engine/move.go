// Package engine is documented in doc.go.
package engine

// Move is a marker interface implemented by all move types.
// The unexported method prevents accidental use of non-move types where a
// Move is expected, while allowing type switches on the three concrete types.
type Move interface{ moveMarker() }

// PlayMove represents placing one or more tiles on the board.
// WordsFormed and Score are populated by ValidatePlacement and Score
// before the move is committed via PlayCommand.Execute.
type PlayMove struct {
	// Placed lists tiles placed this turn in left-to-right / top-to-bottom order.
	Placed []PlacedTile
	// WordsFormed holds all words formed (main word + cross-words), uppercased.
	// Populated by ValidatePlacement; must be non-empty before Score is called.
	WordsFormed []string
	// Score is the total point value for this move including multipliers and bingo.
	// Populated by Score; set on the move before Execute is called.
	Score int
}

func (PlayMove) moveMarker() {}

// ExchangeMove represents returning tiles to the bag and drawing replacements.
type ExchangeMove struct {
	// Tiles are the tiles returned to the bag.
	Tiles []Tile
}

func (ExchangeMove) moveMarker() {}

// PassMove represents skipping a turn without placing or exchanging tiles.
type PassMove struct{}

func (PassMove) moveMarker() {}
