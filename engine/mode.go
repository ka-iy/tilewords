// Package engine is documented in doc.go.
package engine

// GameMode selects a board premium-square layout together with a tile economy
// (letter distribution and point values). It is chosen once at game start and
// persisted with the save so a resumed game uses the same board and economy.
type GameMode int

const (
	// ClassicMode is the standard 15×15 premium-square layout and the standard
	// 100-tile English letter economy. It is the zero value, so older save files
	// (written before game modes existed) decode as ClassicMode.
	ClassicMode GameMode = iota

	// InterestingMode is an independently-designed alternative: a 4-fold rotational
	// ("pinwheel") premium-square layout with even coverage, and a 110-tile economy
	// whose counts and point values are derived from letter-occurrence frequencies
	// over the bundled public-domain / open word lists.
	InterestingMode
)

// String returns a short human-readable name for the mode.
func (m GameMode) String() string {
	if m == InterestingMode {
		return "Interesting"
	}
	return "Classic"
}
