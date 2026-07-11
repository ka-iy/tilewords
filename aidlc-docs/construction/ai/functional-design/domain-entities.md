# Domain Entities — Unit 3: `ai`

## Entity: `MoveCandidate`

**Kind**: Value struct  
**Package**: `ai`

A scored, complete candidate play move produced by `GenerateMoves`. Carries both the
move itself and the metadata needed by `SelectMove` for difficulty-level filtering.

```go
type MoveCandidate struct {
    // Move is the fully-specified play move, ready to pass to PlayCommand.Execute.
    // WordsFormed and Score are populated by GenerateMoves.
    Move engine.PlayMove

    // Score is the total point value for this move (letter × multipliers + bingo).
    // Cached here so SelectMove can sort without re-calling engine.Score.
    Score int

    // OpponentAccess is the count of empty premium squares (DL/TL/DW/TW) that are
    // orthogonally adjacent to any board tile after this move is placed.
    // Used as a tiebreaker at level 10: lower is better (fewer scoring opportunities
    // opened to the opponent). Computed using Option A (total exposure).
    OpponentAccess int
}
```

---

## Entity: `direction` (unexported)

**Kind**: Named bool or int  
**Package**: `ai`

Identifies the orientation of a play being generated.

```go
type direction bool

const (
    horizontal direction = false
    vertical   direction = true
)
```

---

## Entity: `anchorSquare` (unexported)

**Kind**: Value struct  
**Package**: `ai`

An empty board cell eligible to anchor a word placement. Every valid move must
include at least one tile on an anchor square.

```go
type anchorSquare struct {
    row, col int
}
```

**Anchor rules** (BL-AI-01):
- On the first move (empty board): only `{7, 7}` is an anchor.
- On subsequent moves: every empty cell orthogonally adjacent to at least one
  occupied cell is an anchor.

---

## Entity: `crossCheckSet` (unexported)

**Kind**: `[15][15][26]bool`  
**Package**: `ai` (local variable within `GenerateMoves`)

For each empty cell `(r, c)` and each play direction, stores which letters A–Z
may legally appear at that cell without creating an invalid cross-word
(Appel-Jacobson §5, cross-check precomputation).

```
crossCheckSet[r][c][letter-'A'] = true
    ↔ placing letter at (r,c) does not form an invalid perpendicular word
```

If no perpendicular tiles exist at (r, c), all 26 entries are `true`.

---

## Entity: `aiRequest` (unexported)

**Kind**: Value struct  
**Package**: `ai`

The payload sent from the UI goroutine to the AIWorker goroutine via the request channel.

```go
type aiRequest struct {
    state *engine.GameState   // deep clone — owned by the AI goroutine
    dict  *dictionary.Dictionary
    level int
    rng   *rand.Rand          // seeded from time at Request() call
}
```

---

## Entity: `AIWorker`

**Kind**: Pointer-to-struct  
**Package**: `ai`

Owns exactly one background goroutine that performs AI move computation. The UI
goroutine communicates with it via channels.

```go
type AIWorker struct {
    reqCh  chan aiRequest   // capacity 1; UI sends here
    resCh  chan engine.Move // capacity 1; AI sends result here
    busy   bool            // true while a request is in flight (UI side only)
}
```

**Lifecycle**:
- Created by `NewAIWorker()`; goroutine started immediately.
- `Request()` sends a clone of game state + params; panics if `busy`.
- `Poll()` is non-blocking; returns `(move, true)` when result is ready.
- The goroutine runs until the process exits (no shutdown signal needed for this game).
