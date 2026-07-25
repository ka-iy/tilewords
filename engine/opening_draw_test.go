// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"math/rand"
	"testing"
)

// TestDrawForFirstTurn_NearestToAWins verifies the opening-draw ordering: the
// player whose drawn letter is nearest the start of the alphabet plays first, and
// a blank (letter 0) beats any lettered tile. newTestBag draws from the end of the
// slice, so the last two tiles are drawn[0] (human) and drawn[1] (AI) respectively.
func TestDrawForFirstTurn_NearestToAWins(t *testing.T) {
	const blank = byte(0)
	cases := []struct {
		name      string
		human, ai byte
		wantFirst Turn
	}{
		{"human nearer", 'A', 'B', HumanTurn},
		{"ai nearer", 'B', 'A', AITurn},
		{"human blank beats A", blank, 'A', HumanTurn},
		{"ai blank beats A", 'A', blank, AITurn},
		{"far apart, ai wins", 'Z', 'Y', AITurn},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Slice order: tiles[len-2] is drawn first (human), tiles[len-1] second (AI).
			bag := newTestBag([]Tile{{Letter: tc.human, IsBlank: tc.human == 0}, {Letter: tc.ai, IsBlank: tc.ai == 0}})
			first, human, ai := drawForFirstTurn(bag, rand.New(rand.NewSource(1)))
			if first != tc.wantFirst {
				t.Errorf("first = %v, want %v", first, tc.wantFirst)
			}
			if human != tc.human || ai != tc.ai {
				t.Errorf("drawn letters = (%q, %q), want (%q, %q)", human, ai, tc.human, tc.ai)
			}
			// All tiles are returned to the bag before dealing.
			if bag.Count() != 2 {
				t.Errorf("bag count after draw = %d, want 2 (tiles must be returned)", bag.Count())
			}
		})
	}
}

// TestDrawForFirstTurn_TieRedraws confirms that a tie (equal letters) is re-drawn
// and the routine still terminates with two distinct letters whose order matches
// the returned first player. The bag holds duplicate 'A's to force the tie path.
func TestDrawForFirstTurn_TieRedraws(t *testing.T) {
	bag := newTestBag([]Tile{{Letter: 'A'}, {Letter: 'B'}, {Letter: 'A'}})
	first, human, ai := drawForFirstTurn(bag, rand.New(rand.NewSource(7)))
	if human == ai {
		t.Fatalf("drawn letters tied (%q == %q); a tie must be re-drawn", human, ai)
	}
	wantFirst := HumanTurn
	if ai < human {
		wantFirst = AITurn
	}
	if first != wantFirst {
		t.Errorf("first = %v, want %v for letters (%q, %q)", first, wantFirst, human, ai)
	}
	if bag.Count() != 3 {
		t.Errorf("bag count = %d, want 3 (all tiles returned)", bag.Count())
	}
}

// TestNew_FirstPlayerVaries is the regression guard for the reported bug ("the AI
// always starts"): across many seeds, New must select each player first at least
// once, and OpeningDraw must agree with CurrentTurn and the drawn letters.
func TestNew_FirstPlayerVaries(t *testing.T) {
	humanFirst, aiFirst := 0, 0
	for seed := int64(1); seed <= 300; seed++ {
		state := New(testDict.Name(), 5, rand.New(rand.NewSource(seed)))

		od := state.OpeningDraw
		if od == nil {
			t.Fatalf("seed %d: OpeningDraw is nil", seed)
		}
		if od.First != state.CurrentTurn {
			t.Fatalf("seed %d: OpeningDraw.First = %v but CurrentTurn = %v", seed, od.First, state.CurrentTurn)
		}
		if od.HumanLetter == od.AILetter {
			t.Fatalf("seed %d: opening draw recorded a tie (%q == %q)", seed, od.HumanLetter, od.AILetter)
		}
		wantFirst := HumanTurn
		if od.AILetter < od.HumanLetter {
			wantFirst = AITurn
		}
		if od.First != wantFirst {
			t.Fatalf("seed %d: First = %v, want %v for letters (%q, %q)", seed, od.First, wantFirst, od.HumanLetter, od.AILetter)
		}

		switch state.CurrentTurn {
		case HumanTurn:
			humanFirst++
		case AITurn:
			aiFirst++
		}
	}
	if humanFirst == 0 || aiFirst == 0 {
		t.Fatalf("first player did not vary: humanFirst=%d aiFirst=%d", humanFirst, aiFirst)
	}
}
