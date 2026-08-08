# Components — Squabble

> **Correction addendum** — Some choices below changed during implementation — notably
> "Squabble" → **TileWords**, **Ebitengine → Fyne**, and the dictionary set
> (`enable`/`wordnik`/`atebits-letterpress`). See `aidlc-docs/corrections.md` for the
> authoritative corrections, and `aidlc-docs/aidlc-state.md` for post-v1 additions.

## Module Structure

Single `go.mod` at repository root. Top-level package directories:

```
squabble/
├── go.mod
├── go.sum
├── cmd/squabble/        # main package — entry point
├── dictionary/          # Unit 1: GADDAG engine and word validation
├── engine/              # Unit 2: Board, rules, scoring, game state, undo
├── ai/                  # Unit 3: Move generation, difficulty model
├── ui/                  # Unit 4: Ebitengine game loop, screens, save/load
└── assets/              # Embedded assets (pre-built GADDAGs, images)
    ├── dictionaries/    # Serialised GADDAG files (one per dict + combined)
    └── images/          # Board and tile images (open-licensed)
```

---

## Component: `dictionary`

**Purpose**: Build, load, and query GADDAG word graph data structures. Provides word validation used by both the rules engine and the AI move generator.

**Responsibilities**:
- Deserialise pre-built GADDAG binary files embedded via `//go:embed`
- Provide O(1) word membership testing via GADDAG traversal
- Expose GADDAG node traversal API for use by the AI move generator
- Represent each of the 5 dictionaries and the combined (deduplicated) set as a named `Dictionary` value

**Key types**: `GADDAG`, `Node`, `NodeID`, `Dictionary`, `DictName`, `Loader`

**Constraints**:
- Read-only after construction; no mutation after `Load`
- Thread-safe for concurrent reads (AI goroutine + validation on main goroutine)
- Pre-built GADDAG binaries produced by a `go generate` build tool (offline step)

---

## Component: `engine`

**Purpose**: Implement all Scrabble game rules, board state management, scoring, tile bag, player racks, and the command/undo system.

**Responsibilities**:
- Maintain the 15×15 board as a grid of `Cell` values (each holding a placed `Tile` and its `SquareType`)
- Represent tiles (letter, point value, blank flag, assigned-letter-when-blank)
- Manage the tile bag (draw, return, count) with the standard 100-tile NA English distribution
- Manage player racks (add, remove, replenish from bag)
- Validate word placement: connectivity, first-move centre rule, cross-word formation
- Score a `PlayMove` using standard multiplier rules (DL, TL, DW, TW, bingo)
- Enforce game-end conditions (rack exhausted + empty bag; six consecutive passes)
- Apply end-game score adjustment (remaining tile redistribution)
- Implement the command/inverse-command undo pattern for `PlayMove`, `ExchangeMove`, `PassMove`
- Hold the canonical `GameState` (board, racks, bag, scores, pass counter, turn indicator, last command)

**Key types**: `Board`, `Cell`, `SquareType`, `Tile`, `Bag`, `Rack`, `GameState`, `Move` (interface), `PlayMove`, `ExchangeMove`, `PassMove`, `Command` (interface), `PlayCommand`, `ExchangeCommand`, `PassCommand`, `Scorer`, `Rules`

**Constraints**:
- Coordinates are `(row, col)`, row 0 = top, col 0 = left
- `GameState` is the single source of truth; no duplicate state elsewhere
- Commands are the only mechanism for mutating `GameState` (enforces undo correctness)

---

## Component: `ai`

**Purpose**: Implement the Appel-Jacobson (1998) move generator and the 10-level difficulty model.

**Responsibilities**:
- Enumerate all valid moves for a given board, rack, and GADDAG using the GADDAG left-extension algorithm
- Score each candidate move using the `engine.Scorer`
- Implement difficulty levels 1–10: level 1 selects uniformly at random from all valid moves; level 10 selects the highest-scoring move (ties broken by minimising opponent premium-square access); levels 2–9 interpolate by rank percentile
- Expose an async interface: receives a computation request and returns the result via a channel (runs in its own goroutine)
- Never produce an invalid move at any difficulty level

