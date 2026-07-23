# Domain Entities — Unit 4: `ui`

> **Correction addendum** — This document reflects the initial **Ebitengine** design. The UI
> was implemented on **Fyne** instead, so parts below (game loop, `Screen` FSM, pixel
> renderers, input polling, fixed 960×640 resolution) do not match the shipped code. See
> `aidlc-docs/corrections.md` and the post-v1 `ui` functional-design addenda for the actual
> design. The project name is **TileWords**, not "Squabble".

## 1. Game

```go
type Game struct {
    screen  Screen               // current active screen
    dict    *dictionary.Dictionary
    assets  *Assets              // pre-loaded images and fonts (nil in v1 — programmatic render)
}
```

Implements `ebiten.Game`. Routes all `Update`, `Draw`, and `Layout` calls to the current
`Screen`. Holds the dictionary loaded at game start (passed through from SetupScreen to
GameScreen). Transitions between screens by replacing `screen` in `Update`.

---

## 2. Screen (interface)

```go
type Screen interface {
    // Update advances screen logic by one game-loop tick.
    // Returns the next screen to display (may return itself).
    Update(g *Game) (Screen, error)
    // Draw renders the screen to the Ebitengine backbuffer.
    Draw(dst *ebiten.Image, g *Game)
}
```

The FSM of the application. Concrete types: `MainMenuScreen`, `SetupScreen`, `GameScreen`,
`EndGameScreen`. `Game.Update` calls `screen.Update`; if the returned screen differs, the
transition occurs immediately.

---

## 3. MainMenuScreen

```go
type MainMenuScreen struct {
    newGameBtn  button
    loadGameBtn button
    quitBtn     button
    errMsg      string // non-empty when load failed
}
```

Displays: title "Squabble", New Game, Load Game (if a save exists), and Quit buttons.
Clicking New Game → `SetupScreen`. Clicking Load Game → attempts `SaveManager.Load`;
on success → `GameScreen`; on failure → shows `errMsg`.

---

## 4. SetupScreen

```go
type SetupScreen struct {
    selectedDict  dictionary.DictName  // currently highlighted dictionary
    selectedLevel int                  // 1–10
    dictButtons   [6]button            // CSW, SOWPODS, OSPD, NASPA, OTCWL, All
    levelButtons  [10]button           // difficulty 1–10
    startBtn      button
    backBtn       button
}
```

Displays dictionary options (6 buttons) and difficulty level (10 numbered buttons or a
slider). Clicking Start → loads the selected dictionary via `dictionary.Load`, creates a
new `engine.GameState`, and transitions to `GameScreen`.

---

## 5. GameScreen

```go
type GameScreen struct {
    state        *engine.GameState
    dict         *dictionary.Dictionary
    staged       []StagedTile        // tiles placed but not committed
    interaction  TileInteraction     // current drag/select state
    worker       *ai.AIWorker
    saveManager  *SaveManager
    blankPicker  *BlankPickerOverlay // non-nil while choosing letter for a blank
    statusMsg    string              // shown in status bar
    undoReady    bool                // true after human move, cleared after AI responds
    showAIRack   bool               // false by default; toggled by the AI rack toggle button
    board        BoardRenderer
    humanRack    RackRenderer
    aiRack       RackRenderer
    scorePanel   ScorePanel
    controls     ControlPanel
}
```

The central gameplay screen. Owns the `engine.GameState` and the `ai.AIWorker`. Calls
`worker.Poll()` every `Update()` frame while `state.CurrentTurn == AITurn`.

---

## 6. EndGameScreen

```go
type EndGameScreen struct {
    state      *engine.GameState  // final state (read-only display)
    endReason  engine.EndReason
    playAgain  button
    mainMenu   button
}
```

Displays: player names, final scores, score adjustments from remaining rack tiles, winner
declaration, end reason ("Tiles exhausted" or "Six consecutive passes"), and two buttons
(Play Again → `SetupScreen`, Main Menu → `MainMenuScreen`).

---

## 7. StagedTile

```go
type StagedTile struct {
    Tile   engine.Tile // the tile (with AssignedLetter set if blank)
    Row    int
    Col    int
    fromRackIdx int   // index in human rack — used to return tile on recall
}
```

A tile the human has placed on the board but not yet committed. Visually distinct from
committed tiles. Removed from the in-memory rack during staging; restored on recall.

---

## 8. TileInteraction

