# Component Methods — Squabble

Detailed business logic is defined in Functional Design (CONSTRUCTION phase). This document records method signatures and high-level purpose only.

---

## Package: `dictionary`

### Type: `DictName`
```go
type DictName string
const (
    DictCSW     DictName = "csw"
    DictSOWPODS DictName = "sowpods"
    DictOSPD    DictName = "ospd"
    DictNASPA   DictName = "naspa"
    DictOTCWL   DictName = "otcwl"
    DictAll     DictName = "all"
)
```

### Type: `NodeID`
```go
type NodeID uint32
```
Opaque identifier for a node in the GADDAG graph.

### Type: `GADDAG`
```go
// Load deserialises a GADDAG from gob-encoded bytes produced by the build tool.
func Load(data []byte) (*GADDAG, error)

// Contains reports whether word (uppercase) is present in this GADDAG.
func (g *GADDAG) Contains(word string) bool

// Successor returns the NodeID reached by following edge letter from node,
// and whether such an edge exists.
func (g *GADDAG) Successor(node NodeID, letter byte) (NodeID, bool)

// IsTerminal reports whether node marks the end of a valid word.
func (g *GADDAG) IsTerminal(node NodeID) bool

// Root returns the root NodeID for starting traversal.
func (g *GADDAG) Root() NodeID
```

### Type: `Dictionary`
```go
// Name returns the human-readable dictionary name for display.
func (d *Dictionary) Name() DictName

// WordCount returns the number of unique words in the dictionary.
func (d *Dictionary) WordCount() int

// Validate reports whether word (case-insensitive) is valid in this dictionary.
func (d *Dictionary) Validate(word string) bool

// GADDAG returns the underlying graph for AI move generation traversal.
func (d *Dictionary) GADDAG() *GADDAG
```

### Type: `Loader`
```go
// Load loads the named dictionary (or combined set) from embedded assets.
// If multiple names are provided, their word sets are unioned and deduplicated
// before the combined GADDAG is returned.
func Load(names ...DictName) (*Dictionary, error)
```

---

## Package: `engine`

### Type: `SquareType`
```go
type SquareType int
const (
    Normal, DoubleLetter, TripleLetter, DoubleWord, TripleWord, Centre SquareType = iota
)
```

### Type: `Tile`
```go
type Tile struct {
    Letter         byte  // uppercase A-Z; 0 = blank in rack
    Points         int
    IsBlank        bool
    AssignedLetter byte  // letter chosen when blank is played; 0 if not blank
}
```

### Type: `Cell`
```go
type Cell struct {
    Tile       *Tile      // nil if empty
    Square     SquareType
}
```

### Type: `Board`
```go
// Cell returns the cell at (row, col). Panics if out of bounds.
func (b *Board) Cell(row, col int) Cell

// Place sets the tile at (row, col). Returns error if cell is occupied.
func (b *Board) Place(row, col int, tile Tile) error

// Remove clears the tile at (row, col). Used by undo.
func (b *Board) Remove(row, col int)

// IsEmpty reports whether (row, col) has no tile.
func (b *Board) IsEmpty(row, col int) bool

// Clone returns a deep copy of the board.
func (b *Board) Clone() *Board

// NewBoard returns a 15x15 board initialised with standard premium square layout.
func NewBoard() *Board
```

### Type: `Bag`
```go
// NewBag returns a shuffled standard 100-tile NA English bag.
func NewBag(rng *rand.Rand) *Bag

// Draw removes and returns up to n tiles from the bag.
func (b *Bag) Draw(n int) []Tile

// Return adds tiles back to the bag and reshuffles.
func (b *Bag) Return(tiles []Tile, rng *rand.Rand)

// Count returns the number of tiles remaining.
func (b *Bag) Count() int
```

### Type: `Rack`
```go
// Tiles returns a copy of the rack's current tiles.
func (r *Rack) Tiles() []Tile

// Remove removes the specified tiles from the rack. Returns error if any tile not found.
func (r *Rack) Remove(tiles []Tile) error

// Add appends tiles to the rack (up to 7).
func (r *Rack) Add(tiles []Tile)

// Replenish draws tiles from bag to bring rack to 7 (or bag count, whichever is less).
func (r *Rack) Replenish(bag *Bag, rng *rand.Rand)

// Count returns the current number of tiles on the rack.
func (r *Rack) Count() int
```

### Type: `Move` (interface)
```go
type Move interface{ moveMarker() }
```
Implemented by `PlayMove`, `ExchangeMove`, `PassMove`.

### Type: `PlayMove`
```go
type PlacedTile struct {
    Tile       Tile
    Row, Col   int
}
type PlayMove struct {
    Placed     []PlacedTile  // tiles placed this turn (in board order)
    WordsFormed []string      // populated after validation
    Score      int            // populated after scoring
}
```

### Type: `ExchangeMove`
```go
type ExchangeMove struct {
    Tiles []Tile // tiles returned to bag
}
```

### Type: `PassMove`
```go
type PassMove struct{}
```

### Type: `Command` (interface)
```go
type Command interface {
    // Execute applies the move to state, mutating it.
    Execute(state *GameState, rng *rand.Rand) error
    // Undo reverses the effect of Execute on state.
    Undo(state *GameState)
}
```
Implemented by `PlayCommand`, `ExchangeCommand`, `PassCommand`.

