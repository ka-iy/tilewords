// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"reflect"
	"testing"
)

// TestRecomputeCPUHighlight verifies that the red-border highlight tracks the cells
// of the CPU's most recent play still on the board, derived purely from history.
func TestRecomputeCPUHighlight(t *testing.T) {
	// The highlight is derived from each entry's placed cells (as populated by logCommand).
	play := func(player string, cells ...[2]int) historyEntry {
		return historyEntry{player: player, cells: cells}
	}
	cpuPass := historyEntry{player: "CPU"} // a pass places no cells

	cases := []struct {
		name    string
		history []historyEntry
		want    map[[2]int]bool
	}{
		{"empty history", nil, nil},
		{
			"single CPU play highlights its cells",
			[]historyEntry{play("CPU", [2]int{7, 7}, [2]int{7, 8})},
			map[[2]int]bool{{7, 7}: true, {7, 8}: true},
		},
		{
			"human play does not change the CPU highlight",
			[]historyEntry{play("CPU", [2]int{7, 7}), play("You", [2]int{8, 7})},
			map[[2]int]bool{{7, 7}: true},
		},
		{
			"CPU pass keeps the earlier CPU word highlighted",
			[]historyEntry{play("CPU", [2]int{7, 7}), play("You", [2]int{8, 7}), cpuPass},
			map[[2]int]bool{{7, 7}: true},
		},
		{
			"most recent CPU play wins",
			[]historyEntry{play("CPU", [2]int{7, 7}), play("CPU", [2]int{0, 0}, [2]int{0, 1})},
			map[[2]int]bool{{0, 0}: true, {0, 1}: true},
		},
		{"only a CPU pass highlights nothing", []historyEntry{cpuPass}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gs := &gameScreen{history: tc.history}
			gs.recomputeCPUHighlight()
			if !reflect.DeepEqual(gs.cpuLastPlaced, tc.want) {
				t.Errorf("cpuLastPlaced = %v, want %v", gs.cpuLastPlaced, tc.want)
			}
		})
	}
}
