// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"math/rand"
	"testing"
)

// letterBag returns a bag holding one tile for each of the given letters, in that order.
func letterBag(letters string) *Bag {
	tiles := make([]Tile, 0, len(letters))
	for i := 0; i < len(letters); i++ {
		tiles = append(tiles, Tile{Letter: letters[i], Points: 1})
	}
	return newTestBag(tiles)
}

// A multi-tile draw takes every tile from a freshly shuffled bag, so each tile still in
// the bag is equally likely to arrive at any position of the draw. Were the bag shuffled
// only once per game (or not at all before a draw), a tile's chance of being drawn would
// depend on where it happened to sit, and tiles returned to a known position would come
// straight back out.
func TestBagDrawIsUniformAtEveryDrawPosition(t *testing.T) {
	const (
		letters = "ABCDEF"
		perDraw = 3
		trials  = 60000
	)
	rng := rand.New(rand.NewSource(7))

	// counts[position][letter] — how often each letter arrived at each draw position.
	counts := make([]map[byte]int, perDraw)
	for i := range counts {
		counts[i] = map[byte]int{}
	}

	for n := 0; n < trials; n++ {
		drawn := letterBag(letters).Draw(perDraw, rng)
		if len(drawn) != perDraw {
			t.Fatalf("Draw(%d) returned %d tiles", perDraw, len(drawn))
		}
		seen := map[byte]bool{}
		for i, tile := range drawn {
			if seen[tile.Letter] {
				t.Fatalf("Draw(%d) returned letter %q twice: a draw takes tiles without replacement",
					perDraw, tile.Letter)
			}
			seen[tile.Letter] = true
			counts[i][tile.Letter]++
		}
	}

	// Each of the 6 letters should reach each position in about 1/6 of the trials. The
	// 15% band is far wider than the sampling noise at this trial count (a standard
	// deviation is ~0.4% of the expected count) and far tighter than any positional bias
	// a missing reshuffle would produce.
	want := float64(trials) / float64(len(letters))
	for pos := range counts {
		for i := 0; i < len(letters); i++ {
			got := float64(counts[pos][letters[i]])
			if got < want*0.85 || got > want*1.15 {
				t.Errorf("letter %q arrived at draw position %d %v times, want ~%.0f",
					letters[i], pos, got, want)
			}
		}
	}
}

// A nil rng skips the reshuffle, so a draw is the tail of the bag in its existing order.
// Callers rely on this to assert an exact bag (see Bag.Shuffle).
func TestBagDrawNilRNGKeepsBagOrder(t *testing.T) {
	bag := letterBag("ABCDEF")
	drawn := bag.Draw(3, nil)

	if got := string([]byte{drawn[0].Letter, drawn[1].Letter, drawn[2].Letter}); got != "DEF" {
		t.Errorf("Draw(3, nil) = %q, want %q", got, "DEF")
	}
	if bag.Count() != 3 {
		t.Errorf("after Draw(3, nil): count = %d, want 3", bag.Count())
	}
	remaining := bag.Draw(3, nil)
	if got := string([]byte{remaining[0].Letter, remaining[1].Letter, remaining[2].Letter}); got != "ABC" {
		t.Errorf("second Draw(3, nil) = %q, want %q", got, "ABC")
	}
}

// Replenish passes rng through to the draw, so a rack's replacement tiles are subject to
// the same per-tile reshuffle. This covers both players: the human and the AI replenish
// through PlayCommand.Execute, which routes here.
func TestRackReplenishDrawsRandomly(t *testing.T) {
	const trials = 2000
	rng := rand.New(rand.NewSource(11))

	// With a 26-letter bag and a 7-tile replenish, the draw taking the bag's tail every
	// time would yield the same 7 letters on every trial.
	firstLetters := map[byte]int{}
	for n := 0; n < trials; n++ {
		bag := letterBag("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		rack := &Rack{}
		drawn := rack.Replenish(bag, rng)
		if len(drawn) != MaxRackSize {
			t.Fatalf("Replenish drew %d tiles, want %d", len(drawn), MaxRackSize)
		}
		if rack.Count() != MaxRackSize {
			t.Fatalf("after Replenish: rack holds %d tiles, want %d", rack.Count(), MaxRackSize)
		}
		firstLetters[drawn[0].Letter]++
	}
	if len(firstLetters) != 26 {
		t.Errorf("first replacement tile was one of only %d distinct letters over %d replenishments, want all 26",
			len(firstLetters), trials)
	}
}

// An exchange draws its replacements while the exchanged tiles are out of the bag, so no
// reshuffle during the draw can hand a player back a tile they just exchanged away.
func TestExchangeNeverReturnsTheExchangedTiles(t *testing.T) {
	const trials = 500
	rng := rand.New(rand.NewSource(3))

	for n := 0; n < trials; n++ {
		state := newGameState()
		// Exchange the whole rack, so any exchanged tile coming back is unmistakable.
		exchanged := state.HumanRack.Tiles()

		// The multiset the replacements must come from: the bag as it stood before the
		// exchange. An exchanged 'E' may legitimately be replaced by a different 'E' that
		// was already in the bag, so the check has to count letters rather than compare
		// tiles one by one.
		available := map[byte]int{}
		for _, tile := range state.Bag.tiles {
			available[tile.Letter]++
		}

		cmd := &ExchangeCommand{Move: ExchangeMove{Tiles: exchanged}}
		if err := cmd.Execute(state, testDict, rng); err != nil {
			t.Fatalf("ExchangeCommand.Execute: %v", err)
		}

		for _, tile := range cmd.drawnTiles {
			available[tile.Letter]--
			if available[tile.Letter] < 0 {
				t.Fatalf("exchange drew more %q tiles than the bag held before the exchange: an exchanged tile was drawn back",
					tile.Letter)
			}
		}
		if state.HumanRack.Count() != len(exchanged) {
			t.Fatalf("after exchange: rack holds %d tiles, want %d",
				state.HumanRack.Count(), len(exchanged))
		}
	}
}
