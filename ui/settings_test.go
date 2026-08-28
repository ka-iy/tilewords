// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"
	"pgregory.net/rapid"

	"tilewords/cpu"
	"tilewords/dictionary"
	"tilewords/engine"
)

// testAvail is a fixed set of dictionary names used by the settings tests. sanitize/decode
// validate a dictionary by membership in the passed slice, not global availability, so these
// synthetic names exercise the logic without depending on which assets are embedded.
var testAvail = []dictionary.DictName{"alpha", "beta", "gamma"}

// TestDefaultGameSettings covers the built-in defaults: first available dictionary, Classic
// mode, mid difficulty, notation off, board headers off; and the empty-avail case (blank
// dictionary).
func TestDefaultGameSettings(t *testing.T) {
	got := defaultGameSettings(testAvail)
	want := GameSettings{
		Dict:         "alpha",
		Mode:         engine.ClassicMode,
		Difficulty:   defaultDifficulty,
		Notation:     false,
		BoardHeaders: false,
	}
	if got != want {
		t.Fatalf("defaultGameSettings = %+v, want %+v", got, want)
	}
	if d := defaultGameSettings(nil); d.Dict != "" {
		t.Fatalf("defaultGameSettings(nil).Dict = %q, want empty", d.Dict)
	}
}

// TestGameSettingsDisplay maps the persisted display options onto the value startNewGame
// takes, so a mix-up between the two flags shows up here rather than on screen.
func TestGameSettingsDisplay(t *testing.T) {
	got := GameSettings{Notation: true, BoardHeaders: false}.display()
	if want := (displayPrefs{notation: true, boardHeaders: false}); got != want {
		t.Errorf("display() = %+v, want %+v", got, want)
	}
	got = GameSettings{Notation: false, BoardHeaders: true}.display()
	if want := (displayPrefs{notation: false, boardHeaders: true}); got != want {
		t.Errorf("display() = %+v, want %+v", got, want)
	}
}

// TestSanitize covers the load-time validation rules (BR-1..BR-3).
func TestSanitize(t *testing.T) {
	def := defaultGameSettings(testAvail)

	// BR-1: an unavailable dictionary falls back to the first available.
	if got := sanitize(GameSettings{Dict: "nope", Mode: engine.ClassicMode, Difficulty: 5}, testAvail); got.Dict != def.Dict {
		t.Errorf("unavailable dict: Dict = %q, want %q", got.Dict, def.Dict)
	}
	// An available dictionary is preserved.
	if got := sanitize(GameSettings{Dict: "beta", Mode: engine.ClassicMode, Difficulty: 5}, testAvail); got.Dict != "beta" {
		t.Errorf("available dict not preserved: Dict = %q, want beta", got.Dict)
	}
	// BR-2: a difficulty outside the CPU's level range resets to the default.
	for _, bad := range []int{0, -3, cpu.MaxLevel + 1, 100} {
		if got := sanitize(GameSettings{Dict: "alpha", Difficulty: bad}, testAvail); got.Difficulty != def.Difficulty {
			t.Errorf("difficulty %d: got %d, want default %d", bad, got.Difficulty, def.Difficulty)
		}
	}
	// Every level the CPU accepts is preserved, including the demigod-mode top level.
	for _, ok := range []int{cpu.MinLevel, 8, cpu.NearBestLevel, cpu.DemigodModeLevel} {
		if got := sanitize(GameSettings{Dict: "alpha", Difficulty: ok}, testAvail); got.Difficulty != ok {
			t.Errorf("in-range difficulty %d not preserved: got %d", ok, got.Difficulty)
		}
	}
	// BR-3: an unknown mode falls back to Classic; Interesting is preserved.
	if got := sanitize(GameSettings{Dict: "alpha", Mode: engine.GameMode(99), Difficulty: 5}, testAvail); got.Mode != engine.ClassicMode {
		t.Errorf("unknown mode: got %v, want Classic", got.Mode)
	}
	if got := sanitize(GameSettings{Dict: "alpha", Mode: engine.InterestingMode, Difficulty: 5}, testAvail); got.Mode != engine.InterestingMode {
		t.Errorf("Interesting mode not preserved: got %v", got.Mode)
	}
}

// TestDecodeInvalid covers BR-4: empty or malformed documents yield the built-in defaults.
func TestDecodeInvalid(t *testing.T) {
	def := defaultGameSettings(testAvail)
	for _, raw := range []string{"", "{not json", "[]", "null", "42"} {
		if got := decode(raw, testAvail); got != def {
			t.Errorf("decode(%q) = %+v, want defaults %+v", raw, got, def)
		}
	}
}

// TestSettingsStoreRoundTrip verifies save→load through a real in-memory fyne.Preferences.
func TestSettingsStoreRoundTrip(t *testing.T) {
	app := test.NewApp()
	s := newSettingsStore(app.Preferences())

	// Before any save, load returns the built-in defaults.
	if got := s.load(testAvail); got != defaultGameSettings(testAvail) {
		t.Fatalf("empty store: load = %+v, want defaults", got)
	}

	want := GameSettings{Dict: "gamma", Mode: engine.InterestingMode, Difficulty: 9, Notation: true, BoardHeaders: true}
	s.save(want)
	if got := s.load(testAvail); got != want {
		t.Fatalf("after save: load = %+v, want %+v", got, want)
	}
}

// TestPBT_Settings_RoundTrip is the round-trip property: any valid GameSettings survives
// encode→decode unchanged.
func TestPBT_Settings_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dict := testAvail[rapid.IntRange(0, len(testAvail)-1).Draw(t, "dictIdx")]
		mode := engine.ClassicMode
		if rapid.Bool().Draw(t, "interesting") {
			mode = engine.InterestingMode
		}
		s := GameSettings{
			Dict:         dict,
			Mode:         mode,
			Difficulty:   rapid.IntRange(1, 10).Draw(t, "difficulty"),
			Notation:     rapid.Bool().Draw(t, "notation"),
			BoardHeaders: rapid.Bool().Draw(t, "boardHeaders"),
		}

		raw, err := encode(s)
		if err != nil {
			t.Fatalf("encode(%+v) errored: %v", s, err)
		}
		if got := decode(raw, testAvail); got != s {
			t.Fatalf("round-trip: decode(encode(%+v)) = %+v", s, got)
		}
	})
}

// TestPBT_Settings_LoadRobustness is the load-robustness property: decode never panics on an
// arbitrary stored string and always returns settings satisfying the validation rules.
func TestPBT_Settings_LoadRobustness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		raw := rapid.String().Draw(t, "raw")
		got := decode(raw, testAvail) // must not panic

		if !dictInList(got.Dict, testAvail) {
			t.Fatalf("decode(%q).Dict = %q, not in avail", raw, got.Dict)
		}
		if got.Difficulty < 1 || got.Difficulty > 10 {
			t.Fatalf("decode(%q).Difficulty = %d, out of 1..10", raw, got.Difficulty)
		}
		if got.Mode != engine.ClassicMode && got.Mode != engine.InterestingMode {
			t.Fatalf("decode(%q).Mode = %v, invalid", raw, got.Mode)
		}
	})
}
