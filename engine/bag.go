// Package engine is documented in doc.go.
package engine

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/rand"
)

// Bag holds the pool of tiles available for drawing during the game.
type Bag struct {
	tiles []Tile // current remaining tiles; draw pops from the end
}

// NewBag returns a shuffled standard 100-tile North American English bag.
// The shuffle uses the Fisher-Yates algorithm seeded from rng.
func NewBag(rng *rand.Rand) *Bag {
	var tiles []Tile
	for _, entry := range tileDistribution {
		for i := 0; i < entry.count; i++ {
			tiles = append(tiles, Tile{
				Letter:  entry.letter,
				Points:  entry.points,
				IsBlank: entry.letter == 0,
			})
		}
	}
	// Fisher-Yates shuffle: for i from n-1 down to 1, swap tiles[i] with
	// a uniformly random tiles[j] where 0 ≤ j ≤ i.
	for i := len(tiles) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		tiles[i], tiles[j] = tiles[j], tiles[i]
	}
	return &Bag{tiles: tiles}
}

// Draw removes and returns up to n tiles from the bag.
// If fewer than n tiles remain, all remaining tiles are returned.
func (b *Bag) Draw(n int) []Tile {
	if n > len(b.tiles) {
		n = len(b.tiles)
	}
	// Pop from the end of the slice (O(1)).
	drawn := make([]Tile, n)
	copy(drawn, b.tiles[len(b.tiles)-n:])
	b.tiles = b.tiles[:len(b.tiles)-n]
	return drawn
}

// Return adds tiles back to the bag. If rng is non-nil the bag is reshuffled
// after the tiles are inserted; pass nil to skip the reshuffle (used by Undo).
func (b *Bag) Return(tiles []Tile, rng *rand.Rand) {
	b.tiles = append(b.tiles, tiles...)
	if rng != nil {
		for i := len(b.tiles) - 1; i > 0; i-- {
			j := rng.Intn(i + 1)
			b.tiles[i], b.tiles[j] = b.tiles[j], b.tiles[i]
		}
	}
}

// Count returns the number of tiles remaining in the bag.
func (b *Bag) Count() int { return len(b.tiles) }

// Clone returns an independent deep copy of the bag.
func (b *Bag) Clone() *Bag {
	cp := make([]Tile, len(b.tiles))
	copy(cp, b.tiles)
	return &Bag{tiles: cp}
}

// GobEncode serialises the bag for save/load. Required because tiles is unexported.
func (b *Bag) GobEncode() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(b.tiles); err != nil {
		return nil, fmt.Errorf("engine.Bag.GobEncode: %w", err)
	}
	return buf.Bytes(), nil
}

// GobDecode deserialises the bag from save/load data.
func (b *Bag) GobDecode(data []byte) error {
	if err := gob.NewDecoder(bytes.NewReader(data)).Decode(&b.tiles); err != nil {
		return fmt.Errorf("engine.Bag.GobDecode: %w", err)
	}
	return nil
}

// restoreSnapshot replaces the bag's tile slice with a copy of snap.
// Used exclusively by ExchangeCommand.Undo to reverse the post-reshuffle bag state.
func (b *Bag) restoreSnapshot(snap []Tile) {
	b.tiles = make([]Tile, len(snap))
	copy(b.tiles, snap)
}
