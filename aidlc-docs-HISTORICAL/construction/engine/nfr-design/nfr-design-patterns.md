# NFR Design Patterns — Unit 2: `engine`

## Pattern 1: Deep-Copy Model for Thread Safety (NFR-ENG-T1)

**Problem**: The AI goroutine and the UI goroutine both need access to `GameState`, but
concurrent read+write without synchronisation causes data races.

**Solution**: The UI goroutine passes a deep copy of `GameState` to the AI worker before
starting the AI goroutine. The AI goroutine reads only from its copy; the UI goroutine
owns the original. No locks, no channels for state access.

```
UI goroutine owns:                AI goroutine receives:
  state *GameState   ──Clone()──▶  stateCopy *GameState  (independent)
  (mutated by Commands)            (read-only for move gen)
```

**`GameState.Clone()` contract**:
```go
func (s *GameState) Clone() *GameState {
    return &GameState{
        Board:             s.Board.Clone(),        // deep copy [15][15]Cell
        HumanRack:         s.HumanRack.Clone(),   // deep copy []Tile
        AIRack:            s.AIRack.Clone(),       // deep copy []Tile
        Bag:               s.Bag.Clone(),          // deep copy []Tile
        HumanScore:        s.HumanScore,           // value copy
        AIScore:           s.AIScore,
        ConsecutivePasses: s.ConsecutivePasses,
        CurrentTurn:       s.CurrentTurn,
        MoveNumber:        s.MoveNumber,
        DictName:          s.DictName,
        AILevel:           s.AILevel,
        // LastHumanCommand and LastAICommand intentionally omitted:
        // AI operates on a read-only snapshot; undo state is irrelevant.
    }
}
```

**Go memory model guarantee**: A value fully written before a goroutine is started is
safely readable by that goroutine without synchronisation. The clone is complete before
`go aiWorker.run(stateCopy)` is called.

---

## Pattern 2: Fixed-Array Board for O(1) Premium-Square Access (NFR-ENG-P1)

**Problem**: `Score` and `ValidatePlacement` read premium square types for each tile
position on every call. A map lookup or slice-of-slices adds indirection and allocation.

**Solution**: The board grid is a `[15][15]Cell` fixed array. `Cell.Square` is set once
by `NewBoard` and never changes. Premium-square lookup is a direct array index: O(1),
zero allocation, cache-friendly.

```go
type Board struct {
    cells [15][15]Cell  // fixed size; allocated on heap as part of Board struct
}

// Scoring hot path — no map lookup, no allocation:
sq := b.cells[row][col].Square
switch sq {
case DoubleLetter: letterMult = 2
case TripleLetter: letterMult = 3
case DoubleWord, Centre: wordMultiplier *= 2
case TripleWord:   wordMultiplier *= 3
}
```

**Board.Clone()** copies the entire `[15][15]Cell` array by value:
```go
func (b *Board) Clone() *Board {
    clone := &Board{}
    clone.cells = b.cells  // array copy — all 225 cells copied by value
    // Tile pointers within cells: tiles on board are immutable after placement,
    // so shallow pointer copy is safe for read-only AI use.
    return clone
}
```

---

## Pattern 3: Inverse Command Pattern for Undo (BR-E14, FR-09)

**Problem**: Undo must restore `GameState` exactly to its prior state, including bag order
after a tile exchange (which reshuffles the bag).

**Solution**: Each command captures pre-Execute state in unexported fields at Execute time.
Undo reads those fields to reverse every mutation. No separate memento object is needed
because the command itself is the memento.

```go
// PlayCommand captures enough to reverse a play.
type PlayCommand struct {
    Move       PlayMove  // the move being executed (exported for gob if needed)
    drawnTiles []Tile    // tiles drawn from bag (to return on Undo)
    prevPasses int       // ConsecutivePasses before Execute
}

// ExchangeCommand captures the bag's full state before the reshuffle.
type ExchangeCommand struct {
    Move        ExchangeMove
    drawnTiles  []Tile   // tiles drawn from bag
    bagSnapshot []Tile   // full copy of bag.tiles before Execute
    prevPasses  int
}

// PassCommand — only ConsecutivePasses changes.
type PassCommand struct {
    prevPasses int
}
```

**Why `bagSnapshot` for ExchangeCommand**: `Bag.Return` reshuffles the bag using `rng`,
making the post-Execute bag order non-deterministic. A snapshot of the bag before Execute
is the only way to restore exact bag state on Undo. The snapshot is at most 100 tiles
(≈3 KB), so the cost is acceptable.

**Undo ordering** for compound undo (FR-09: one full round):
```go
func UndoLastRound(state *GameState) {
    // AI moved second; undo AI first, then human.
    if state.LastAICommand != nil {
        state.LastAICommand.Undo(state)
        state.LastAICommand = nil
    }
    if state.LastHumanCommand != nil {
        state.LastHumanCommand.Undo(state)
        state.LastHumanCommand = nil
    }
}
```

---

## Pattern 4: Injected Randomness for Deterministic Tests (NFR-ENG-TEST-1)

