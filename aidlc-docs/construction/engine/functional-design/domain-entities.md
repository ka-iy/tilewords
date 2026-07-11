# Domain Entities — Unit 2: `engine`

## Entity: `SquareType`

**Kind**: Named integer enum  
**Package**: `engine`

Identifies the premium-square classification of a board cell. Premium multipliers apply
only when a new tile is placed on the square during the current turn.

```go
type SquareType int

const (
    Normal       SquareType = iota // no multiplier
    DoubleLetter                   // letter score ×2
    TripleLetter                   // letter score ×3
    DoubleWord                     // word score ×2 (includes Centre)
    TripleWord                     // word score ×3
    Centre                         // word score ×2; first move must cover this square
)
```

---

## Entity: `Tile`

**Kind**: Value struct  
**Package**: `engine`

Represents a single game tile, either in a rack or placed on the board.

```go
type Tile struct {
    Letter         byte  // uppercase A-Z; 0 = blank (unassigned)
    Points         int   // face value (0 for blanks)
    IsBlank        bool  // true for blank tiles
    AssignedLetter byte  // set when a blank is played; 0 in rack
}
```

**Invariants**:
- `IsBlank == true` ⟹ `Points == 0` always
- `IsBlank == true && AssignedLetter != 0` ⟹ tile is placed on the board with a chosen letter
- `IsBlank == true && AssignedLetter == 0` ⟹ blank tile still in rack (letter not yet chosen)
- `IsBlank == false` ⟹ `Letter` is 'A'–'Z' and `Points` matches the standard tile value table

---

## Entity: `PlacedTile`

**Kind**: Value struct  
**Package**: `engine`

A `Tile` paired with its board coordinates. Used in `PlayMove` and by the AI generator.

```go
type PlacedTile struct {
    Tile     Tile
    Row, Col int  // 0-indexed; row 0 = top, col 0 = left
}
```

---

## Entity: `Cell`

**Kind**: Value struct  
**Package**: `engine`

A single cell on the 15×15 board. `Tile` is nil when the cell is unoccupied.

```go
type Cell struct {
    Tile   *Tile      // nil if empty
    Square SquareType
}
```

---

## Entity: `Board`

**Kind**: Pointer-to-struct  
**Package**: `engine`

The 15×15 playing grid. Constructed by `NewBoard()` which applies the standard
premium-square layout. All coordinates are `(row, col)` with row 0 at the top.

```go
type Board struct {
    cells [15][15]Cell  // unexported; accessed via methods
}
```

**Premium square layout** (0-indexed, enforced in `NewBoard`):

| Type         | Coordinates                                                                           |
|---|---|
| TripleWord   | (0,0) (0,7) (0,14) (7,0) (7,14) (14,0) (14,7) (14,14)                               |
| Centre/DW    | (7,7)                                                                                 |
| DoubleWord   | (1,1)(2,2)(3,3)(4,4)(10,10)(11,11)(12,12)(13,13) and their reflections               |
| TripleLetter | (1,5)(1,9)(5,1)(5,5)(5,9)(5,13)(9,1)(9,5)(9,9)(9,13)(13,5)(13,9)                    |
| DoubleLetter | (0,3)(0,11)(2,6)(2,8)(3,0)(3,7)(3,14)(6,2)(6,6)(6,8)(6,12)(7,3)(7,11) and reflections |

---

## Entity: `Bag`

**Kind**: Pointer-to-struct  
**Package**: `engine`

The tile draw bag. Initialised with the standard 100-tile North American English
distribution and shuffled using `rng` at construction.

```go
type Bag struct {
    tiles []Tile  // current remaining tiles (unexported)
}
```

**Standard tile distribution** (total = 100):

| Letter | Count | Points | Letter | Count | Points |
|--------|-------|--------|--------|-------|--------|
| Blank  | 2     | 0      | N      | 6     | 1      |
| A      | 9     | 1      | O      | 8     | 1      |
| B      | 2     | 3      | P      | 2     | 3      |
| C      | 2     | 3      | Q      | 1     | 10     |
| D      | 4     | 2      | R      | 6     | 1      |
| E      | 12    | 1      | S      | 4     | 1      |
| F      | 2     | 4      | T      | 6     | 1      |
| G      | 3     | 2      | U      | 4     | 1      |
| H      | 2     | 4      | V      | 2     | 4      |
| I      | 9     | 1      | W      | 2     | 4      |
| J      | 1     | 8      | X      | 1     | 8      |
| K      | 1     | 5      | Y      | 2     | 4      |
| L      | 4     | 1      | Z      | 1     | 10     |
| M      | 2     | 3      |        |       |        |

