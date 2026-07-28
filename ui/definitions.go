// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"strings"

	"fyne.io/fyne/v2"

	"tilewords/defs"
)

// defsWordBuffer bounds the words awaiting definition lookup. A game forms far fewer
// words than this, so the buffer never fills in practice; the bound only guarantees
// dispatchDefinitions can never block the UI goroutine.
const defsWordBuffer = 256

// defsUnavailableNote is shown in the Definitions tab when the definitions asset was
// not embedded in the build (i.e. 'make defs' was not run).
const defsUnavailableNote = "Definitions are unavailable in this build."

// startDefinitions wires up the Definitions tab: it opens the dispatch channel and
// launches the lookup worker. When the definitions asset is absent it instead shows
// a note and leaves defsWordCh nil, so dispatchDefinitions becomes a no-op.
func (gs *gameScreen) startDefinitions() {
	if !defs.Available() {
		gs.defsLabel.SetText(defsUnavailableNote)
		return
	}
	gs.defsWordCh = make(chan defsRequest, defsWordBuffer)
	go gs.runDefinitionsWorker()
	gs.dispatchHistoryDefinitions()
}

// defsRequest is one queued lookup: a word and the index of the history turn that formed it.
type defsRequest struct {
	// word is the word to look up.
	word string
	// turn is the history index of the play that formed word. The entry belongs to the
	// panel only while the history still reaches that index.
	turn int
}

// defsEntry is one rendered definition shown in the Definitions tab.
type defsEntry struct {
	// text is the formatted entry, ready to display.
	text string
	// turn is the history index of the play that formed the word; see defsRequest.turn.
	turn int
}

// dispatchHistoryDefinitions queues the words of every play already in the move history
// for lookup. It repopulates the Definitions tab when a saved game is loaded; for a new
// game the history holds no plays yet, so it is a no-op.
func (gs *gameScreen) dispatchHistoryDefinitions() {
	for i, e := range gs.history {
		gs.dispatchDefinitionsForTurn(e.words, i)
	}
}

// dropUndoneDefinitions removes the entries belonging to turns the history no longer has,
// called after an undo. Entries already displayed are filtered here; entries still in flight
// are rejected on arrival by appendDefinition, since a lookup dispatched before the undo can
// be delivered after it.
func (gs *gameScreen) dropUndoneDefinitions() {
	kept := gs.defsEntries[:0]
	for _, e := range gs.defsEntries {
		if e.turn < len(gs.history) {
			kept = append(kept, e)
		}
	}
	gs.defsEntries = kept
	if gs.defsLabel != nil {
		gs.defsLabel.SetText(gs.definitionsText())
		// The panel shrank, so the scroll offset can now point past the end of the text and
		// show nothing at all. Re-scrolling clamps it back onto the shortened content, so
		// the remaining entries stay visible without the user having to scroll.
		gs.scrollDefinitionsToEnd()
	}
}

// stopDefinitions closes the dispatch channel so the worker goroutine exits when the
// screen is left. It is safe to call more than once.
func (gs *gameScreen) stopDefinitions() {
	if gs.defsWordCh != nil && !gs.defsClosed {
		gs.defsClosed = true
		close(gs.defsWordCh)
	}
}

// dispatchDefinitions queues each word a move formed for definition lookup. Sends are
// non-blocking: if the buffer were ever full the word is dropped rather than stall the
// UI goroutine, since a missing definition is a cosmetic loss. It is a no-op when the
// definitions asset is unavailable (defsWordCh is nil).
// It is called from logCommand before the turn is appended to the history, so the turn the
// words belong to is the index that append will occupy.
func (gs *gameScreen) dispatchDefinitions(words []string) {
	gs.dispatchDefinitionsForTurn(words, len(gs.history))
}

// dispatchDefinitionsForTurn queues words as belonging to the given history turn.
func (gs *gameScreen) dispatchDefinitionsForTurn(words []string, turn int) {
	if gs.defsWordCh == nil {
		return
	}
	for _, w := range words {
		select {
		case gs.defsWordCh <- defsRequest{word: w, turn: turn}:
		default:
		}
	}
}

// runDefinitionsWorker loads the definitions DB, then looks up each dispatched word
// and marshals its formatted entry back onto the UI goroutine. It runs off the UI
// goroutine (the decode is slow and the lookups are pure work) and exits when the
// dispatch channel is closed by stopDefinitions.
func (gs *gameScreen) runDefinitionsWorker() {
	db, err := defs.Load()
	if err != nil {
		fyne.Do(func() {
			if !gs.abandoned {
				gs.defsLabel.SetText(defsUnavailableNote)
			}
		})
		for range gs.defsWordCh { // drain until closed so senders never block
		}
		return
	}
	for req := range gs.defsWordCh {
		entry := defsEntry{text: formatDefinitionEntry(db, req.word), turn: req.turn}
		fyne.Do(func() { gs.appendDefinition(entry) })
	}
}

// appendDefinition adds one entry to the Definitions tab and scrolls it into view.
// It must run on the UI goroutine and is a no-op once the screen has been left.
//
// An entry whose turn the history no longer reaches is dropped: lookups run off the UI
// goroutine, so one dispatched before an undo can arrive after it, and appending it would
// put an undone word back into the panel that dropUndoneDefinitions had just cleaned.
func (gs *gameScreen) appendDefinition(entry defsEntry) {
	if gs.abandoned {
		return
	}
	if entry.turn >= len(gs.history) {
		return
	}
	gs.defsEntries = append(gs.defsEntries, entry)
	if gs.defsLabel != nil {
		gs.defsLabel.SetText(gs.definitionsText())
		gs.scrollDefinitionsToEnd()
	}
}

// definitionsText joins the definition entries with a blank line between them, so each
// word's entry is visually separated from the previous one.
func (gs *gameScreen) definitionsText() string {
	texts := make([]string, len(gs.defsEntries))
	for i, e := range gs.defsEntries {
		texts[i] = e.text
	}
	return strings.Join(texts, "\n\n")
}

// scrollDefinitionsToEnd keeps the last definition entry in view after the panel's text
// changes, mirroring scrollHistoryToEnd: Refresh re-measures the label so ScrollToBottom
// targets the new bottom. It is needed in both directions — a grown panel would otherwise
// hide the newest entry below the fold, and a shrunken one would leave the offset past the
// end of the text, showing an empty pane.
func (gs *gameScreen) scrollDefinitionsToEnd() {
	if gs.defsScroll == nil {
		return
	}
	gs.defsScroll.Refresh()
	gs.defsScroll.ScrollToBottom()
}

// formatDefinitionEntry renders one Definitions-tab entry for a played word: the word
// in upper case, then each sense on its own line (part of speech and gloss), then any
// "form of" reading for a word that is also an inflection of another lemma. A word
// with no definition still gets an entry, so the panel accounts for every word played.
func formatDefinitionEntry(db *defs.DB, word string) string {
	header := strings.ToUpper(word)
	res, ok := db.Lookup(word)
	if !ok || res.Entry == nil {
		return header + "\n(no definition found)"
	}
	var b strings.Builder
	b.WriteString(header)
	for _, s := range res.Entry.Senses {
		b.WriteByte('\n')
		if s.POS != "" {
			b.WriteString(s.POS)
			b.WriteString(" - ")
		}
		b.WriteString(s.Gloss)
	}
	if res.AlsoForm != nil && len(res.AlsoForm.Senses) > 0 {
		b.WriteString("\nalso form of ")
		b.WriteString(res.AlsoFormWord)
		b.WriteString(": ")
		b.WriteString(res.AlsoForm.Senses[0].Gloss)
	}
	return b.String()
}