**Key types**: `MoveCandidate`, `Generator`, `DifficultyModel`, `AIPlayer`, `AIWorker`

**Constraints**:
- `Generator` is stateless; safe to call from any goroutine
- `AIWorker` owns exactly one goroutine; communicates with `ui` via channels
- Must complete move selection in ≤500 ms on target hardware at all difficulty levels

---

## Component: `ui`

**Purpose**: Implement the Ebitengine game loop, all game screens, board/tile rendering, player input handling, and local save/load persistence.

**Responsibilities**:
- Implement `ebiten.Game` interface (`Update`, `Draw`, `Layout`)
- Manage screen transitions: `MainMenuScreen` → `SetupScreen` → `GameScreen` → `EndGameScreen`
- Render the 15×15 board with open-licensed images; overlay placed tiles
- Render the human rack (interactive) and AI rack (display-only, always fully visible)
- Handle tile drag-and-drop and click-to-place input for desktop and touch for Android
- Display running scores, remaining bag count, current turn indicator, AI difficulty/dictionary labels
- Show "AI thinking..." indicator while `AIWorker` goroutine is running
- Commit or cancel staged (uncommitted) tile placements
- Trigger undo by invoking the inverse command from `engine`
- Save and load `engine.GameState` using `encoding/gob` to/from the platform app data directory
- Display end-game screen with final scores, end condition, and winner

**Key types**: `Game`, `Screen` (interface), `MainMenuScreen`, `SetupScreen`, `GameScreen`, `EndGameScreen`, `BoardRenderer`, `RackRenderer`, `ScorePanel`, `InputHandler`, `SaveManager`, `AIWorker`

**Constraints**:
- All `ebiten` calls must occur on the main goroutine (Ebitengine requirement)
- `AIWorker` communicates results to `GameScreen` via a non-blocking channel poll in `Update()`
- Asset images loaded once at startup via `//go:embed`; no runtime file I/O for images
- Save files written only to OS app-data directory with user-only permissions

---

## Component: `defs` (post-v1)

**Purpose**: Provide the definition of a word formed during gameplay. Sources meanings from
Wiktionary (kaikki.org `wiktextract`), filters them to the shipped word lists at build time,
and resolves a played word to a definition at runtime.

**Responsibilities**:
- Build a gzip-compressed gob asset of headword definitions + inflection edges, filtered to
  the union of the bundled word lists (offline `tools/builddefs`)
- Resolve a played word via a layered matcher: exact → form-of → rule-based stem →
  orthographic variant; never invent a definition from an unrelated near-spelling
- Surface a homograph's inflected reading additively (`Result.AlsoForm`)
- Load the asset once (cached, off the UI goroutine) and gate the feature on its presence
  (`Available()`)

**Key types**: `DB`, `Entry`, `Sense`, `MatchKind`, `Result`, `WordList`, `Report`

**Constraints**:
- Read-only after decode; `Lookup` is thread-safe for concurrent use
- Definitions are CC BY-SA (Wiktionary); attribution required
- Optional asset — the game runs without it

---

## Post-v1 additions to existing components

- **`engine`** — `GameMode` (Classic/Interesting: mode-parameterised premium layout + tile
  economy), Scrabble-notation formatting (`AnnotatedWords`), and persisted display records
  (`MoveRecord`, `OpeningDraw`, `GameState.{Mode, History, ScrabbleNotation}`).
- **`ui`** — the actual toolkit is **Fyne** (not Ebitengine as in the original design). Added:
  the Move history / Definitions two-tab panel (`tabPanel`) with a Copy button and mobile
  drag-scroll/long-press-copy overlay; the definitions worker; the game-mode chooser + preview
  dialog; the Scrabble-notation toggle; and the single-slot atomic `SaveManager`.