---

## Entity: `Rack`

**Kind**: Pointer-to-struct  
**Package**: `engine`

A player's hand of up to 7 tiles.

```go
type Rack struct {
    tiles []Tile  // current tiles (unexported); len ≤ 7
}
```

---

## Entity: `Turn`

**Kind**: Named integer enum  
**Package**: `engine`

```go
type Turn int
const (
    HumanTurn Turn = iota
    AITurn
)
```

---

## Entity: `EndReason`

**Kind**: Named integer enum  
**Package**: `engine`

```go
type EndReason int
const (
    NotOver              EndReason = iota
    RackExhausted                  // a player emptied rack while bag is empty
    SixConsecutivePasses           // ConsecutivePasses reached 6
)
```

---

## Entity: `GameState`

**Kind**: Pointer-to-struct  
**Package**: `engine`

The canonical, single source of truth for all mutable game data. All mutations must
go through a `Command.Execute` call; all reversals through `Command.Undo`.

```go
type GameState struct {
    Board             *Board
    HumanRack         *Rack
    AIRack            *Rack
    Bag               *Bag
    HumanScore        int
    AIScore           int
    ConsecutivePasses int
    CurrentTurn       Turn
    MoveNumber        int             // increments on each Execute; 0 before first move
    LastHumanCommand  Command         // nil before human's first move; set by PlayCommand/ExchangeCommand/PassCommand
    LastAICommand     Command         // nil before AI's first move; set by AI's command
    DictName          dictionary.DictName
    AILevel           int             // 1–10
}
```

---

## Entity: `Move` (interface)

**Kind**: Marker interface  
**Package**: `engine`

All move types implement this interface. The marker method prevents accidental use of
non-move types where a move is expected.

```go
type Move interface{ moveMarker() }
```

---

## Entity: `PlayMove`

**Kind**: Value struct; implements `Move`  
**Package**: `engine`

Represents placing one or more tiles on the board. `WordsFormed` and `Score` are
populated by `ValidatePlacement` and `Score` before the command is executed.

```go
type PlayMove struct {
    Placed      []PlacedTile // tiles placed this turn, in left-to-right / top-to-bottom order
    WordsFormed []string     // all words formed (main word + cross-words), uppercase
    Score       int          // total score for this move including all multipliers and bingo
}
```

---

## Entity: `ExchangeMove`

**Kind**: Value struct; implements `Move`  
**Package**: `engine`

Represents returning tiles to the bag and drawing replacements.

```go
type ExchangeMove struct {
    Tiles []Tile // tiles returned to the bag
}
```

---

## Entity: `PassMove`

**Kind**: Value struct; implements `Move`  
**Package**: `engine`

Represents skipping a turn without placing or exchanging tiles.

```go
type PassMove struct{}
```

---

## Entity: `Command` (interface)

**Kind**: Interface  
**Package**: `engine`

The sole mechanism for mutating `GameState`. Every mutation has an inverse.

```go
type Command interface {
    Execute(state *GameState, rng *rand.Rand) error
    Undo(state *GameState)
}
```

---

## Entities: `PlayCommand`, `ExchangeCommand`, `PassCommand`

**Kind**: Pointer-to-struct; each implements `Command`  
**Package**: `engine`

Each command stores enough data at construction/Execute time to fully reverse the
operation via `Undo`.

```go
// PlayCommand executes a PlayMove and can reverse it.
type PlayCommand struct {
    Move           PlayMove
    drawnTiles     []Tile   // tiles drawn from bag during Execute (stored for Undo)
    prevPasses     int      // ConsecutivePasses value before Execute (stored for Undo)
}

// ExchangeCommand executes an ExchangeMove and can reverse it.
type ExchangeCommand struct {
    Move           ExchangeMove
    drawnTiles     []Tile   // tiles drawn from bag during Execute (stored for Undo)
    prevPasses     int      // ConsecutivePasses value before Execute
}

// PassCommand executes a PassMove and can reverse it.
type PassCommand struct {
    prevPasses int // ConsecutivePasses value before Execute
}
```
