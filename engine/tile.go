// Package engine is documented in doc.go.
package engine

// Tile represents a single game tile, in a rack or placed on the board.
type Tile struct {
	// Letter is the uppercase A-Z letter. 0 means the tile is a blank in the rack
	// (letter not yet assigned).
	Letter byte

	// Points is the face value of the tile. Always 0 for blank tiles.
	Points int

	// IsBlank is true for blank tiles regardless of whether they have been played.
	IsBlank bool

	// AssignedLetter is the letter chosen when a blank tile is played onto the board.
	// 0 when the blank is still in the rack.
	AssignedLetter byte
}

// DisplayLetter returns the letter to display for this tile.
// For a played blank, returns AssignedLetter; for an unplayed tile, returns Letter.
func (t Tile) DisplayLetter() byte {
	if t.IsBlank && t.AssignedLetter != 0 {
		return t.AssignedLetter
	}
	return t.Letter
}

// wellFormed reports whether the tile is one the game could have produced: a blank, or a
// tile whose Letter is an uppercase A-Z. Every tile the engine creates satisfies this, so
// it only ever fails for a tile that came from outside — a decoded save file, where a
// single corrupted byte can leave a letter the rest of the engine has no meaning for.
func (t Tile) wellFormed() bool {
	if t.IsBlank {
		// A blank's assignment is either absent or a real letter.
		return t.AssignedLetter == 0 || (t.AssignedLetter >= 'A' && t.AssignedLetter <= 'Z')
	}
	return t.Letter >= 'A' && t.Letter <= 'Z'
}

// PlacedTile pairs a Tile with its board coordinates.
type PlacedTile struct {
	Tile     Tile
	Row, Col int // 0-indexed; row 0 = top, col 0 = left
}

// tileSpec is one entry of a tile distribution: a letter (0 = blank), its face value,
// and how many such tiles the bag holds.
type tileSpec struct {
	letter byte
	points int
	count  int
}

// tileDistribution is the ClassicMode 100-tile North American English tile set.
// Source: standard crossword-game tournament rules; total tile count must equal 100.
var tileDistribution = []tileSpec{
	{0, 0, 2},    // blank ×2
	{'A', 1, 9},  // A ×9
	{'B', 3, 2},  // B ×2
	{'C', 3, 2},  // C ×2
	{'D', 2, 4},  // D ×4
	{'E', 1, 12}, // E ×12
	{'F', 4, 2},  // F ×2
	{'G', 2, 3},  // G ×3
	{'H', 4, 2},  // H ×2
	{'I', 1, 9},  // I ×9
	{'J', 8, 1},  // J ×1
	{'K', 5, 1},  // K ×1
	{'L', 1, 4},  // L ×4
	{'M', 3, 2},  // M ×2
	{'N', 1, 6},  // N ×6
	{'O', 1, 8},  // O ×8
	{'P', 3, 2},  // P ×2
	{'Q', 10, 1}, // Q ×1
	{'R', 1, 6},  // R ×6
	{'S', 1, 4},  // S ×4
	{'T', 1, 6},  // T ×6
	{'U', 1, 4},  // U ×4
	{'V', 4, 2},  // V ×2
	{'W', 4, 2},  // W ×2
	{'X', 8, 1},  // X ×1
	{'Y', 4, 2},  // Y ×2
	{'Z', 10, 1}, // Z ×1
}

// tileDistributionTotal is the ClassicMode tile count (verified in tests).
const tileDistributionTotal = 100

// tileDistributionInteresting is the InterestingMode 110-tile set: 106 letters + 4
// blanks. Counts and point values are independently derived from letter-occurrence
// frequencies over the bundled public-domain / open word lists (rarer letters score
// more and are scarcer), not copied from any existing game.
var tileDistributionInteresting = []tileSpec{
	{0, 0, 4},    // blank ×4
	{'A', 1, 8},  // A ×8
	{'B', 4, 2},  // B ×2
	{'C', 2, 4},  // C ×4
	{'D', 2, 3},  // D ×3
	{'E', 1, 12}, // E ×12
	{'F', 4, 1},  // F ×1
	{'G', 3, 3},  // G ×3
	{'H', 3, 3},  // H ×3
	{'I', 1, 9},  // I ×9
	{'J', 10, 1}, // J ×1
	{'K', 5, 2},  // K ×2
	{'L', 2, 6},  // L ×6
	{'M', 3, 3},  // M ×3
	{'N', 1, 7},  // N ×7
	{'O', 1, 7},  // O ×7
	{'P', 2, 3},  // P ×3
	{'Q', 10, 1}, // Q ×1
	{'R', 1, 7},  // R ×7
	{'S', 1, 6},  // S ×6
	{'T', 1, 7},  // T ×7
	{'U', 2, 3},  // U ×3
	{'V', 5, 2},  // V ×2
	{'W', 8, 2},  // W ×2
	{'X', 10, 1}, // X ×1
	{'Y', 4, 2},  // Y ×2
	{'Z', 8, 1},  // Z ×1
}

// tileDistributionInterestingTotal is the InterestingMode tile count (verified in tests).
const tileDistributionInterestingTotal = 110

// deriveLetterPoints builds an A–Z point-value table from a distribution, so the point
// tables and the tile distributions share a single source of truth.
func deriveLetterPoints(dist []tileSpec) [26]int {
	var t [26]int
	for _, d := range dist {
		if d.letter != 0 {
			t[d.letter-'A'] = d.points
		}
	}
	return t
}

// letterPointsTable / letterPointsInteresting map each uppercase letter A–Z to its
// point value in the corresponding mode.
var (
	letterPointsTable       = deriveLetterPoints(tileDistribution)
	letterPointsInteresting = deriveLetterPoints(tileDistributionInteresting)
)

// distributionForMode returns the tile distribution used to fill the bag for mode.
func distributionForMode(mode GameMode) []tileSpec {
	if mode == InterestingMode {
		return tileDistributionInteresting
	}
	return tileDistribution
}

// letterPointsForMode returns the A–Z point table for mode.
func letterPointsForMode(mode GameMode) *[26]int {
	if mode == InterestingMode {
		return &letterPointsInteresting
	}
	return &letterPointsTable
}

// LetterPoints returns the ClassicMode point value for an uppercase letter A–Z.
// It returns 0 for any other byte, including the blank sentinel (0): blank tiles
// always score 0 regardless of the letter they represent. Mode-aware point lookup
// during play goes through Board.LetterPoints so each mode scores correctly.
func LetterPoints(letter byte) int {
	if letter < 'A' || letter > 'Z' {
		return 0
	}
	return letterPointsTable[letter-'A']
}

// TileSpec is an exported view of one tile-distribution entry, used by the UI to
// display a mode's tile economy. Letter 0 denotes the blank.
type TileSpec struct {
	// Letter is the uppercase A–Z letter, or 0 for a blank.
	Letter byte
	// Points is the tile's face value (0 for a blank).
	Points int
	// Count is how many such tiles are in the bag.
	Count int
}

// Distribution returns the tile economy for mode as a slice of TileSpec, for display.
func Distribution(mode GameMode) []TileSpec {
	dist := distributionForMode(mode)
	out := make([]TileSpec, len(dist))
	for i, d := range dist {
		out[i] = TileSpec{Letter: d.letter, Points: d.points, Count: d.count}
	}
	return out
}

// TotalTiles returns the number of tiles in the bag for mode.
func TotalTiles(mode GameMode) int {
	if mode == InterestingMode {
		return tileDistributionInterestingTotal
	}
	return tileDistributionTotal
}
