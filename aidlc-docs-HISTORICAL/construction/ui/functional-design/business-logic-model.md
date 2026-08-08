# Business Logic Model — Unit 4: `ui`

> **Correction addendum** — This document reflects the initial **Ebitengine** design. The UI
> was implemented on **Fyne** instead, so parts below (game loop, `Screen` FSM, pixel
> renderers, input polling, fixed 960×640 resolution) do not match the shipped code. See
> `aidlc-docs/corrections.md` and the post-v1 `ui` functional-design addenda for the actual
> design. The project name is **TileWords**, not "Squabble".

## Algorithm 1: Ebitengine Game Loop

```
Game.Layout(outsideW, outsideH) → (960, 640)   // fixed logical resolution

Game.Update():
    next, err := screen.Update(g)
    if err: return err
    screen = next

Game.Draw(dst *ebiten.Image):
    screen.Draw(dst, g)
```

`Layout` always returns (960, 640) regardless of physical window size. Ebitengine scales
automatically. This is the single logical coordinate system for all rendering.

---

## Algorithm 2: Screen Transition FSM

```
MainMenuScreen
    → [New Game]  → SetupScreen
    → [Load Game] → GameScreen (with loaded state) | MainMenuScreen (on error)

SetupScreen
    → [Start]  → GameScreen (new state)
    → [Back]   → MainMenuScreen

GameScreen
    → [game over] → EndGameScreen
    → [save]      → stays in GameScreen

EndGameScreen
    → [Play Again] → SetupScreen
    → [Main Menu]  → MainMenuScreen
```

Transitions occur by returning a different Screen value from `Screen.Update`.

---

## Algorithm 3: Human Turn Flow

```
State: HumanTurn, no staged tiles

Input processing (each Update tick):
  1. If blankPicker != nil:     route all input to BlankPickerOverlay.Update
  2. Else:                      route input to TileInteraction + ControlPanel

Tile placement (click-to-select model):
  Mouse down on rack tile →  set interaction.selected = rack index
  Mouse down on board cell:
    if interaction.selected >= 0:  stageTile(rackIdx, row, col)
    elif cell has staged tile:     recallStagedTile(row, col)
  Mouse down on staged board tile: recallStagedTile(row, col)

stageTile(rackIdx, row, col):
  t = humanRack.tiles[rackIdx]
  remove tile from in-memory rack
  append StagedTile{Tile:t, Row:row, Col:col, fromRackIdx:rackIdx} to staged
  interaction.selected = -1
  if t.IsBlank: show BlankPickerOverlay

recallStagedTile(row, col):
  find staged tile at (row, col)
  restore tile to humanRack at fromRackIdx (or append if slot taken)
  remove from staged

Button actions:
  [Play]     → commitPlay()
  [Exchange] → commitExchange()
  [Pass]     → commitPass()
  [Undo]     → doUndo()
  [Save]     → doSave()
```

---

## Algorithm 4: Commit Play (Human)

```
commitPlay():
  if len(staged) == 0: statusMsg = "Place at least one tile."; return
  if any staged tile has IsBlank and AssignedLetter == 0:
      statusMsg = "Assign a letter to the blank tile."; return

  placed = []PlacedTile from staged (Row, Col, Tile)
  move = PlayMove{Placed: placed}
  if _, err = engine.ValidatePlacement(state.Board, &move, dict); err:
      statusMsg = "Invalid word: " + err.Error(); return

  cmd = PlayCommand{}
  if err = cmd.Execute(state, dict, rng); err:
      statusMsg = "Could not play: " + err.Error(); return

  state.LastHumanCommand = cmd
  staged = nil
  undoReady = false
  statusMsg = fmt.Sprintf("You played for %d points.", move.Score)

  if engine.IsGameOver(state): transition to EndGameScreen
  else: startAITurn()
```

---

## Algorithm 5: Commit Exchange (Human)

```
commitExchange():
  selected = tiles marked for exchange in rack UI
  if len(selected) == 0: statusMsg = "Select tiles to exchange."; return
  if state.Bag.Count() < engine.MaxRackSize:
      statusMsg = "Not enough tiles in bag to exchange."; return

  cmd = ExchangeCommand{Tiles: selected}
  cmd.Execute(state, dict, rng)
  state.LastHumanCommand = cmd
  undoReady = false
  statusMsg = "Tiles exchanged."

  if engine.IsGameOver(state): transition to EndGameScreen
  else: startAITurn()
```

---

## Algorithm 6: Commit Pass (Human)

```
commitPass():
  cmd = PassCommand{}
  cmd.Execute(state, dict, rng)
  state.LastHumanCommand = cmd
  undoReady = false
  statusMsg = "Turn passed."

  if engine.IsGameOver(state): transition to EndGameScreen
  else: startAITurn()
```

---

## Algorithm 7: AI Turn Flow

