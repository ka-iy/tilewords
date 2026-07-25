// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"
	"time"
)

// TestRegisterStagedPress covers the double-press detector shared by the tap path
// (onBoardTap) and the touch drag-release path (onBoardDragEnd).
func TestRegisterStagedPress(t *testing.T) {
	gs := newPlacementHarness(t)

	if gs.registerStagedPress(7, 7) {
		t.Fatal("first press on a cell reported as a double-press")
	}
	if !gs.registerStagedPress(7, 7) {
		t.Fatal("quick second press on the same cell was not a double-press")
	}
	// The detector resets after a double-press, so the next press is single again.
	if gs.registerStagedPress(7, 7) {
		t.Fatal("press immediately after a double-press should be single")
	}
	// A press on a different cell is never a double-press.
	if gs.registerStagedPress(7, 8) {
		t.Fatal("press on a different cell reported as a double-press")
	}
	// A second press outside the window is not a double-press.
	gs.lastPressAt = time.Now().Add(-time.Hour)
	if gs.registerStagedPress(7, 8) {
		t.Fatal("slow second press reported as a double-press")
	}
}

// TestDoublePress_TapPathRecalls verifies a double-press via the tap path returns the
// staged tile to the rack (single tap only picks it up).
func TestDoublePress_TapPathRecalls(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)

	gs.onBoardTap(7, 7) // first press → pick up, not recalled
	if _, ok := gs.stagedAt(7, 7); !ok {
		t.Fatal("first tap recalled the tile instead of picking it up")
	}
	gs.onBoardTap(7, 7) // quick second press → double-press → recall
	if _, ok := gs.stagedAt(7, 7); ok {
		t.Fatal("double-press did not return the staged tile to the rack")
	}
	if gs.pickedUp != [2]int{-1, -1} {
		t.Fatalf("pickedUp not cleared after a double-press recall: %v", gs.pickedUp)
	}
}
