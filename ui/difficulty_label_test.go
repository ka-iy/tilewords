package ui

import (
	"strings"
	"testing"

	"tilewords/ai"
)

// TestDifficultyLabelText verifies the slider caption names the top level rather than only
// numbering it, and keeps the easy/hard hint at every other level.
func TestDifficultyLabelText(t *testing.T) {
	god := difficultyLabelText(ai.GodModeLevel)
	if !strings.Contains(god, "GOD MODE") {
		t.Errorf("label at level %d = %q, want it to name GOD MODE", ai.GodModeLevel, god)
	}
	if !strings.Contains(god, "11") {
		t.Errorf("label at level %d = %q, want it to show the level number", ai.GodModeLevel, god)
	}

	// The hint text lists the range's endpoints, so a player can see god mode exists before
	// dragging the slider all the way over.
	mid := difficultyLabelText(defaultDifficulty)
	for _, want := range []string{"3", "easy", "hard", "GOD MODE"} {
		if !strings.Contains(mid, want) {
			t.Errorf("label at level %d = %q, want it to mention %q", defaultDifficulty, mid, want)
		}
	}
}

// TestDefaultDifficultyIsInRange guards the default against drifting outside the levels the AI
// accepts, which sanitize would then silently replace.
func TestDefaultDifficultyIsInRange(t *testing.T) {
	if defaultDifficulty < ai.MinLevel || defaultDifficulty > ai.MaxLevel {
		t.Fatalf("defaultDifficulty = %d, outside the AI's range [%d,%d]",
			defaultDifficulty, ai.MinLevel, ai.MaxLevel)
	}
	// A default at the top of the range would start every new player in god mode.
	if defaultDifficulty >= ai.NearBestLevel {
		t.Errorf("defaultDifficulty = %d, too high for a first game", defaultDifficulty)
	}
}
