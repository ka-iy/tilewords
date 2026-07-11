# Code Generation Plan — Unit 4: `ui`

## Pre-Generation Steps

Before generating Go source, add Ebitengine and x/image to go.mod:
```
go get github.com/hajimehoshi/ebiten/v2@latest
go get golang.org/x/image@latest
```

---

## Files to Generate

| Step | File | Description |
|---|---|---|
| 1 | `ui/doc.go` | Package GoDoc: Ebitengine game loop, screen FSM, usage |
| 2 | `ui/screen.go` | `Screen` interface; `button` helper (rect, enabled, hover, IsClicked, Draw) |
| 3 | `ui/colors.go` | Colour constants for board, tiles, premium squares, UI elements |
| 4 | `ui/input.go` | Pure functions: `cellAt`, `rackTileAt`; tile interaction constants |
| 5 | `ui/save.go` | `SaveManager`: atomic Save, Load, Exists; configRoot injection |
| 6 | `ui/render.go` | `BoardRenderer`, `RackRenderer`, `ScorePanel`, shared draw helpers (`drawTile`, `fillRect`, `drawText`) |
| 7 | `ui/blank_picker.go` | `BlankPickerOverlay`: modal A–Z letter selection, Cancel |
| 8 | `ui/mainmenu.go` | `MainMenuScreen`: New Game, Load Game, Quit |
| 9 | `ui/setup.go` | `SetupScreen`: dict buttons, difficulty buttons, Start, Back; async dict load via goroutine+channel |
| 10 | `ui/endgame.go` | `EndGameScreen`: final scores, winner, end reason, Play Again / Main Menu |
| 11 | `ui/gamescreen.go` | `GameScreen`: 5-phase Update, staged tile management, AI poller, ControlPanel dispatch |
| 12 | `ui/game.go` | `Game` struct: `Update`, `Draw`, `Layout`; `NewGame` constructor |
| 13 | `ui/ui_test.go` | Example tests for pure/headless logic |
| 14 | `ui/ui_pbt_test.go` | PBT tests (PBT-UI-01 through PBT-UI-04) |
| 15 | `aidlc-docs/construction/ui/code/code-summary.md` | Code summary |

---

## Step Details

### Step 1 — `ui/doc.go`
Package-level GoDoc explaining: Ebitengine game loop, Screen FSM, GameScreen phases,
SaveManager atomic write, headless testability of pure functions. Include usage example
showing `ebiten.RunGame(ui.NewGame())`.

### Step 2 — `ui/screen.go`
```go
type Screen interface {
    Update(g *Game) (Screen, error)
    Draw(dst *ebiten.Image, g *Game)
}

type button struct {
    label   string
    rect    image.Rectangle
    enabled bool
}
func (b *button) IsClicked(mx, my int, justPressed bool) bool
func (b *button) Draw(dst *ebiten.Image, hovered bool)
```
`IsClicked` checks `enabled` first, then bounds. `Draw` fills rect in enabled/disabled/hover
colour, then draws centred label text.

### Step 3 — `ui/colors.go`
Named colour constants (all `color.RGBA`):
- Board/premium: `colorBoardBg`, `colorDW`, `colorTW`, `colorDL`, `colorTL`, `colorCentre`, `colorGrid`
- Tiles: `colorTileBg`, `colorTileStagedBg`, `colorTileStagedBorder`, `colorTileLetter`,
  `colorTileBlankLetter`, `colorTilePoints`, `colorTileFaceDown`
- UI: `colorBtnEnabled`, `colorBtnDisabled`, `colorBtnHover`, `colorBtnText`, `colorPanel`,
  `colorStatusBar`, `colorOverlay`

