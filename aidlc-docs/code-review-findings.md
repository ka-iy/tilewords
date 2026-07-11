# Code Review Findings

Reviewed: 2026-04-19  
Scope: all Go source files in `engine/`, `dictionary/`, `ai/`, `ui/`, `cmd/`, `tools/`  
Status: all issues fixed; `go build ./...` and `go test -race ./...` pass clean.

---

## Summary

| Severity | Count | All fixed? |
|---|---|---|
| Bug — functional | 3 | ✓ |
| Bug — rendering | 1 | ✓ |
| Dead code | 2 | ✓ |
| Style / convention | 1 | ✓ |
| Test quality | 1 | ✓ |

---

## BUG-01 — Exchange mode unreachable (functional)

**File:** `ui/gamescreen.go`  
**Severity:** Bug — the Exchange action could never be invoked by the player.

### Root cause

`isExchangeMode()` returned `len(gs.exchangeSel) > 0`, and the only code that populated `exchangeSel` was guarded by `if gs.isExchangeMode()`. This was a circular dependency: exchange selection could never be initiated because the condition required it to already be non-empty.

The Exchange button was also only enabled when `hasExchangeSel && !hasStagedTiles`, which further entrenched the deadlock.

### Fix

Added an explicit `exchangeMode bool` field to `GameScreen`. The Exchange button now acts as a toggle:

- **First click** (exchangeMode = false): enters exchange mode and prompts the player to click tiles.
- **Second click** (exchangeMode = true, tiles selected): commits the exchange.
- **Second click** (exchangeMode = true, no tiles selected): cancels exchange mode.

`syncButtons` now enables the Exchange button whenever it is the human's turn with no staged tiles, regardless of exchangeSel state. The button label updates dynamically to `"Exchange"` / `"Confirm Exchange"` / `"Cancel Exchange"`.

`handleTileInput` now reads `gs.exchangeMode` directly instead of calling the removed `isExchangeMode()` helper. `recallAll` and `commitExchange` both reset `exchangeMode = false` and clear `exchangeSel`.

Pass and Undo buttons are additionally disabled while `exchangeMode` is active, preventing conflicting actions mid-exchange.

---

## BUG-02 — `wrapText` drops a character on hard cuts (functional)

**File:** `ui/render.go`, `wrapText` function  
**Severity:** Bug — long words without spaces caused one character to be silently dropped per line.

### Root cause

```go
if cut == 0 {
    cut = maxChars // no space found — hard cut
}
lines = append(lines, text[:cut])
text = text[cut+1:] // BUG: skips text[maxChars] when cut==maxChars
```

When no space was found within `maxChars` characters, `cut` was reset to `maxChars`. The slice advancement `text[cut+1:]` then skipped the character at position `maxChars` as if it were a space, silently dropping it from the output.

### Fix

Separated the two cases:

```go
if cut == 0 {
    // No space within maxChars: hard cut, no space character to skip.
    lines = append(lines, text[:maxChars])
    text = text[maxChars:]
} else {
    lines = append(lines, text[:cut])
    text = text[cut+1:] // skip the space
}
```

---

## BUG-03 — Error status colour never applied (rendering)

**File:** `ui/render.go` (`ScorePanel.Draw`), `ui/gamescreen.go`  
**Severity:** Bug — error messages were displayed in the same green colour as success messages.

### Root cause

`GameScreen` tracked `statusIsErr bool` via `setStatus(msg, isErr)`, but `ScorePanel.Draw` received only `statusMsg string`; the `isErr` flag was never passed and never consulted. All status messages rendered in `colorStatusOK` (light green) regardless of whether they were errors.

### Fix

Added `statusErr bool` as a parameter to `ScorePanel.Draw`. The panel now selects `colorStatusErr` (light red) when `statusErr` is true, `colorText` when it is the AI's turn, and `colorStatusOK` (light green) otherwise. `GameScreen.Draw` passes `gs.statusIsErr` to the panel.

---

## DEAD-01 — `BoardRenderer.opts` field never used

**File:** `ui/render.go`  
**Severity:** Dead code — the field was added per NFR-UI-M2 (pre-allocate draw options to avoid heap allocation), but `BoardRenderer.Draw` never references it. The board is rendered entirely through `fillRect` and `vector.StrokeRect`, neither of which takes `ebiten.DrawImageOptions`.

### Fix

Removed the `opts ebiten.DrawImageOptions` field from `BoardRenderer`. The struct is now empty (`type BoardRenderer struct{}`).

---

## DEAD-02 — `HumanRackOriginX()` exported for no reason

**File:** `ui/gamescreen.go`  
**Severity:** Dead code / unnecessary export — the function simply returned `RackOriginX` and was only called once, within the same package.

```go
// Before
func HumanRackOriginX() int { return RackOriginX }
// ...
rackTileAt(HumanRackOriginX(), HumanRackOriginY, mx, my)
```

### Fix

Removed `HumanRackOriginX()`. The single call site now uses `RackOriginX` directly:

```go
rackTileAt(RackOriginX, HumanRackOriginY, mx, my)
```

---

## STYLE-01 — Split import statements in `ai/crosscheck.go`

**File:** `ai/crosscheck.go`, lines 4–5  
**Severity:** Style / convention — Go convention (enforced by `gofmt`) is a single import block.

```go
// Before
import "squabble/dictionary"
import "squabble/engine"

// After
import (
    "squabble/dictionary"
    "squabble/engine"
)
```

---

## TEST-01 — `TestSaveManager_RoundTrip` only verified `AILevel`

