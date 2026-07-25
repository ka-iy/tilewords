// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"math/rand"
	"testing"
)

// TestInterestingDistributionTotals verifies the Interesting economy is 106 letters + 4
// blanks = 110 tiles, matching tileDistributionInterestingTotal.
func TestInterestingDistributionTotals(t *testing.T) {
	letters, blanks, sum := 0, 0, 0
	for _, d := range tileDistributionInteresting {
		sum += d.count
		if d.letter == 0 {
			blanks += d.count
		} else {
			letters += d.count
		}
	}
	if sum != tileDistributionInterestingTotal {
		t.Fatalf("interesting total = %d, want %d", sum, tileDistributionInterestingTotal)
	}
	if letters != 106 || blanks != 4 {
		t.Fatalf("interesting split = %d letters + %d blanks, want 106 + 4", letters, blanks)
	}
}

// TestNewBagForModeCounts verifies each mode fills the bag with the right tile count.
func TestNewBagForModeCounts(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	if n := NewBagForMode(rng, ClassicMode).Count(); n != 100 {
		t.Fatalf("classic bag = %d, want 100", n)
	}
	if n := NewBagForMode(rng, InterestingMode).Count(); n != 110 {
		t.Fatalf("interesting bag = %d, want 110", n)
	}
}

// TestNewWithModeSetsMode verifies the mode is recorded on both the state and its board.
func TestNewWithModeSetsMode(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	s := NewWithMode("enable", 5, InterestingMode, rng)
	if s.Mode != InterestingMode {
		t.Fatalf("state.Mode = %v, want Interesting", s.Mode)
	}
	if s.Board.Mode() != InterestingMode {
		t.Fatalf("board.Mode = %v, want Interesting", s.Board.Mode())
	}
	// Tile conservation for the Interesting economy.
	total := s.Bag.Count() + s.HumanRack.Count() + s.AIRack.Count()
	if total != tileDistributionInterestingTotal {
		t.Fatalf("interesting tile conservation: got %d, want %d", total, tileDistributionInterestingTotal)
	}
}

// TestBoardModePointsPersist verifies mode-aware letter points, and that Clone and the
// gob round-trip preserve the board's mode.
func TestBoardModePointsPersist(t *testing.T) {
	b := NewBoardForMode(InterestingMode)
	// 'U' is worth 2 in Interesting mode but 1 in Classic — a clear per-mode difference.
	if got := b.LetterPoints('U'); got != 2 {
		t.Fatalf("interesting U points = %d, want 2", got)
	}
	if got := NewBoardForMode(ClassicMode).LetterPoints('U'); got != 1 {
		t.Fatalf("classic U points = %d, want 1", got)
	}
	if b.LetterPoints('_') != 0 {
		t.Fatal("non-letter should score 0")
	}
	if b.Clone().Mode() != InterestingMode {
		t.Fatal("Clone dropped the board mode")
	}
	data, err := b.GobEncode()
	if err != nil {
		t.Fatalf("GobEncode: %v", err)
	}
	var b2 Board
	if err := b2.GobDecode(data); err != nil {
		t.Fatalf("GobDecode: %v", err)
	}
	if b2.Mode() != InterestingMode {
		t.Fatal("gob round-trip dropped the board mode")
	}
}

// TestInterestingPremiumLayout verifies the pinwheel premium counts and centre position.
func TestInterestingPremiumLayout(t *testing.T) {
	b := NewBoardForMode(InterestingMode)
	var tw, dw, tl, dl, ctr int
	for r := 0; r < 15; r++ {
		for c := 0; c < 15; c++ {
			switch b.Cell(r, c).Square {
			case TripleWord:
				tw++
			case DoubleWord:
				dw++
			case TripleLetter:
				tl++
			case DoubleLetter:
				dl++
			case Centre:
				ctr++
			}
		}
	}
	if tw != 4 || dw != 12 || tl != 12 || dl != 28 || ctr != 1 {
		t.Fatalf("interesting premium counts = TW%d DW%d TL%d DL%d centre%d, want 4/12/12/28/1",
			tw, dw, tl, dl, ctr)
	}
	if b.Cell(7, 7).Square != Centre {
		t.Fatal("centre square must be at (7,7)")
	}
	// Corners must be empty (a defining difference from the Classic layout).
	for _, cell := range [][2]int{{0, 0}, {0, 14}, {14, 0}, {14, 14}} {
		if b.Cell(cell[0], cell[1]).Square != Normal {
			t.Fatalf("corner (%d,%d) must be Normal in the interesting layout", cell[0], cell[1])
		}
	}
}
