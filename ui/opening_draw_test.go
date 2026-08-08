// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math/rand/v2"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

func newOpeningHarness(t *testing.T, od *engine.OpeningDraw) *gameScreen {
	t.Helper()
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewPCG(1, 0)))
	state.OpeningDraw = od
	state.CurrentTurn = engine.HumanTurn
	gs := newGameScreen(nil, state, dict)
	gs.build()
	return gs
}

// TestHistory_ShowsOpeningDraw verifies the opening-draw result is the first move-history
// line from game start and stays at the top after moves are logged.
func TestHistory_ShowsOpeningDraw(t *testing.T) {
	gs := newOpeningHarness(t, &engine.OpeningDraw{HumanLetter: 'C', CPULetter: 'T', First: engine.HumanTurn})
	want := "Opening draw: you drew C, CPU drew T - you go first"

	if got := gs.historyLabel.Text; got != want {
		t.Errorf("history at start = %q, want %q", got, want)
	}

	gs.logCommand("You", &engine.PlayCommand{Move: engine.PlayMove{WordsFormed: []string{"CAT"}, Score: 5}})
	// Entries are separated by a blank line, so split on the separator rather than on "\n".
	entries := historyEntries(gs)
	if len(entries) != 2 || entries[0] != want {
		t.Errorf("after a move, history = %v; want opening line first then the move", entries)
	}
}

// historyEntries returns the move-history panel's entries, splitting on the blank line that
// separates them so a wrapped or multi-line entry still counts once.
func historyEntries(gs *gameScreen) []string {
	text := gs.historyLabel.Text
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n\n")
}

// TestOpeningDrawLine covers the CPU-first wording and the blank rendering.
func TestOpeningDrawLine(t *testing.T) {
	got := openingDrawLine(&engine.OpeningDraw{HumanLetter: 0, CPULetter: 'Q', First: engine.CPUTurn})
	want := "Opening draw: you drew (blank), CPU drew Q - CPU goes first"
	if got != want {
		t.Errorf("openingDrawLine = %q, want %q", got, want)
	}
	if openingDrawLine(nil) != "" {
		t.Error("openingDrawLine(nil) must be empty")
	}
}