**File:** `ui/ui_test.go`  
**Severity:** Test quality — the round-trip test saved a `GameState` and loaded it back, but only checked `AILevel`. If `GobEncode`/`GobDecode` for `Board`, `Rack`, or `Bag` were broken, the test would not catch it. (These methods were added to `engine/rack.go` and `engine/bag.go` specifically to support save/load.)

### Fix

Expanded the test to assert:

- `HumanScore` and `AIScore` survive the round-trip
- `Board` is non-nil after load
- `HumanRack.Count()` matches the original
- `Bag.Count()` matches the original

---

## False positives from automated review (no fix needed)

The following agent findings were investigated and determined to be non-issues:

| Finding | Verdict |
|---|---|
| `commands.go` accesses `state.Bag.tiles` (unexported) | `commands.go` is in the same `engine` package; intra-package access to unexported fields is valid Go. |
| Data race on `LastHumanCommand` / `LastAICommand` | Both fields are written only on the UI goroutine. The AI goroutine receives a `Clone()` that deliberately omits them. No concurrent access. |
| `ai/select.go` level-formula boundary conditions | Formula is mathematically correct; level 1 → k=total, level 10 → deterministic best, as documented. |
| `engine.GameState.Clone` shallow-copies tile pointers | Intentional and documented: placed tiles are immutable after placement, so shared pointers are safe for the read-only AI snapshot. |

---

## Verified clean after fixes (Review 1)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

# Review 2 — 2026-04-19

Reviewed: 2026-04-19  
Scope: all Go source files in `engine/`, `dictionary/`, `ai/`, `ui/`, `cmd/`, `tools/`  
Status: all issues fixed; `go build ./...` and `go test -race ./...` pass clean.

---

## Summary

| ID | Severity | File | All fixed? |
|---|---|---|---|
| R2-BUG-01 | Bug — concurrency / UI freeze | `ui/mainmenu.go` | ✓ |
| R2-DEAD-01 | Dead code | `dictionary/gaddag.go` | ✓ |
| R2-DEAD-02 | Style / dead parameter | `ui/mainmenu.go`, `ui/setup.go`, `ui/endgame.go` | ✓ |
| R2-DEAD-03 | Dead code — trivial wrapper | `engine/score.go` | ✓ |

---

## R2-BUG-01 — `dictionary.Load` blocks the UI goroutine in `mainmenu.go`

**File:** `ui/mainmenu.go`  
**Severity:** Bug — UI freezes for the duration of GADDAG decoding when the player clicks Load Game.

### Root cause

The Load Game handler called `dictionary.Load(state.DictName)` synchronously on the Ebitengine game-loop goroutine. `dictionary.Load` decodes a multi-megabyte GADDAG, taking hundreds of milliseconds on slow storage. During this time Ebitengine's `Update`/`Draw` cycle stalls, causing a visible freeze and potential watchdog warnings on mobile.

`SetupScreen` already used the correct async pattern (goroutine + buffered channel + non-blocking poll), but this pattern was not applied to the main-menu load path.

### Fix

Added `loading bool`, `loadCh chan loadResult`, and `pendingState *engine.GameState` fields to `MainMenuScreen`. On Load Game click:

1. `saveManager.Load()` runs synchronously (fast — reads a small gob file).
2. The loaded `*engine.GameState` is stored in `pendingState`.
3. A goroutine is spawned to call `dictionary.Load`; the result is sent on the buffered `loadCh`.
4. `Update` polls `loadCh` non-blocking each tick; on receipt it constructs `GameScreen` and returns it.

`Draw` shows a "Loading dictionary…" message while `loading` is true, and the Load Game button is disabled during loading to prevent double-clicks.

---

## R2-DEAD-01 — `_ = i` in `gaddag.go addString`

**File:** `dictionary/gaddag.go`  
**Severity:** Dead code — the loop index `i` was never used, and the explicit blank discard was misleading.

### Root cause

```go
for i, b := range seq {
    // ... i is never referenced ...
    _ = i
}
```

The `_ = i` suppressed a compiler error at some earlier revision. After removing all uses of `i`, the blank discard was left in.

### Fix

Changed `for i, b := range seq` to `for _, b := range seq` and removed the `_ = i` statement.

---

## R2-DEAD-02 — `g *Game` unused in `Draw`/`Update` method signatures

**Files:** `ui/mainmenu.go`, `ui/setup.go`, `ui/endgame.go`  
**Severity:** Style — the `Screen` interface requires `g *Game` as a parameter, but none of these three screens ever reference the argument. Go allows unused function parameters without error, but naming them signals that they should be used.

### Fix

Replaced `g *Game` with `_ *Game` in all six affected method signatures (`Update` and `Draw` on each of the three screens). This makes the intent explicit: the parameter is required by the interface but intentionally ignored by these implementations.

---

## R2-DEAD-03 — `isNewTile` one-liner wrapper in `engine/score.go`

**File:** `engine/score.go`  
**Severity:** Dead code — `isNewTile` was a single-line wrapper around `covers` with no added semantics, documentation, or test coverage of its own.

```go
// Before
func isNewTile(placed []PlacedTile, row, col int) bool {
    return covers(placed, row, col)
}
```

### Fix

Removed `isNewTile`. The single call site in `sumWord` now calls `covers` directly:

```go
if covers(placed, r, c) {
```

---

## Verified clean after fixes (Review 2)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

# Review 3 — 2026-04-19

Reviewed: 2026-04-19  
Scope: all Go source files in `engine/`, `dictionary/`, `ai/`, `ui/`, `cmd/`, `tools/`  
Status: all issues fixed; `go build ./...` and `go test -race ./...` pass clean.

---

## Summary

