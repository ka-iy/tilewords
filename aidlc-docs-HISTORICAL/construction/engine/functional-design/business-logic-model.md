# Business Logic Model — Unit 2: `engine`

## BL-E01: Board Initialisation (`NewBoard`)

```
NewBoard() *Board:
  1. Allocate a [15][15]Cell array; all cells start with Square=Normal, Tile=nil
  2. Apply the premium square table (see domain-entities.md) by setting Cell.Square for
     each premium coordinate. The layout is symmetric; all 8 TWS, 17 DWS (including
     Centre), 12 TLS, and 24 DLS positions are hard-coded from the standard layout.
  3. Return pointer to the initialised Board.
```

**Invariant**: `NewBoard` is deterministic (no randomness). Calling it twice produces
identical board layouts.

---

## BL-E02: Bag Initialisation (`NewBag`)

```
NewBag(rng *rand.Rand) *Bag:
  1. Build tiles []Tile by iterating the distribution table (domain-entities.md),
     appending count copies of each Tile{Letter, Points, IsBlank=Letter==0} entry.
  2. Fisher-Yates shuffle using rng.
  3. Return pointer to the initialised Bag.
```

**PBT-E01**: `len(NewBag(rng).tiles) == 100` for any rng seed.

---

## BL-E03: GameState Initialisation (`engine.New`)

```
New(dictName DictName, aiLevel int, rng *rand.Rand) *GameState:
  1. board := NewBoard()
  2. bag := NewBag(rng)
  3. humanRack, aiRack := &Rack{}, &Rack{}
  4. humanRack.Replenish(bag, rng)   // draws 7 tiles
  5. aiRack.Replenish(bag, rng)      // draws 7 tiles
  6. Determine first turn: flip := rng.Intn(2); if flip == 0 → HumanTurn, else → AITurn
  7. Return &GameState{Board: board, HumanRack: humanRack, AIRack: aiRack, Bag: bag,
       CurrentTurn: <above>, DictName: dictName, AILevel: aiLevel}
```

**PBT-E02 (Tile Conservation)**: After `New`, `len(bag.tiles) + humanRack.Count() + aiRack.Count() == 100`.

---

## BL-E04: Move Validation (`ValidatePlacement`)

```
ValidatePlacement(board *Board, move *PlayMove, dict *Dictionary) ([]string, error):

  Step 1 — Sanity checks:
    if len(move.Placed) == 0 → error "no tiles placed"
    if len(move.Placed) > 7  → error "too many tiles placed"

  Step 2 — Orientation detection:
    allSameRow := all PlacedTile have the same Row
    allSameCol := all PlacedTile have the same Col
    if !allSameRow && !allSameCol → error "tiles not in a single row or column"

  Step 3 — Occupancy check:
    for each PlacedTile pt in move.Placed:
      if !board.IsEmpty(pt.Row, pt.Col) → error "cell (r,c) is already occupied"

  Step 4 — Contiguity check (no gaps):
    Determine the range [minIdx, maxIdx] in the varying dimension.
    For each index i in [minIdx, maxIdx]:
      cell must be non-empty (either an existing tile or a new PlacedTile)
      → error "gap at position i" if both board and move.Placed have nothing there

  Step 5 — First-move rule:
    if board has no placed tiles (MoveNumber == 0):
      if none of move.Placed covers (7,7) → error "first move must cover centre square"
    else:
      adjacentToExisting := false
      for each PlacedTile pt:
        check all four orthogonal neighbours of (pt.Row, pt.Col)
        if any neighbour has an existing board tile → adjacentToExisting = true
      if !adjacentToExisting → error "move must connect to an existing tile"

  Step 6 — Word extraction:
    words := extractWords(board, move)  // see BL-E05

  Step 7 — Word validation:
    for each w in words:
      if len(w) < 2 → error "word too short: %q"  (cross-words can be length 1 after filtering)
      if !dict.Validate(w) → error "invalid word: %q"

  Step 8 — Populate move:
    move.WordsFormed = words
    return words, nil
```

---

## BL-E05: Word Extraction (`extractWords`)

An internal function called by `ValidatePlacement` and `Score`.

```
extractWords(board *Board, move *PlayMove) []string:

  Build a virtual board: the current board with move.Placed overlaid (for lookup).

  Determine orientation (horizontal if allSameRow, else vertical).

  Main word:
    Extend from the first placed tile in the primary direction to the left/up
    until an empty cell is found (following existing tiles on the virtual board).
    Extend right/down similarly.
    Collect all tile letters in order → main word string.

  Cross-words:
    For each PlacedTile pt in move.Placed:
      Extend perpendicular to the primary direction from pt, collecting letters.
      If the resulting string has length ≥ 2 → add to cross-word list.

  Return [main word] + cross-words (in left-to-right, top-to-bottom board order).
```

