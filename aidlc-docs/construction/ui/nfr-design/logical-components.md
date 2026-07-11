# Logical Components — Unit 4: `ui`

## Component Map

```
Game (ebiten.Game)
├── Screen FSM
│   ├── MainMenuScreen
│   ├── SetupScreen
│   ├── GameScreen
│   │   ├── BoardRenderer
│   │   ├── RackRenderer (human)
│   │   ├── RackRenderer (AI, showAIRack-aware)
│   │   ├── ScorePanel
│   │   ├── ControlPanel (incl. AI rack toggle)
│   │   ├── BlankPickerOverlay (conditional)
│   │   ├── InputHandler (hit-test + staged tile management)
│   │   └── AIPoller (timeout-guarded Poll loop)
│   └── EndGameScreen
└── SaveManager (used by MainMenuScreen + GameScreen)
```

---

## Component 1: Game

**Type**: `Game` (implements `ebiten.Game`)

**Responsibility**: Ebitengine entry point. Delegates all logic to the current `Screen`.
Holds the loaded `*dictionary.Dictionary` once set by `SetupScreen` so it can be passed
to `GameScreen` without reloading.

**Methods**:
```go
func (g *Game) Update() error          // routes to g.screen.Update
func (g *Game) Draw(dst *ebiten.Image) // routes to g.screen.Draw
func (g *Game) Layout(ow, oh int) (int, int) // always returns (960, 640)
```

**NFR trace**: NFR-UI-P6 (O(1) transition); NFR-UI-T1 (main goroutine only).

---

## Component 2: MainMenuScreen

**Responsibility**: Title display, New Game / Load Game / Quit navigation.

**Logic**:
- `newGameBtn` → returns `NewSetupScreen()`
- `loadGameBtn` (shown only when `saveManager.Exists()`) → calls `saveManager.Load()`;
  on success returns `NewGameScreen(state, dict, saveManager)`; on error sets `errMsg`
  and returns `self`
- `quitBtn` → calls `os.Exit(0)`
- `errMsg` shown when load fails (sanitised)

**NFR trace**: SECURITY-UI-2 (sanitised error); NFR-UI-R2 (no partial state on load failure).

---

## Component 3: SetupScreen

**Responsibility**: Dictionary and difficulty selection before game start.

**Logic**:
- Six dictionary buttons (CSW, SOWPODS, OSPD, NASPA, OTCWL, All); one highlighted.
- Ten difficulty buttons (1–10); one highlighted.
- `startBtn` → calls `dictionary.Load(selectedDict)`; on success returns
  `NewGameScreen(engine.New(selectedDict, selectedLevel, rng), dict, saveManager)`.
- `backBtn` → returns `NewMainMenuScreen()`.
- Dictionary loading may take up to 3 s (NFR-UI-P5); status text "Loading dictionary…"
  shown while in progress. Since Ebitengine Update is synchronous, the load is initiated
  on click and completed in a goroutine; result delivered via a channel polled in Update.

**NFR trace**: NFR-UI-P5; NFR-UI-P6.

---

## Component 4: GameScreen

**Responsibility**: Central gameplay screen. Owns `engine.GameState`, `ai.AIWorker`,
staged tile list, and `showAIRack` toggle state.

**Sub-responsibilities delegated to sub-components** (all called from `GameScreen.Update`
and `GameScreen.Draw`):
1. **InputHandler** — hit-test, stageTile, recallStagedTile, tile selection
2. **BlankPickerOverlay** — letter assignment for blanks (blocks other input while active)
3. **AIPoller** — timeout-guarded `worker.Poll()` during AITurn
4. **ControlPanel** — button dispatch (play/exchange/pass/undo/save/toggle)

**Phase structure of `Update()`**:
```
Phase 1 — BlankPicker: if blankPicker != nil, route all input here; return early
Phase 2 — AITurn poll: if AITurn, check timeout, poll worker; return early
Phase 3 — Human input: process mouse/touch for tile staging
Phase 4 — Button dispatch: check ControlPanel button clicks
Phase 5 — End-game check: call engine.IsGameOver after any state change
```

**NFR trace**: NFR-UI-P1 (≤2 ms Update); NFR-UI-R4 (timeout guard in Phase 2).

---

## Component 5: BoardRenderer

**Responsibility**: Draw the 15×15 board including premium squares, committed tiles,
staged tiles, grid lines, and cursor highlight.

**Draw passes** (in order):
1. Board background fill (`#228B22` forest green)
2. Premium square cells — per-cell fill + short label ("DL", "TW", "★", etc.)
3. Committed tiles (from `state.Board`)
4. Staged tiles (from `GameScreen.staged`) — distinct visual style
5. Grid lines

**Geometry**: origin at `(10, 10)`; cell size 32 px; board = 480×480 px.

**Tile draw**: `drawTile(dst, tile, px, py, committed bool)` — fills rect, draws letter
(large, centred), draws point value (small, bottom-right). Blank tiles on board use
blue letter colour. Staged tiles use lemon-chiffon background + goldenrod border.

**NFR trace**: NFR-UI-M2 (pre-allocated opts); NFR-UI-C3 (five draw passes commented).

---

## Component 6: RackRenderer (×2)

**Responsibility**: Draw a 7-tile rack row. Used for both human rack and AI rack.

**Human rack**: tiles are clickable/draggable (`interactive=true`). Staged tiles shown
as empty outlined placeholders.