| ID | Severity | File | All fixed? |
|---|---|---|---|
| R3-BUG-01 | Bug — fragile sentinel comparison | `cmd/squabble/main.go`, `ui/game.go` | ✓ |
| R3-DEAD-01 | Dead code — stdlib reimplementation | `ui/save.go` | ✓ |
| R3-DEAD-02 | Dead code — exported no-op method | `ai/worker.go` | ✓ |
| R3-DEAD-03 | Style — unused parameter | `ui/gamescreen.go` | ✓ |
| R3-PERF-01 | Performance — unnecessary heap allocation | `ai/record.go` | ✓ |

---

## R3-BUG-01 — Error sentinel compared by string value in `main.go`

**Files:** `cmd/squabble/main.go`, `ui/game.go`  
**Severity:** Bug — fragile: string comparison on an error breaks silently if the error message ever changes.

### Root cause

`ui/game.go` declared `errQuit` as unexported. `cmd/squabble/main.go` had no access to the sentinel, so it compared by message text:

```go
if err.Error() != "quit" {
    log.Fatalf(...)
}
```

If the message were ever changed (e.g., to "user quit" during a refactor), the check would silently accept it as a crash and call `log.Fatalf`, making the game log a fatal error every time the user quits normally.

### Fix

Renamed `errQuit → ErrQuit` and exported it from `ui`. Both internal call sites (`game.go:Update`, `mainmenu.go:Update`) updated. `main.go` now uses the idiomatic `errors.Is`:

```go
if !errors.Is(err, ui.ErrQuit) {
    log.Fatalf("squabble: %v", err)
}
```

---

## R3-DEAD-01 — `indexOf` reimplements `strings.Index` in `ui/save.go`

**File:** `ui/save.go`  
**Severity:** Dead code / unnecessary reimplementation — the function is a byte-by-byte reimplementation of the standard library function `strings.Index`.

### Root cause

`sanitiseError` needed to find the first ": " separator to strip function-name prefixes. Rather than importing `"strings"`, a local `indexOf` helper was written.

### Fix

Removed `indexOf`. Added `"strings"` import. The stripping loop now uses `strings.Cut`, which also simplifies the code:

```go
// Before
for {
    i := indexOf(msg, ": ")
    if i < 0 { break }
    msg = msg[i+2:]
}

// After
for {
    _, after, found := strings.Cut(msg, ": ")
    if !found { break }
    msg = after
}
```

---

## R3-DEAD-02 — `AIWorker.Start()` is an exported no-op

**File:** `ai/worker.go`  
**Severity:** Dead code — the method has an empty body and is called nowhere in the codebase.

### Root cause

The method was added "for API symmetry" during an early design pass but the actual goroutine launch was moved into `NewAIWorker`. `Start()` was never removed, leaving an exported symbol with no behaviour.

### Fix

Removed `Start()` entirely. The goroutine is documented as launched by `NewAIWorker`.

---

## R3-DEAD-03 — `GameScreen.Draw` receives unused `g *Game`

**File:** `ui/gamescreen.go`  
**Severity:** Style — consistent with R2-DEAD-02 (fixed in Review 2 for three other screens).

### Fix

Renamed `g *Game` to `_ *Game` in the `GameScreen.Draw` signature. All four `Screen` implementations now uniformly use `_` for the `*Game` parameter they do not reference.

---

## R3-PERF-01 — `computeOpponentAccess` allocates a heap map per candidate

**File:** `ai/record.go`  
**Severity:** Performance — called once per generated move candidate. On boards with many anchor squares, this function is called hundreds or thousands of times per AI turn; each call previously allocated a `map[pos]bool` with capacity 225.

### Root cause

```go
type pos struct{ r, c int }
occupied := make(map[pos]bool, 225)
```

Map allocation involves heap pressure. The grid is always 15×15, so the size is statically known.

### Fix

Replaced the map with a stack-allocated array:

```go
var occupied [15][15]bool
```

The neighbour bounds check is made explicit (previously the map returned `false` for out-of-bounds keys implicitly; the array requires an explicit guard):

```go
nr, nc := r+d[0], c+d[1]
if nr >= 0 && nr < 15 && nc >= 0 && nc < 15 && occupied[nr][nc] {
```

---

## Verified clean after fixes (Review 3)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

# Review 4 — 2026-04-20

Reviewed: 2026-04-20  
Scope: all Go source files in `engine/`, `dictionary/`, `ai/`, `ui/`, `cmd/`, `tools/`  
Status: all issues fixed; `go build ./...` and `go test -race ./...` pass clean.

---

## Summary

| ID | Severity | File | All fixed? |
|---|---|---|---|
| R4-BUG-01 | Bug — post-timeout panic | `ai/worker.go`, `ui/gamescreen.go` | ✓ |
| R4-TEST-01 | Test quality — PlayMove never applied | `ai/ai_pbt_test.go` | ✓ |
| R4-DEAD-01 | Style — unused parameter | `ui/gamescreen.go` | ✓ |
| R4-STYLE-01 | Style — wrong column count in comment | `ui/blank_picker.go` | ✓ |
| R4-STYLE-02 | Dead code — redundant blank identifier | `ai/ai_pbt_test.go` | ✓ |

---

## R4-BUG-01 — AI timeout leaves `worker.busy = true`, causing next `Request` to panic

**Files:** `ai/worker.go`, `ui/gamescreen.go`  
**Severity:** Bug — after the 10-second AI timeout fires, the game panics on the next human move.

### Root cause

When `Poll` returns a result, it clears `worker.busy`. But the timeout path in Phase 2 of `GameScreen.Update` calls `applyAIMove(PassMove{})` directly, bypassing `Poll`:

```go
gs.applyAIMove(engine.PassMove{})   // turn flips to human; worker.busy still true
```

