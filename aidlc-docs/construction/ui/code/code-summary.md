# Code Summary — Unit 4: `ui`

> **Correction addendum** — This document reflects the initial **Ebitengine** design. The UI
> was implemented on **Fyne** instead, so parts below (game loop, `Screen` FSM, pixel
> renderers, input polling, fixed 960×640 resolution) do not match the shipped code. See
> `aidlc-docs/corrections.md` and the post-v1 `ui` functional-design addenda for the actual
> design. The project name is **TileWords**, not "Squabble".

## Files Generated

| File | Description |
|---|---|
| `ui/doc.go` | Package GoDoc — Screen FSM, GameScreen phases, SaveManager, headless testability |
| `ui/screen.go` | `Screen` interface; `button` helper (IsClicked, IsHovered, Draw); `drawTextCentred`, `drawTextAt`, `fillRect` |
| `ui/colors.go` | All `color.RGBA` constants: board/premium squares, tile face-up/staged/face-down, UI controls |
| `ui/input.go` | Layout constants; pure functions `cellAt` and `rackTileAt` (no ebiten dependency) |
| `ui/save.go` | `SaveManager` — atomic gob save (mkdir→temp→rename), load with nil-guard, `Exists`; `sanitiseError` |
| `ui/render.go` | `BoardRenderer`, `RackRenderer`, `ScorePanel`, `StagedTile`; `drawTile`, `premiumColor`, `wrapText` |
| `ui/blank_picker.go` | `BlankPickerOverlay` — modal A–Z grid + Cancel; `Update` returns `(letter, dismissed)` |
| `ui/mainmenu.go` | `MainMenuScreen` — New Game, Load Game (enabled when save exists), Quit |
| `ui/setup.go` | `SetupScreen` — async dictionary load via goroutine+channel; dict/level buttons; `loadResult` type |
| `ui/endgame.go` | `EndGameScreen` — final scores, end reason, winner, Play Again / Main Menu |
| `ui/gamescreen.go` | `GameScreen` — 5-phase Update; staged tile management; `controlPanel`; AI worker integration |
| `ui/game.go` | `Game` (ebiten.Game): `Update`, `Draw`, `Layout(960,640)`; `NewGame`; `errQuit` sentinel |
| `ui/ui_test.go` | Headless example tests: `cellAt`, `rackTileAt`, `sanitiseError`, `SaveManager`, `button` |
| `ui/ui_pbt_test.go` | PBT-UI-01 through PBT-UI-04 (rapid-based): cellAt inverse, outside-board, rackTileAt slot, sanitiseError invariants |

## Engine Additions (required for save/load)

- `engine/rack.go` — added `GobEncode`/`GobDecode` (tiles field is unexported)
- `engine/bag.go` — added `GobEncode`/`GobDecode` (tiles field is unexported)

## Design Decisions

- **showAIRack defaults to false** — AI rack hidden on game start per user change request (BR-UI-07).
  Toggle button label alternates "Show AI Rack" / "Hide AI Rack" (always enabled).
- **Async dict load** — `SetupScreen` spawns a goroutine; polls a buffered channel each tick.
  Keeps the Ebitengine loop non-blocking during GADDAG decode.
- **libXxf86vm** — The runtime library exists at `/usr/lib/x86_64-linux-gnu/libXxf86vm.so.1`.
  A symlink `/usr/local/lib/libXxf86vm.so → libXxf86vm.so.1` was created so the linker can
  resolve `-lXxf86vm` without the `libxxf86vm-dev` package.
- **Pre-allocated DrawImageOptions** — `BoardRenderer` holds one `ebiten.DrawImageOptions`
  instance reused per frame (NFR-UI-M2).
- **Staged tile invariant** — Staged tiles stay in `Rack.tiles` during staging; only `fromRackIdx`
  tracks which are "on the board". `PlayCommand.Execute` removes them on commit.
- **configRoot injection** — `NewSaveManager(tmpDir)` used in all tests to avoid touching
  the real user config directory (NFR-UI-TEST-1).

## Verification

```
go build ./ui/... ./cmd/squabble/...      # compiles cleanly
go test -race ./...                       # all tests pass, no regressions
```