**AI rack**: `interactive=false`. When `showAIRack=false`, each tile slot is drawn as a
face-down rectangle (uniform dark-green fill, no letter). When `showAIRack=true`, tiles
render identically to human rack tiles.

**Geometry**: AI rack at `(500, 80)`; human rack at `(500, 400)`. Tile size 44×44 px.
Gap between tiles: 4 px. Total row width: 7×44 + 6×4 = 332 px (fits in right panel).

---

## Component 7: ScorePanel

**Responsibility**: Draw scores, bag count, turn indicator, status message.

**Drawn items** (all in right panel, `x=500..950`):
- "Squabble" title text at top
- Human score: "You: [N]"
- AI score: "AI: [N]"
- Bag: "[N] tiles remaining"
- Turn: "Your turn" / "AI is thinking…"
- Status bar: `statusMsg` (validation errors, action confirmations)

Uses `ebitenutil.DebugPrintAt` for all text in v1.

---

## Component 8: ControlPanel

**Responsibility**: Draw and hit-test the six game-control buttons.

**Button layout** (right panel, below human rack):
```
[Play]  [Exchange]  [Pass]
[Undo]  [Save]      [Show AI Rack / Hide AI Rack]
```

**Enable/disable logic**:
| Button | Enabled when |
|---|---|
| Play | HumanTurn, len(staged)>0, no unassigned blank |
| Exchange | HumanTurn, ≥1 rack tile selected, bag≥7 |
| Pass | HumanTurn |
| Undo | HumanTurn, undoReady==true |
| Save | Always (both turns) |
| Toggle AI Rack | Always (both turns) |

**NFR trace**: NFR-UI-U3 (toggle label changes immediately); NFR-UI-U5 (hover state).

---

## Component 9: BlankPickerOverlay

**Responsibility**: Modal letter-assignment UI shown when a blank tile is staged.

**Geometry**: Full-screen semi-transparent dark overlay (960×640). Centred panel
320×280 px. 4 rows × 7 columns of letter buttons (A–Z = 26; remaining 2 cells = Cancel
button + empty). Each button 40×40 px.

**Input**: Blocks all `GameScreen` input while active. On letter click → assigns letter
to `staged[stagedIdx].Tile.AssignedLetter`; dismisses. On Cancel → calls
`recallStagedTile`; dismisses.

**NFR trace**: BR-UI-03 (blank assigned before commit); NFR-UI-U6 (44 px touch target —
40 px close enough, acceptable for overlay buttons).

---

## Component 10: InputHandler (pure functions)

**Responsibility**: Convert raw mouse/touch events to staged tile operations.

**Pure functions** (headless-testable — Pattern 7):
```go
func cellAt(originX, originY, cellSize, px, py int) (row, col int, ok bool)
func rackTileAt(originX, originY, tileSize, gap, px, py int) (idx int, ok bool)
```

**Stateful logic** (in `GameScreen.Update` Phase 3):
- `stageTile(rackIdx, row, col)` — removes from rack, appends to staged, shows
  BlankPickerOverlay if blank.
- `recallStagedTile(row, col)` — removes from staged, restores to rack.
- `recallAllStagedTiles()` — batch recall (used by Exchange and Pass buttons to clear
  any accidentally staged tiles before executing the command).

**NFR trace**: PBT-UI-01 (cellAt round-trip); PBT-UI-03 (tile-count invariant).

---

## Component 11: SaveManager

**Responsibility**: Persist and restore `engine.GameState` via gob.

**API**:
```go
func NewSaveManager(configRoot string) *SaveManager  // configRoot="" → os.UserConfigDir()
func (sm *SaveManager) Save(state *engine.GameState) error
func (sm *SaveManager) Load() (*engine.GameState, error)
func (sm *SaveManager) Exists() bool
```

`configRoot` injection enables headless testing (NFR-UI-TEST-1). In production,
`configRoot=""` triggers `os.UserConfigDir()/squabble/`.

**Save path**: `{configRoot}/squabble/savegame.gob`
**Write**: atomic temp-file rename (Pattern 3); permissions 0600/0700.
**Load**: gob decode into `*engine.GameState`; validates non-nil result (SECURITY-UI-3).

**NFR trace**: NFR-UI-R2, NFR-UI-R3, NFR-UI-R5; SECURITY-UI-1, SECURITY-UI-3, SECURITY-UI-4.

---

## Infrastructure Dependencies

| Dependency | Usage |
|---|---|
| `github.com/hajimehoshi/ebiten/v2` | `Game` interface, `ebiten.Image`, input, window |
| `github.com/hajimehoshi/ebiten/v2/vector` | `vector.DrawFilledRect`, `vector.StrokeLine` |
| `github.com/hajimehoshi/ebiten/v2/ebitenutil` | `DebugPrintAt` for text |
| `golang.org/x/image/font/basicfont` | Tile letter bitmap font |
| `squabble/engine` | `GameState`, all command types, `ValidatePlacement`, `IsGameOver`, `ApplyEndgameScoring` |
| `squabble/ai` | `AIWorker`, `ChooseMove` |
| `squabble/dictionary` | `Dictionary`, `Load`, `DictName` |
| `encoding/gob`, `os`, `path/filepath` | Save/load persistence |
| `fmt`, `strings`, `time`, `image`, `image/color` | Error formatting, timeout, geometry, colour |