### Step 4 — `ui/input.go`
```go
const (
    BoardOriginX = 10
    BoardOriginY = 10
    CellSize     = 32
    RackTileSize = 44
    RackGap      = 4
)

// cellAt converts pixel coordinates to board (row, col). ok=false if outside board.
func cellAt(px, py int) (row, col int, ok bool)

// rackTileAt converts pixel coordinates to rack tile index. ok=false if outside rack.
// originX, originY are the rack's top-left pixel position.
func rackTileAt(originX, originY, px, py int) (idx int, ok bool)
```
Both are pure functions with no ebiten dependency — headless testable.

### Step 5 — `ui/save.go`
```go
type SaveManager struct { path string }

func NewSaveManager(configRoot string) (*SaveManager, error)
func (sm *SaveManager) Save(state *engine.GameState) error
func (sm *SaveManager) Load() (*engine.GameState, error)
func (sm *SaveManager) Exists() bool
```
`NewSaveManager("")` → resolves `os.UserConfigDir()/squabble/savegame.gob`.
`NewSaveManager(tmpDir)` → uses `tmpDir/squabble/savegame.gob` (test injection).
Save: mkdir 0700 → write temp 0600 → rename. Load: open → gob.Decode → validate non-nil.

### Step 6 — `ui/render.go`
```go
type BoardRenderer struct { opts ebiten.DrawImageOptions }
func (r *BoardRenderer) Draw(dst *ebiten.Image, board *engine.Board, staged []StagedTile)

type RackRenderer struct { originX, originY int; interactive bool }
func (r *RackRenderer) Draw(dst *ebiten.Image, tiles []engine.Tile, stagedSet map[int]bool, showTiles bool)

type ScorePanel struct{}
func (s *ScorePanel) Draw(dst *ebiten.Image, state *engine.GameState, statusMsg string, aiTurn bool)
```
Shared helpers (unexported):
- `fillRect(dst, x, y, w, h int, col color.RGBA)` — fills using `vector.DrawFilledRect`
- `drawText(dst, text string, x, y int, col color.RGBA)` — renders with basicfont
- `drawTile(dst *ebiten.Image, t engine.Tile, px, py, size int, staged bool)` — full tile render

### Step 7 — `ui/blank_picker.go`
```go
type BlankPickerOverlay struct {
    stagedIdx int
    buttons   [26]button
    cancelBtn button
}
func NewBlankPickerOverlay(stagedIdx int) *BlankPickerOverlay
// Update returns (letter byte, dismissed bool).
// letter==0 and dismissed==true means cancel.
func (bp *BlankPickerOverlay) Update() (byte, bool)
func (bp *BlankPickerOverlay) Draw(dst *ebiten.Image)
```
Layout: semi-transparent full-screen overlay; centred panel; 4×7 grid of letter buttons
(A–Z in rows), Cancel in the 27th slot.

### Step 8 — `ui/mainmenu.go`
```go
type MainMenuScreen struct {
    newGameBtn  button
    loadGameBtn button
    quitBtn     button
    saveManager *SaveManager
    errMsg      string
}
func NewMainMenuScreen(saveManager *SaveManager, errMsg string) *MainMenuScreen
func (s *MainMenuScreen) Update(g *Game) (Screen, error)
func (s *MainMenuScreen) Draw(dst *ebiten.Image, g *Game)
```
`Update`: handle button clicks → transition or set errMsg. `loadGameBtn` enabled only
when `saveManager.Exists()`.

### Step 9 — `ui/setup.go`
```go
type SetupScreen struct {
    dictBtns     [6]button
    levelBtns    [10]button
    startBtn     button
    backBtn      button
    selectedDict dictionary.DictName
    selectedLevel int
    saveManager  *SaveManager
    // async dict load
    loading      bool
    loadResult   chan loadResult
}
type loadResult struct { dict *dictionary.Dictionary; err error }
func NewSetupScreen(saveManager *SaveManager) *SetupScreen
func (s *SetupScreen) Update(g *Game) (Screen, error)
func (s *SetupScreen) Draw(dst *ebiten.Image, g *Game)
```
On Start click: spawn `go func() { dict, err := dictionary.Load(sel); ch <- loadResult{dict,err} }()`;
set `loading=true`. Each Update tick polls `loadResult` channel (non-blocking select).
On result: if ok → `NewGameScreen(...)`, else show error.