### Type: `Scorer`
```go
// Score calculates the total score for move on board, including all cross-words
// and the bingo bonus. Board must not yet have move's tiles placed.
func Score(board *Board, move *PlayMove) (int, error)
```

### Type: `Rules`
```go
// ValidatePlacement checks that move is legally placeable on board given dict.
// Returns all new words formed (main word + cross-words) or an error.
func ValidatePlacement(board *Board, move *PlayMove, dict *dictionary.Dictionary) ([]string, error)

// IsGameOver reports whether state meets any end condition.
func IsGameOver(state *GameState) (over bool, reason EndReason)

// ApplyEndgameScoring adjusts scores when game ends by rack exhaustion.
func ApplyEndgameScoring(state *GameState)
```

### Type: `GameState`
```go
type Turn int
const (HumanTurn, AITurn Turn = iota)

type EndReason int
const (NotOver, RackExhausted, SixConsecutivePasses EndReason = iota)

type GameState struct {
    Board             *Board
    HumanRack         *Rack
    AIRack            *Rack
    Bag               *Bag
    HumanScore        int
    AIScore           int
    ConsecutivePasses int
    CurrentTurn       Turn
    LastCommand       Command   // non-nil after first move; used for undo
    DictName          dictionary.DictName
    AILevel           int
    MoveNumber        int
}

// New initialises a fresh GameState: shuffled bag, 7 tiles to each rack.
func New(dictName dictionary.DictName, aiLevel int, rng *rand.Rand) *GameState
```

---

## Package: `ai`

### Type: `MoveCandidate`
```go
type MoveCandidate struct {
    Move              engine.PlayMove
    Score             int
    OpponentAccess    int  // count of premium squares opened to opponent; used for tie-breaking
}
```

### Type: `Generator`
```go
// GenerateMoves enumerates all valid play moves for rack on board using the
// GADDAG left-extension algorithm (Appel & Jacobson, 1998).
// Returns candidates sorted by Score descending.
func GenerateMoves(
    board *engine.Board,
    rack *engine.Rack,
    dict *dictionary.Dictionary,
) []MoveCandidate
```

### Type: `DifficultyModel`
```go
// SelectMove chooses a move from candidates according to level (1-10).
// Level 1: uniform random. Level 10: highest score, tie-break by OpponentAccess asc.
// Levels 2-9: select uniformly from the top k candidates by rank percentile.
func SelectMove(candidates []MoveCandidate, level int, rng *rand.Rand) MoveCandidate
```

### Type: `AIPlayer`
```go
// ChooseMove selects the AI's move given current game state and difficulty.
// Returns a PassMove if no valid play or exchange move exists.
func ChooseMove(
    state *engine.GameState,
    dict *dictionary.Dictionary,
    level int,
    rng *rand.Rand,
) engine.Move
```

### Type: `AIWorker`
```go
// Request sends an async move computation request.
// Panics if a request is already in flight.
func (w *AIWorker) Request(state *engine.GameState, dict *dictionary.Dictionary, level int)

// Poll returns the computed move and true if ready; (nil, false) otherwise.
func (w *AIWorker) Poll() (engine.Move, bool)

// NewAIWorker creates a worker with its goroutine and channels initialised.
func NewAIWorker(player *AIPlayer) *AIWorker
```

---

## Package: `ui`

### Type: `Screen` (interface)
```go
type Screen interface {
    Update(g *Game) (next Screen, err error)
    Draw(screen *ebiten.Image, g *Game)
}
```

### Type: `Game`
```go
// Implements ebiten.Game.
func (g *Game) Update() error
func (g *Game) Draw(screen *ebiten.Image)
func (g *Game) Layout(outsideW, outsideH int) (screenW, screenH int)
```

### Type: `BoardRenderer`
```go
// Draw renders the board background, premium square overlays, and all placed tiles.
func (r *BoardRenderer) Draw(dst *ebiten.Image, board *engine.Board, staged []engine.PlacedTile)

// CellAt returns the board (row, col) for a screen pixel coordinate, or (-1,-1) if outside.
func (r *BoardRenderer) CellAt(screenX, screenY int) (row, col int)
```

### Type: `RackRenderer`
```go
// Draw renders a rack of tiles. interactive=true enables drag-and-drop hit detection.
func (r *RackRenderer) Draw(dst *ebiten.Image, rack *engine.Rack, interactive bool)

// TileAt returns the rack index for a screen pixel coordinate, or -1 if none.
func (r *RackRenderer) TileAt(screenX, screenY int) int
```

### Type: `InputHandler`
```go
// Update processes mouse/touch input each frame; updates staged tile positions.
func (h *InputHandler) Update(board *BoardRenderer, rack *RackRenderer, staged *[]engine.PlacedTile)
```

### Type: `SaveManager`
```go
// Save encodes state to gob and writes to SavePath() with user-only permissions.
func (s *SaveManager) Save(state *engine.GameState) error

// Load reads and decodes a saved GameState from SavePath().
// Returns (nil, nil) if no save file exists.
func (s *SaveManager) Load() (*engine.GameState, error)

// Exists reports whether a save file is present.
func (s *SaveManager) Exists() bool

// SavePath returns the platform-appropriate app data file path.
func (s *SaveManager) SavePath() string
```
