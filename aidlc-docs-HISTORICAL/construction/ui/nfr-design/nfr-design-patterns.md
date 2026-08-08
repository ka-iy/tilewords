# NFR Design Patterns — Unit 4: `ui`

> **Correction addendum** — This document reflects the initial **Ebitengine** design. The UI
> was implemented on **Fyne** instead, so parts below (game loop, `Screen` FSM, pixel
> renderers, input polling, fixed 960×640 resolution) do not match the shipped code. See
> `aidlc-docs/corrections.md` and the post-v1 `ui` functional-design addenda for the actual
> design. The project name is **TileWords**, not "Squabble".

## Pattern 1: Screen Interface FSM (NFR-UI-T1, NFR-UI-P6)

**Problem**: The application transitions between four screens (MainMenu, Setup, Game,
EndGame). Coupling transition logic into `Game` creates a monolithic update function.

**Solution**: Each screen is a value implementing `Screen`. `Game.Update` calls
`screen.Update(g)` and replaces `screen` with the returned value. Transition is a
single pointer assignment — O(1), no blocking I/O on the hot path.

```go
type Screen interface {
    Update(g *Game) (Screen, error)
    Draw(dst *ebiten.Image, g *Game)
}

func (g *Game) Update() error {
    next, err := g.screen.Update(g)
    if err != nil {
        return err
    }
    g.screen = next
    return nil
}
```

A screen that returns itself stays active. Returning a different screen transitions
immediately on the next frame. This eliminates the need for a separate state machine
enum and centralises transition logic inside each screen.

---

## Pattern 2: Pre-Allocated Draw Options (NFR-UI-M2)

**Problem**: `ebiten.DrawImageOptions` is a struct allocated on every `Draw` call if
created inline. With 225 board cells + 14 rack tiles rendered each frame, this produces
hundreds of allocations per frame, triggering GC pauses that drop frame rate.

**Solution**: Each renderer pre-allocates a single `*ebiten.DrawImageOptions` and resets
it before each use by assigning the zero value of the embedded `GeoM`.

```go
type BoardRenderer struct {
    opts ebiten.DrawImageOptions // allocated once at struct construction
    ...
}

func (r *BoardRenderer) drawCell(dst *ebiten.Image, px, py int, col color.RGBA) {
    r.opts.GeoM.Reset()
    r.opts.GeoM.Translate(float64(px), float64(py))
    dst.DrawImage(cellImg, &r.opts)
}
```

Since `Draw` always runs on the main goroutine (Ebitengine guarantee), there is no
concurrent access to `opts`. Zero allocations in steady-state `Draw`.

---

## Pattern 3: Atomic Save via Temp-File Rename (NFR-UI-R3, SECURITY-UI-1)

**Problem**: Writing directly to `savegame.gob` risks a half-written file if the process
is killed mid-write, corrupting the only save slot.

**Solution**: Write to a sibling temp file, then `os.Rename` atomically replaces the
target. On POSIX systems (Linux, macOS, Android) `rename(2)` is atomic. On Windows,
Go's `os.Rename` handles the cross-file-system case.

```go
func (sm *SaveManager) Save(state *engine.GameState) error {
    tmp := sm.path + ".tmp"
    f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
    if err != nil {
        return fmt.Errorf("ui.SaveManager.Save: %w", err)
    }
    if err = gob.NewEncoder(f).Encode(state); err != nil {
        f.Close()
        os.Remove(tmp)
        return fmt.Errorf("ui.SaveManager.Save: encode: %w", err)
    }
    if err = f.Close(); err != nil {
        return fmt.Errorf("ui.SaveManager.Save: close: %w", err)
    }
    if err = os.Rename(tmp, sm.path); err != nil {
        return fmt.Errorf("ui.SaveManager.Save: rename: %w", err)
    }
    return nil
}
```

---

## Pattern 4: AI Timeout Guard in Update (NFR-UI-R4)

**Problem**: `ai.AIWorker` is guaranteed to finish in ≤500 ms on desktop hardware (NFR-AI-P1),
but on slow devices or edge cases it might stall. `Poll()` is non-blocking, so the UI
keeps running — but the human is stuck in "AI thinking…" indefinitely.

**Solution**: Record `aiRequestTime` when `worker.Request` is called. Each `Update` tick
during AITurn checks the elapsed time. If >10 s, log the timeout and transition to an
error state.

```go
// In GameScreen.Update during AITurn:
if time.Since(gs.aiRequestTime) > 10*time.Second {
    // AI goroutine is stalled; transition to safe state.
    return NewMainMenuScreen("AI timed out. Returning to menu."), nil
}
if move, ok := gs.worker.Poll(); ok {
    gs.applyAIMove(move)
}
```

The 10-second threshold is generous; normal move generation completes in <500 ms.

---

## Pattern 5: Staged Tile Invariant via Rack Shadow (Correctness)

**Problem**: Tiles staged on the board are temporarily absent from the rack. If the staged
slice and the in-memory rack become inconsistent (e.g. due to a bug), the player could
commit more tiles than they hold.

**Solution**: `stageTile` removes the tile from `state.HumanRack` immediately (not just
marks it). `recallStagedTile` adds it back. This makes the rack the authoritative source
of truth; staged tiles are physically absent from it. `commitPlay` builds `Placed` directly
from `staged` and calls `engine.ValidatePlacement` as the gate — if the rack was somehow
corrupted, the engine detects the inconsistency.

The PBT-UI-03 property (`count(rack) + count(staged)` is invariant across stage/recall)
verifies this at test time.

---

## Pattern 6: Error Sanitisation at UI Boundary (SECURITY-UI-2)

**Problem**: Errors from `engine.ValidatePlacement`, `dictionary.Load`, and `os` file
operations may contain internal paths, type names, or verbose Go error chains that
should not be shown to the user.

**Solution**: A `sanitiseError` helper trims the error to a single human-readable
sentence. It is called at every point where an error message is assigned to `statusMsg`
or `errMsg`.

```go
// sanitiseError returns a short user-facing message for err.
// It never exposes file paths, type names, or stack frames.
func sanitiseError(err error) string {
    if err == nil {
        return ""
    }
    msg := err.Error()
    // Strip any path prefix (everything before the last ": ").
    if i := strings.LastIndex(msg, ": "); i >= 0 {
        msg = msg[i+2:]
    }
    if len(msg) > 120 {
        msg = msg[:120] + "…"
    }
    return msg
}
```

Engine and dictionary errors already carry a function-name prefix per SECURITY-AI-1 and
SECURITY-15; `sanitiseError` strips that prefix so users see only the trailing description.

---

## Pattern 7: Headless-Testable Pure Functions (NFR-UI-TEST-5)

**Problem**: Ebitengine's `ebiten.RunGame` requires a display; running it in `go test`
without a display causes a panic or hangs. This would make all `ui` tests display-dependent.

**Solution**: All logic that does not call any `ebiten.*` function is extracted into plain
Go functions or methods on non-ebiten types. These are tested directly without
starting the game loop.

Examples of headless-testable functions:
- `cellAt(originX, originY, cellSize, px, py int) (row, col int, ok bool)` — board hit-test
- `SaveManager.Save / Load` — file I/O only
- `sanitiseError(err error) string` — pure
- `button.IsClicked(mx, my int) bool` — pure geometry check
- Screen transition predicates (e.g. `shouldTransitionToEndGame`)

Ebitengine `Draw` and `Update` methods are NOT unit-tested; they are exercised through
manual UI testing and the integration test in `cmd/squabble`.