---

## BL-E06: Scoring (`Score`)

```
Score(board *Board, move *PlayMove) (int, error):
  if move.WordsFormed is empty:
    return 0, error "Score called before ValidatePlacement"

  total := 0

  Determine which cells are "new" (from move.Placed vs. existing board tiles).

  For each word w in move.WordsFormed (as a sequence of (row,col) positions):
    wordScore := 0
    wordMultiplier := 1
    for each position (r,c) in w:
      tile := virtualBoard(r,c)  // placed tile or existing board tile
      sq := board.Cell(r,c).Square
      letterMult := 1
      if isNewTile(r,c, move):
        switch sq:
          case DoubleLetter, Centre: letterMult = 2   // Centre acts as DW for word, not letter
          case TripleLetter:         letterMult = 3
          case DoubleWord, Centre:   wordMultiplier *= 2
          case TripleWord:           wordMultiplier *= 3

      // Note: Centre is a DoubleWord square; handle as DW (word×2), not letter×2
      wordScore += tile.Points * letterMult
    total += wordScore * wordMultiplier

  Bingo bonus:
    if len(move.Placed) == 7: total += 50

  move.Score = total
  return total, nil
```

**Note on Centre**: Centre is classified as `DoubleWord` in SquareType. The letter multiplier
at (7,7) is 1× (it is NOT a DoubleLetter square); only the word multiplier is 2×.

**PBT-E03 (Score non-negative)**: `Score(board, move) ≥ 0` for any valid move.  
**PBT-E04 (Bingo)**: `len(move.Placed) == 7` ⟹ `Score ≥ face_value_sum + 50`.  
**PBT-E05 (No multiplier on existing tiles)**: Placing a word entirely on occupied squares (via test double) yields exactly the sum of face values (multiplier = 1 for all).

---

## BL-E07: PlayCommand — Execute and Undo

```
PlayCommand.Execute(state *GameState, rng *rand.Rand) error:
  1. rack := currentRack(state)  // HumanRack or AIRack based on CurrentTurn
  2. ValidatePlacement(state.Board, &cmd.Move, dict)  → error if invalid
     (caller must supply dict; see architectural note below)
  3. Score(state.Board, &cmd.Move)
  4. rack.Remove(cmd.Move.Placed tiles) → error if any tile not in rack
  5. for each PlacedTile: state.Board.Place(pt.Row, pt.Col, pt.Tile)
  6. cmd.prevPasses = state.ConsecutivePasses
  7. state.ConsecutivePasses = 0
  8. cmd.drawnTiles = rack.Replenish(state.Bag, rng)  // up to 7-len(rack)
  9. addScore(state, cmd.Move.Score)
  10. state.CurrentTurn = opposite(state.CurrentTurn)
  11. state.MoveNumber++
  12. return nil

PlayCommand.Undo(state *GameState):
  rack := currentRack(state, opposite turn — the player who made this move)
  1. subtractScore(state, cmd.Move.Score)
  2. rack.Remove(cmd.drawnTiles)
  3. state.Bag.Return(cmd.drawnTiles, nil)  // no reshuffle on undo
  4. for each PlacedTile: state.Board.Remove(pt.Row, pt.Col)
  5. rack.Add(cmd.Move.Placed tiles)
  6. state.ConsecutivePasses = cmd.prevPasses
  7. state.CurrentTurn = opposite(state.CurrentTurn)
  8. state.MoveNumber--
```

