// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math/rand/v2"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

// newHeadersHarness returns a built game screen whose game shows the board's row and
// column headers (or not), sized to a phone viewport.
func newHeadersHarness(t *testing.T, headers bool) *gameScreen {
	t.Helper()
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "DOG", "AT", "TO", "GO"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewPCG(1, 0)))
	state.BoardHeaders = headers
	gs := newGameScreen(nil, state, dict)
	content := gs.build()
	content.Resize(phoneSize)
	return gs
}

// boardLabelTexts returns the canvas.Text children the board carries past its cells —
// the row/column headers.
func boardLabelTexts(board *fyne.Container) []*canvas.Text {
	var out []*canvas.Text
	for _, o := range board.Objects[boardDim*boardDim:] {
		if t, ok := o.(*canvas.Text); ok {
			out = append(out, t)
		}
	}
	return out
}

// The board carries its row and column headers when the player asked for them.
func TestBoardHeaders_ShownWhenAskedFor(t *testing.T) {
	gs := newHeadersHarness(t, true)

	if got, want := len(boardLabelTexts(gs.boardBox)), 2*boardDim; got != want {
		t.Errorf("board headers: got %d want %d (%d columns + %d rows)", got, want, boardDim, boardDim)
	}
	if got := len(gs.boardLabels); got != 2*boardDim {
		t.Errorf("retained headers: got %d want %d — the theme recolour would miss them", got, 2*boardDim)
	}
}

// Unasked for, the headers are absent and the board reserves no gutter for them: the
// layout and the hit test must both work off the same unlabelled geometry, otherwise a tap
// lands on a different cell than the one drawn under the finger.
func TestBoardHeaders_HiddenWhenNotAskedFor(t *testing.T) {
	gs := newHeadersHarness(t, false)

	if got := len(gs.boardBox.Objects); got != boardDim*boardDim {
		t.Errorf("board children: got %d want %d (the cells alone)", got, boardDim*boardDim)
	}
	if got := len(gs.boardLabels); got != 0 {
		t.Errorf("retained headers: got %d want 0", got)
	}
	l, ok := gs.boardBox.Layout.(boardLayout)
	if !ok {
		t.Fatalf("board layout: got %T want boardLayout", gs.boardBox.Layout)
	}
	if l.labelled {
		t.Error("the board reserves a label gutter even though it has no labels")
	}
}

// Whichever way the headers are set, a tap at the grid origin reported by the board's own
// geometry lands on the top-left cell.
func TestBoardHitTest_MatchesLabelledState(t *testing.T) {
	for _, headers := range []bool{true, false} {
		gs := newHeadersHarness(t, headers)
		size := gs.boardBox.Size()
		_, offX, offY := boardGeometry(size.Width, size.Height, headers)
		row, col, ok := cellAtRel(fyne.NewPos(offX, offY), size, headers)
		if !ok || row != 0 || col != 0 {
			t.Errorf("headers=%v: grid origin maps to (%d,%d,%v), want (0,0,true)", headers, row, col, ok)
		}
	}
}

// The two display options are independent: the move-history format does not decide whether
// the board shows its row and column headers.
func TestBoardHeaders_IndependentOfNotation(t *testing.T) {
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT", "DOG", "AT", "TO", "GO"})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		notation, headers bool
		wantLabels        int
	}{
		{notation: true, headers: false, wantLabels: 0},
		{notation: false, headers: true, wantLabels: 2 * boardDim},
	}
	for _, c := range cases {
		state := engine.New(dict.Name(), 5, rand.New(rand.NewPCG(1, 0)))
		state.OfficialNotation = c.notation
		state.BoardHeaders = c.headers
		gs := newGameScreen(nil, state, dict)
		gs.build().Resize(phoneSize)
		if got := len(boardLabelTexts(gs.boardBox)); got != c.wantLabels {
			t.Errorf("notation=%v headers=%v: board headers = %d, want %d",
				c.notation, c.headers, got, c.wantLabels)
		}
	}
}