**Problem**: `NewBag` and `engine.New` use randomness. Tests need deterministic behaviour.

**Solution**: All randomness is injected via `*rand.Rand`. No package-level or global RNG.
Tests pass `rand.New(rand.NewSource(fixedSeed))` for reproducible sequences.

```go
// Production call:
rng := rand.New(rand.NewSource(time.Now().UnixNano()))
state := engine.New(dict.Name(), level, rng)

// Test call — always the same bag order:
rng := rand.New(rand.NewSource(42))
bag := engine.NewBag(rng)
```

Additionally, `NewTestBag(tiles []Tile)` constructs a bag from a caller-supplied slice
in that exact order (no shuffle). This gives tests full control over draw order.

---

## Pattern 5: Contextual Error Wrapping (SECURITY-ENG-1 / SECURITY-15)

**Problem**: Errors from validation and rack operations are opaque without context.

**Solution**: Every exported function that returns an error wraps it with the
package-qualified function name as prefix:

```go
func ValidatePlacement(board *Board, move *PlayMove, dict *dictionary.Dictionary) ([]string, error) {
    if len(move.Placed) == 0 {
        return nil, fmt.Errorf("engine.ValidatePlacement: no tiles placed")
    }
    ...
    if !dict.Validate(word) {
        return nil, fmt.Errorf("engine.ValidatePlacement: invalid word %q", word)
    }
    ...
}

func (r *Rack) Remove(tiles []Tile) error {
    ...
    return fmt.Errorf("engine.Rack.Remove: tile %v not found in rack", t)
}
```

**Rule**: No exported function returns a bare `errors.New` string. Every error includes
the call site and wraps upstream errors with `%w`.

---

## Pattern 6: Board GobEncode/GobDecode for Unexported Field Serialisation (NFR-ENG-S1)

**Problem**: `Board.cells` is unexported and silently skipped by `encoding/gob`. A saved
game with an empty board on load would be a silent data loss bug.

**Solution**: `Board` implements `GobEncode` and `GobDecode` using an exported wire struct:

```go
// boardWire is the gob-compatible representation of Board.
type boardWire struct {
    Cells [15][15]Cell  // exported field; gob encodes it
}

func (b *Board) GobEncode() ([]byte, error) {
    var buf bytes.Buffer
    enc := gob.NewEncoder(&buf)
    if err := enc.Encode(boardWire{Cells: b.cells}); err != nil {
        return nil, fmt.Errorf("engine.Board.GobEncode: %w", err)
    }
    return buf.Bytes(), nil
}

func (b *Board) GobDecode(data []byte) error {
    var w boardWire
    dec := gob.NewDecoder(bytes.NewReader(data))
    if err := dec.Decode(&w); err != nil {
        return fmt.Errorf("engine.Board.GobDecode: %w", err)
    }
    b.cells = w.Cells
    return nil
}
```

**Note**: `Cell.Tile` is `*Tile`. A nil pointer gob-encodes correctly as absent; a non-nil
pointer encodes the pointed-to `Tile` value. No special handling needed.

---

## Pattern 7: Word Extraction Reuse Between Validation and Display (BL-E05)

**Problem**: `ValidatePlacement` and the UI's move-confirmation display both need the list
of words formed by a play. Duplicating the word-extraction logic would risk divergence.

**Solution**: `extractWords(board *Board, move *PlayMove) []string` is a single unexported
function used by both `ValidatePlacement` and `Score`. The result is cached in
`PlayMove.WordsFormed` by `ValidatePlacement` so `Score` does not re-extract:

```
ValidatePlacement:
  words := extractWords(board, move)   // extract + validate all words
  move.WordsFormed = words             // cache for Score and UI display
  return words, nil

Score:
  if len(move.WordsFormed) == 0 {
      return 0, fmt.Errorf("engine.Score: called before ValidatePlacement")
  }
  // use move.WordsFormed directly — no re-extraction
```

---

## Pattern 8: GoDoc + Algorithm Commentary (NFR-10)

Mandatory structure for algorithm-critical files:

```go
// Package engine implements all game rules, board state, tile management,
// scoring, and the command/undo system for the Squabble crossword board game.
//
// All GameState mutations go through a Command.Execute call.
// All reversals go through Command.Undo. This invariant is the foundation
// of the undo system (FR-09) and must not be bypassed.
package engine

// Score calculates the total score for move on board, including all cross-word
// scores and the bingo bonus (+50 for playing all 7 tiles).
// Premium square multipliers apply only to tiles placed during this move;
// existing board tiles contribute only their face value.
// Score must be called after ValidatePlacement has populated move.WordsFormed.
func Score(board *Board, move *PlayMove) (int, error) { ... }

// extractWords returns all words formed by placing move.Placed on board:
// the main word (along the primary axis of the play) followed by any
// cross-words (perpendicular words of length ≥ 2 formed by the new tiles).
// Words are returned in left-to-right, top-to-bottom board order.
func extractWords(board *Board, move *PlayMove) []string { ... }
```
