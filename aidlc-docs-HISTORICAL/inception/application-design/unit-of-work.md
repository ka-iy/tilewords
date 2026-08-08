# Units of Work — Squabble

> **Correction addendum** — Some choices below changed during implementation — notably
> "Squabble" → **TileWords**, **Ebitengine → Fyne**, and the dictionary set
> (`enable`/`wordnik`/`atebits-letterpress`). See `aidlc-docs/corrections.md` for the
> authoritative corrections, and `aidlc-docs/aidlc-state.md` for post-v1 additions.

## Overview

The system is decomposed into **5 units**, developed strictly sequentially. Each unit must have all code written and all tests passing before the next unit begins.

```
Unit 1: dictionary  -->  Unit 2: engine  -->  Unit 3: ai  -->  Unit 4: ui  -->  Unit 5: cmd
```

---

## Unit 1: `dictionary`

**Package path**: `dictionary/`
**Build tool**: `tools/buildgaddag/` (part of this unit)

**Description**: GADDAG word graph data structure, word validation, multi-dictionary loading, and the offline build tool that produces pre-built `.gob` files from raw word lists.

**Responsibilities**:
- `tools/buildgaddag/` — CLI tool that reads raw word list `.txt` files (one word per line), builds a GADDAG, serialises it to `.gob`, and writes to `assets/dictionaries/`. Run via `go generate ./dictionary/...`.
- `dictionary/gaddag.go` — GADDAG struct: deserialise from `.gob`, root access, `Successor`, `IsTerminal`
- `dictionary/dictionary.go` — `Dictionary` wrapper: `Name`, `WordCount`, `Validate`, `GADDAG` accessor
- `dictionary/loader.go` — `Load(names ...DictName)`: selects one or more dicts, produces a combined `Dictionary` with deduplication when multiple names requested
- `dictionary/names.go` — `DictName` constants (csw, sowpods, ospd, naspa, otcwl, all)
- Raw word list `.txt` source files (not committed — developer-sourced) and resulting `.gob` files (committed to `assets/dictionaries/`)

**Deliverables**:
- `tools/buildgaddag/main.go`
- `dictionary/*.go` (gaddag, dictionary, loader, names, doc)
- `assets/dictionaries/csw.gob`, `sowpods.gob`, `ospd.gob`, `naspa.gob`, `otcwl.gob`, `all.gob`
- `dictionary/dictionary_test.go` (unit tests + PBT)

**Done when**: `go test ./dictionary/...` passes; all 6 `.gob` files present and loadable; `dictionary.Validate` correctly accepts and rejects known words for all 6 dictionary configurations.

---

## Unit 2: `engine`

**Package path**: `engine/`

**Description**: All Scrabble game rules, board management, tile bag, player racks, scoring, game state, and the command/inverse-command undo system.

**Responsibilities**:
- `engine/board.go` — `Board` (15×15 `Cell` grid), `Cell`, `SquareType`, `NewBoard` with standard premium layout
- `engine/tile.go` — `Tile` struct, standard 100-tile NA English distribution constant
- `engine/bag.go` — `Bag`, `NewBag`, `Draw`, `Return`
- `engine/rack.go` — `Rack`, `Remove`, `Add`, `Replenish`
- `engine/move.go` — `Move` interface, `PlayMove`, `ExchangeMove`, `PassMove`, `PlacedTile`
- `engine/command.go` — `Command` interface, `PlayCommand`, `ExchangeCommand`, `PassCommand` (each stores enough data to `Undo`)
- `engine/scorer.go` — `Score(board, move)`: letter multipliers → word multipliers → bingo bonus → cross-word sums
- `engine/rules.go` — `ValidatePlacement`, `IsGameOver`, `ApplyEndgameScoring`
- `engine/state.go` — `GameState`, `Turn`, `EndReason`, `New`

**Deliverables**:
- `engine/*.go` (all files above + doc.go)
- `engine/engine_test.go` (unit tests + PBT)

**Done when**: `go test ./engine/...` passes; scorer correctly handles all premium combinations and bingo; `ValidatePlacement` correctly rejects disconnected, uncovered-centre, and invalid-cross-word placements; command execute/undo round-trips restore state exactly.

---

## Unit 3: `ai`

**Package path**: `ai/`

**Description**: Appel-Jacobson (1998) move generator using GADDAG traversal, 10-level difficulty model, and async worker goroutine.

**Responsibilities**:
- `ai/generator.go` — `GenerateMoves`: implements the GADDAG left-extension algorithm from the 1998 paper; returns all valid `MoveCandidate` values sorted by score descending
- `ai/difficulty.go` — `SelectMove`: level 1 = uniform random; level 10 = highest score, tie-break by `OpponentAccess` ascending; levels 2–9 = top-k percentile selection
- `ai/player.go` — `AIPlayer.ChooseMove`: orchestrates generator + difficulty; falls back to `ExchangeMove` or `PassMove` if no play moves available
- `ai/worker.go` — `AIWorker`: goroutine + buffered channels; `Request` sends computation task; `Poll` returns result non-blocking

**Deliverables**:
- `ai/*.go` (generator, difficulty, player, worker, doc)
- `ai/ai_test.go` (unit tests + PBT; oracle tests comparing generator against brute-force for small boards)

