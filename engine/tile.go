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

// PlacedTile pairs a Tile with its board coordinates.
type PlacedTile struct {
	Tile     Tile
	Row, Col int // 0-indexed; row 0 = top, col 0 = left
}

// tileDistribution defines the standard 100-tile North American English tile set.
// Source: Official Scrabble tournament rules; total tile count must equal 100.
var tileDistribution = []struct {
	letter byte
	points int
	count  int
}{
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

// tileDistributionTotal is the expected sum of all tile counts (verified in tests).
const tileDistributionTotal = 100

// letterPointsTable maps each uppercase letter A–Z to its standard point value,
// derived from tileDistribution so there is a single source of truth.
var letterPointsTable = func() [26]int {
	var t [26]int
	for _, d := range tileDistribution {
		if d.letter != 0 {
			t[d.letter-'A'] = d.points
		}
	}
	return t
}()

// LetterPoints returns the standard point value for an uppercase letter A–Z.
// It returns 0 for any other byte, including the blank sentinel (0): blank tiles
// always score 0 regardless of the letter they represent. The AI move generator
// uses this to stamp the correct face value onto tiles it places, since its
// rack representation discards per-tile point values.
func LetterPoints(letter byte) int {
	if letter < 'A' || letter > 'Z' {
		return 0
	}
	return letterPointsTable[letter-'A']
}
