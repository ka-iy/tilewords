package ui

import (
	"math/rand"
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
	state := engine.New(dict.Name(), 5, rand.New(rand.NewSource(1)))
	state.OpeningDraw = od
	state.CurrentTurn = engine.HumanTurn
	gs := newGameScreen(nil, state, dict)
	gs.build()
	return gs
}

// TestHistory_ShowsOpeningDraw verifies the opening-draw result is the first move-history
// line from game start and stays at the top after moves are logged.
func TestHistory_ShowsOpeningDraw(t *testing.T) {
	gs := newOpeningHarness(t, &engine.OpeningDraw{HumanLetter: 'C', AILetter: 'T', First: engine.HumanTurn})
	want := "Opening draw: you drew C, AI drew T — you go first"

	if got := gs.historyLabel.Text; got != want {
		t.Errorf("history at start = %q, want %q", got, want)
	}

	gs.logCommand("You", &engine.PlayCommand{Move: engine.PlayMove{WordsFormed: []string{"CAT"}, Score: 5}})
	lines := strings.Split(gs.historyLabel.Text, "\n")
	if len(lines) != 2 || lines[0] != want {
		t.Errorf("after a move, history = %v; want opening line first then the move", lines)
	}
}

// TestOpeningDrawLine covers the AI-first wording and the blank rendering.
func TestOpeningDrawLine(t *testing.T) {
	got := openingDrawLine(&engine.OpeningDraw{HumanLetter: 0, AILetter: 'Q', First: engine.AITurn})
	want := "Opening draw: you drew (blank), AI drew Q — AI goes first"
	if got != want {
		t.Errorf("openingDrawLine = %q, want %q", got, want)
	}
	if openingDrawLine(nil) != "" {
		t.Error("openingDrawLine(nil) must be empty")
	}
}
