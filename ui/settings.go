// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"encoding/json"
	"log"

	"fyne.io/fyne/v2"

	"tilewords/cpu"
	"tilewords/dictionary"
	"tilewords/engine"
)

// settingsPrefKey is the single preferences key under which the player's default New Game
// Setup choices are stored, as one JSON document (see GameSettings).
const settingsPrefKey = "defaultGameSettings"

// defaultDifficulty is the CPU difficulty used when no valid saved value exists; it matches
// the setup screen's initial slider position. It sits low in the range so a first game is
// winnable while the player is still learning the board — the difficulty is theirs to raise.
const defaultDifficulty = 3

// GameSettings is the persisted set of New Game Setup choices — saved as the player's
// defaults and restored the next time the setup screen opens.
//
// It is serialised as one JSON document, which is what makes the feature extensible: a setup
// option added in the future is persisted automatically by adding a field here and wiring its
// control, with no change to the store or the encode/decode functions below.
type GameSettings struct {
	// Dict is the chosen word list.
	Dict dictionary.DictName `json:"dict"`
	// Mode is the chosen board layout + tile economy.
	Mode engine.GameMode `json:"mode"`
	// Difficulty is the CPU difficulty level, cpu.MinLevel (easy) to cpu.MaxLevel (demigod mode).
	Difficulty int `json:"difficulty"`
	// Notation selects Scrabble-notation move history when true.
	Notation bool `json:"notation"`
}

// defaultGameSettings returns the built-in defaults shown when nothing valid is saved: the
// first available dictionary, Classic mode, mid difficulty, and plain (non-notation) history.
func defaultGameSettings(avail []dictionary.DictName) GameSettings {
	var d dictionary.DictName
	if len(avail) > 0 {
		d = avail[0]
	}
	return GameSettings{Dict: d, Mode: engine.ClassicMode, Difficulty: defaultDifficulty, Notation: false}
}

// dictInList reports whether name is one of the available (bundled) dictionaries in avail.
func dictInList(name dictionary.DictName, avail []dictionary.DictName) bool {
	for _, a := range avail {
		if a == name {
			return true
		}
	}
	return false
}

// sanitize coerces every field of gs into a valid value, substituting the built-in default
// for anything out of range: an unavailable dictionary becomes the first available one, a
// difficulty outside the CPU's level range becomes defaultDifficulty, and an unknown mode
// becomes ClassicMode.
func sanitize(gs GameSettings, avail []dictionary.DictName) GameSettings {
	def := defaultGameSettings(avail)
	if !dictInList(gs.Dict, avail) {
		gs.Dict = def.Dict
	}
	if gs.Difficulty < cpu.MinLevel || gs.Difficulty > cpu.MaxLevel {
		gs.Difficulty = def.Difficulty
	}
	if gs.Mode != engine.ClassicMode && gs.Mode != engine.InterestingMode {
		gs.Mode = engine.ClassicMode
	}
	return gs
}

// encode marshals gs to its JSON document form for storage.
func encode(gs GameSettings) (string, error) {
	b, err := json.Marshal(gs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// decode parses a stored settings document into valid GameSettings. Empty or malformed input
// yields the built-in defaults; a parsed document is sanitised. It never returns an error and
// never panics, so persisted data is safe to treat as untrusted input.
func decode(raw string, avail []dictionary.DictName) GameSettings {
	if raw == "" {
		return defaultGameSettings(avail)
	}
	var gs GameSettings
	if err := json.Unmarshal([]byte(raw), &gs); err != nil {
		return defaultGameSettings(avail)
	}
	return sanitize(gs, avail)
}

// settingsStore persists GameSettings to a single fyne.Preferences key. It depends only on
// the fyne.Preferences interface, so tests can back it with an in-memory preferences.
type settingsStore struct {
	// prefs is the backing preference store (App.fapp.Preferences() in production).
	prefs fyne.Preferences
}

// newSettingsStore returns a store backed by prefs.
func newSettingsStore(prefs fyne.Preferences) *settingsStore {
	return &settingsStore{prefs: prefs}
}

// load reads and validates the saved settings, falling back to built-in defaults for absent,
// empty, or malformed data. It never errors.
func (s *settingsStore) load(avail []dictionary.DictName) GameSettings {
	return decode(s.prefs.StringWithFallback(settingsPrefKey, ""), avail)
}

// save persists gs as the player's defaults. It is best-effort: an encode failure is logged
// and ignored so that persisting defaults never blocks starting a game.
func (s *settingsStore) save(gs GameSettings) {
	raw, err := encode(gs)
	if err != nil {
		log.Printf("settings: could not encode default settings: %v", err)
		return
	}
	s.prefs.SetString(settingsPrefKey, raw)
}

// defaultsFor returns the player's saved default settings, or the built-in defaults when no
// settings store is configured (e.g. a headless-constructed App in tests).
func (a *App) defaultsFor(avail []dictionary.DictName) GameSettings {
	if a.settings != nil {
		return a.settings.load(avail)
	}
	return defaultGameSettings(avail)
}
