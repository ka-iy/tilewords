// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"tilewords/defs"
)

// testDefsDB builds a small definitions DB covering the words used in these tests.
func testDefsDB() *defs.DB {
	return defs.NewDB(map[string]*defs.Entry{
		"unmix": {Word: "unmix", Senses: []defs.Sense{{POS: "verb", Gloss: "To separate what has been mixed."}}},
		"mouse": {Word: "mouse", Senses: []defs.Sense{{POS: "noun", Gloss: "A small rodent."}}},
		"mice":  {Word: "mice", Senses: []defs.Sense{{POS: "verb", Gloss: "To hunt mice."}}},
		"cat":   {Word: "cat", Senses: []defs.Sense{{POS: "noun", Gloss: "A small feline."}}},
		"abac":  {Word: "abac", Senses: []defs.Sense{{POS: "noun", Gloss: "Synonym of nomogram."}}},
		"nomogram": {Word: "nomogram", Senses: []defs.Sense{
			{POS: "noun", Gloss: "A diagram of the relationship between variables."}}},
	}, map[string]defs.Inflection{
		"mice": {Lemma: "mouse", Relation: "plural"},
		"cats": {Lemma: "cat", Relation: "plural"},
	})
}

// TestFormatDefinitionEntryInflection covers a played word that only Wiktionary's own
// inflection edge explains: the senses shown are the lemma's, so the entry has to say
// whose they are.
func TestFormatDefinitionEntryInflection(t *testing.T) {
	got := formatDefinitionEntry(testDefsDB(), "CATS")
	if !strings.Contains(got, "noun - plural of cat: A small feline.") {
		t.Errorf("entry does not name the lemma the senses belong to; got %q", got)
	}
}

// TestFormatDefinitionEntryRedirect covers the other half: a word whose own sense only
// points at another word arrives from Lookup with that word's definition already joined
// on, and is rendered as the single line it is.
func TestFormatDefinitionEntryRedirect(t *testing.T) {
	got := formatDefinitionEntry(testDefsDB(), "ABAC")
	want := "noun - Synonym of nomogram: A diagram of the relationship between variables."
	if !strings.Contains(got, want) {
		t.Errorf("entry = %q, want it to contain %q", got, want)
	}
}

func TestFormatDefinitionEntry(t *testing.T) {
	db := testDefsDB()
	got := formatDefinitionEntry(db, "UNMIX")
	if !strings.HasPrefix(got, "UNMIX\n") {
		t.Errorf("entry should start with the upper-case word header; got %q", got)
	}
	if !strings.Contains(got, "verb - To separate what has been mixed.") {
		t.Errorf("entry missing the sense line; got %q", got)
	}
}

func TestFormatDefinitionEntryLowercaseInput(t *testing.T) {
	// WordsFormed is uppercase in play, but the header must be upper-case regardless.
	if got := formatDefinitionEntry(testDefsDB(), "unmix"); !strings.HasPrefix(got, "UNMIX\n") {
		t.Errorf("header not upper-cased; got %q", got)
	}
}

func TestFormatDefinitionEntryNotFound(t *testing.T) {
	got := formatDefinitionEntry(testDefsDB(), "ZZZZZ")
	if got != "ZZZZZ\n(no definition found)" {
		t.Errorf("unexpected not-found entry: %q", got)
	}
}

func TestFormatDefinitionEntryInflectionMerge(t *testing.T) {
	// "mice" is a headword and also the plural of "mouse"; both readings must appear.
	got := formatDefinitionEntry(testDefsDB(), "MICE")
	if !strings.Contains(got, "verb - To hunt mice.") {
		t.Errorf("entry missing the word's own sense; got %q", got)
	}
	if !strings.Contains(got, "also plural of mouse: A small rodent.") {
		t.Errorf("entry missing the inflection reading; got %q", got)
	}
}

func TestDefinitionsBlankLineSeparation(t *testing.T) {
	// Two turns of history, so both entries belong to plays the history still holds.
	gs := &gameScreen{history: []historyEntry{
		{player: "You", words: []string{"UNMIX"}},
		{player: "CPU", words: []string{"MOUSE"}},
	}}
	gs.appendDefinition(defsEntry{text: "UNMIX\nverb - To separate.", turn: 0, word: "UNMIX"})
	gs.appendDefinition(defsEntry{text: "MOUSE\nnoun - A small rodent.", turn: 1, word: "MOUSE"})

	want := "UNMIX\nverb - To separate.\n\nMOUSE\nnoun - A small rodent."
	if got := gs.definitionsText(); got != want {
		t.Errorf("definitions text = %q, want %q", got, want)
	}
}

