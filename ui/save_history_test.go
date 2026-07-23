package ui

import (
	"math/rand"
	"strings"
	"testing"

	"fyne.io/fyne/v2/test"

	"tilewords/dictionary"
	"tilewords/engine"
)

// TestSaveManager_PersistsMoveHistory verifies that the move history — and the status
// summary and AI-word highlight derived from it — survive a save/load round trip, while the
// restored moves are not undoable.
func TestSaveManager_PersistsMoveHistory(t *testing.T) {
	_ = test.NewApp()
	dict, err := dictionary.NewFromWords("test", []string{"CAT"})
	if err != nil {
		t.Fatal(err)
	}
	state := engine.New(dict.Name(), 5, rand.New(rand.NewSource(1)))
	state.OpeningDraw = &engine.OpeningDraw{HumanLetter: 'C', AILetter: 'T', First: engine.HumanTurn}
	state.CurrentTurn = engine.HumanTurn
	gs := newGameScreen(nil, state, dict)
	gs.build()

	gs.logCommand("You", &engine.PlayCommand{Move: engine.PlayMove{
		Placed:      []engine.PlacedTile{{Tile: engine.Tile{Letter: 'C'}, Row: 7, Col: 7}},
		WordsFormed: []string{"CAT"}, Score: 5,
	}})
	gs.logCommand("AI", &engine.PlayCommand{Move: engine.PlayMove{
		Placed:      []engine.PlacedTile{{Tile: engine.Tile{Letter: 'Q'}, Row: 8, Col: 8}},
		WordsFormed: []string{"QI"}, Score: 22,
	}})

	// Save (as doSave does) and load into a fresh screen.
	dir := t.TempDir()
	sm, err := NewSaveManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	gs.state.History = gs.moveRecords()
	if err := sm.Save(gs.state); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	gs2 := newGameScreen(nil, loaded, dict)
	gs2.build()

	// History lines restored verbatim.
	if len(gs2.history) != 2 {
		t.Fatalf("restored history length = %d, want 2", len(gs2.history))
	}
	if gs2.history[0].line != gs.history[0].line || gs2.history[1].line != gs.history[1].line {
		t.Errorf("restored lines = %q,%q; want %q,%q",
			gs2.history[0].line, gs2.history[1].line, gs.history[0].line, gs.history[1].line)
	}

	// Words restored so the definitions panel can repopulate on load.
	if got := gs2.history[0].words; len(got) != 1 || got[0] != "CAT" {
		t.Errorf("restored words[0] = %v, want [CAT]", got)
	}
	if got := gs2.history[1].words; len(got) != 1 || got[0] != "QI" {
		t.Errorf("restored words[1] = %v, want [QI]", got)
	}

	// Status summary derived from the restored history.
	if gs2.lastHumanPts != 5 || gs2.lastAIPts != 22 {
		t.Errorf("restored points = You %d / AI %d, want 5 / 22", gs2.lastHumanPts, gs2.lastAIPts)
	}

	// AI-word highlight restored from the stored cells.
	if !gs2.aiLastPlaced[[2]int{8, 8}] {
		t.Errorf("AI highlight not restored: %v", gs2.aiLastPlaced)
	}

	// Restored entries carry no command, so they are not undoable.
	if gs2.canUndo() {
		t.Error("restored move history must not be undoable")
	}

	// The panel shows the opening-draw line followed by the two moves.
	if lines := strings.Split(gs2.historyLabel.Text, "\n"); len(lines) != 3 {
		t.Errorf("history panel = %d lines, want 3 (opening + 2 moves): %q", len(lines), lines)
	}
}
