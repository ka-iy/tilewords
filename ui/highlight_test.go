// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"reflect"
	"testing"
)

// TestRecomputeAIHighlight verifies that the red-border highlight tracks the cells
// of the AI's most recent play still on the board, derived purely from history.
func TestRecomputeAIHighlight(t *testing.T) {
	// The highlight is derived from each entry's placed cells (as populated by logCommand).
	play := func(player string, cells ...[2]int) historyEntry {
		return historyEntry{player: player, cells: cells}
	}
	aiPass := historyEntry{player: "AI"} // a pass places no cells

	cases := []struct {
		name    string
		history []historyEntry
		want    map[[2]int]bool
	}{
		{"empty history", nil, nil},
		{
			"single AI play highlights its cells",
			[]historyEntry{play("AI", [2]int{7, 7}, [2]int{7, 8})},
			map[[2]int]bool{{7, 7}: true, {7, 8}: true},
		},
		{
			"human play does not change the AI highlight",
			[]historyEntry{play("AI", [2]int{7, 7}), play("You", [2]int{8, 7})},
			map[[2]int]bool{{7, 7}: true},
		},
		{
			"AI pass keeps the earlier AI word highlighted",
			[]historyEntry{play("AI", [2]int{7, 7}), play("You", [2]int{8, 7}), aiPass},
			map[[2]int]bool{{7, 7}: true},
		},
		{
			"most recent AI play wins",
			[]historyEntry{play("AI", [2]int{7, 7}), play("AI", [2]int{0, 0}, [2]int{0, 1})},
			map[[2]int]bool{{0, 0}: true, {0, 1}: true},
		},
		{"only an AI pass highlights nothing", []historyEntry{aiPass}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gs := &gameScreen{history: tc.history}
			gs.recomputeAIHighlight()
			if !reflect.DeepEqual(gs.aiLastPlaced, tc.want) {
				t.Errorf("aiLastPlaced = %v, want %v", gs.aiLastPlaced, tc.want)
			}
		})
	}
}
