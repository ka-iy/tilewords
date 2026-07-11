# Code Generation Plan — Unit 2: `engine`

## Unit Context
- **Workspace root**: `/home/kartik/PROGS/SQUABBLE-Scrabble_Vibe_coded`
- **Package path**: `engine/`
- **Stories**: US-06, US-08, US-09, US-10, US-11, US-14, US-17, US-18
- **Dependencies**: `squabble/dictionary` (for `ValidatePlacement`), `math/rand` (stdlib)
- **Project type**: Greenfield single-module monolith

## Stories Implemented
- [x] US-06 (Tile bag) — `NewBag`, standard 100-tile NA English distribution
- [x] US-08 (Word validation) — `ValidatePlacement` delegates to `dictionary.Dictionary`
- [x] US-09 (Move scoring) — `Score`, letter/word multipliers, bingo bonus
- [x] US-10 (Game rules) — end conditions, end-game scoring, exchange rules
- [x] US-11 (Undo) — `PlayCommand.Undo`, `ExchangeCommand.Undo`, `UndoLastRound`
- [x] US-14 (Save/load types) — exported `GameState` fields; `Board.GobEncode`/`GobDecode`
- [x] US-17 (AI rack visibility) — `GameState.AIRack` always accessible via `engine.New`
- [x] US-18 (Tournament rules) — bingo, 6-pass, rack exhaustion, end-game scoring

---

## Step 1: Package Doc — `engine/doc.go`
- [ ] Package declaration with GoDoc comment:
  - Purpose of the package (game rules, board state, scoring, command/undo)
  - Reference to command/inverse-command pattern with undo semantics
  - Note on thread safety: `GameState` owned by UI goroutine; AI receives `Clone()`
  - Brief usage example: `New` → `PlayCommand.Execute` → `IsGameOver`

## Step 2: Tile Types — `engine/tile.go`
- [ ] Package `engine`
- [ ] `Tile` struct (Letter, Points, IsBlank, AssignedLetter) with GoDoc on all fields
- [ ] `PlacedTile` struct (Tile, Row, Col) with GoDoc
- [ ] `tileDistribution` — unexported package-level var: `[]struct{ letter byte; points, count int }` for all 27 entries (blank + A–Z), with comment referencing the standard 100-tile NA English distribution and asserting total = 100

## Step 3: Board — `engine/board.go`
- [ ] `SquareType` named int enum: `Normal`, `DoubleLetter`, `TripleLetter`, `DoubleWord`, `TripleWord`, `Centre` with GoDoc on each constant
- [ ] `Cell` struct (Tile *Tile, Square SquareType) with GoDoc
- [ ] `Board` struct with unexported `cells [15][15]Cell`
- [ ] `NewBoard() *Board` — allocates and populates premium square layout; hard-coded coordinate table with comment referencing the standard layout and noting the symmetric structure
- [ ] Premium square coordinate table as a package-level unexported `var` (slice of `struct{ row, col int; sq SquareType }`) — one entry per premium cell; allows inspection in tests
- [ ] `(b *Board) Cell(row, col int) Cell` — panics with message on out-of-bounds
- [ ] `(b *Board) Place(row, col int, tile Tile) error` — returns error if occupied
- [ ] `(b *Board) Remove(row, col int)` — no-op if empty
- [ ] `(b *Board) IsEmpty(row, col int) bool`
- [ ] `(b *Board) Clone() *Board` — copies `[15][15]Cell` by array value; see Pattern 2 note on `*Tile` pointers (shallow copy is safe for read-only AI use; comment explains why)
- [ ] `(b *Board) GobEncode() ([]byte, error)` and `(b *Board) GobDecode([]byte) error` using `boardWire` exported wire struct

## Step 4: Bag — `engine/bag.go`
- [ ] `Bag` struct with unexported `tiles []Tile`
- [ ] `NewBag(rng *rand.Rand) *Bag` — builds tiles from `tileDistribution`, Fisher-Yates shuffle with comment citing algorithm name
- [ ] `NewTestBag(tiles []Tile) *Bag` — test helper: no shuffle, exact order; exported so `engine_test.go` can use it from within `package engine` (internal test)
- [ ] `(b *Bag) Draw(n int) []Tile` — pops from end; returns `min(n, len)` tiles
- [ ] `(b *Bag) Return(tiles []Tile, rng *rand.Rand)` — appends; if rng non-nil, reshuffles; nil rng = no reshuffle (used by `PlayCommand.Undo`)
- [ ] `(b *Bag) Count() int`
- [ ] `(b *Bag) Clone() *Bag` — deep copy of `tiles` slice

