# Logical Components — Unit 2: `engine`

## Overview

The `engine` unit is a pure in-memory library with no network, database, queue, or cache
infrastructure. All state lives in `GameState`; all mutation goes through commands.

---

## Component 1: Board Manager (`engine.Board`)

**Type**: In-process struct methods  
**Files**: `engine/board.go`

```
[NewBoard()]
      |
      v
[15][15]Cell array — Square types fixed at construction from premium layout table
      |
      v
[Place / Remove / IsEmpty / Cell / Clone]
  called by: Commands (Place/Remove), AI Clone, Rules (Cell read)
```

**Responsibilities**:
- Initialise 15×15 grid with standard premium square layout
- Provide O(1) cell access, tile placement, and tile removal
- Produce deep-copy clone for AI goroutine
- Implement GobEncode/GobDecode for save/load

---

## Component 2: Tile Bag (`engine.Bag`)

**Type**: In-process struct methods  
**Files**: `engine/bag.go`

```
[NewBag(rng)]
      |
      v
[100 Tile slice] — shuffled with Fisher-Yates
      |
[Draw(n)] ──▶ []Tile (drawn from end; O(1) pop)
[Return(tiles, rng)] ──▶ appends + reshuffles
[Count()] ──▶ int
[Clone()] ──▶ independent copy (for GameState.Clone)
```

**Responsibilities**:
- Maintain the standard 100-tile distribution
- Provide tile draw (random) and return (with reshuffle)
- Support deterministic test construction via `NewTestBag(tiles)`

---

## Component 3: Rack Manager (`engine.Rack`)

**Type**: In-process struct methods  
**Files**: `engine/rack.go`

```
[Rack{tiles []Tile}]
      |
[Add / Remove / Replenish / Tiles / Count / Clone]
  called by: Commands, AI (read via GameState clone)
```

**Responsibilities**:
- Hold up to 7 tiles
- Remove specific tiles (returning error if not found)
- Replenish from bag to capacity

---

## Component 4: Rules Engine (`engine/rules.go`)

**Type**: Package-level functions (stateless)  
**Files**: `engine/rules.go`

```
ValidatePlacement(board, move, dict):
  [Sanity] → [Orientation] → [Occupancy] → [Contiguity]
           → [First-move centre] → [Adjacency] → [extractWords] → [dict.Validate each]
           → ([]string words, error)

IsGameOver(state):
  [ConsecutivePasses ≥ 6?] OR [Bag empty AND rack empty?]
  → (bool, EndReason)

ApplyEndgameScoring(state):
  [identify empty-rack player] → [adjust HumanScore / AIScore]
```

**Consumers**: `PlayCommand.Execute` (validation), `ui.GameScreen` (game-over check after each command), `ui.GameScreen` (end-game scoring at game over).

---

## Component 5: Scoring Engine (`engine/score.go`)

**Type**: Package-level functions (stateless)  
**Files**: `engine/score.go`

```
Score(board, move):
  [check WordsFormed populated] → [for each word: letter×mult + word×mult]
                                → [bingo check] → int score

extractWords(board, move):  [unexported]
  [build virtual board overlay] → [main word along primary axis]
                               → [cross-words perpendicular to each new tile]
                               → []string
```

**Called by**: `PlayCommand.Execute` (after ValidatePlacement populates WordsFormed).

---

## Component 6: Command System (`engine/commands.go`)

**Type**: Interface + three concrete structs  
**Files**: `engine/commands.go`

```
Command interface { Execute(*GameState, *rand.Rand) error; Undo(*GameState) }

PlayCommand.Execute:
  [ValidatePlacement] → [Score] → [Rack.Remove] → [Board.Place ×n]
  → [save prevPasses] → [reset ConsecutivePasses] → [Rack.Replenish] → [addScore]
  → [flip CurrentTurn] → [MoveNumber++]

PlayCommand.Undo:
  [subtractScore] → [Rack.Remove(drawn)] → [Bag.Return(drawn, nil)]
  → [Board.Remove ×n] → [Rack.Add(placed)] → [restore prevPasses]
  → [flip CurrentTurn] → [MoveNumber--]

ExchangeCommand.Execute:
  [bag ≥ 7 check] → [Rack.Remove] → [snapshot bag] → [Bag.Draw]
  → [Rack.Add(drawn)] → [Bag.Return(exchanged, rng)]
  → [prevPasses++] → [flip CurrentTurn] → [MoveNumber++]

ExchangeCommand.Undo:
  [Rack.Remove(drawn)] → [restore bag from snapshot]
  → [Rack.Add(exchanged)] → [restore prevPasses]
  → [flip CurrentTurn] → [MoveNumber--]

PassCommand.Execute:
  [prevPasses++] → [flip CurrentTurn] → [MoveNumber++]

PassCommand.Undo:
  [restore prevPasses] → [flip CurrentTurn] → [MoveNumber--]

UndoLastRound(state):
  [LastAICommand.Undo] → [LastHumanCommand.Undo] → nil both fields
```

---

## Component 7: Game State (`engine/state.go`)

**Type**: Struct + constructor  
**Files**: `engine/state.go`

```
engine.New(dictName, aiLevel, rng):
  [NewBoard] → [NewBag(rng)] → [drawForFirstTurn(bag, rng)]
  → [HumanRack.Replenish] → [AIRack.Replenish] → *GameState

GameState.Clone():
  [Board.Clone] + [HumanRack.Clone] + [AIRack.Clone] + [Bag.Clone]
  + scalar fields → *GameState (independent deep copy)
```

---

## Infrastructure Components: None

| Infrastructure Type | Present | Rationale |
|---|---|---|
| Network / HTTP | No | Pure local library |
| Database / persistent store | No | State held in memory; gob serialisation in `ui` |
| Message queue | No | Synchronous library calls; AIWorker channels are in `ai` package |
| Cache (LRU, Redis, etc.) | No | No repeated expensive lookups |
| Circuit breaker | No | No external dependencies |
| Rate limiter | No | Internal library |
| Load balancer | No | Single-process application |
| CDN / object storage | No | No assets in this unit |
