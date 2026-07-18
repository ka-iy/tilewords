package ui

import (
	"math/rand"
	"testing"

	"tilewords/engine"
)

// TestSaveManager_PersistsScrabbleNotation verifies the move-history format preference
// survives a save/load round trip, so a resumed game keeps the chosen notation.
func TestSaveManager_PersistsScrabbleNotation(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewSaveManager(dir)
	if err != nil {
		t.Fatalf("NewSaveManager: %v", err)
	}

	state := engine.New("csw", 5, rand.New(rand.NewSource(42)))
	state.ScrabbleNotation = true
	if err := sm.Save(state); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !loaded.ScrabbleNotation {
		t.Error("ScrabbleNotation did not persist across save/load")
	}
}