```go
type TileInteraction struct {
    dragging    bool
    tile        engine.Tile
    fromRack    int        // rack index, or -1 if from board (re-drag staged)
    fromBoard   struct{ row, col int } // set when re-dragging a staged tile
    cursorX     int
    cursorY     int
    selected    int   // rack index of click-selected tile (-1 if none)
}
```

Tracks the in-progress drag-and-drop or click-select state. Reset to zero value on
successful placement, recall, or click cancellation.

---

## 9. BlankPickerOverlay

```go
type BlankPickerOverlay struct {
    stagedIdx int        // index into GameScreen.staged for the blank tile being assigned
    buttons   [26]button // A–Z letter buttons arranged in 4×7 grid (26 + cancel)
    cancelBtn button
}
```

Shown when a blank tile is placed on the board. Blocks all other input until dismissed.
On letter selection, sets `staged[stagedIdx].Tile.AssignedLetter` and hides the overlay.
On Cancel, returns the blank tile to the rack.

---

## 10. SaveManager

```go
type SaveManager struct {
    path string // resolved once at construction: os.UserConfigDir()/squabble/savegame.gob
}
```

Provides `Save(state *engine.GameState) error` and `Load() (*engine.GameState, error)`.
Uses `encoding/gob`. Writes to the platform's user config directory with 0600 permissions.
`Exists() bool` reports whether a save file is present (used to enable/disable the Load
button in MainMenuScreen).

---

## 11. BoardRenderer

```go
type BoardRenderer struct {
    originX, originY int // pixel offset of (0,0) cell in logical coords
    cellSize         int // 32 px (= 480 / 15)
}
```

Draws the 15×15 board background, premium-square colours, committed tiles, and staged
tiles using Ebitengine vector/fill primitives. No external image dependency. Premium square
colours: DW=orange, TW=red, DL=light-blue, TL=dark-blue, Centre=gold.

---

## 12. RackRenderer

```go
type RackRenderer struct {
    originX, originY int
    tileSize         int  // 44 px
    interactive      bool // true for human rack, false for AI rack
}
```

Draws a row of up to 7 tile cells. For the human rack, tiles are clickable/draggable.
Staged tiles (already on board) are shown as empty placeholders. For the AI rack
(`interactive=false`), tiles are rendered with letters visible only when
`GameScreen.showAIRack == true`; otherwise each tile is drawn as a face-down placeholder
(coloured rectangle, no letter or point value).

---

## 13. ScorePanel

Draws: human score, AI score, tiles remaining in bag, current turn indicator
("Your turn" / "AI thinking…"), last-action description (e.g. "AI played QUARTZ for 84"),
and the `statusMsg` from `GameScreen` (validation errors, undo confirmation, etc.).

---

## 14. ControlPanel

```go
type ControlPanel struct {
    playBtn        button // commit staged tiles as a PlayMove
    exchangeBtn    button // exchange selected rack tiles
    passBtn        button // pass this turn
    undoBtn        button // undo last human+AI round
    saveBtn        button // save current game
    toggleAIRack   button // "Show AI Rack" / "Hide AI Rack" toggle
}
```

Buttons are enabled/disabled based on game state:
- `playBtn`: enabled when `len(staged) > 0` and no blank awaiting letter assignment.
- `exchangeBtn`: enabled when human has selected tiles AND `bag.Count() >= MaxRackSize`.
- `undoBtn`: enabled only when `undoReady == true`.
- `toggleAIRack`: always enabled (available during both human and AI turns).
- All action buttons (play/exchange/pass/undo): disabled during AITurn.

`toggleAIRack` label is "Show AI Rack" when `showAIRack == false` and "Hide AI Rack"
when `showAIRack == true`. Clicking it flips `GameScreen.showAIRack`.

---

## 15. button (internal helper)

```go
type button struct {
    label   string
    rect    image.Rectangle
    enabled bool
    hovered bool
}
```

Thin struct used by all screens. `IsClicked(x, y int) bool` checks mouse/touch point
against `rect` when `enabled`. Drawn as a filled rectangle with label text centred.

---

## 16. Assets

```go
type Assets struct{} // v1: empty — all rendering is programmatic
```

Reserved for future image/font loading. In v1, all rendering uses Ebitengine's built-in
`ebitenutil.DebugPrint` for text and `vector` package primitives for shapes. A real font
(e.g. via `golang.org/x/image/font/basicfont`) will be used for tile letters.
