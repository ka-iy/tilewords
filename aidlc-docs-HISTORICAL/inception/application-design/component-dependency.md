# Component Dependencies — Squabble

## Dependency Matrix

| Depends on →  | `dictionary` | `engine` | `ai` | `ui` | `assets` |
|---|---|---|---|---|---|
| `dictionary`  |      —       |    No    |  No  |  No  |   Yes    |
| `engine`      |     Yes      |    —     |  No  |  No  |    No    |
| `ai`          |     Yes      |   Yes    |  —   |  No  |    No    |
| `ui`          |     Yes      |   Yes    | Yes  |  —   |   Yes    |
| `cmd/squabble`|     Yes      |   Yes    | Yes  | Yes  |    No    |

**No circular dependencies.** Build order: `dictionary` → `engine` → `ai` → `ui` → `cmd/squabble`

---

## Dependency Detail

### `dictionary` → `assets/dictionaries/`
- Loads pre-built serialised GADDAG binary files via `//go:embed assets/dictionaries/*.gob`
- `Loader.Load` reads the appropriate `.gob` file and deserialises the `GADDAG`
- The combined ("all") GADDAG is also a pre-built file (produced by build tool with deduplication)
- **Communication**: direct `//go:embed` file bytes; no runtime file I/O

### `engine` → `dictionary`
- `engine.Rules.ValidatePlacement` calls `dictionary.Dictionary.Validate` for each word formed
- `engine` receives a `*dictionary.Dictionary` as a parameter to rule functions (dependency injection)
- `engine` does NOT import `dictionary` as a package-level dependency; it uses the `dictionary.Dictionary` type via an interface-compatible pattern (passed at call sites)
  - This keeps `engine` independently testable without real dictionary data

### `ai` → `dictionary`
- `ai.Generator.GenerateMoves` traverses the `dictionary.GADDAG` using `Successor` and `IsTerminal`
- Receives `*dictionary.Dictionary` via `GenerateMoves` parameter (dependency injection)

### `ai` → `engine`
- `ai.Generator` reads `engine.Board` and `engine.Rack` to enumerate moves
- `ai.AIPlayer.ChooseMove` returns an `engine.Move` value
- `ai.MoveCandidate` wraps `engine.PlayMove`
- `ai.DifficultyModel.SelectMove` uses `engine.Scorer` to validate scoring

### `ui` → `engine`
- `ui.GameScreen` holds and mutates `*engine.GameState` exclusively via `engine.Command` implementations
- `ui.BoardRenderer` reads `engine.Board` and `engine.Cell` for rendering
- `ui.RackRenderer` reads `engine.Rack` for rendering
- `ui.SaveManager` encodes/decodes `engine.GameState` using `encoding/gob`

### `ui` → `ai`
- `ui.AIWorker` holds an `*ai.AIPlayer` and calls `AIPlayer.ChooseMove` on its goroutine
- `ui.GameScreen` sends a copy of `engine.GameState` to `AIWorker.Request`

### `ui` → `dictionary`
- `ui.SetupScreen` calls `dictionary.Loader.Load` with the user-selected `DictName`
- Loaded `*dictionary.Dictionary` is stored in `GameState` (or passed alongside it)

### `ui` → `assets/images/`
- Board and tile images loaded once at startup via `//go:embed assets/images/*`
- Stored in `ui` package-level variables as `*ebiten.Image` after decoding

### `cmd/squabble` → all
- Constructs top-level objects (`SaveManager`, `AIPlayer`, `Game`) and calls `ebiten.RunGame`

---

## Data Flow: Human Move

```
InputHandler (ui)
    |-- staged []PlacedTile
    v
GameScreen.Update (ui)
    |-- engine.Rules.ValidatePlacement(board, move, dict)
    |       |-- dictionary.Dictionary.Validate(word)   [per word formed]
    |-- engine.Scorer.Score(board, move)
    |-- engine.PlayCommand.Execute(state, rng)
    |       |-- engine.Board.Place(row, col, tile)
    |       |-- engine.Rack.Remove(tiles)
    |       |-- engine.Rack.Replenish(bag, rng)
    |-- engine.Rules.IsGameOver(state)
    |-- AIWorker.Request(stateCopy, dict, level)       [if not game over]
```

## Data Flow: AI Move

```
AIWorker goroutine
    |-- ai.Generator.GenerateMoves(board, rack, dict)
    |       |-- dictionary.GADDAG.Successor(node, letter)   [traversal]
    |       |-- engine.Scorer.Score(board, candidate)
    |-- ai.DifficultyModel.SelectMove(candidates, level, rng)
    |-- result --> channel

GameScreen.Update (ui) [next frame]
    |-- AIWorker.Poll() --> engine.Move
    |-- engine.PlayCommand.Execute(state, rng)  [or ExchangeCommand/PassCommand]
    |-- engine.Rules.IsGameOver(state)
```

---

## Communication Patterns

| Link | Pattern | Notes |
|---|---|---|
| `ui` ↔ `ai` goroutine | Go channels (buffered, size 1) | Non-blocking poll in game loop |
| `ui` → `engine` state changes | Command pattern (Execute/Undo) | Only mutation path for GameState |
| `ai` → `engine` reads | Direct struct field reads | Read-only; safe since AI gets a copy of GameState |
| `dictionary` → callers | Method calls (stateless after Load) | Thread-safe reads after construction |
| `ui` → `assets` | `//go:embed` byte slices | Decoded once at startup |