After the turn flips, Phase 2 is skipped (human's turn). When the human plays and `startAITurn` calls `worker.Request`, it sees `busy == true` and panics:

```
panic: ai.AIWorker.Request: called while a previous request is still in flight
```

### Fix

Added `Reset()` to `AIWorker`. It drains any pending result from `resCh` (non-blocking) and clears `busy`:

```go
func (w *AIWorker) Reset() {
    select {
    case <-w.resCh:
    default:
    }
    w.busy = false
}
```

The timeout path now calls `Reset()` before `applyAIMove`:

```go
gs.worker.Reset()
gs.applyAIMove(engine.PassMove{})
gs.setStatus("AI timed out — pass applied.", true)
```

---

## R4-TEST-01 — `boardStateGen` in `ai_pbt_test.go` never applies `PlayMove`

**File:** `ai/ai_pbt_test.go`  
**Severity:** Test quality — the board-variety generator silently failed to evolve state on every iteration, making all PBT scenarios run on an empty board regardless of the "moves" draw.

### Root cause

```go
case engine.PlayMove:
    cmd := &engine.PlayCommand{}              // Move field left zero
    if err := cmd.Execute(state, testDict, rng); err == nil {
        _ = m                                 // m (the actual move) never used
    }
```

`PlayCommand.Execute` calls `ValidatePlacement` on `cmd.Move.Placed`, which is nil. That returns `"no tiles placed"`, so the `err == nil` branch is never entered. The board never changes. `_ = m` suppressed the type-assertion variable but the move itself was discarded.

### Fix

```go
case engine.PlayMove:
    cmd := &engine.PlayCommand{Move: m}
    _ = cmd.Execute(state, testDict, rng)
```

The `_ = move` after the switch was also removed: `move` is already consumed by the `switch` statement and the extra blank is dead code.

---

## R4-DEAD-01 — `GameScreen.Update` receives unused `g *Game`

**File:** `ui/gamescreen.go`  
**Severity:** Style — `g` is never referenced inside `Update`. `GameScreen.Draw` was fixed in Review 3; `Update` was missed.

### Fix

`func (gs *GameScreen) Update(g *Game)` → `func (gs *GameScreen) Update(_ *Game)`.  
All `Screen` method signatures now uniformly use `_` for the `*Game` parameter they do not reference.

---

## R4-STYLE-01 — Stale "4-column" comment in `blank_picker.go`

**File:** `ui/blank_picker.go`  
**Severity:** Style — the doc comment for `NewBlankPickerOverlay` said "A–Z in a 4-column grid" but the code uses `cols = 7`.

### Fix

Updated the comment to "7-column grid" and replaced the vague "27th slot" description with "below the last letter row".

---

## R4-STYLE-02 — `_ = move` dead code in `boardStateGen`

**File:** `ai/ai_pbt_test.go`  
**Severity:** Style — `move` is already consumed by the `switch m := move.(type)` statement immediately before. The subsequent `_ = move` is a no-op.

### Fix

Removed the line as part of the R4-TEST-01 fix.

---

## Verified clean after fixes (Review 4)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

# Review 5 — 2026-04-20

Reviewed: 2026-04-20  
Scope: all Go source files in `engine/`, `dictionary/`, `ai/`, `ui/`, `cmd/`, `tools/`  
Status: all issues fixed; `go build ./...` and `go test -race ./...` pass clean.

---

## Summary

| ID | Severity | File | All fixed? |
|---|---|---|---|
| R5-BUG-01 | Bug — save fails mid-game | `ui/save.go`, `ui/ui_test.go` | ✓ |
| R5-STYLE-01 | Style — wrong rule-ref prefix | `engine/rules.go` | ✓ |
| R5-STYLE-02 | Style — wrong method name in comment | `dictionary/gaddag.go` | ✓ |
| R5-STYLE-03 | Style — unchecked Close error | `tools/buildgaddag/main.go` | ✓ |

---

## R5-BUG-01 — `SaveManager.Save` fails mid-game with "gob: type not registered for interface"

**Files:** `ui/save.go`, `ui/ui_test.go`  
**Severity:** Bug — saving a game in progress silently fails after the first move is played.

### Root cause

`engine.GameState.LastHumanCommand` and `LastAICommand` are exported fields of type `engine.Command` (an interface). When `SaveManager.Save` encodes the state with `gob.Encode`, gob must serialise every exported field. For interface-typed fields:

- **nil value**: encodes safely (empty interface slot).
- **non-nil value**: gob requires the concrete type to be pre-registered via `gob.Register`. Without registration, `Encode` returns `"gob: type not registered for interface: engine.Command"`.

`engine.GameState` holds non-nil commands after any move is played (`*PlayCommand`, `*ExchangeCommand`, `*PassCommand`). The existing `TestSaveManager_RoundTrip` only tests a freshly-created state (both fields nil), so the bug was not caught.

The design intent is explicit: `// These fields are intentionally excluded from save files`—undo history is discarded on save/load.

### Fix

`Save` now encodes a shallow struct copy with both interface fields zeroed before passing to `gob.Encode`:

```go
saveable := *state
saveable.LastHumanCommand = nil
saveable.LastAICommand = nil
encErr := gob.NewEncoder(f).Encode(&saveable)
```

The shallow copy is safe because the pointer fields (`Board`, `HumanRack`, `AIRack`, `Bag`) are only read by gob, never mutated.

A regression-preventing test (`TestSaveManager_RoundTrip_WithCommands`) was added to `ui/ui_test.go`. It sets both command fields to `&engine.PassCommand{}` before saving, verifying the save succeeds and the loaded state has nil commands.

---

## R5-STYLE-01 — "BL-E10/11" should be "BR-E10/11" in `engine/rules.go`

**File:** `engine/rules.go`  
**Severity:** Style — two doc comments used the prefix `BL-` instead of `BR-` (Business Rule), inconsistent with every other rule cross-reference in the codebase.

```go
// Before
// IsGameOver reports whether state meets any end condition (BL-E10).
// ApplyEndgameScoring adjusts scores when the game ends (BL-E11, BR-E12).

// After
// IsGameOver reports whether state meets any end condition (BR-E10).
// ApplyEndgameScoring adjusts scores when the game ends (BR-E11, BR-E12).
```

---

## R5-STYLE-02 — `words()` doc comment names the wrong method

**File:** `dictionary/gaddag.go`  
**Severity:** Style — the doc comment read `"wordCount returns the number of distinct words…"` but the method is named `words()`, not `wordCount`.

```go
// Before
// wordCount returns the number of distinct words stored in this GADDAG.
func (g *GADDAG) words() int { return int(g.wordCount) }

// After
// words returns the number of distinct words stored in this GADDAG.
func (g *GADDAG) words() int { return int(g.wordCount) }
```

---

## R5-STYLE-03 — `f.Close()` error silently dropped in `buildgaddag/main.go`

**File:** `tools/buildgaddag/main.go`  
**Severity:** Style — `f.Close()` was called but its return value was discarded with no check.

### Fix

Checked `scanner.Err()` first (scan errors take priority), then checked and returned the `Close` error:

```go
// Before
f.Close()
if err := scanner.Err(); err != nil { ... }

// After
if err := scanner.Err(); err != nil {
    f.Close()
    return nil, fmt.Errorf("scan %q: %w", path, err)
}
if err := f.Close(); err != nil {
    return nil, fmt.Errorf("close %q: %w", path, err)
}
```

---

## Verified clean after fixes (Review 5)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

## Review 6 — 2026-04-19

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/).

