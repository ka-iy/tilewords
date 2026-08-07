// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math/rand/v2"
	"testing"

	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

// playCAT commits CAT horizontally across row 7 (cols 6–8) and logs it, returning the newest
// history line. scrabbleNotation selects the format.
func playCATLine(t *testing.T, scrabbleNotation bool) string {
	t.Helper()
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewPCG(1, 0)))
	state.CurrentTurn = engine.HumanTurn
	state.ScrabbleNotation = scrabbleNotation
	gs := newGameScreen(nil, state, dict)
	gs.build()

	placed := []engine.PlacedTile{
		{Tile: engine.Tile{Letter: 'C', Points: 3}, Row: 7, Col: 6},
		{Tile: engine.Tile{Letter: 'A', Points: 1}, Row: 7, Col: 7},
		{Tile: engine.Tile{Letter: 'T', Points: 1}, Row: 7, Col: 8},
	}
	for _, pt := range placed {
		if err := state.Board.Place(pt.Row, pt.Col, pt.Tile); err != nil {
			t.Fatal(err)
		}
	}
	gs.logCommand("You", &engine.PlayCommand{Move: engine.PlayMove{
		Placed: placed, WordsFormed: []string{"CAT"}, Score: 5,
	}})
	return gs.history[len(gs.history)-1].line
}

// TestHistoryLine_PlainVsNotation verifies the move-history line uses the plain word list by
// default and Scrabble coordinate notation when the option is enabled.
func TestHistoryLine_PlainVsNotation(t *testing.T) {
	if got, want := playCATLine(t, false), "You: CAT (+5)"; got != want {
		t.Errorf("plain history line = %q, want %q", got, want)
	}
	// Row index 7 -> row number 8; column index 6 -> G; horizontal -> number then letter.
	if got, want := playCATLine(t, true), "You: 8G CAT +5"; got != want {
		t.Errorf("notation history line = %q, want %q", got, want)
	}
}

// TestHistoryLine_NotationListsCrossWords verifies the notation history line lists the main
// word and any perpendicular cross-words, with existing letters parenthesised.
func TestHistoryLine_NotationListsCrossWords(t *testing.T) {
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "ST"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewPCG(1, 0)))
	state.CurrentTurn = engine.HumanTurn
	state.ScrabbleNotation = true
	gs := newGameScreen(nil, state, dict)
	gs.build()

	// Existing S above; play CAT across so the T forms the cross-word ST.
	if err := state.Board.Place(6, 7, engine.Tile{Letter: 'S', Points: 1}); err != nil {
		t.Fatal(err)
	}
	placed := []engine.PlacedTile{
		{Tile: engine.Tile{Letter: 'C', Points: 3}, Row: 7, Col: 5},
		{Tile: engine.Tile{Letter: 'A', Points: 1}, Row: 7, Col: 6},
		{Tile: engine.Tile{Letter: 'T', Points: 1}, Row: 7, Col: 7},
	}
	for _, pt := range placed {
		if err := state.Board.Place(pt.Row, pt.Col, pt.Tile); err != nil {
			t.Fatal(err)
		}
	}
	gs.logCommand("You", &engine.PlayCommand{Move: engine.PlayMove{
		Placed: placed, WordsFormed: []string{"CAT", "ST"}, Score: 9,
	}})
	if got, want := gs.history[len(gs.history)-1].line, "You: 8F CAT, H7 (S)T +9"; got != want {
		t.Errorf("notation line = %q, want %q", got, want)
	}
}