### Step 10 — `ui/endgame.go`
```go
type EndGameScreen struct {
    state       *engine.GameState
    endReason   engine.EndReason
    playAgain   button
    mainMenuBtn button
    saveManager *SaveManager
}
func NewEndGameScreen(state *engine.GameState, reason engine.EndReason, sm *SaveManager) *EndGameScreen
func (s *EndGameScreen) Update(g *Game) (Screen, error)
func (s *EndGameScreen) Draw(dst *ebiten.Image, g *Game)
```
Draw: "Game Over" title, human/AI scores, rack adjustments, end reason, winner, two buttons.

### Step 11 — `ui/gamescreen.go`
The largest file. Key elements:
```go
type StagedTile struct { Tile engine.Tile; Row, Col int; fromRackIdx int }

type GameScreen struct {
    state          *engine.GameState
    dict           *dictionary.Dictionary
    staged         []StagedTile
    rackSelected   int        // index in human rack, -1 if none
    exchangeSel    map[int]bool // rack indices selected for exchange
    worker         *ai.AIWorker
    saveManager    *SaveManager
    blankPicker    *BlankPickerOverlay
    statusMsg      string
    undoReady      bool
    showAIRack     bool
    aiRequestTime  time.Time
    rng            *rand.Rand
    board          BoardRenderer
    humanRackR     RackRenderer
    aiRackR        RackRenderer
    scorePanel     ScorePanel
    controls       controlPanel
}
```
`Update`: 5 phases as per business logic model.
`stageTile(rackIdx, row, col)` + `recallStagedTile(row, col)` + `recallAll()`.
`commitPlay()`, `commitExchange()`, `commitPass()`, `doUndo()`, `doSave()`.
`startAITurn()`, `applyAIMove(move engine.Move)`.

Internal `controlPanel` struct holds all 6 buttons; `controlPanel.draw(dst, gs)` and
`controlPanel.handleClick(mx, my, gs) action` (returns an enum so GameScreen.Update
dispatches without coupling).

### Step 12 — `ui/game.go`
```go
type Game struct {
    screen Screen
    sm     *SaveManager
}
func NewGame() (*Game, error)  // creates SaveManager, starts at MainMenuScreen
func (g *Game) Update() error
func (g *Game) Draw(dst *ebiten.Image)
func (g *Game) Layout(outsideW, outsideH int) (int, int)  // returns 960, 640
```

### Steps 13–14 — Tests
`ui/ui_test.go`: tests for `cellAt`, `rackTileAt`, `SaveManager` (round-trip, corrupt-load,
missing-load, first-run mkdir), `sanitiseError`, `button.IsClicked`.

`ui/ui_pbt_test.go`: rapid-based tests for PBT-UI-01 through PBT-UI-04.

---

## Dependency Order

Steps 1–5 have no internal `ui` package dependencies.
Step 6 (`render.go`) depends on `engine` types only.
Step 7 (`blank_picker.go`) depends on `screen.go` (button) and `colors.go`.
Steps 8–10 depend on steps 2–6.
Step 11 (`gamescreen.go`) depends on steps 2–7 and `ai.AIWorker`.
Step 12 (`game.go`) depends on all screens (8–11).
Steps 13–14 depend on steps 4–5 (pure functions + SaveManager).

Generation order: 1→2→3→4→5→6→7→8→9→10→11→12→13→14→15.

---

## Verification

After all files generated:
```
go get github.com/hajimehoshi/ebiten/v2@latest
go get golang.org/x/image@latest
go build ./ui/... ./cmd/squabble/...
go test -race ./ui/...
go test -race ./...
```

`go build` of `cmd/squabble` is the integration smoke test confirming all packages wire
together. `go test -race ./ui/...` runs only the headless tests (no Ebitengine loop).