### Summary

| ID | File(s) | Category | Description |
|----|---------|----------|-------------|
| R6-DEAD-01 | engine/rack.go, engine/state.go, engine/commands.go | Dead parameter | `Rack.Replenish` accepted `rng *rand.Rand` that was never used in the body |
| R6-DEAD-02 | engine/bag.go → engine/testhelpers_test.go | Dead export | `NewTestBag` exported into the production binary but only used in tests |
| R6-DEAD-03 | ui/render.go, ui/gamescreen.go | Dead field | `RackRenderer.interactive bool` set in two initialisers but never read |

### R6-DEAD-01 — `Rack.Replenish` unused `rng` parameter

**Root cause**: `Bag.Draw` pops tiles from the end of the pre-shuffled slice, so drawing is already random; no further use of `rng` is needed inside `Replenish`. The parameter was a legacy carry-over from an earlier design where the method itself shuffled.

**Fix**: Removed `rng *rand.Rand` from the `Replenish` signature and the `"math/rand"` import from `engine/rack.go`. Updated three call sites: `engine/state.go` (two: `humanRack.Replenish(bag, rng)` → `humanRack.Replenish(bag)`, `aiRack.Replenish(bag, rng)` → `aiRack.Replenish(bag)`) and `engine/commands.go` (one: `rack.Replenish(state.Bag, rng)` → `rack.Replenish(state.Bag)`).

### R6-DEAD-02 — `NewTestBag` exported from production binary

**Root cause**: `NewTestBag` was placed in `engine/bag.go` (compiled into the production binary) but is only referenced from `engine/engine_test.go`. Exported test helpers in non-test files bloat the production binary and expose internal constructors as public API.

**Fix**: Removed `NewTestBag` from `engine/bag.go`; added an unexported equivalent `newTestBag` to `engine/testhelpers_test.go`. Updated all three call sites in `engine/engine_test.go` (`NewTestBag(` → `newTestBag(`). Since both files are in `package engine`, the test file can access unexported bag internals directly.

### R6-DEAD-03 — `RackRenderer.interactive` field never read

**Root cause**: The `interactive bool` field was added to `RackRenderer` to distinguish the human rack (click-enabled) from the AI rack (display-only). Click detection is handled entirely in `gamescreen.go` via `rackTileAt`, not in `RackRenderer.Draw`, so the field became dead weight.

**Fix**: Removed `interactive bool` from `RackRenderer` struct in `ui/render.go`. Removed `interactive: true` and `interactive: false` from the two `RackRenderer` initialisers in `ui/gamescreen.go`. Updated doc comment on the struct (removed the stale reference to the field).

---

## Verified clean after fixes (Review 6)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

## Review 7 — 2026-04-20

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files.

### Summary

| ID | File(s) | Category | Description |
|----|---------|----------|-------------|
| R7-DEAD-01 | engine/board.go → engine/testhelpers_test.go | Dead export | `newFlatBoard` test-only helper sitting in production binary |
| R7-STYLE-01 | ui/game.go | Stale comment | Doc comment on `Update` still says `errQuit` after R3 renamed it to `ErrQuit` |
| R7-STYLE-02 | engine/engine_test.go | Shadowed builtin | Variable named `copy` shadowed the `copy` built-in function |

### R7-DEAD-01 — `newFlatBoard` in production binary

**Root cause**: Same pattern as R6-DEAD-02 (`NewTestBag`). `newFlatBoard()` returns a zero-value `*Board` (all Normal squares, no tiles) specifically to give tests predictable face-value-only scoring. It lives in `engine/board.go` (production code) but is only referenced from `engine/engine_test.go` and `engine/engine_pbt_test.go`.

**Fix**: Removed `newFlatBoard` from `engine/board.go`; added it to `engine/testhelpers_test.go`. Because both files are in `package engine`, the test-file version has full access to the unexported `Board` zero value. No callers needed updating (function name and signature unchanged; only file of residence changed).

