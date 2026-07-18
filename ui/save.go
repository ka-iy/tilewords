// Package ui is documented in doc.go.
package ui

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tilewords/engine"
)

// SaveManager persists and restores engine.GameState to a single save slot.
// The save file is written atomically (temp file → rename) to prevent corruption
// on process death mid-write — Pattern 3 / NFR-UI-R3.
//
// Use NewSaveManager("") for production (resolves os.UserConfigDir/tilewords/).
// Inject a temp directory in tests to avoid touching the real config directory (NFR-UI-TEST-1).
type SaveManager struct {
	path string // absolute path to savegame.gob
}

// NewSaveManager constructs a SaveManager. If configRoot is empty, it resolves to
// os.UserConfigDir()/tilewords/savegame.gob. Otherwise configRoot/tilewords/savegame.gob
// is used (enables test injection without touching the real config directory).
func NewSaveManager(configRoot string) (*SaveManager, error) {
	if configRoot == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("ui.NewSaveManager: cannot determine user config directory: %w", err)
		}
		configRoot = dir
	}
	return &SaveManager{
		path: filepath.Join(configRoot, "tilewords", "savegame.gob"),
	}, nil
}

// Save encodes state to the save file atomically. The parent directory is created if
// it does not exist (NFR-UI-R5). File permissions: directory 0700, file 0600 (SECURITY-UI-1).
func (sm *SaveManager) Save(state *engine.GameState) error {
	dir := filepath.Dir(sm.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("ui.SaveManager.Save: mkdir: %w", err)
	}

	// Write to a sibling temp file first; rename for atomicity (POSIX rename(2) is atomic).
	tmp := sm.path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("ui.SaveManager.Save: open temp: %w", err)
	}

	// LastHumanCommand and LastAICommand are Command interface fields. Gob
	// cannot encode non-nil interface values without gob.Register for each
	// concrete type. Per the design intent, undo history is not persisted:
	// encode a shallow copy with those fields zeroed.
	saveable := *state
	saveable.LastHumanCommand = nil
	saveable.LastAICommand = nil

	encErr := gob.NewEncoder(f).Encode(&saveable)
	closeErr := f.Close()
	if encErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("ui.SaveManager.Save: encode: %w", encErr)
	}
	if closeErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("ui.SaveManager.Save: close: %w", closeErr)
	}

	if err := os.Rename(tmp, sm.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("ui.SaveManager.Save: rename: %w", err)
	}
	return nil
}

// Load decodes and returns the saved GameState. Returns an error if the save file is
// missing, corrupt, or produces a nil state (SECURITY-UI-3 / NFR-UI-R2).
func (sm *SaveManager) Load() (*engine.GameState, error) {
	f, err := os.Open(sm.path)
	if err != nil {
		return nil, fmt.Errorf("ui.SaveManager.Load: open: %w", err)
	}
	defer f.Close()

	var state engine.GameState
	if err := gob.NewDecoder(f).Decode(&state); err != nil {
		return nil, fmt.Errorf("ui.SaveManager.Load: decode: %w", err)
	}
	// Defensive nil-pointer guard: gob may succeed but leave fields zeroed.
	if state.Board == nil || state.HumanRack == nil || state.AIRack == nil || state.Bag == nil {
		return nil, fmt.Errorf("ui.SaveManager.Load: corrupt save file (nil game fields)")
	}
	return &state, nil
}

// Exists reports whether a save file is present. Used to enable/disable the Load button
// on MainMenuScreen.
func (sm *SaveManager) Exists() bool {
	_, err := os.Stat(sm.path)
	return err == nil
}

// Delete removes the save file. A missing file is not an error (the slot is already
// empty), so Delete is idempotent; any other removal failure is reported.
func (sm *SaveManager) Delete() error {
	if err := os.Remove(sm.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("ui.SaveManager.Delete: remove: %w", err)
	}
	return nil
}

// sanitiseError converts an error to a short user-facing string. It strips any
// function-name prefix and truncates at 120 characters so that internal paths,
// type names, and Go error chains are never shown in the UI (SECURITY-UI-2).
func sanitiseError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Strip everything up to and including the last ": " (function-name prefix).
	for {
		_, after, found := strings.Cut(msg, ": ")
		if !found {
			break
		}
		msg = after
	}
	if len(msg) > 120 {
		msg = msg[:120] + "…"
	}
	if msg == "" {
		msg = "unknown error"
	}
	return msg
}
