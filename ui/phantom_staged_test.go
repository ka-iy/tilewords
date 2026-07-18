package ui

import (
	"testing"

	"tilewords/engine"
)

// TestApplyAIMove_ClearsStaleStaged verifies the human's turn always begins with no
// staged tiles: a stale staged entry lingering when the AI's move is applied (which
// would otherwise blank a rack slot and make the rack look short, with the recall
// button wrongly enabled) is cleared.
func TestApplyAIMove_ClearsStaleStaged(t *testing.T) {
	gs := newPlacementHarness(t)
	stageOneTile(t, gs, 7, 7)
	if len(gs.staged) == 0 {
		t.Fatal("setup: expected a staged tile")
	}

	// Simulate that it is now the AI's turn (as if the human had moved) and the AI
	// replies with a pass, handing the turn back to the human.
	gs.state.CurrentTurn = engine.AITurn
	gs.aiThinking = true
	gs.applyAIMove(engine.PassMove{}, false)

	if gs.state.CurrentTurn != engine.HumanTurn {
		t.Fatalf("after AI pass, turn = %v, want HumanTurn", gs.state.CurrentTurn)
	}
	if len(gs.staged) != 0 {
		t.Fatalf("the human's turn began with %d stale staged tile(s); the rack would look short", len(gs.staged))
	}
}

// TestStageTile_IgnoresAlreadyStagedSlot verifies staging a rack slot that is already
// staged is a no-op, so no duplicate FromRackIdx (phantom) can be created.
func TestStageTile_IgnoresAlreadyStagedSlot(t *testing.T) {
	gs := newPlacementHarness(t)
	slot := firstNonBlankSlot(gs)
	if slot < 0 {
		t.Skip("no non-blank rack tile")
	}

	gs.stageTile(slot, 7, 7)
	if len(gs.staged) != 1 {
		t.Fatalf("first stage: staged=%d want 1", len(gs.staged))
	}
	gs.stageTile(slot, 7, 8) // same slot again
	if len(gs.staged) != 1 {
		t.Fatalf("staging an already-staged slot created a duplicate: staged=%d want 1", len(gs.staged))
	}
}
