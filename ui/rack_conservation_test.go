// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math/rand"
	"sort"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

// shownRackLetters returns the letters the human rack's slots are currently displaying,
// sorted. A face-down or empty slot contributes nothing.
func shownRackLetters(gs *gameScreen) []byte {
	var out []byte
	for _, slot := range gs.humanRack {
		if slot.tile == nil || slot.faceDown {
			continue
		}
		out = append(out, slot.tile.Letter)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// heldRackLetters returns the letters the engine says are on the human rack, sorted, minus
// any currently staged on the board (whose slots render empty).
func heldRackLetters(gs *gameScreen) []byte {
	stagedOut := map[int]bool{}
	for _, st := range gs.staged {
		stagedOut[st.FromRackIdx] = true
	}
	var out []byte
	for i, t := range gs.state.HumanRack.Tiles() {
		if stagedOut[i] || i == gs.dragRackSrc {
			continue
		}
		out = append(out, t.Letter)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sameLetters(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// The rack a player looks at must be exactly the rack the engine holds — no tile rendered
// twice, none dropped. A duplicate here would look precisely like a bad shuffle (the same
// letter appearing several times) while the bag was in fact dealt correctly, so the display
// is checked against the engine rather than assumed to follow it.
//
// The rack is dragged into new orders throughout, since reordering rebuilds the tile slice
// (see engine.Rack.MoveTile) and is the operation with the most scope to duplicate a tile.
func TestRackDisplayMatchesEngineRack(t *testing.T) {
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "DOG", "AT", "TO", "GO"})
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(9))

	for game := 0; game < 200; game++ {
		state := engine.New(dict.Name(), 5, rng)
		state.CurrentTurn = engine.HumanTurn
		gs := newGameScreen(nil, state, dict)
		gs.build().Resize(fyne.NewSize(400, 800))

		if got, want := shownRackLetters(gs), heldRackLetters(gs); !sameLetters(got, want) {
			t.Fatalf("game %d: freshly dealt rack shows %q, engine holds %q", game, got, want)
		}

		for step := 0; step < 20; step++ {
			n := gs.state.HumanRack.Count()
			if n < 2 {
				break
			}
			gs.reorderRack(rng.Intn(n), rng.Intn(n))
			gs.refresh()

			got, want := shownRackLetters(gs), heldRackLetters(gs)
			if !sameLetters(got, want) {
				t.Fatalf("game %d step %d: rack shows %q, engine holds %q", game, step, got, want)
			}
			if len(want) != n {
				t.Fatalf("game %d step %d: engine rack holds %d tiles, want %d — a reorder lost or duplicated one",
					game, step, len(want), n)
			}
		}
	}
}

// The tiles a replenishment puts on screen must be the tiles the bag handed out. This walks
// real turns — play tiles, refill, redraw — and checks the rack after each, so a duplicate
// introduced by the play/refill path (rather than by the draw) would be caught.
func TestRackDisplayMatchesEngineRackAcrossPlays(t *testing.T) {
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "DOG", "AT", "TO", "GO"})
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(21))

	for game := 0; game < 100; game++ {
		state := engine.New(dict.Name(), 5, rng)
		state.CurrentTurn = engine.HumanTurn
		gs := newGameScreen(nil, state, dict)
		gs.build().Resize(fyne.NewSize(400, 800))

		for turn := 0; turn < 10 && gs.state.Bag.Count() >= 4; turn++ {
			// Discard four tiles and refill, which is what a play does to the rack.
			held := gs.state.HumanRack.Tiles()
			if len(held) < 4 {
				break
			}
			if err := gs.state.HumanRack.Remove(held[:4]); err != nil {
				t.Fatalf("game %d turn %d: %v", game, turn, err)
			}
			gs.state.HumanRack.Replenish(gs.state.Bag, gs.rng)
			gs.refresh()

			got, want := shownRackLetters(gs), heldRackLetters(gs)
			if !sameLetters(got, want) {
				t.Fatalf("game %d turn %d: after refill the rack shows %q, engine holds %q",
					game, turn, got, want)
			}
			if n := gs.state.HumanRack.Count(); n != engine.MaxRackSize {
				t.Fatalf("game %d turn %d: rack holds %d tiles after refill, want %d",
					game, turn, n, engine.MaxRackSize)
			}
		}
	}
}