### R7-STYLE-01 — Stale `errQuit` reference in `ui/game.go` doc comment

**Root cause**: Review 3 (R3-BUG-01) exported the quit sentinel as `ErrQuit` and updated all code paths. The doc comment on `Game.Update` was overlooked and still read `"Returns errQuit to terminate."`.

**Fix**: Updated the comment to `"Returns ErrQuit to terminate."` at `ui/game.go:36`.

### R7-STYLE-02 — `copy` variable shadows built-in in `engine/engine_test.go`

**Root cause**: `TestBoard_GobRoundTrip` used `copy := b2.cells[r][c]` to hold a decoded cell, shadowing Go's built-in `copy` function within that loop body. Go does not error on built-in shadowing, but it confuses static analysis tools and readers.

**Fix**: Renamed `copy` → `got` and `copyEmpty` → `gotEmpty` throughout the loop body in `TestBoard_GobRoundTrip`.

---

## Verified clean after fixes (Review 7)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

## Review 8 — 2026-04-20

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files, with particular focus on package-level doc.go files not examined in previous reviews.

### Summary

| ID | File | Category | Description |
|----|------|----------|-------------|
| R8-DOC-01 | engine/doc.go | Wrong example | Usage example calls `cmd.Execute(state, rng)` — missing the `dict` argument |
| R8-DOC-02 | ai/doc.go | Wrong example | Usage example calls `worker.Start()` (deleted in R3-DEAD-02) and passes `state.Clone()` to `Request` (which clones internally) |

### R8-DOC-01 — `engine/doc.go` usage example wrong signature

**Root cause**: The package-level usage example predates the `dict *dictionary.Dictionary` parameter being added to `Command.Execute`. The example showed `cmd.Execute(state, rng)` but the actual interface requires three arguments: `Execute(state *GameState, dict *dictionary.Dictionary, rng *rand.Rand) error`. A developer copy-pasting from the doc would get a compile error.

**Fix**: Updated example to `cmd.Execute(state, dict, rng)` in `engine/doc.go`.

### R8-DOC-02 — `ai/doc.go` usage example references deleted method and causes redundant clone

**Root cause** (two sub-issues):
1. `worker.Start()` — this no-op method was removed in R3-DEAD-02 but remained in the doc example. Any code following the docs verbatim would fail to compile.
2. `worker.Request(state.Clone(), dict, level)` — `AIWorker.Request` already calls `state.Clone()` internally (SECURITY-AI-2: the clone ensures the AI goroutine and UI goroutine never share a pointer). A caller passing `state.Clone()` causes a redundant double-clone: one heap allocation for the caller's copy, immediately discarded once `Request` clones it again.

**Fix**: Removed the `worker.Start()` call; changed `worker.Request(state.Clone(), dict, level)` to `worker.Request(state, dict, level)` with a clarifying comment; updated `ai/doc.go`.

---

## Verified clean after fixes (Review 8)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

## Review 9 — 2026-04-20

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files, with particular attention to control-flow conditions in ui/.

### Summary

| ID | File | Category | Description |
|----|------|----------|-------------|
| R9-DEAD-01 | ui/input.go | Dead condition | `row < 0`, `col < 0` in `cellAt`; `idx < 0` in `rackTileAt` are unreachable |

### R9-DEAD-01 — Unreachable negative-index conditions in `ui/input.go`

**Root cause**: Both `cellAt` and `rackTileAt` guard against negative results from integer division, but the division operand is always non-negative by the time it is computed:

- In `cellAt`: the early-return `if px < BoardOriginX || py < BoardOriginY` guarantees `px - BoardOriginX >= 0` and `py - BoardOriginY >= 0`, so `col` and `row` are always ≥ 0. The subsequent `row < 0 || col < 0` arms of the bounds check were dead.
- In `rackTileAt`: the early-return `if px < originX` guarantees `px - originX >= 0`, and `stride = RackTileSize + RackGap = 48 > 0`, so `idx = (px - originX) / stride >= 0` always. The `idx < 0` arm of the bounds check was dead.

**Fix**: Simplified the conditions to remove the dead arms:
```go
// cellAt: was "row < 0 || row > 14 || col < 0 || col > 14"
if row > 14 || col > 14 {

// rackTileAt: was "idx < 0 || idx > 6"
if idx > 6 {
```

---

## Verified clean after fixes (Review 9)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

## Review 10 — 2026-04-20

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files.

### Summary

| ID | File | Category | Description |
|----|------|----------|-------------|
| R10-DEAD-01 | dictionary/names.go | Dead export | `DisplayName` declared but never called anywhere in the codebase |

### R10-DEAD-01 — `DisplayName` never called

**Root cause**: `DisplayName(DictName) string` was added to `dictionary/names.go` to provide human-readable labels for the setup screen's dictionary buttons. The UI setup screen (`ui/setup.go`) instead uses `string(name)` (the raw internal value, e.g. "csw") as each button label. `DisplayName` was never wired up to any caller, production or test.

**Fix**: Removed the dead function from `dictionary/names.go`. The `string(name)` labels ("csw", "sowpods", etc.) remain as the button labels — consistent with the existing 160 px button width constraint (the longest human-readable label, "Official Tournament and Club Word List (OTCWL)", would overflow at 46 chars × 7 px/char = 322 px).

---

## Verified clean after fixes (Review 10)

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

---

## Review 11 — 2026-04-20

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files. Full re-read of every file — 47 files examined.

### Summary

No issues found. The codebase is clean.

---

## Verified clean after Review 11

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

**Review cycle complete.** Eleven consecutive reviews were performed. Issues were found and fixed in Reviews 1–10; Review 11 found no further issues.

---

