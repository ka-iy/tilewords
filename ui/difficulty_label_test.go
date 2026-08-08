// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"strings"
	"testing"

	"tilewords/cpu"
)

// TestDifficultyLabelText verifies the slider caption names the top level rather than only
// numbering it, and keeps the easy/hard hint at every other level.
func TestDifficultyLabelText(t *testing.T) {
	demigod := difficultyLabelText(cpu.DemigodModeLevel)
	if !strings.Contains(demigod, "DEMIGOD MODE") {
		t.Errorf("label at level %d = %q, want it to name DEMIGOD MODE", cpu.DemigodModeLevel, demigod)
	}
	if !strings.Contains(demigod, "11") {
		t.Errorf("label at level %d = %q, want it to show the level number", cpu.DemigodModeLevel, demigod)
	}

	// The hint text lists the range's endpoints, so a player can see demigod mode exists before
	// dragging the slider all the way over.
	mid := difficultyLabelText(defaultDifficulty)
	for _, want := range []string{"3", "easy", "hard", "DEMIGOD MODE"} {
		if !strings.Contains(mid, want) {
			t.Errorf("label at level %d = %q, want it to mention %q", defaultDifficulty, mid, want)
		}
	}
}

// TestDefaultDifficultyIsInRange guards the default against drifting outside the levels the CPU
// accepts, which sanitize would then silently replace.
func TestDefaultDifficultyIsInRange(t *testing.T) {
	if defaultDifficulty < cpu.MinLevel || defaultDifficulty > cpu.MaxLevel {
		t.Fatalf("defaultDifficulty = %d, outside the CPU's range [%d,%d]",
			defaultDifficulty, cpu.MinLevel, cpu.MaxLevel)
	}
	// A default at the top of the range would start every new player in demigod mode.
	if defaultDifficulty >= cpu.NearBestLevel {
		t.Errorf("defaultDifficulty = %d, too high for a first game", defaultDifficulty)
	}
}
