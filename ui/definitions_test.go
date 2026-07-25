package ui

import (
	"strings"
	"testing"

	"tilewords/defs"
)

// testDefsDB builds a small definitions DB covering the words used in these tests.
func testDefsDB() *defs.DB {
	return defs.NewDB(map[string]*defs.Entry{
		"unmix": {Word: "unmix", Senses: []defs.Sense{{POS: "verb", Gloss: "To separate what has been mixed."}}},
		"mouse": {Word: "mouse", Senses: []defs.Sense{{POS: "noun", Gloss: "A small rodent."}}},
		"mice":  {Word: "mice", Senses: []defs.Sense{{POS: "verb", Gloss: "To hunt mice."}}},
	}, map[string]string{"mice": "mouse"})
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
	if !strings.Contains(got, "also form of mouse: A small rodent.") {
		t.Errorf("entry missing the inflection reading; got %q", got)
	}
}

func TestDefinitionsBlankLineSeparation(t *testing.T) {
	// Two turns of history, so both entries belong to turns the history still reaches.
	gs := &gameScreen{history: []historyEntry{{player: "You"}, {player: "AI"}}}
	gs.appendDefinition(defsEntry{text: "UNMIX\nverb - To separate.", turn: 0})
	gs.appendDefinition(defsEntry{text: "MOUSE\nnoun - A small rodent.", turn: 1})

	want := "UNMIX\nverb - To separate.\n\nMOUSE\nnoun - A small rodent."
	if got := gs.definitionsText(); got != want {
		t.Errorf("definitions text = %q, want %q", got, want)
	}
}

// TestDefinitionsDroppedOnUndo verifies the Definitions tab stops describing a word once the
// turn that played it is undone, and that a lookup still in flight for that turn is refused
// on arrival — lookups run off the UI goroutine, so one can be delivered after the undo.
func TestDefinitionsDroppedOnUndo(t *testing.T) {
	gs := &gameScreen{history: []historyEntry{{player: "You"}, {player: "AI"}}}
	gs.appendDefinition(defsEntry{text: "CRANE", turn: 0})
	gs.appendDefinition(defsEntry{text: "ZEBRA", turn: 1})

	// Undo the AI's turn.
	gs.history = gs.history[:1]
	gs.dropUndoneDefinitions()

	if got := gs.definitionsText(); got != "CRANE" {
		t.Errorf("after undo definitions text = %q, want %q", got, "CRANE")
	}

	// A late arrival for the undone turn must not reinstate it.
	gs.appendDefinition(defsEntry{text: "ZEBRA", turn: 1})
	if got := gs.definitionsText(); got != "CRANE" {
		t.Errorf("a late lookup for an undone turn was appended: %q", got)
	}

	// Replaying the turn admits its definition again, exactly once.
	gs.history = append(gs.history, historyEntry{player: "AI"})
	gs.appendDefinition(defsEntry{text: "ZEBRA", turn: 1})
	if got := gs.definitionsText(); got != "CRANE\n\nZEBRA" {
		t.Errorf("after replay definitions text = %q, want %q", got, "CRANE\n\nZEBRA")
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
		history:    []historyEntry{{player: "You"}, {player: "AI"}},
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
			{player: "AI", words: []string{"SPELLS"}},
			{player: "You"}, // a pass: no words
			{player: "AI", words: []string{"OW", "DO"}},
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
