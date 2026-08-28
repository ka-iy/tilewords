// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package engine

import (
	"strings"
	"testing"
)

// validState returns a minimal state that ValidateDecodedState must accept.
func validState() *GameState {
	return &GameState{
		Board:     newFlatBoard(),
		HumanRack: &Rack{tiles: []Tile{{Letter: 'A', Points: 1}, {IsBlank: true}}},
		CPURack:   &Rack{tiles: []Tile{{Letter: 'Z', Points: 10}}},
		Bag:       newTestBag([]Tile{{Letter: 'E', Points: 1}, {IsBlank: true}}),
	}
}

// TestValidateDecodedState_AcceptsWellFormedState checks the fixture the rejection cases
// below are derived from is itself accepted, so those cases fail for the intended reason.
func TestValidateDecodedState_AcceptsWellFormedState(t *testing.T) {
	if err := ValidateDecodedState(validState()); err != nil {
		t.Fatalf("rejected a well-formed state: %v", err)
	}
}

// TestValidateDecodedState_AcceptsPlayedBlank verifies a blank that has been played, and so
// carries an assigned letter, is still accepted.
func TestValidateDecodedState_AcceptsPlayedBlank(t *testing.T) {
	s := validState()
	if err := s.Board.Place(7, 7, Tile{IsBlank: true, AssignedLetter: 'Q'}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateDecodedState(s); err != nil {
		t.Fatalf("rejected a played blank: %v", err)
	}
}

// TestValidateDecodedState_RejectsUnplayableTiles verifies that a tile no game could have
// produced is refused wherever it is held. A decoded save is outside data, and the letter is
// used to index letter-keyed arrays downstream, so accepting one crashes the process later
// — from the CPU goroutine, where nothing recovers.
func TestValidateDecodedState_RejectsUnplayableTiles(t *testing.T) {
	// A rack blank whose IsBlank flag is lost decodes as exactly this tile.
	poison := Tile{Letter: 0, IsBlank: false}

	cases := []struct {
		name    string
		corrupt func(*GameState)
		want    string
	}{
		{"human rack", func(s *GameState) { s.HumanRack.tiles = append(s.HumanRack.tiles, poison) }, "human rack"},
		{"CPU rack", func(s *GameState) { s.CPURack.tiles = append(s.CPURack.tiles, poison) }, "CPU rack"},
		// The bag matters as much as the racks: a malformed tile there is drawn onto a rack
		// turns later, so the crash it causes is delayed rather than avoided.
		{"bag", func(s *GameState) { s.Bag.tiles = append(s.Bag.tiles, poison) }, "bag"},
		{"board", func(s *GameState) { s.Board.cells[3][4].Tile = &poison }, "board"},
		{"lowercase letter", func(s *GameState) {
			s.HumanRack.tiles = append(s.HumanRack.tiles, Tile{Letter: 'a'})
		}, "human rack"},
		{"letter past Z", func(s *GameState) {
			s.CPURack.tiles = append(s.CPURack.tiles, Tile{Letter: '\\'})
		}, "CPU rack"},
		{"blank with a bogus assignment", func(s *GameState) {
			s.HumanRack.tiles = append(s.HumanRack.tiles, Tile{IsBlank: true, AssignedLetter: '!'})
		}, "human rack"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := validState()
			tc.corrupt(s)
			err := ValidateDecodedState(s)
			if err == nil {
				t.Fatal("accepted a state holding an unplayable tile")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to name %q", err, tc.want)
			}
		})
	}
}

// TestValidateDecodedState_RejectsOversizedRack verifies an over-capacity rack is refused.
// Such a rack cannot be played: an exchange removes the selected tiles, draws that many
// back, and then cannot fit them, which the command layer treats as unreachable.
func TestValidateDecodedState_RejectsOversizedRack(t *testing.T) {
	for _, which := range []string{"human", "CPU"} {
		t.Run(which, func(t *testing.T) {
			s := validState()
			big := make([]Tile, MaxRackSize+3)
			for i := range big {
				big[i] = Tile{Letter: 'E', Points: 1}
			}
			if which == "human" {
				s.HumanRack.tiles = big
			} else {
				s.CPURack.tiles = big
			}
			err := ValidateDecodedState(s)
			if err == nil {
				t.Fatal("accepted a rack over capacity")
			}
			if !strings.Contains(err.Error(), "capacity") {
				t.Errorf("error = %q, want it to mention the capacity", err)
			}
		})
	}
}

// TestValidateDecodedState_RejectsMissingFields verifies the nil guards, since gob can
// decode successfully while leaving pointer fields unset.
func TestValidateDecodedState_RejectsMissingFields(t *testing.T) {
	if err := ValidateDecodedState(nil); err == nil {
		t.Error("accepted a nil state")
	}
	for _, tc := range []struct {
		name    string
		corrupt func(*GameState)
	}{
		{"board", func(s *GameState) { s.Board = nil }},
		{"human rack", func(s *GameState) { s.HumanRack = nil }},
		{"CPU rack", func(s *GameState) { s.CPURack = nil }},
		{"bag", func(s *GameState) { s.Bag = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := validState()
			tc.corrupt(s)
			if err := ValidateDecodedState(s); err == nil {
				t.Errorf("accepted a state with a nil %s", tc.name)
			}
		})
	}
}

// TestClone_CarriesEveryField guards against Clone silently dropping a field. Clone copies
// the struct wholesale and overrides only what must not be shared, so a newly added field
// is carried by default; this pins that, and pins the deep copies it must still make.
func TestClone_CarriesEveryField(t *testing.T) {
	s := validState()
	s.HumanScore, s.CPUScore = 40, 25
	s.ConsecutivePasses, s.MoveNumber, s.CPULevel = 2, 9, 7
	s.CurrentTurn = CPUTurn
	s.Mode = InterestingMode
	s.EndgameScored = true
	s.OfficialNotation = true
	s.History = []MoveRecord{{Player: "You", Line: "8D CAT +10", Points: 10}}
	s.OpeningDraw = &OpeningDraw{First: HumanTurn}

	c := s.Clone()

	if !c.EndgameScored {
		t.Error("Clone dropped EndgameScored: a clone of a finished game would be scored again")
	}
	if !c.OfficialNotation {
		t.Error("Clone dropped OfficialNotation")
	}
	if c.OpeningDraw == nil || c.OpeningDraw.First != HumanTurn {
		t.Error("Clone dropped OpeningDraw")
	}
	if c.Mode != InterestingMode {
		t.Errorf("Clone Mode = %v, want InterestingMode", c.Mode)
	}
	if c.HumanScore != 40 || c.CPUScore != 25 || c.ConsecutivePasses != 2 ||
		c.MoveNumber != 9 || c.CPULevel != 7 || c.CurrentTurn != CPUTurn {
		t.Error("Clone dropped a scalar field")
	}

	// History is deliberately not shared: the UI appends to it while the CPU holds the clone.
	if c.History != nil {
		t.Errorf("Clone shared History (%v); the UI would append to it under the CPU goroutine", c.History)
	}
	// The mutable game objects must be independent copies.
	if c.Board == s.Board || c.HumanRack == s.HumanRack || c.CPURack == s.CPURack || c.Bag == s.Bag {
		t.Fatal("Clone shared a mutable game object with the original")
	}
	if err := c.Board.Place(0, 0, Tile{Letter: 'X', Points: 8}); err != nil {
		t.Fatal(err)
	}
	if !s.Board.IsEmpty(0, 0) {
		t.Error("mutating the clone's board changed the original")
	}
}
