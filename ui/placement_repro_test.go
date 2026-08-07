// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math/rand/v2"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

// firstNonBlankSlot returns a human rack slot index holding a non-blank tile.
func firstNonBlankSlot(gs *gameScreen) int {
	for i, t := range gs.state.HumanRack.Tiles() {
		if !t.IsBlank {
			return i
		}
	}
	return -1
}

// newPlacementHarness builds a real gameScreen forced to the human's turn, laid out at a
// phone-like size.
//
// The layout matters: a widget that was never sized occupies no space, which no widget does on
// screen, and the page scroll only exists once the column knows it overflows its viewport.
func newPlacementHarness(t *testing.T) *gameScreen {
	t.Helper()
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "DOG", "AT", "TO", "GO"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewPCG(1, 0)))
	state.CurrentTurn = engine.HumanTurn
	gs := newGameScreen(nil, state, dict)
	gs.build().Resize(fyne.NewSize(400, 800))
	return gs
}

// TestRepro_BasicPlacement: select a rack tile, tap an empty cell, expect it staged.
func TestRepro_BasicPlacement(t *testing.T) {
	gs := newPlacementHarness(t)
	slot := firstNonBlankSlot(gs)
	if slot < 0 {
		t.Skip("no non-blank tile in rack")
	}
	gs.onRackTap(slot)
	if gs.rackSelected != slot {
		t.Fatalf("after rack tap, rackSelected=%d want %d", gs.rackSelected, slot)
	}
	gs.onBoardTap(7, 7)
	if len(gs.staged) != 1 {
		t.Fatalf("placement failed: staged=%d want 1", len(gs.staged))
	}
}

// TestRepro_PlaceTwo: place two tiles in a row.
func TestRepro_PlaceTwo(t *testing.T) {
	gs := newPlacementHarness(t)
	tiles := gs.state.HumanRack.Tiles()
	placed := 0
	col := 7
	for i := range tiles {
		if tiles[i].IsBlank {
			continue
		}
		gs.onRackTap(i)
		gs.onBoardTap(7, col)
		col++
		placed++
		if placed == 2 {
			break
		}
	}
	if len(gs.staged) != 2 {
		t.Fatalf("two placements: staged=%d want 2", len(gs.staged))
	}
}

// TestRepro_PlaceAfterAITurn: simulate an AI turn (the thinking flag set, then the
// move applied) and verify the human can place again afterward.
func TestRepro_PlaceAfterAITurn(t *testing.T) {
	gs := newPlacementHarness(t)

	// Mimic startAITurn's flag and an AI reply landing via applyAIMove.
	gs.state.CurrentTurn = engine.AITurn
	gs.aiThinking = true
	gs.refresh()

	// While the AI is "thinking", placement must be blocked.
	slot := firstNonBlankSlot(gs)
	gs.onRackTap(slot)
	gs.onBoardTap(7, 7)
	if len(gs.staged) != 0 {
		t.Fatalf("placement should be blocked during AI turn, staged=%d", len(gs.staged))
	}

	// AI replies with a pass; turn returns to the human.
	gs.applyAIMove(engine.PassMove{}, false)
	if gs.aiThinking {
		t.Fatal("aiThinking still set after applyAIMove")
	}
	if gs.state.CurrentTurn != engine.HumanTurn {
		t.Fatalf("after AI pass, CurrentTurn=%v want HumanTurn", gs.state.CurrentTurn)
	}

	// Now placement must work.
	slot = firstNonBlankSlot(gs)
	gs.onRackTap(slot)
	gs.onBoardTap(7, 7)
	if len(gs.staged) != 1 {
		t.Fatalf("placement after AI turn failed: staged=%d want 1", len(gs.staged))
	}
}
