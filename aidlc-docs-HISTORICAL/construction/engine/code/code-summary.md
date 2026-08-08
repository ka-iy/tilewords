# Code Summary — Unit 2: `engine`

## Files Created

| File | Description |
|---|---|
| `engine/doc.go` | Package GoDoc — command pattern, thread-safety model (Clone before AI goroutine), usage example |
| `engine/tile.go` | `Tile`, `PlacedTile`, `tileDistribution` (27-entry table, 100 tiles total), `tileDistributionTotal` constant |
| `engine/board.go` | `SquareType` enum, `Cell`, `Board` ([15][15]Cell), `NewBoard` (premium layout from `premiumSquares` table), `newFlatBoard` (test helper), `Clone`, `GobEncode`/`GobDecode` via `boardWire` |
| `engine/bag.go` | `Bag`, `NewBag` (Fisher-Yates shuffle), `NewTestBag` (no shuffle, exact order), `Draw` (O(1) pop from end), `Return` (nil rng = no reshuffle for undo), `restoreSnapshot` (for ExchangeCommand.Undo) |
| `engine/rack.go` | `Rack`, `Add` (cap 7), `Remove` (atomic: all-or-nothing), `Replenish` (returns drawn tiles for undo recording), `Clone`, `tilesMatch` (blank-aware matching) |
| `engine/move.go` | `Move` marker interface, `PlayMove` (Placed/WordsFormed/Score), `ExchangeMove`, `PassMove` |
| `engine/state.go` | `Turn`, `EndReason`, `GameState` (all fields exported for gob), `New` (coin-flip first turn via rng.Intn(2)), `Clone` (deep copy excluding command fields), helpers `currentRack`, `addScore`, `subtractScore`, `opposite` |
| `engine/rules.go` | `ValidatePlacement` (7-step: sanity→orientation→occupancy→contiguity→first-move/adjacency→extractWords→dict), `IsGameOver`, `ApplyEndgameScoring`, helpers `checkContiguity`, `hasAdjacent`, `sumRackPoints` |
| `engine/score.go` | `Score` (letter×mult + word×mult + bingo), `extractWords`, `extractWordPositions`, `mainWordPositions`, `crossWordPositions`, `virtualTile`, `isNewTile`, `covers`, `isHorizontal` |
| `engine/commands.go` | `Command` interface (Execute/Undo), `PlayCommand`, `ExchangeCommand` (with `bagSnapshot`), `PassCommand`, `UndoLastRound` |
| `dictionary/words.go` | `dictionary.NewFromWords` — builds a Dictionary from a word list without embedded assets; used by engine tests |
| `engine/testhelpers_test.go` | `TestMain` builds test dictionary via `dictionary.NewFromWords`; `deterministicRNG`, `newGameState` helpers |
| `engine/engine_test.go` | 28 example-based tests covering bag, board, rack, ValidatePlacement, Score, all three commands, IsGameOver, ApplyEndgameScoring, GameState.Clone |
| `engine/engine_pbt_test.go` | 7 PBT properties: BagCount, TileConservation, ScoreNonNegative, BingoBonus, ScoreFlat=FaceValues, ExecuteUndo round-trip, ExistingTilesNoMultiplier |

## Key Design Decisions

**`Board.GobEncode`/`GobDecode`**: The `cells` field is unexported, which gob silently skips. Custom encode/decode methods using the exported `boardWire` struct ensure the board state is correctly serialised for save/load.

**`ExchangeCommand.bagSnapshot`**: `Bag.Return` reshuffles the bag with an RNG, making the post-execute bag non-deterministically ordered. A pre-execute snapshot of `bag.tiles` enables exact restoration on `Undo` without requiring a deterministic reverse-shuffle.

**`Bag.Return(tiles, nil)`**: Passing `nil` for the RNG skips the reshuffle. This is the path used by `PlayCommand.Undo` to return drawn tiles without altering bag order unnecessarily.

**`Rack.Replenish` returns `[]Tile`**: The drawn tiles are returned so `PlayCommand` can record them for undo without a second call to `Bag.Draw`.

**`dictionary.NewFromWords`**: Added to the dictionary package (not test-only) to enable engine tests to construct a valid `*dictionary.Dictionary` from a curated word list without needing pre-built `.gob` assets.

**Single-tile word length**: `extractWordPositions` requires ≥2 positions to emit a word; `Score` therefore returns 0 for a single isolated tile — consistent with Scrabble rules (words must be ≥2 letters).

## Dependency Note

`engine` imports `squabble/dictionary` for the `ValidatePlacement` function signature. This is a one-way dependency: `dictionary` has no knowledge of `engine`.

## Test Results

```
ok  squabble/dictionary  1.946s
ok  squabble/engine      1.054s
```

Race detector enabled (`-race`). 28 example tests + 7 PBT property checks pass.