## Review 12 — 2026-04-20

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files. Full re-read of every file — 47 files examined. Triggered by Makefile rewrite that introduced stale documentation.

### Issue R12-DOC-01 — Stale `go generate` references after Makefile GADDAG integration

**Files:** `dictionary/doc.go`, `dictionary/loader.go`  
**Severity:** Documentation / stale developer guidance — no functional impact.

#### Root cause

The Makefile was rewritten to invoke `go run ./tools/buildgaddag` via proper file-target rules and to embed the resulting `.gob` assets into every build target. This superseded the `//go:generate` workflow, but two source files still referenced it:

1. `dictionary/doc.go` line 22: package doc comment said `go generate ./dictionary/...`.  
2. `dictionary/doc.go` line 37: a non-functional `//go:generate` placeholder directive remained in the file.  
3. `dictionary/loader.go` line 39: the runtime error message for a missing embedded asset said `"run 'go generate ./dictionary/...' to build it"`.

#### Fix applied

- `dictionary/doc.go`: Updated the "Word List Assets" section to say `make gaddag`; removed the non-functional `//go:generate` placeholder directive.
- `dictionary/loader.go`: Updated the runtime error message to say `"run 'make gaddag' to build it"`.

#### Verification

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

### Summary

One documentation issue found and fixed (R12-DOC-01). No functional bugs, dead code, or test quality issues identified.

---

## Review 13 — 2026-04-21

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files. Full re-read of every file — 47 files examined. Triggered by the ENABLE word list integration and setup screen fixes made in this session.

### Issue R13-QUALITY-01 — State mutation in `SetupScreen.Draw`

**File:** `ui/setup.go`  
**Severity:** Code quality — Draw should be a read-only rendering operation.

#### Root cause

The `Draw` method was computing and writing the `selected` boolean field on each dict and level button every frame:

```go
// in Draw():
for i := range s.dictBtns {
    s.dictBtns[i].selected = dictionary.AllDictNames[i] == s.selectedDict  // mutation
    s.dictBtns[i].Draw(dst, mx, my)
}
for i := range s.levelBtns {
    s.levelBtns[i].selected = i+1 == s.selectedLevel  // mutation
    s.levelBtns[i].Draw(dst, mx, my)
}
```

This was introduced as a side-effect of moving from the (broken) `fillRect`-before-Draw approach (R12 session) to the `button.selected` field approach. The computation is idempotent and caused no data race (Ebitengine runs Update and Draw on the same thread), but it violates the principle that rendering functions must not mutate model state, and would become a race if Ebitengine ever decoupled the draw thread.

#### Fix applied

- `initButtons()`: `selected` is now initialised correctly for both dict buttons (`name == s.selectedDict`) and level buttons (`i+1 == s.selectedLevel`), so the initial state is correct before the first frame.
- `Update()`: dict and level click handlers now iterate by index (not range-copy), clear `selected` on all buttons in the group, and set `selected = true` on the clicked button. This keeps `selected` always in sync with `selectedDict`/`selectedLevel`.
- `Draw()`: the `selected =` assignments are removed; Draw now only reads state.

#### Verification

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

### Summary

One code-quality issue found and fixed (R13-QUALITY-01). No functional bugs, dead code, or documentation issues identified.

---

## Review 14 — 2026-04-21

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files. Full re-read of every file — triggered by the removal of CSW/OSPD/NASPA/OTCWL/All dictionaries and the Makefile auto-build change made in this session.

### Issue R14-DOC-01 — Stale `DictNASPA` reference in `engine/doc.go`

**File:** `engine/doc.go`, line 19  
**Severity:** Documentation — compile-time invisible (comment only), but misleading.

#### Root cause

The package-level usage example referenced `dictionary.DictNASPA`, a constant that was deleted when all dictionaries except ENABLE and SOWPODS were removed:

```go
//	state := engine.New(dictionary.DictNASPA, 5, rng)
```

#### Fix applied

Updated the example to use `dictionary.DictENABLE`:

```go
//	state := engine.New(dictionary.DictENABLE, 5, rng)
```

---

### Issue R14-QUALITY-01 — State mutation in `MainMenuScreen.Draw`

**File:** `ui/mainmenu.go`  
**Severity:** Code quality — same Draw-mutates-state anti-pattern as R13-QUALITY-01.

#### Root cause

`Draw()` was mutating `loadGameBtn.enabled` on each frame:

```go
// in Draw():
s.loadGameBtn.enabled = s.saveManager.Exists() && !s.loading
```

The comment said "Refresh load button enabled state (save may have appeared since last draw)." While the intent was valid (the save file could be created/deleted externally), Draw is a rendering operation and must not mutate model state.

#### Fix applied

Moved the enabled-state update to `Update()`, which runs every tick before any rendering:

```go
// in Update():
s.loadGameBtn.enabled = s.saveManager.Exists() && !s.loading
```

Removed the stale line from `Draw()`.

#### Verification

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

### Summary

Two issues found and fixed: one stale documentation reference (R14-DOC-01) and one Draw-mutates-state anti-pattern (R14-QUALITY-01). No functional bugs or test quality issues identified.

---

## Review 15 — 2026-04-21

Scope: all Go source files (engine/, dictionary/, ai/, ui/, cmd/, tools/) and all test files. Full re-read of every file — focused on game-state correctness, undo invariants, and UI state machine edges.

### Issue R15-BUG-01 — Blank tile displays incorrectly in rack after undo

**File:** `engine/commands.go` — `PlayCommand.Undo`  
**Severity:** Bug — visual corruption of rack state after undoing a move that played a blank tile.

#### Root cause

