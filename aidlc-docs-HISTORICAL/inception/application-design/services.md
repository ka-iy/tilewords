# Services — Squabble

## Overview

Squabble is a local desktop/mobile application with no network services, microservices, or external APIs. Orchestration occurs within the `ui` package via the `GameScreen` acting as the game session coordinator. Two additional services handle cross-cutting concerns: async AI computation and persistence.

---

## Service 1: `GameOrchestrator` (implemented as `ui.GameScreen`)

**Purpose**: Coordinates the human turn, AI turn, game rule enforcement, undo, and screen transitions for a single game session.

**Responsibilities**:
- Receive staged tile placements from `InputHandler`
- On "Play": call `engine.Rules.ValidatePlacement` → `engine.Scorer.Score` → `engine.PlayCommand.Execute`
- On "Exchange": call `engine.ExchangeCommand.Execute`
- On "Pass": call `engine.PassCommand.Execute`
- On "Undo": call `engine.GameState.LastCommand.Undo`
- After each human move: check `engine.Rules.IsGameOver`; if not, trigger `AIWorker.Request`
- Poll `AIWorker` each frame; on result: apply AI move via command, check game over, advance turn
- On game over: call `engine.Rules.ApplyEndgameScoring`, transition to `EndGameScreen`
- On "Save": delegate to `SaveManager.Save`

**Interactions**:
- Reads from: `engine.GameState`, `dictionary.Dictionary`
- Writes via: `engine.Command` implementations
- Delegates to: `AIWorker` (async), `SaveManager`, `BoardRenderer`, `RackRenderer`, `InputHandler`

**Invariants**:
- Only one `Command` is in flight at a time (human or AI, never both)
- `AIWorker` is only requested when it is not already computing
- `GameState.LastCommand` is set before any undo can be triggered

---

## Service 2: `AIWorker` (implemented in `ui`, delegates to `ai.AIPlayer`)

**Purpose**: Isolates AI move computation in a background goroutine to prevent blocking the Ebitengine game loop (which must tick at ≥30 fps).

**Responsibilities**:
- Accept a move request (game state snapshot, dictionary, level) via a request channel
- Invoke `ai.AIPlayer.ChooseMove` on its goroutine
- Deliver the computed `engine.Move` back via a result channel
- Expose non-blocking `Poll()` for the game loop to check for a ready result

**Concurrency contract**:
- One goroutine, one request at a time
- `GameScreen.Update()` calls `Poll()` every frame — O(1), non-blocking
- `GameState` passed to `Request()` must be a value copy (not a pointer) to avoid data races

---

## Service 3: `SaveManager` (implemented in `ui`)

**Purpose**: Provides durable local persistence of a single game slot using `encoding/gob`.

**Responsibilities**:
- Determine the platform-specific app data path at construction time:
  - Linux/macOS: `$XDG_DATA_HOME/squabble/save.gob` or `~/.local/share/squabble/save.gob`
  - Windows: `%APPDATA%\squabble\save.gob`
  - Android: returned by the mobile platform's data directory API
- Encode `engine.GameState` to gob and write atomically (write to temp file, rename)
- Set file permissions to user-only (0600) on creation
- Decode and return `engine.GameState` on load
- Detect and report corrupted save files without crashing (SECURITY-15, NFR-04)

**Interactions**:
- Called by `GameScreen` on save/resume actions
- Called by `MainMenuScreen` to check `Exists()` for enabling "Resume" button

---

## Service Interaction Summary

```
MainMenuScreen
    |-- checks SaveManager.Exists() --> enables/disables Resume
    |-- on New Game --> SetupScreen
    |-- on Resume   --> GameScreen (via SaveManager.Load)

SetupScreen
    |-- on Start --> GameScreen (new engine.GameState)

GameScreen  [GameOrchestrator]
    |-- InputHandler   --> staged tile list
    |-- Play/Exchange/Pass --> engine.Command.Execute --> GameState
    |-- Undo           --> engine.Command.Undo --> GameState
    |-- after human move --> AIWorker.Request (async)
    |-- every frame    --> AIWorker.Poll --> if ready: engine.Command.Execute
    |-- every frame    --> engine.Rules.IsGameOver
    |-- on game over   --> EndGameScreen
    |-- Save           --> SaveManager.Save
    |-- BoardRenderer + RackRenderer --> Draw

AIWorker [goroutine]
    |-- receives request --> ai.AIPlayer.ChooseMove
    |-- returns engine.Move via channel --> GameScreen.Poll

EndGameScreen
    |-- New Game --> SetupScreen
    |-- Quit     --> exit
```