## Step 5: Rack — `engine/rack.go`
- [ ] `Rack` struct with unexported `tiles []Tile`
- [ ] `(r *Rack) Tiles() []Tile` — returns a copy (not the underlying slice)
- [ ] `(r *Rack) Add(tiles []Tile) error` — returns error if adding exceeds cap 7
- [ ] `(r *Rack) Remove(tiles []Tile) error` — removes first match for each tile; returns error if any tile not found
- [ ] `(r *Rack) Replenish(bag *Bag, rng *rand.Rand) []Tile` — draws `min(7-count, bag.Count())` tiles; returns drawn tiles (needed by `PlayCommand` to record `drawnTiles`)
- [ ] `(r *Rack) Count() int`
- [ ] `(r *Rack) Clone() *Rack` — deep copy

## Step 6: Move Types — `engine/move.go`
- [ ] `Move` interface with unexported `moveMarker()` method; GoDoc explaining the marker pattern
- [ ] `PlayMove` struct (Placed []PlacedTile, WordsFormed []string, Score int) implementing `Move`
- [ ] `ExchangeMove` struct (Tiles []Tile) implementing `Move`
- [ ] `PassMove` struct implementing `Move`

## Step 7: Game State — `engine/state.go`
- [ ] `Turn` named int enum: `HumanTurn`, `AITurn` with GoDoc
- [ ] `EndReason` named int enum: `NotOver`, `RackExhausted`, `SixConsecutivePasses` with GoDoc
- [ ] `GameState` struct with all fields (exported for gob) and GoDoc on each field; comment on `LastHumanCommand`/`LastAICommand` being excluded from save
- [ ] `New(dictName dictionary.DictName, aiLevel int, rng *rand.Rand) *GameState` — full initialisation with coin-flip for first turn; comment on BR-E19
- [ ] `(s *GameState) Clone() *GameState` — deep copy excluding command fields; comment explains why commands are omitted

## Step 8: Rules — `engine/rules.go`
- [ ] `ValidatePlacement(board *Board, move *PlayMove, dict *dictionary.Dictionary) ([]string, error)` — full 7-step algorithm from BL-E04; inline comments for each step referencing the BR number
- [ ] `IsGameOver(state *GameState) (bool, EndReason)` — BL-E10
- [ ] `ApplyEndgameScoring(state *GameState)` — BL-E11; comment explains redistribution vs. mutual-reduction cases
- [ ] `isHorizontal(placed []PlacedTile) bool` — unexported helper
- [ ] `covers(placed []PlacedTile, row, col int) bool` — unexported: checks if a position is in the placed set

## Step 9: Scoring — `engine/score.go`
- [ ] `Score(board *Board, move *PlayMove) (int, error)` — BL-E06; inline comments for letter multiplier, word multiplier accumulation, bingo check
- [ ] `extractWords(board *Board, move *PlayMove) []string` — BL-E05; inline comments for main-word extension and cross-word detection
- [ ] `virtualCell(board *Board, placed []PlacedTile, row, col int) *Tile` — unexported: returns tile from placed set first, then board
- [ ] `isNewTile(placed []PlacedTile, row, col int) bool` — unexported: checks if (row,col) is in placed set
- [ ] `sumWord(board *Board, placed []PlacedTile, positions [][2]int) (wordScore, wordMult int)` — unexported: computes raw word score and accumulated word multiplier for a sequence of positions

## Step 10: Commands — `engine/commands.go`
- [ ] `Command` interface (Execute, Undo) with GoDoc naming the inverse-command pattern
- [ ] `PlayCommand` struct (exported Move PlayMove; unexported drawnTiles, prevPasses) — GoDoc on each field explaining undo role
- [ ] `PlayCommand.Execute` — BL-E07; comment each step
- [ ] `PlayCommand.Undo` — BL-E07; comment each reversal step
- [ ] `ExchangeCommand` struct (exported Move ExchangeMove; unexported drawnTiles, bagSnapshot, prevPasses) — GoDoc explaining why bagSnapshot is needed
- [ ] `ExchangeCommand.Execute` — BL-E08
- [ ] `ExchangeCommand.Undo` — BL-E08; comment: "restore bag from snapshot taken before reshuffle"
- [ ] `PassCommand` struct (unexported prevPasses)
- [ ] `PassCommand.Execute` — BL-E09
- [ ] `PassCommand.Undo` — BL-E09
- [ ] `UndoLastRound(state *GameState)` — BL-E12; comment on FR-09 and ordering (AI undo first)
- [ ] `opposite(t Turn) Turn` — unexported helper

## Step 11: Tests — `engine/testhelpers_test.go`, `engine/engine_test.go`, `engine/engine_pbt_test.go`

### `engine/testhelpers_test.go`
- [ ] `package engine` (whitebox — same package for unexported access)
- [ ] `TestMain` — builds a small test GADDAG via `dictionary.Build`; wraps in `dictionary.Dictionary` test double for use across tests
- [ ] `newTestDict() *dictionary.Dictionary` — builds GADDAG from curated word set; reused by all validation tests
- [ ] `newFlatBoard() *Board` — board with all `Normal` squares (no premium multipliers); used for pure face-value scoring tests
- [ ] `newGameState(rng) *GameState` — convenience: `engine.New` with fixed dict/level/seed

