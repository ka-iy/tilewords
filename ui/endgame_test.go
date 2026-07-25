// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"strings"
	"testing"
)

// TestCheckEndGame_StaysOnScreen: when the game ends, the screen stays up (not
// abandoned), the endgame is scored, input is disabled, the rack label shows a red
// GAME OVER, and the Menu button stays usable while play buttons are disabled.
func TestCheckEndGame_StaysOnScreen(t *testing.T) {
	gs := newPlacementHarness(t)
	gs.state.ConsecutivePasses = 6 // six-pass end condition

	if !gs.checkEndGame() {
		t.Fatal("checkEndGame should report the game over")
	}
	if !gs.gameOver {
		t.Fatal("gameOver flag not set")
	}
	if gs.abandoned {
		t.Fatal("ending the game must not abandon/leave the screen")
	}
	if !gs.state.EndgameScored {
		t.Fatal("endgame scoring was not applied")
	}
	if gs.humanInputAllowed() {
		t.Fatal("input should be blocked after the game is over")
	}
	if gs.rackLabel.Text != "GAME OVER" {
		t.Fatalf("rack label = %q, want %q", gs.rackLabel.Text, "GAME OVER")
	}
	if gs.rackLabel.Color != colorGameOver {
		t.Errorf("rack label colour = %v, want red", gs.rackLabel.Color)
	}
	if gs.menuBtn.Disabled() {
		t.Error("Menu button should stay enabled after game over")
	}
	if !gs.playBtn.Disabled() {
		t.Error("Play button should be disabled after game over")
	}
}

// TestEndGameMessage reports the winner from the final scores.
func TestEndGameMessage(t *testing.T) {
	gs := newPlacementHarness(t)
	cases := []struct {
		human, ai int
		want      string
	}{
		{200, 150, "You win!"},
		{150, 200, "AI wins!"},
		{175, 175, "tie"},
	}
	for _, c := range cases {
		gs.state.HumanScore, gs.state.AIScore = c.human, c.ai
		if got := gs.endGameMessage(); !strings.Contains(got, c.want) {
			t.Errorf("endGameMessage(you=%d, ai=%d) = %q, want it to contain %q", c.human, c.ai, got, c.want)
		}
	}
}

// TestDifficultyShownInTopBar: the AI difficulty appears in the top counters row.
func TestDifficultyShownInTopBar(t *testing.T) {
	gs := newPlacementHarness(t) // built with level 5
	if !strings.Contains(gs.levelLabel.Text, "5") {
		t.Errorf("level label = %q, want it to show level 5", gs.levelLabel.Text)
	}
}
