# NFR Requirements — Unit 2: `engine`

## Performance (NFR-01)

| ID | Requirement | Threshold | Rationale |
|---|---|---|---|
| NFR-ENG-P1 | `Score(board, move)` execution time | ≤ 10 µs per call | Called thousands of times per AI move-generation cycle; must not be the bottleneck |
| NFR-ENG-P2 | `ValidatePlacement(board, move, dict)` execution time | ≤ 50 µs per call | Called once per committed move by the UI; imperceptible to the player |
| NFR-ENG-P3 | `engine.New()` (board + bag + racks) startup time | ≤ 10 ms | Called once at game start; negligible relative to GADDAG load |
| NFR-ENG-P4 | `Board.Clone()` execution time | ≤ 5 µs | Called once per AI request to produce an independent state copy |

---

## Memory (NFR-03)

| ID | Requirement | Threshold | Rationale |
|---|---|---|---|
| NFR-ENG-M1 | `GameState` heap footprint | ≤ 512 KB | Single game state; board (225 cells) + racks + bag are small |
| NFR-ENG-M2 | `ExchangeCommand.bagSnapshot` | ≤ 100 × sizeof(Tile) ≈ 3 KB | At most 100 tiles; bounded and small |
| NFR-ENG-M3 | No unbounded allocations in hot paths | — | `Score` and `ValidatePlacement` must not allocate per-call heap objects in the steady state; use pre-allocated buffers where possible |

---

## Thread Safety (NFR-ENG-T1)

`engine` types are **not** concurrency-safe by design. The `GameState` is owned by the UI
goroutine. The AI goroutine receives a **deep copy** of `GameState` (via `GameState.Clone()`)
before its goroutine is started; it never shares state with the UI goroutine. This eliminates
all data races without locks.

Requirements:
- `Board.Clone()` must produce a fully independent deep copy (no shared pointer aliasing).
- `GameState.Clone()` must deep-copy `Board`, `HumanRack`, `AIRack`, and `Bag`; it does
  **not** need to copy `LastHumanCommand` or `LastAICommand` (AI operates on a read-only
  snapshot; undo state is irrelevant to the AI computation).
- No method on any `engine` type may mutate shared state after `Clone`.

---

## Reliability (NFR-04)

| ID | Requirement |
|---|---|
| NFR-ENG-R1 | `ValidatePlacement` and `Score` must return descriptive errors for all invalid inputs; must never panic on any well-typed (but semantically invalid) input |
| NFR-ENG-R2 | `Command.Undo` must never fail. If state is inconsistent, Undo panics with a clear diagnostic message (a panic here indicates a programming error, not a user error) |
| NFR-ENG-R3 | `Board.Place` returns error if cell is already occupied; `Board.Remove` on an empty cell is a no-op (not a panic) |
| NFR-ENG-R4 | `Rack.Remove` returns a descriptive error if any requested tile is not present in the rack |
| NFR-ENG-R5 | `Bag.Draw(n)` returns `min(n, bag.Count())` tiles; never returns more tiles than available |

---

## Serialisation (NFR-ENG-S1)

The `ui.SaveManager` uses `encoding/gob` to serialise `GameState`. Requirements:

- All fields of `GameState`, `Board`, `Bag`, `Rack`, and `Tile` that must survive a
  save/load cycle **must be exported** (gob silently skips unexported fields).
- `LastHumanCommand` and `LastAICommand` are `Command` interface values. Interface values
  require `gob.Register` for each concrete type. To avoid this complexity and keep the
  engine package free of gob coupling, these fields are **excluded from serialisation**:
  - The `ui.SaveManager` saves a `GameStateSave` projection struct (defined in `ui`) that
    omits both command fields.
  - After loading, `LastHumanCommand` and `LastAICommand` are nil; undo is unavailable
    until the first new move is completed. This is consistent with FR-09: "Undo is only
    available immediately after the human's move."
- The `Board.cells` array is unexported. The `Board` type must expose a `SavedBoard` helper
  struct or implement `GobEncode`/`GobDecode` to serialise/deserialise the cell grid.

---

## Security (NFR-08 / SECURITY-10, SECURITY-15)

| ID | Requirement |
|---|---|
| SECURITY-ENG-1 | All error returns are wrapped with the originating function name as prefix: `fmt.Errorf("engine.ValidatePlacement: %w", err)` |
| SECURITY-ENG-2 | No engine error message exposes internal file paths or Go stack traces |
| SECURITY-ENG-3 | `go.sum` integrity is enforced at build time (no `replace` directives for `engine`-specific dependencies; there are none — engine is stdlib-only) |
| SECURITY-ENG-4 | Race detector (`go test -race`) must pass on all engine tests with no reported races |

---

## Testability (NFR-07)

| ID | Requirement |
|---|---|
| NFR-ENG-TEST-1 | All business logic in `engine` is unit-testable without a running UI or real GADDAG (use a mock `dictionary.Dictionary` or a test-built GADDAG from Unit 1's `Build()`) |
| NFR-ENG-TEST-2 | PBT framework `pgregory.net/rapid` must be used for all property-based tests (PBT-E01 through PBT-E08) |
| NFR-ENG-TEST-3 | `go test -race ./engine/...` must pass with no races |
| NFR-ENG-TEST-4 | Test helpers must expose `NewTestBoard()` (board with controlled premium layout for scoring tests) and `NewTestBag(tiles []Tile)` (bag with a fixed tile sequence, no shuffle) for deterministic unit tests |

---

## Code Commentary (NFR-10)

| ID | Requirement |
|---|---|
| NFR-ENG-C1 | Every exported type, constant, variable, and function in `engine` has a GoDoc-compliant comment |
| NFR-ENG-C2 | Algorithm-critical sections (cross-word extraction, scoring multiplier logic, Fisher-Yates shuffle, end-game scoring) include inline comments explaining the algorithm step |
| NFR-ENG-C3 | The premium square layout table in `board.go` references the standard board layout source in a comment |
| NFR-ENG-C4 | `Command.Execute` and `Command.Undo` comment the inverse-command pattern by name and explain the undo data stored |
| NFR-ENG-C5 | The tile distribution table in `bag.go` references the standard distribution with a comment noting the 100-tile count and source |
