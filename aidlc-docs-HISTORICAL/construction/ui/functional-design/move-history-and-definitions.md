# Functional Design Addendum — Move History & Definitions Panel (`ui`)

> Retroactively documented. The right-hand panel beside the board evolved into a two-tab
> area — **Move history** and **Definitions** — with a whole-panel copy affordance and
> touch-friendly scrolling. Implemented with Fyne (the shipped UI toolkit).

## Two-Tab Panel (`ui/tabpanel.go`)

`tabPanel` replaces Fyne's `container.AppTabs` (whose tab bar has a large intrinsic width and
whose touch/nested-scroll handling misbehaved inside the phone layout's vertical scroll —
board pushed past the viewport, missed tab taps, un-scrollable body). It is a plain button
header over a `Stack` that shows one body at a time.
- Header: one button per tab (equal-width grid) + a compact **Copy** icon button (right).
- `selectTab(i)` shows body *i*, hides the rest, and marks the active button.
- Default tab: **Move history**.

## Move-History Tab (`ui/game.go`)

A scrollable, selectable label logging each turn. `logCommand` appends one `historyEntry`
per executed command (player, line, points, placed cells, and the play's `words`).
- **Format** (`playLine`): plain word list (e.g. `You: UNMIX, CROSS (+28)`) or Scrabble
  coordinate notation (e.g. `You: 8D UNMIX +28`) via `engine.AnnotatedWords`, chosen by the
  `scrabbleNotation` preference (persisted in `GameState.ScrabbleNotation`).
- A fixed opening-draw line precedes the moves. The panel auto-scrolls to the newest line.

## Definitions Tab (`ui/definitions.go`)

Shows the meaning of every word played, one entry per word, entries separated by a blank
line. Populated asynchronously:
- `startDefinitions` (called from `App.showGame`) opens a buffered channel and launches
  `runDefinitionsWorker`; if `defs.Available()` is false it shows an "unavailable" note.
- When a move commits, `logCommand` calls `dispatchDefinitions(words)` — a **non-blocking**
  send, so the UI goroutine never stalls.
- `runDefinitionsWorker` loads the DB once (off the UI goroutine), looks up each word
  (`formatDefinitionEntry`), and marshals the entry back with `fyne.Do` → `appendDefinition`.
- `dispatchHistoryDefinitions` re-queues the restored history's words on load, so a resumed
  game's Definitions tab repopulates (relies on `MoveRecord.Words`).
- `formatDefinitionEntry`: the word in caps, each sense on its own line (`pos — gloss`), plus
  an "also form of <lemma>: …" line for homographs (`Result.AlsoForm`); "(no definition
  found)" when unmatched.

## Copy (`ui/tabpanel.go`, `ui/dragscroll.go`)

Because on touch a finger drag scrolls (below) rather than selects, the whole active panel is
copied two ways, both reporting "Copied to clipboard.":
- **Copy button** in the tab bar → copies the active tab's full text.
- **Long press** on the panel → copies the full text (mobile only, via the drag overlay).
- Desktop keeps native drag-select + the Copy button; word/line selection (double/triple-tap)
  still works on touch.

## Touch Scrolling (`ui/dragscroll.go`)

A selectable Fyne label captures touch drags for text selection, so the enclosing scroll
never pans. On **mobile only**, a transparent overlay is layered over the scroll content that
forwards drags to the scroll (`enableTouchScroll`). It implements only `Draggable` +
`SecondaryTappable`, so double-tap/triple-tap selection still falls through to the label, and
a long press copies the panel. Desktop is unaffected (wheel scrolls, drag selects).

## Business Rules
- **BR-UI-MH1**: The move-history format is a persisted preference; a resumed game keeps it.
- **BR-UI-MH2**: Definition lookup is best-effort and off the UI goroutine; a missing asset
  or unmatched word degrades gracefully, never blocking play.
- **BR-UI-MH3**: Copy affordances copy the whole active panel; selection is preserved (native
  on desktop; word/line + long-press-copy on touch).