```
startAITurn():
  state.CurrentTurn = AITurn
  worker.Request(state, dict, state.AILevel)
  statusMsg = "AI is thinking…"

GameScreen.Update() during AITurn:
  if move, ok = worker.Poll(); ok:
      applyAIMove(move)

applyAIMove(move engine.Move):
  switch m := move.(type):
    case PlayMove:
      cmd = PlayCommand{Move: m}  // ValidatePlacement already done by ChooseMove
      cmd.Execute(state, dict, rng)
      state.LastAICommand = cmd
      statusMsg = fmt.Sprintf("AI played %s for %d points.", firstWord(m), m.Score)
    case ExchangeMove:
      cmd = ExchangeCommand{Tiles: m.Tiles}
      cmd.Execute(state, dict, rng)
      state.LastAICommand = cmd
      statusMsg = "AI exchanged tiles."
    case PassMove:
      cmd = PassCommand{}
      cmd.Execute(state, dict, rng)
      state.LastAICommand = cmd
      statusMsg = "AI passed."

  undoReady = true
  state.CurrentTurn = HumanTurn
  // AI rack display updates automatically on next Draw call;
  // showAIRack flag is unchanged by applyAIMove.

  engine.ApplyEndgameScoring if IsGameOver
  if engine.IsGameOver(state): transition to EndGameScreen
```

---

## Algorithm 8: Blank Tile Assignment

```
BlankPickerOverlay.Update(g *Game) → (dismissed bool, letter byte):
  for each of 26 letter buttons:
    if clicked: return true, letter
  if cancelBtn clicked: return true, 0

In GameScreen.Update, after stageTile with blank:
  if blankPicker.dismissed:
    if letter != 0:
      staged[blankPicker.stagedIdx].Tile.AssignedLetter = letter
      staged[blankPicker.stagedIdx].Tile.Letter = letter  // display letter
    else:
      recallStagedTile(staged[blankPicker.stagedIdx].Row, staged[blankPicker.stagedIdx].Col)
    blankPicker = nil
```

---

## Algorithm 9: Undo Last Round

```
doUndo():
  if !undoReady: return
  engine.UndoLastRound(state)   // undoes AI move then human move
  staged = nil
  undoReady = false
  state.CurrentTurn = HumanTurn
  statusMsg = "Move undone."
```

`engine.UndoLastRound` calls `state.LastAICommand.Undo(state)` then
`state.LastHumanCommand.Undo(state)`. If either command is nil (first turn before AI moved),
only the available undo is applied.

---

## Algorithm 10: Save / Load

```
doSave():
  if err = saveManager.Save(state); err:
    statusMsg = "Save failed: " + sanitise(err)
  else:
    statusMsg = "Game saved."

SaveManager.Save(state):
  mkdir -p path.Dir(sm.path) with perm 0700
  gob.Encode(state) → tempFile
  os.Rename(tempFile, sm.path)   // atomic write
  os.Chmod(sm.path, 0600)

MainMenuScreen: Load Game button:
  state, err = saveManager.Load()
  if err: errMsg = "Could not load save: " + sanitise(err)
  else: transition to GameScreen(state)
```

---

## Algorithm 11: Board Hit-Testing

```
cellAt(px, py int) (row, col int, ok bool):
  col = (px - board.originX) / board.cellSize
  row = (py - board.originY) / board.cellSize
  ok = row in [0,14] and col in [0,14]
    and px >= board.originX and py >= board.originY
```

Used by input handler to convert mouse/touch coordinates to board cell.

---

## Algorithm 12: BoardRenderer.Draw

```
Draw(dst, state, staged):
  // 1. Fill board background.
  fill(dst, boardOrigin, 15*cellSize, 15*cellSize, colorBoardBg)

  // 2. Draw premium square cells.
  for each premium square:
    fill(dst, cellRect(r,c), premiumColor[sq])
    drawText(dst, premiumLabel[sq], cellRect(r,c))  // "DL", "TW" etc.

  // 3. Draw committed tiles.
  for r,c where board.Cell(r,c).Tile != nil:
    drawTile(dst, board.Cell(r,c).Tile, cellRect(r,c), committed=true)

  // 4. Draw staged tiles (yellow outline, semi-transparent).
  for each st in staged:
    drawTile(dst, st.Tile, cellRect(st.Row,st.Col), committed=false)

  // 5. Draw grid lines.
  for each row/col boundary:
    vector.StrokeLine(dst, ..., colorGrid)
```

---

## PBT Properties

| ID | Property |
|---|---|
| PBT-UI-01 | `cellAt(boardOriginX + col*cellSize + cellSize/2, boardOriginY + row*cellSize + cellSize/2)` returns `(row, col, true)` for all valid (row, col) |
| PBT-UI-02 | `SaveManager.Save` then `Load` round-trips `GameState` with identical scores, turn, board tiles |
| PBT-UI-03 | After any sequence of `stageTile` + `recallStagedTile` operations, the total tile count (rack + staged) is invariant |
| PBT-UI-04 | `commitPlay` never commits a move that `engine.ValidatePlacement` would reject |
