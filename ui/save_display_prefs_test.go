// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"math/rand/v2"
	"testing"

	"tilewords/engine"
)

// TestSaveManager_PersistsDisplayPrefs verifies the display preferences survive a save/load
// round trip, so a resumed game keeps the chosen move-history format and board headers.
func TestSaveManager_PersistsDisplayPrefs(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSaveManager(dir)
	if err != nil {
		t.Fatalf("NewSaveManager: %v", err)
	}

	state := engine.New("csw", 5, rand.New(rand.NewPCG(42, 0)))
	state.OfficialNotation = true
	state.BoardHeaders = true
	if err := sm.Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.OfficialNotation {
		t.Error("OfficialNotation did not persist across save/load")
	}
	if !loaded.BoardHeaders {
		t.Error("BoardHeaders did not persist across save/load")
	}
}