**Done when**: `go test ./ai/...` passes; generator produces the same valid-move set as a reference brute-force implementation on test boards; level 10 always returns the highest-scoring move; AI worker correctly communicates results without data races (`go test -race ./ai/...` clean).

---

## Unit 4: `ui`

**Package path**: `ui/`

**Description**: Ebitengine game loop, all game screens, board and rack rendering, player input (drag-and-drop + touch), and local save/load persistence.

**Responsibilities**:
- `ui/game.go` — `Game` struct implementing `ebiten.Game`; owns `currentScreen Screen`
- `ui/screen.go` — `Screen` interface
- `ui/mainmenu.go` — `MainMenuScreen`: New Game / Resume Game / Quit; disables Resume when no save
- `ui/setup.go` — `SetupScreen`: dictionary picker (6 options), difficulty slider (1–10), Start button
- `ui/gamescreen.go` — `GameScreen`: `GameOrchestrator`; human input → validate → command → state; AI worker polling; undo; save trigger
- `ui/endgame.go` — `EndGameScreen`: final scores, end condition message, New Game / Quit
- `ui/board_renderer.go` — `BoardRenderer`: draws board image + tile overlays + staged tiles
- `ui/rack_renderer.go` — `RackRenderer`: draws human rack (interactive) and AI rack (display, always fully visible)
- `ui/score_panel.go` — `ScorePanel`: scores, bag count, turn indicator, dict/level labels
- `ui/input_handler.go` — `InputHandler`: mouse drag-and-drop, click-to-place, Android touch events
- `ui/save_manager.go` — `SaveManager`: gob encode/decode; platform-appropriate path; atomic write; user-only permissions; corrupt-file detection

**Deliverables**:
- `ui/*.go` (all files above + doc)
- `assets/images/` — open-licensed board and tile images (sourced and attributed in CREDITS.md)
- `CREDITS.md` — asset attributions + Hasbro non-affiliation disclaimer
- `ui/ui_test.go`

**Done when**: `go test ./ui/...` passes; game is playable end-to-end on Linux desktop (primary dev target); AI rack always visible; undo restores state correctly; save/load round-trips without data loss.

---

## Unit 5: `cmd`

**Package path**: `cmd/squabble/`

**Description**: Application entry point, dependency wiring, and platform-specific build configuration for all target platforms.

**Responsibilities**:
- `cmd/squabble/main.go` — constructs `SaveManager`, `AIPlayer`, `AIWorker`, `dictionary.Loader`, wires into `ui.Game`, calls `ebiten.RunGame`
- `cmd/squabble/main_mobile.go` — `//go:build android` build tag; mobile entry point via `ebitenmobile`
- Platform build scripts:
  - `Makefile` — targets: `build-linux`, `build-windows`, `build-macos`, `build-android`, `build-wasm` (optional), `test`, `generate`
  - `AndroidManifest.xml` + Gradle config (or `ebitenmobile` wrapper)
  - Cross-compilation notes in `BUILD.md`
- `go.mod` — module declaration, all dependency versions pinned

**Deliverables**:
- `cmd/squabble/main.go`, `main_mobile.go`
- `Makefile`
- `BUILD.md`
- `go.mod`, `go.sum` (final, with all dependencies pinned)
- `CREDITS.md` (finalised)

**Done when**: `make build-linux` produces a runnable binary; `make build-android` produces a deployable `.apk` or `.aab`; `make test` runs `go test ./...` clean with `-race`; `go vet ./...` reports no issues.

---

## Development Sequence

```
Week 1:  Unit 1 — dictionary + build tool
Week 2:  Unit 2 — engine
Week 3:  Unit 3 — ai
Week 4:  Unit 4 — ui
Week 5:  Unit 5 — cmd + cross-platform builds
```

*(Timelines are indicative; each unit begins only after the previous unit's tests pass.)*

---

## Post-v1 Additions (retroactively documented)

Work added after the initial five units. Full designs are under `construction/`.

### Unit 6: `defs`

**Package path**: `defs/`
**Build/inspection tools**: `tools/builddefs/`, `tools/defslookup/`, `tools/memcheck/`

**Description**: Word definitions shown during gameplay. Sources meanings from Wiktionary
(kaikki.org `wiktextract`), filters them to the shipped word lists at build time, and resolves
a played word to a definition at runtime via a layered matcher (exact → form-of → stem →
orthographic variant). See `construction/defs/`.

**Deliverables**: `defs/*.go`; `tools/builddefs`, `tools/defslookup`, `tools/memcheck`;
`defs/assets/definitions/definitions.gob.gz` (built locally via `make defs`, gitignored);
`ui/definitions.go`, `ui/tabpanel.go`, `ui/dragscroll.go` (UI integration).

**Done when**: `go test ./defs/...` passes headless; the game shows definitions for played
words when the asset is built, and degrades gracefully when it is not.

### Feature additions to existing units

- **`engine`** — Game modes (`ClassicMode`/`InterestingMode`: mode-parameterised board layout
  + tile economy); Scrabble-coordinate move notation; persisted move records / opening draw.
  See `construction/engine/functional-design/{game-modes, notation-and-move-records}.md`.
- **`ui`** — Move history / Definitions two-tab panel with copy + touch scrolling; game-mode
  selection with a preview dialog; the notation toggle; the single-slot atomic save/resume.
  See `construction/ui/functional-design/{move-history-and-definitions, game-setup-and-modes,
  save-and-resume}.md`.
