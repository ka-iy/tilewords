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
	gs.defsWordCh = make(chan string, defsWordBuffer)
	go gs.runDefinitionsWorker()
	gs.dispatchHistoryDefinitions()
}

// dispatchHistoryDefinitions queues the words of every play already in the move history
// for lookup. It repopulates the Definitions tab when a saved game is loaded; for a new
// game the history holds no plays yet, so it is a no-op.
func (gs *gameScreen) dispatchHistoryDefinitions() {
	for _, e := range gs.history {
		gs.dispatchDefinitions(e.words)
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
func (gs *gameScreen) dispatchDefinitions(words []string) {
	if gs.defsWordCh == nil {
		return
	}
	for _, w := range words {
		select {
		case gs.defsWordCh <- w:
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
	for word := range gs.defsWordCh {
		entry := formatDefinitionEntry(db, word)
		fyne.Do(func() { gs.appendDefinition(entry) })
	}
}

// appendDefinition adds one entry to the Definitions tab and scrolls it into view.
// It must run on the UI goroutine and is a no-op once the screen has been left.
func (gs *gameScreen) appendDefinition(entry string) {
	if gs.abandoned {
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
	return strings.Join(gs.defsEntries, "\n\n")
}

// scrollDefinitionsToEnd keeps the newest definition entry in view after the panel
// grows, mirroring scrollHistoryToEnd: Refresh re-measures the taller label so
// ScrollToBottom targets the new bottom.
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