### `engine/engine_test.go`
- [ ] `TestNewBag_Count` — 100 tiles
- [ ] `TestNewBag_Distribution` — correct count per letter
- [ ] `TestBag_Draw_Replenish` — draw 7, return 7, count restored
- [ ] `TestBoard_PremiumLayout` — spot-check known TWS/DWS/TLS/DLS positions
- [ ] `TestBoard_PlaceRemove` — place, verify non-empty, remove, verify empty
- [ ] `TestBoard_Place_OccupiedError` — placing on occupied cell returns error
- [ ] `TestBoard_GobRoundTrip` — encode → decode → identical cell grid
- [ ] `TestValidatePlacement_FirstMove_Centre` — valid first move covering (7,7)
- [ ] `TestValidatePlacement_FirstMove_MissesCentre` — error
- [ ] `TestValidatePlacement_Gap` — error when gap in placed tiles
- [ ] `TestValidatePlacement_NotAdjacent` — second move not adjacent to any tile
- [ ] `TestValidatePlacement_InvalidWord` — word rejected by dict
- [ ] `TestValidatePlacement_CrossWordFormed` — correctly identifies and validates cross-word
- [ ] `TestScore_FaceValues` — flat board, no multipliers: score = sum of face values
- [ ] `TestScore_DoubleLetter` — DL square doubles one tile's value
- [ ] `TestScore_TripleWord` — TW square triples word total
- [ ] `TestScore_BingoBonus` — 7-tile play adds 50
- [ ] `TestScore_BeforeValidate_Error` — calling Score before ValidatePlacement returns error
- [ ] `TestPlayCommand_ExecuteUndo` — execute play, verify state; undo, verify restored
- [ ] `TestExchangeCommand_ExecuteUndo` — execute exchange, verify tiles swapped; undo, verify restored
- [ ] `TestPassCommand_ExecuteUndo` — ConsecutivePasses increments then restores
- [ ] `TestIsGameOver_SixPasses` — 6 consecutive passes triggers end
- [ ] `TestIsGameOver_RackExhausted` — empty rack + empty bag triggers end
- [ ] `TestApplyEndgameScoring_RackExhausted` — emptier gains other player's tile sum
- [ ] `TestApplyEndgameScoring_SixPasses` — both players lose own tile values
- [ ] `TestGameState_Clone_Independent` — mutating clone does not affect original

### `engine/engine_pbt_test.go`
- [ ] `TestPBT_BagCount` — `NewBag(rng).Count() == 100` (PBT-E01)
- [ ] `TestPBT_TileConservation` — after any sequence of Execute calls, bag + racks + board = 100 (PBT-E07)
- [ ] `TestPBT_ScoreNonNegative` — `Score(board, move) ≥ 0` for any valid move (PBT-E03)
- [ ] `TestPBT_BingoBonus` — 7-tile move: Score ≥ face sum + 50 (PBT-E04)
- [ ] `TestPBT_ExecuteUndo_RoundTrip` — for any command, Execute then Undo restores identical GameState (PBT-E06)
- [ ] `TestPBT_ScoreFlat_FaceValues` — on flat board (no premium squares), Score == sum of face values (PBT-E08)
- [ ] `TestPBT_ExistingTilesNoMultiplier` — tiles already on board do not receive premium multipliers (PBT-E05)

## Step 12: Code Documentation Summary
- [ ] Create `aidlc-docs/construction/engine/code/code-summary.md`:
  - All files with one-line description
  - Note on `Board.GobEncode`/`GobDecode` for unexported field serialisation
  - Note on `ExchangeCommand.bagSnapshot` for exact undo
  - Note that `engine` imports `squabble/dictionary` (one-way dependency)

---

## File Manifest

| File | Type | Step |
|---|---|---|
| `engine/doc.go` | Go source | 1 |
| `engine/tile.go` | Go source | 2 |
| `engine/board.go` | Go source | 3 |
| `engine/bag.go` | Go source | 4 |
| `engine/rack.go` | Go source | 5 |
| `engine/move.go` | Go source | 6 |
| `engine/state.go` | Go source | 7 |
| `engine/rules.go` | Go source | 8 |
| `engine/score.go` | Go source | 9 |
| `engine/commands.go` | Go source | 10 |
| `engine/testhelpers_test.go` | Go test | 11 |
| `engine/engine_test.go` | Go test | 11 |
| `engine/engine_pbt_test.go` | Go test | 11 |
| `aidlc-docs/construction/engine/code/code-summary.md` | Documentation | 12 |
