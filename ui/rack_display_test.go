// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math/rand"
	"testing"

	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

// displayedRackCount mirrors refresh()'s rack-rendering rule: a slot shows a tile
// unless it is beyond the rack length, staged out, or the active drag source.
func displayedRackCount(gs *gameScreen) int {
	stagedOut := make(map[int]bool, len(gs.staged))
	for _, st := range gs.staged {
		stagedOut[st.FromRackIdx] = true
	}
	n := gs.state.HumanRack.Count()
	shown := 0
	for i := 0; i < engine.MaxRackSize; i++ {
		if i >= n || stagedOut[i] || i == gs.dragRackSrc {
			continue
		}
		shown++
	}
	return shown
}

// checkRackInvariant asserts that every tile in the engine rack is accounted for
// exactly once: either shown in a rack slot or staged on the board.
func checkRackInvariant(t *testing.T, gs *gameScreen, label string) {
	t.Helper()
	shown := displayedRackCount(gs)
	staged := len(gs.staged)
	total := gs.state.HumanRack.Count()
	if shown+staged != total {
		t.Errorf("%s: shown(%d) + staged(%d) = %d, want engine rack count %d",
			label, shown, staged, shown+staged, total)
	}
	// Staged FromRackIdx must be distinct and in range.
	seen := make(map[int]bool)
	for _, st := range gs.staged {
		if st.FromRackIdx < 0 || st.FromRackIdx >= total {
			t.Errorf("%s: staged FromRackIdx=%d out of range [0,%d)", label, st.FromRackIdx, total)
		}
		if seen[st.FromRackIdx] {
			t.Errorf("%s: duplicate staged FromRackIdx=%d (collision)", label, st.FromRackIdx)
		}
		seen[st.FromRackIdx] = true
	}
}

func newRackHarness(t *testing.T) *gameScreen {
	t.Helper()
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "DOG", "AT", "TO", "GO"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewSource(1)))
	state.CurrentTurn = engine.HumanTurn
	gs := newGameScreen(nil, state, dict)
	gs.build()
	return gs
}

// TestRackDisplay_StageThenReorder reproduces the reported rack discrepancy: stage
// two tiles onto the board, then reorder the rack, and verify the display still
// accounts for all seven tiles.
func TestRackDisplay_StageThenReorder(t *testing.T) {
	gs := newRackHarness(t)

	// Stage two rack tiles onto two empty board cells.
	gs.onRackTap(0)
	gs.onBoardTap(7, 7)
	gs.onRackTap(1)
	gs.onBoardTap(7, 8)
	checkRackInvariant(t, gs, "after staging 2 tiles")

	// Now reorder a non-staged rack tile to another slot while tiles are staged.
	gs.reorderRack(6, 2)
	checkRackInvariant(t, gs, "after reorder while 2 staged")

	// Stage a third tile after the reorder.
	slot := -1
	for i := 0; i < gs.state.HumanRack.Count(); i++ {
		if !gs.isStagedSlot(i) {
			slot = i
			break
		}
	}
	gs.onRackTap(slot)
	gs.onBoardTap(7, 9)
	checkRackInvariant(t, gs, "after staging a 3rd tile post-reorder")
}
