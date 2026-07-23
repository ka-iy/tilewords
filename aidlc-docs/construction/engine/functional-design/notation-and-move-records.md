# Functional Design Addendum — Move Notation & Move Records (`engine`)

> Retroactively documented. Two related additions to the engine after the initial design:
> Scrabble-coordinate move notation (a display format) and the persisted move-history /
> opening-draw records that let a resumed game restore its move log.

## Scrabble Notation (`engine/notation.go`)

A display formatter for a committed `PlayMove`, used by the UI when the player enables
"Show move history in Scrabble notation".

- `AnnotatedWords(board, move) []string` — returns a notation string for every word the play
  forms: the main word (along the play's axis) first, then perpendicular cross-words. Each is
  `"<coord> <word>"`.
- `AnnotatedMainWord(board, move) (coord, word string, ok bool)` — the primary word only.
- **Coordinate convention** (`notationCoord`): a **horizontal** word is written
  row-number-then-column-letter (e.g. `8D`); a **vertical** word is column-letter-then-row-
  number (e.g. `D8`).
- **Word rendering** (`annotate`): tiles already on the board before this move (the existing
  word(s) the play hooks onto) are shown parenthesised in maximal runs; blank tiles are shown
  as lowercase letters.
- Called **after** the move is committed, so the board is read with the tiles in place.

### Business Rules
- **BR-NOTE-1**: Notation is a pure display transform over committed board state; it changes
  no rules or scoring.
- **BR-NOTE-2**: The plain format (`"UNMIX, CROSS"`) and the Scrabble format
  (`"8D UNMIX +28"`) present the same move; the choice is a UI preference (below).

## Move Records & Opening Draw (`engine/state.go`)

The persisted, non-executable data a resumed game needs to restore its UI move log and its
opening-draw summary — kept out of the command/undo system on purpose.

### Entity: `MoveRecord`
```
MoveRecord {
    Player string     // "You" or "AI"
    Line   string     // already-formatted move-history display line
    Points int        // score this move contributed (0 for pass/exchange)
    Cells  [][2]int   // cells this move placed, for the AI last-word highlight; nil otherwise
    Words  []string   // words this play formed (main + cross), so the definitions panel
                      // repopulates on load; nil for pass/exchange
}
```
`GameState.History []MoveRecord` holds the log; it stores rendered display data, **not**
executable commands — undo is intentionally not restored across a save/load.

### Entity: `OpeningDraw`
Records how the first turn was decided under the opening-draw rule (each player draws one
tile; nearest to the start of the alphabet plays first, blank earliest). `GameState.OpeningDraw`
is set by `New` and is nil for directly-constructed states (e.g. tests).

### Persisted UI-preference field
- `GameState.ScrabbleNotation bool` — the move-history display preference, persisted so a
  resumed game keeps its format. Older saves without it decode as `false` (plain).

### Business Rules
- **BR-REC-1**: `History`, `OpeningDraw`, `ScrabbleNotation`, `Words` are all optional in the
  save format; older saves decode them as empty/nil/false (backward compatible).
- **BR-REC-2**: Records are display data; restoring them never makes restored moves undoable
  (consistent with the zeroed `LastHumanCommand`/`LastAICommand`).
