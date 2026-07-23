package ui

import (
	"testing"

	"fyne.io/fyne/v2/test"

	"tilewords/engine"
)

// newSettingsApp returns an App backed by a fresh in-memory preferences store, for testing
// the setup screen's load/save wiring without a real window.
func newSettingsApp(t *testing.T) *App {
	t.Helper()
	app := test.NewApp()
	return &App{fapp: app, settings: newSettingsStore(app.Preferences())}
}

// TestDefaultsFor_NilStore verifies the headless nil-guard: an App with no settings store
// still yields the built-in defaults rather than panicking.
func TestDefaultsFor_NilStore(t *testing.T) {
	a := &App{}
	if got := a.defaultsFor(testAvail); got != defaultGameSettings(testAvail) {
		t.Fatalf("defaultsFor with nil store = %+v, want built-in defaults", got)
	}
}

// TestDefaultsFor_RoundTripThroughApp verifies that settings saved via the App's store are
// returned by defaultsFor (the path buildSetup uses to pre-populate the controls).
func TestDefaultsFor_RoundTripThroughApp(t *testing.T) {
	a := newSettingsApp(t)
	avail := availableDicts()
	if len(avail) == 0 {
		t.Skip("no dictionaries embedded in this test build")
	}

	want := defaultGameSettings(avail)
	want.Mode = engine.InterestingMode
	want.Difficulty = 7
	want.Notation = true
	a.settings.save(want)

	if got := a.defaultsFor(avail); got != want {
		t.Fatalf("defaultsFor after save = %+v, want %+v", got, want)
	}
}

// TestBuildSetup_LoadMappingAndNoPanic guards the load-into-controls wiring: the loaded
// default dictionary's display label must be one of the radio's labels (so SetSelected
// matches a real row), and buildSetup must construct the screen without panicking when
// saved defaults are present.
func TestBuildSetup_LoadMappingAndNoPanic(t *testing.T) {
	a := newSettingsApp(t)
	avail := availableDicts()
	if len(avail) == 0 {
		t.Skip("no dictionaries embedded in this test build")
	}

	seed := defaultGameSettings(avail)
	seed.Mode = engine.InterestingMode
	seed.Difficulty = 3
	seed.Notation = true
	a.settings.save(seed)

	// The loaded dictionary must map to an existing radio label.
	gs := a.defaultsFor(avail)
	labels := make(map[string]bool, len(avail))
	for _, d := range avail {
		labels[dictDisplayName(d)] = true
	}
	if !labels[dictDisplayName(gs.Dict)] {
		t.Fatalf("loaded dict %q has label %q not present among radio labels", gs.Dict, dictDisplayName(gs.Dict))
	}

	if content := a.buildSetup(); content == nil {
		t.Fatal("buildSetup returned nil content")
	}
}
