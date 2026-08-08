# Application Design — Squabble (Consolidated)

> **Correction addendum** — Some choices below changed during implementation — notably
> "Squabble" → **TileWords**, **Ebitengine → Fyne**, and the dictionary set
> (`enable`/`wordnik`/`atebits-letterpress`). See `aidlc-docs/corrections.md` for the
> authoritative corrections, and `aidlc-docs/aidlc-state.md` for post-v1 additions.

## Design Decisions Summary

| Decision | Choice | Rationale |
|---|---|---|
| Module structure | Single `go.mod`, top-level packages | Simplest for a monorepo game; no need for separate versioning |
| Undo mechanism | Command/inverse-command pattern | Avoids full state copies; minimal memory overhead for 1-level undo |
| AI concurrency | Goroutine + channel, non-blocking poll | Keeps Ebitengine game loop at ≥30 fps regardless of AI computation time |
| Save format | `encoding/gob` binary | Compact, fast, Go-native; no external dependency |
| Blank tile representation | Wildcard byte `0` in rack; letter + flag on board | Matches Appel-Jacobson paper approach; avoids GADDAG explosion |
| Board coordinates | `(row, col)`, row 0 top-left | Standard matrix notation; consistent with paper |
| Dictionary embedding | Pre-built serialised GADDAGs embedded via `//go:embed` | Fast startup; no build-time GADDAG construction on target device |

---

## Package Structure

```
squabble/
├── go.mod                        # module squabble
├── go.sum
├── CLAUDE.md
├── CREDITS.md                    # asset attributions + Hasbro non-affiliation disclaimer
├── cmd/
│   └── squabble/
│       └── main.go               # entry point; wires all packages, calls ebiten.RunGame
├── dictionary/
│   ├── doc.go                    # package-level doc comment
│   ├── gaddag.go                 # GADDAG struct, Load, traversal methods
│   ├── dictionary.go             # Dictionary struct, Validate, Name, WordCount
│   ├── loader.go                 # Loader.Load — selects and merges dicts
│   ├── names.go                  # DictName constants
│   └── dictionary_test.go
├── engine/
│   ├── doc.go
│   ├── board.go                  # Board, Cell, SquareType, NewBoard
│   ├── tile.go                   # Tile, standard tile distribution
│   ├── bag.go                    # Bag, NewBag, Draw, Return
│   ├── rack.go                   # Rack, Remove, Add, Replenish
│   ├── move.go                   # Move interface, PlayMove, ExchangeMove, PassMove
│   ├── command.go                # Command interface, PlayCommand, ExchangeCommand, PassCommand
│   ├── scorer.go                 # Score function — multipliers, bingo
│   ├── rules.go                  # ValidatePlacement, IsGameOver, ApplyEndgameScoring
│   ├── state.go                  # GameState, Turn, EndReason, New
│   └── engine_test.go
├── ai/
│   ├── doc.go
│   ├── generator.go              # GenerateMoves — GADDAG left-extension algorithm
│   ├── difficulty.go             # DifficultyModel.SelectMove — level 1-10 interpolation
│   ├── player.go                 # AIPlayer.ChooseMove
│   ├── worker.go                 # AIWorker — goroutine, channels, Request/Poll
│   └── ai_test.go
├── ui/
│   ├── doc.go
│   ├── game.go                   # Game struct — ebiten.Game implementation
│   ├── screen.go                 # Screen interface
│   ├── mainmenu.go               # MainMenuScreen
│   ├── setup.go                  # SetupScreen — dict + difficulty selection
│   ├── gamescreen.go             # GameScreen — GameOrchestrator
│   ├── endgame.go                # EndGameScreen
│   ├── board_renderer.go         # BoardRenderer
│   ├── rack_renderer.go          # RackRenderer
│   ├── score_panel.go            # ScorePanel
│   ├── input_handler.go          # InputHandler — drag-and-drop, click-to-place, touch
│   ├── save_manager.go           # SaveManager — gob encode/decode, platform path
│   └── ui_test.go
└── assets/
    ├── dictionaries/
    │   ├── csw.gob               # pre-built GADDAG for CSW
    │   ├── sowpods.gob
    │   ├── ospd.gob
    │   ├── naspa.gob
    │   ├── otcwl.gob
    │   └── all.gob               # pre-built combined GADDAG (deduplicated)
    └── images/
        ├── board.png             # 15x15 board background
        ├── tile_*.png            # tile images A-Z + blank (open-licensed)
        └── ui_*.png              # button and UI element images
```

---

## Component Responsibilities (Summary)

| Package | Core Responsibility |
|---|---|
| `dictionary` | GADDAG construction, load, word validation, traversal API |
| `engine` | Board, bag, rack, scoring, rules, game state, command/undo |
| `ai` | Move enumeration (Appel-Jacobson), difficulty interpolation, async worker |
| `ui` | Ebitengine game loop, all screens, rendering, input, save/load |
| `assets` | Embedded binary GADDAGs and open-licensed game images |
| `cmd/squabble` | Wiring and entry point only; minimal logic |

---

## Key Interfaces

```go
// engine.Move — all move types implement this marker interface
type Move interface{ moveMarker() }

// engine.Command — execute/undo pattern for all moves
type Command interface {
    Execute(state *GameState, rng *rand.Rand) error
    Undo(state *GameState)
}

// ui.Screen — all game screens implement this
type Screen interface {
    Update(g *Game) (next Screen, err error)
    Draw(screen *ebiten.Image, g *Game)
}
```

---

## Security & NFR Cross-Cutting Notes

- **SECURITY-03**: Structured logging initialised in `cmd/squabble/main.go`; all packages receive a `*slog.Logger` via dependency injection (no global logger).
- **SECURITY-09**: Error responses in UI show generic messages; internal errors logged only, never displayed.
- **SECURITY-15**: All external I/O (file read/write) wrapped in explicit error handling; resources closed via `defer`.
- **NFR-10**: Every exported symbol has a GoDoc comment; algorithm-critical sections in `dictionary/gaddag.go`, `ai/generator.go`, `engine/scorer.go` include inline comments referencing the Appel-Jacobson paper.
- **PBT-09**: `rapid` added as a test dependency; all packages with business logic include property-based tests.
- **NFR-09**: The word "Scrabble" appears nowhere in package names, exported symbols, UI strings, or asset file names.