When a blank tile is played, `ui/gamescreen.go:assignBlankLetter` sets both `Tile.AssignedLetter` and `Tile.Letter` to the chosen letter (e.g. `'A'`). Both fields are stored in `cmd.Move.Placed[i].Tile`. `PlayCommand.Undo` returned the tile verbatim:

```go
placedTiles[i] = pt.Tile  // AssignedLetter='A', Letter='A' still set
```

After `rack.Add(placedTiles)`, the rack held a tile with `IsBlank=true, AssignedLetter='A'`. `Tile.DisplayLetter()` returns `AssignedLetter` when `IsBlank && AssignedLetter != 0`, so the rack rendered it as the letter 'A' rather than as a blank tile.

#### Fix applied

Reset `Letter` and `AssignedLetter` for blank tiles before returning them to the rack:

```go
t := pt.Tile
if t.IsBlank {
    t.Letter = 0
    t.AssignedLetter = 0
}
placedTiles[i] = t
```

---

### Issue R15-BUG-02 — Stale `rackSelected` persists into exchange mode

**File:** `ui/gamescreen.go` — enter-exchange-mode branch and `commitExchange`  
**Severity:** Bug — after entering exchange mode with a tile already selected, clicking the board on the next frame would stage that tile despite being in exchange mode.

#### Root cause

When the player selects a rack tile (`rackSelected = N`), then clicks Exchange (entering exchange mode), `rackSelected` is not cleared. On any subsequent frame, `handleTileInput` checks `if gs.rackSelected >= 0` before checking exchange mode, so clicking a board cell calls `stageTile(N, row, col)` — staging a tile into exchange mode, making the Exchange button go disabled and leaving the UI in an inconsistent state.

#### Fix applied

Two changes:
1. When entering exchange mode, reset `rackSelected = -1` immediately.
2. In `commitExchange()`, reset `rackSelected = -1` after the exchange succeeds (defence in depth).

#### Verification

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

### Summary

Two bugs found and fixed: R15-BUG-01 (blank tile display after undo) and R15-BUG-02 (stale tile selection leaking into exchange mode). No documentation or code-quality issues identified.

---

## Review 16 — 2026-04-21

Scope: all Go source files. Focused on game-rule edge cases and engine correctness.

### Issue R16-BUG-01 — `ValidatePlacement` accepts a move that forms no word

**File:** `engine/rules.go` — `ValidatePlacement`  
**Severity:** Bug — a placement that forms no word of length ≥ 2 is incorrectly accepted by `ValidatePlacement`, causing `Score` to fail later with an internal-sounding error message.

#### Root cause

After `extractWords` is called, `ValidatePlacement` validated words in the returned slice:

```go
words := extractWords(board, move)
for _, w := range words {   // vacuously true when words == []
    if !dict.Validate(w) { return nil, error }
}
move.WordsFormed = words    // WordsFormed = []
return words, nil           // ← success, but no word was formed
```

When `words` is empty (e.g. a single tile placed on an empty board with no adjacent tiles), the loop body never executes and `ValidatePlacement` returns success with `WordsFormed = []`. Subsequent `Score` call then failed with:

> "engine.Score: WordsFormed is empty — call ValidatePlacement first"

which `sanitiseError` surfaced to the user as "call ValidatePlacement first" — a confusing internal message.

The only game path that reaches this state in normal play is a single-tile first move (one tile placed at the centre, no adjacent tiles, no cross-words).

#### Fix applied

Added an explicit check between word extraction and dictionary validation:

```go
if len(words) == 0 {
    return nil, fmt.Errorf("engine.ValidatePlacement: placement forms no valid word")
}
```

The user now sees "placement forms no valid word" instead of an internal-sounding message.

#### Verification

```
go build ./...         # zero errors
go test -race ./...    # all packages pass, no data-race reports
```

### Summary

One bug found and fixed (R16-BUG-01). No documentation or code-quality issues identified.

---

## Review 17 — 2026-04-21

Scope: all Go source files and test files. Focused on test coverage gaps for bugs fixed in Reviews 15 and 16.

### Issue R17-TEST-01 — No regression tests for R15-BUG-01 and R16-BUG-01

**File:** `engine/engine_test.go`  
**Severity:** Test quality — bugs R15-BUG-01 and R16-BUG-01 had no test coverage, leaving them free to regress silently.

#### R15-BUG-01 gap

`TestPlayCommand_ExecuteUndo` used only letter tiles; no test verified that a blank tile's `Letter` and `AssignedLetter` fields are cleared after undo.

#### R16-BUG-01 gap

No test covered the single-tile first-move path through `ValidatePlacement`. The old behaviour (vacuous success when `words == []`) was untested; the new rejection is also untested.

#### Fix applied

Added two regression tests to `engine/engine_test.go`:

- `TestPlayCommand_Undo_BlankTileReset`: plays "CAT" with a blank as 'A', undoes the move, then asserts `tile.Letter == 0` and `tile.AssignedLetter == 0` for the returned blank.
- `TestValidatePlacement_SingleTileNoWord`: places a single tile on an empty board and asserts that `ValidatePlacement` returns a non-nil error.

#### Verification

```
go test -race ./...    # all packages pass, including the two new tests
```

### Summary

One test-quality gap found and fixed (R17-TEST-01). No new functional bugs, documentation issues, or code-quality issues identified.

---

## Review 18 — 2026-04-21

Scope: all Go source files and test files. Checked: remaining panic sites (all legitimate invariant-enforcement), go vet (zero warnings), TODO/FIXME markers (none), cross-package boundary uses of unexported fields (all within-package and correct), Gob encoding paths, and concurrency invariants in the AI worker.

No issues found.

`go build ./...`, `go vet ./...`, and `go test -race ./... -count=1` all pass clean.
