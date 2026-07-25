// Package ui is documented in doc.go.
package ui

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"tilewords/engine"
)

// SaveManager persists and restores engine.GameState to a single save slot.
// The save file is written atomically (temp file → fsync → rename) so neither process death
// nor power loss mid-write can corrupt it — Pattern 3 / NFR-UI-R3.
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

	// GameState carries no undo state, so there is nothing to strip before encoding: the
	// executed commands live with the UI's history log and are deliberately not persisted.
	encErr := gob.NewEncoder(f).Encode(state)
	// Flush the encoded bytes to the device before the rename publishes them. Without this
	// the rename can reach the disk while the data behind it has not, so a power loss just
	// after a save leaves a truncated file where the previous good save used to be.
	syncErr := error(nil)
	if encErr == nil {
		syncErr = f.Sync()
	}
	closeErr := f.Close()
	if encErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("ui.SaveManager.Save: encode: %w", encErr)
	}
	if syncErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("ui.SaveManager.Save: sync: %w", syncErr)
	}
	if closeErr != nil {
		os.Remove(tmp)
		return fmt.Errorf("ui.SaveManager.Save: close: %w", closeErr)
	}

	if err := os.Rename(tmp, sm.path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("ui.SaveManager.Save: rename: %w", err)
	}

	// Make the directory entry itself durable, so the rename survives a power loss too. A
	// failure here is not reported: the save is already written and renamed, so the game is
	// safe against everything except an abrupt power cut, and failing the save would be
	// more misleading than the residual risk. Not supported on every platform.
	if dh, err := os.Open(dir); err == nil {
		_ = dh.Sync()
		_ = dh.Close()
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
	// A save file is outside data: gob only reports what the encoding itself makes visible,
	// so check the game invariants before handing the state to code that assumes them.
	if err := engine.ValidateDecodedState(&state); err != nil {
		return nil, fmt.Errorf("ui.SaveManager.Load: corrupt save file: %w", err)
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

// sanitiseError converts an error to a short user-facing string. It strips the Go
// function-name prefixes that wrapping adds and truncates the result, so internal paths and
// type names are never shown in the UI (SECURITY-UI-2).
//
// Only leading segments that look like a qualified Go symbol are removed. Cutting at the
// last ": " instead would keep just the innermost segment, discarding the part of the chain
// written to be read — "asset not found; run 'make gaddag'" would arrive as
// "file does not exist", and unrelated failures would collapse to the same words.
//
// A filesystem error names the path it failed on. Only that path is removed, not the whole
// message, so an explanation wrapped around it still reaches the player.
func sanitiseError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	for {
		before, after, found := strings.Cut(msg, ": ")
		// A wrapping prefix is a package-qualified symbol: dotted and unspaced, like
		// "ui.SaveManager.Load". Anything else is prose meant for the reader, so stop.
		if !found || strings.ContainsAny(before, " \t") || !strings.Contains(before, ".") {
			break
		}
		msg = after
	}

	msg = scrubPaths(err, msg)
	msg = truncateRunes(msg, 120)
	if msg == "" {
		msg = "unknown error"
	}
	return msg
}

// scrubPaths replaces the filesystem paths named anywhere in err's chain with a neutral
// phrase wherever they appear in msg, so a path never reaches the UI (SECURITY-UI-2) while
// the surrounding explanation survives.
func scrubPaths(err error, msg string) string {
	replace := func(path string) {
		if path != "" {
			msg = strings.ReplaceAll(msg, path, "the file")
		}
	}
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		replace(pathErr.Path)
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		replace(linkErr.Old)
		replace(linkErr.New)
	}
	return msg
}

// truncateRunes shortens s to at most max runes, appending an ellipsis when it cuts. It
// counts runes rather than bytes so a multi-byte character is never split into an invalid
// fragment, which would render as a replacement glyph.
func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	n := 0
	for i := range s {
		if n == max {
			return s[:i] + "…"
		}
		n++
	}
	return s
}