// TestDefinitionsDroppedOnUndo verifies the Definitions tab stops describing a word once the
// turn that played it is undone, and that a lookup still in flight for that turn is refused
// on arrival — lookups run off the UI goroutine, so one can be delivered after the undo.
func TestDefinitionsDroppedOnUndo(t *testing.T) {
	gs := &gameScreen{history: []historyEntry{
		{player: "You", words: []string{"CRANE"}},
		{player: "CPU", words: []string{"ZEBRA"}},
	}}
	gs.appendDefinition(defsEntry{text: "CRANE", turn: 0, word: "CRANE"})
	gs.appendDefinition(defsEntry{text: "ZEBRA", turn: 1, word: "ZEBRA"})

	// Undo the CPU's turn.
	gs.history = gs.history[:1]
	gs.dropUndoneDefinitions()

	if got := gs.definitionsText(); got != "CRANE" {
		t.Errorf("after undo definitions text = %q, want %q", got, "CRANE")
	}

	// A late arrival for the undone turn must not reinstate it.
	gs.appendDefinition(defsEntry{text: "ZEBRA", turn: 1, word: "ZEBRA"})
	if got := gs.definitionsText(); got != "CRANE" {
		t.Errorf("a late lookup for an undone turn was appended: %q", got)
	}

	// Replaying the same word at that turn admits its definition again, exactly once.
	gs.history = append(gs.history, historyEntry{player: "CPU", words: []string{"ZEBRA"}})
	gs.appendDefinition(defsEntry{text: "ZEBRA", turn: 1, word: "ZEBRA"})
	if got := gs.definitionsText(); got != "CRANE\n\nZEBRA" {
		t.Errorf("after replay definitions text = %q, want %q", got, "CRANE\n\nZEBRA")
	}
}

// TestDefinitionsRefusedForReplacedTurn verifies a lookup in flight across an undo is refused when
// the turn it belongs to has been replaced by a DIFFERENT play.
//
// The turn index is a position, not an identity: the replacement play takes the index the undone
// one had, so a length check alone readmits the old word once the history regrows past it. The
// panel would then describe a word that was never on the board. The window is real at startup,
// where the worker blocks on the one-time decode of the definitions asset while every dispatched
// word waits in the queue.
func TestDefinitionsRefusedForReplacedTurn(t *testing.T) {
	gs := &gameScreen{history: []historyEntry{
		{player: "You", words: []string{"CRANE"}},
		{player: "You", words: []string{"DOG"}},
		{player: "CPU", words: []string{"OX"}},
	}}
	gs.appendDefinition(defsEntry{text: "CRANE", turn: 0, word: "CRANE"})

	// Undo back past DOG, then play PIG instead and let the CPU reply. History is the same
	// length it was, so a positional check would admit the queued DOG lookup.
	gs.history = gs.history[:1]
	gs.dropUndoneDefinitions()
	gs.history = append(gs.history,
		historyEntry{player: "You", words: []string{"PIG"}},
		historyEntry{player: "CPU", words: []string{"OX"}},
	)

	gs.appendDefinition(defsEntry{text: "DOG", turn: 1, word: "DOG"})

	if got := gs.definitionsText(); got != "CRANE" {
		t.Errorf("definitions text = %q, want %q: the panel described DOG, which the replacement "+
			"play never formed", got, "CRANE")
	}
}

// TestDefinitionsScrollClampedOnUndo verifies the Definitions panel still shows its text
// after an undo shortens it: the offset left over from the taller panel MUST be clamped back
// onto the remaining entries, otherwise the pane renders blank until the user scrolls it.
func TestDefinitionsScrollClampedOnUndo(t *testing.T) {
	gs := newRackHarness(t)
	// Render the screen so the definitions scroll and label have real sizes.
	w := test.NewWindow(gs.build())
	defer w.Close()
	w.Resize(fyne.NewSize(900, 400))

	// Enough entries that the panel is taller than its viewport and scrolled to the bottom.
	const turns = 50
	for i := 0; i < turns; i++ {
		gs.history = append(gs.history, historyEntry{player: "You", words: []string{"CRANE"}})
		gs.appendDefinition(defsEntry{text: "CRANE\nnoun - A tall wading bird.", turn: i, word: "CRANE"})
	}
	viewH := gs.defsScroll.Size().Height
	if gs.defsLabel.MinSize().Height <= viewH {
		t.Fatalf("setup: definitions (%.0f) not taller than the panel (%.0f); cannot verify scrolling",
			gs.defsLabel.MinSize().Height, viewH)
	}

	// Undo back to a single turn, leaving one entry — far shorter than the panel.
	gs.history = gs.history[:1]
	gs.dropUndoneDefinitions()

	contentH := gs.defsLabel.MinSize().Height
	maxOffset := fyne.Max(contentH-viewH, 0)
	if got := gs.defsScroll.Offset.Y; got > maxOffset+1 {
		t.Errorf("definitions scrolled past the end of its text: offset.Y=%.1f, want <=%.1f (content %.0f, view %.0f)",
			got, maxOffset, contentH, viewH)
	}
}