**Architectural note**: `ValidatePlacement` requires a `*dictionary.Dictionary`. This is
injected at the call site (`ui` passes the game session's dictionary to the command). The
`engine` package imports `dictionary` for the `ValidatePlacement` signature only; there is
no circular dependency.

---

## BL-E08: ExchangeCommand — Execute and Undo

```
ExchangeCommand.Execute(state *GameState, rng *rand.Rand) error:
  1. if state.Bag.Count() < 7 → error "cannot exchange: fewer than 7 tiles in bag"
  2. rack := currentRack(state)
  3. rack.Remove(cmd.Move.Tiles) → error if any tile not in rack
  4. cmd.drawnTiles = state.Bag.Draw(len(cmd.Move.Tiles))
  5. rack.Add(cmd.drawnTiles)
  6. state.Bag.Return(cmd.Move.Tiles, rng)  // reshuffle into bag
  7. cmd.prevPasses = state.ConsecutivePasses
  8. state.ConsecutivePasses++              // Q2: exchange counts as pass (Option B)
  9. state.CurrentTurn = opposite(state.CurrentTurn)
  10. state.MoveNumber++
  11. return nil

ExchangeCommand.Undo(state *GameState):
  rack := currentRack(state, opposite turn)
  1. rack.Remove(cmd.drawnTiles)
  2. state.Bag.UndoReturn(cmd.Move.Tiles)  // remove the exchanged tiles from bag
  3. rack.Add(cmd.Move.Tiles)
  4. state.Bag.UndoReturn(cmd.drawnTiles)  // oops — see note below
  --- Simplified approach ---
  The bag's state must be fully saved before Execute to support exact Undo.
  ExchangeCommand stores bagSnapshot []Tile (a copy of bag.tiles before Execute).
  Undo restores bag.tiles = bagSnapshot, rack tiles as above, prevPasses, CurrentTurn.
```

**Revised ExchangeCommand struct**:
```go
type ExchangeCommand struct {
    Move        ExchangeMove
    drawnTiles  []Tile
    bagSnapshot []Tile  // copy of bag.tiles before Execute; needed for exact bag Undo
    prevPasses  int
}
```

---

## BL-E09: PassCommand — Execute and Undo

```
PassCommand.Execute(state *GameState, rng *rand.Rand) error:
  1. cmd.prevPasses = state.ConsecutivePasses
  2. state.ConsecutivePasses++
  3. state.CurrentTurn = opposite(state.CurrentTurn)
  4. state.MoveNumber++
  5. return nil

PassCommand.Undo(state *GameState):
  1. state.ConsecutivePasses = cmd.prevPasses
  2. state.CurrentTurn = opposite(state.CurrentTurn)
  3. state.MoveNumber--
```

---

## BL-E10: Game-End Detection (`IsGameOver`)

```
IsGameOver(state *GameState) (over bool, reason EndReason):
  if state.ConsecutivePasses >= 6:
    return true, SixConsecutivePasses
  if state.Bag.Count() == 0:
    if state.HumanRack.Count() == 0 || state.AIRack.Count() == 0:
      return true, RackExhausted
  return false, NotOver
```

**PBT-E06**: `IsGameOver` is deterministic given the same state; no side effects.

---

## BL-E11: End-Game Score Adjustment (`ApplyEndgameScoring`)

```
ApplyEndgameScoring(state *GameState):
  humanRemaining := sumTilePoints(state.HumanRack)
  aiRemaining    := sumTilePoints(state.AIRack)

  if state.HumanRack.Count() == 0 && state.Bag.Count() == 0:
    // Human emptied rack: human gains AI's remaining; AI loses AI's remaining
    state.HumanScore += aiRemaining
    state.AIScore    -= aiRemaining
  else if state.AIRack.Count() == 0 && state.Bag.Count() == 0:
    // AI emptied rack: AI gains human's remaining; human loses human's remaining
    state.AIScore    += humanRemaining
    state.HumanScore -= humanRemaining
  else:
    // 6-pass end: both lose their own remaining tiles (no redistribution)
    state.HumanScore -= humanRemaining
    state.AIScore    -= aiRemaining
```

---

## BL-E12: Undo Orchestration (Compound Undo)

The UI calls `UndoLastRound(state)` which reverts both the AI's most recent move and the
human's move before that (one full round), per FR-09.

```
UndoLastRound(state *GameState):
  if state.LastAICommand != nil:
    state.LastAICommand.Undo(state)
    state.LastAICommand = nil
  if state.LastHumanCommand != nil:
    state.LastHumanCommand.Undo(state)
    state.LastHumanCommand = nil
```

`UndoLastRound` is only callable when `CurrentTurn == HumanTurn` and both
`LastHumanCommand` and `LastAICommand` are non-nil (enforced by the UI layer).

---

## PBT Property Summary

| ID | Property | Oracle |
|---|---|---|
| PBT-E01 | `NewBag` always contains exactly 100 tiles | count after construction |
| PBT-E02 | After `New`, bag + human rack + AI rack = 100 | sum of counts |
| PBT-E03 | `Score(board, move) ≥ 0` for any valid move | ≥ 0 check |
| PBT-E04 | 7-tile move always adds ≥ 50 beyond face value sum | score - face_sum ≥ 50 |
| PBT-E05 | Tiles on pre-occupied squares get no multiplier | place on occupied cell in test double |
| PBT-E06 | `Execute then Undo` restores GameState exactly | deep equality before/after |
| PBT-E07 | Tile conservation: total tiles constant at 100 throughout any move sequence | running count |
| PBT-E08 | `Score` on a board with no premium squares = sum of face values | flat board test double |
