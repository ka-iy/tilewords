package ui

import (
	"testing"

	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"squabble/engine"
)

// segText returns a RichText segment's text and colour name.
func segText(t *testing.T, seg widget.RichTextSegment) (string, string) {
	t.Helper()
	ts, ok := seg.(*widget.TextSegment)
	if !ok {
		t.Fatalf("segment is not a *TextSegment: %T", seg)
	}
	return ts.Text, string(ts.Style.ColorName)
}

// TestStatusSummary_ShowsBothPlayerPoints verifies the status line shows each player's
// most-recent-move points, prefixed with "You"/"AI" — the human's in green (Success), the
// AI's in amber (Warning), separated by " / " — that a play clears a transient message,
// and that a pass or exchange counts as +0.
func TestStatusSummary_ShowsBothPlayerPoints(t *testing.T) {
	gs := newRackHarness(t)

	// Before any move: an em dash for each, still green / amber.
	segs := gs.statusSegments()
	if len(segs) != 3 {
		t.Fatalf("initial: got %d segments, want 3", len(segs))
	}
	if txt, col := segText(t, segs[0]); txt != "You —" || col != string(theme.ColorNameSuccess) {
		t.Errorf("initial human segment = %q/%s, want \"You —\"/success", txt, col)
	}
	if txt, col := segText(t, segs[2]); txt != "AI —" || col != string(theme.ColorNameWarning) {
		t.Errorf("initial AI segment = %q/%s, want \"AI —\"/warning", txt, col)
	}

	// A stale transient message is present, then the human plays.
	gs.setStatus("Game saved.", false)
	gs.logCommand("You", &engine.PlayCommand{Move: engine.PlayMove{WordsFormed: []string{"UNMIX"}, Score: 28}})
	if gs.statusMsg != "" {
		t.Errorf("a play must clear the transient message, got %q", gs.statusMsg)
	}
	gs.logCommand("AI", &engine.PlayCommand{Move: engine.PlayMove{WordsFormed: []string{"QI"}, Score: 22}})

	segs = gs.statusSegments()
	if txt, col := segText(t, segs[0]); txt != "You +28" || col != string(theme.ColorNameSuccess) {
		t.Errorf("human segment = %q/%s, want \"You +28\"/success", txt, col)
	}
	if txt, _ := segText(t, segs[1]); txt != " / " {
		t.Errorf("separator = %q, want %q", txt, " / ")
	}
	if txt, col := segText(t, segs[2]); txt != "AI +22" || col != string(theme.ColorNameWarning) {
		t.Errorf("AI segment = %q/%s, want \"AI +22\"/warning", txt, col)
	}

	// A pass counts as +0 for the human; the AI's last play stands.
	gs.logCommand("You", &engine.PassCommand{})
	segs = gs.statusSegments()
	if txt, _ := segText(t, segs[0]); txt != "You +0" {
		t.Errorf("after pass, human segment = %q, want \"You +0\"", txt)
	}
	if txt, _ := segText(t, segs[2]); txt != "AI +22" {
		t.Errorf("after human pass, AI segment = %q, want \"AI +22\" (unchanged)", txt)
	}

	// An exchange also counts as +0.
	gs.logCommand("AI", &engine.ExchangeCommand{Move: engine.ExchangeMove{Tiles: []engine.Tile{{Letter: 'A'}, {Letter: 'B'}}}})
	if txt, _ := segText(t, gs.statusSegments()[2]); txt != "AI +0" {
		t.Errorf("after AI exchange, AI segment = %q, want \"AI +0\"", txt)
	}
}

// TestStatusSummary_ThinkingAndErrorOverride verifies AI-thinking and error messages take
// precedence over the score summary.
func TestStatusSummary_ThinkingAndErrorOverride(t *testing.T) {
	gs := newRackHarness(t)
	gs.logCommand("You", &engine.PlayCommand{Move: engine.PlayMove{WordsFormed: []string{"CAT"}, Score: 10}})

	gs.aiThinking = true
	if segs := gs.statusSegments(); len(segs) != 1 {
		t.Errorf("while thinking: got %d segments, want 1 (the notice)", len(segs))
	}
	gs.aiThinking = false

	gs.setStatus("Place at least one tile before playing.", true)
	segs := gs.statusSegments()
	if len(segs) != 1 {
		t.Fatalf("on error: got %d segments, want 1", len(segs))
	}
	if _, col := segText(t, segs[0]); col != string(theme.ColorNameError) {
		t.Errorf("error segment colour = %s, want error", col)
	}
}

// TestRecomputeLastPoints_FromHistory verifies the points are derived from the move history
// (so removing the last entry, as undo does, restores the prior move's points).
func TestRecomputeLastPoints_FromHistory(t *testing.T) {
	gs := newRackHarness(t)

	gs.recomputeLastPoints()
	if gs.lastHumanPts != -1 || gs.lastAIPts != -1 {
		t.Fatalf("empty history: got human=%d ai=%d, want -1/-1", gs.lastHumanPts, gs.lastAIPts)
	}

	gs.history = []historyEntry{
		{cmd: &engine.PlayCommand{Move: engine.PlayMove{Score: 30}}, player: "You"},
		{cmd: &engine.PlayCommand{Move: engine.PlayMove{Score: 15}}, player: "AI"},
		{cmd: &engine.PassCommand{}, player: "You"},
	}
	gs.recomputeLastPoints()
	if gs.lastHumanPts != 0 || gs.lastAIPts != 15 {
		t.Errorf("after pass: got human=%d ai=%d, want human=0 (pass) ai=15", gs.lastHumanPts, gs.lastAIPts)
	}

	// Undo drops the pass; the human's prior play (30) is restored.
	gs.history = gs.history[:2]
	gs.recomputeLastPoints()
	if gs.lastHumanPts != 30 {
		t.Errorf("after undo of the pass: human=%d, want 30", gs.lastHumanPts)
	}
}

// TestHistory_IsCopyable verifies the move-history label is selectable so its text can be
// copied.
func TestHistory_IsCopyable(t *testing.T) {
	gs := newRackHarness(t)
	if !gs.historyLabel.Selectable {
		t.Error("move-history label is not Selectable, so it cannot be copied")
	}
}