// TestDefinitionsLabelMatchesHistoryStyle verifies the two tabs of the shared panel render
// their text identically, so switching tabs does not change the typeface.
func TestDefinitionsLabelMatchesHistoryStyle(t *testing.T) {
	gs := newRackHarness(t)
	if !gs.defsLabel.TextStyle.Monospace {
		t.Error("definitions label is not monospace")
	}
	if gs.defsLabel.TextStyle != gs.historyLabel.TextStyle {
		t.Errorf("definitions text style %+v differs from the history panel's %+v",
			gs.defsLabel.TextStyle, gs.historyLabel.TextStyle)
	}
}

func TestDispatchDefinitionsNoChannelIsNoOp(t *testing.T) {
	// With no definitions asset the channel is nil; dispatch must not panic or block.
	gs := &gameScreen{}
	gs.dispatchDefinitions([]string{"CAT", "DOG"})
}

func TestDispatchDefinitionsQueuesWords(t *testing.T) {
	gs := &gameScreen{defsWordCh: make(chan defsRequest, defsWordBuffer)}
	gs.dispatchDefinitions([]string{"CAT", "HAT"})
	gs.stopDefinitions()

	var got []string
	for w := range gs.defsWordCh {
		got = append(got, w.word)
	}
	if len(got) != 2 || got[0] != "CAT" || got[1] != "HAT" {
		t.Errorf("queued words = %v, want [CAT HAT]", got)
	}
}

// TestDispatchDefinitionsStampsPendingTurn verifies a dispatched word is tagged with the turn
// it will occupy. dispatchDefinitions runs before the turn is appended to the history, so the
// stamp is the index that append is about to fill.
func TestDispatchDefinitionsStampsPendingTurn(t *testing.T) {
	gs := &gameScreen{
		defsWordCh: make(chan defsRequest, defsWordBuffer),
		history:    []historyEntry{{player: "You"}, {player: "CPU"}},
	}
	gs.dispatchDefinitions([]string{"CAT"})
	gs.stopDefinitions()

	req, ok := <-gs.defsWordCh
	if !ok {
		t.Fatal("no request queued")
	}
	if req.turn != 2 {
		t.Errorf("queued turn = %d, want 2 (the index the pending turn will occupy)", req.turn)
	}
}

func TestStopDefinitionsIsIdempotent(t *testing.T) {
	gs := &gameScreen{defsWordCh: make(chan defsRequest, 1)}
	gs.stopDefinitions()
	gs.stopDefinitions() // must not panic by double-closing
}

func TestDispatchHistoryDefinitions(t *testing.T) {
	// A loaded game's restored history carries each play's words; they must be queued so the
	// Definitions tab repopulates. Pass/exchange entries (nil words) contribute nothing.
	gs := &gameScreen{
		defsWordCh: make(chan defsRequest, defsWordBuffer),
		history: []historyEntry{
			{player: "CPU", words: []string{"SPELLS"}},
			{player: "You"}, // a pass: no words
			{player: "CPU", words: []string{"OW", "DO"}},
		},
	}
	gs.dispatchHistoryDefinitions()
	gs.stopDefinitions()

	var got []string
	var turns []int
	for w := range gs.defsWordCh {
		got = append(got, w.word)
		turns = append(turns, w.turn)
	}
	want := []string{"SPELLS", "OW", "DO"}
	if len(got) != len(want) {
		t.Fatalf("dispatched words = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dispatched words = %v, want %v", got, want)
		}
	}
	// Each restored word is stamped with the history turn it came from, so a later undo
	// removes exactly the right entries.
	wantTurns := []int{0, 2, 2}
	for i := range wantTurns {
		if turns[i] != wantTurns[i] {
			t.Fatalf("dispatched turns = %v, want %v", turns, wantTurns)
		}
	}
}
