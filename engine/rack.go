// Package engine is documented in doc.go.
package engine

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
)

// MaxRackSize is the maximum number of tiles a player may hold.
const MaxRackSize = 7

// Rack holds a player's current hand of up to MaxRackSize tiles.
type Rack struct {
	tiles []Tile
}

// Tiles returns a copy of the rack's current tiles. The returned slice is
// independent; modifying it does not affect the rack.
func (r *Rack) Tiles() []Tile {
	cp := make([]Tile, len(r.tiles))
	copy(cp, r.tiles)
	return cp
}

// Count returns the number of tiles currently on the rack.
func (r *Rack) Count() int { return len(r.tiles) }

// Add appends tiles to the rack. Returns an error if adding would exceed MaxRackSize.
func (r *Rack) Add(tiles []Tile) error {
	if len(r.tiles)+len(tiles) > MaxRackSize {
		return fmt.Errorf("engine.Rack.Add: adding %d tile(s) would exceed rack capacity %d (current: %d)",
			len(tiles), MaxRackSize, len(r.tiles))
	}
	r.tiles = append(r.tiles, tiles...)
	return nil
}

// Remove removes the first occurrence of each tile in tiles from the rack.
// Returns an error (without modifying the rack) if any tile is not found.
func (r *Rack) Remove(tiles []Tile) error {
	// Work on a copy to preserve atomicity: either all succeed or none are removed.
	work := make([]Tile, len(r.tiles))
	copy(work, r.tiles)

	for _, want := range tiles {
		idx := -1
		for i, have := range work {
			if tilesMatch(have, want) {
				idx = i
				break
			}
		}
		if idx == -1 {
			return fmt.Errorf("engine.Rack.Remove: tile {Letter:%c IsBlank:%v} not found in rack",
				want.Letter, want.IsBlank)
		}
		// Remove element at idx by replacing with the last element and shrinking.
		work[idx] = work[len(work)-1]
		work = work[:len(work)-1]
	}
	r.tiles = work
	return nil
}

// Replenish draws tiles from bag until the rack holds MaxRackSize tiles or the
// bag is empty. Returns the drawn tiles so callers (e.g. PlayCommand) can
// record them for undo.
func (r *Rack) Replenish(bag *Bag) []Tile {
	need := MaxRackSize - len(r.tiles)
	if need <= 0 || bag.Count() == 0 {
		return nil
	}
	drawn := bag.Draw(need)
	r.tiles = append(r.tiles, drawn...)
	return drawn
}

// MoveTile reorders the rack so the tile at index from is repositioned to index to,
// the other tiles shifting to fill the gap. Tile order is purely cosmetic (it does
// not affect play), so this only changes display order. Out-of-range or no-op
// (from == to) requests are ignored.
func (r *Rack) MoveTile(from, to int) {
	n := len(r.tiles)
	if from < 0 || from >= n || to < 0 || to >= n || from == to {
		return
	}
	tiles := make([]Tile, n)
	copy(tiles, r.tiles)
	moved := tiles[from]
	rest := append(tiles[:from], tiles[from+1:]...) // remove `from` from the copy
	out := make([]Tile, 0, n)
	out = append(out, rest[:to]...)
	out = append(out, moved)
	out = append(out, rest[to:]...)
	r.tiles = out
}

// Shuffle randomises the order of the tiles in the rack using rng. Order is purely
// cosmetic; the set of tiles is unchanged.
func (r *Rack) Shuffle(rng *rand.Rand) {
	for i := len(r.tiles) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		r.tiles[i], r.tiles[j] = r.tiles[j], r.tiles[i]
	}
}

// Clone returns an independent deep copy of the rack.
func (r *Rack) Clone() *Rack {
	cp := make([]Tile, len(r.tiles))
	copy(cp, r.tiles)
	return &Rack{tiles: cp}
}

// GobEncode serialises the rack for save/load. Required because tiles is unexported.
func (r *Rack) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(r.tiles); err != nil {
		return nil, fmt.Errorf("engine.Rack.GobEncode: %w", err)
	}
	return buf.Bytes(), nil
}

// GobDecode deserialises the rack from save/load data.
func (r *Rack) GobDecode(data []byte) error {
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&r.tiles); err != nil {
		return fmt.Errorf("engine.Rack.GobDecode: %w", err)
	}
	return nil
}

// tilesMatch reports whether two tiles are equivalent for rack-removal purposes.
// Blank tiles match only other blank tiles; lettered tiles match by letter.
// AssignedLetter is intentionally ignored: a played blank in the rack (impossible
// in normal play, but defensive) is treated as a blank.
func tilesMatch(a, b Tile) bool {
	if a.IsBlank != b.IsBlank {
		return false
	}
	if a.IsBlank {
		return true // all blanks are interchangeable in the rack
	}
	return a.Letter == b.Letter
}
