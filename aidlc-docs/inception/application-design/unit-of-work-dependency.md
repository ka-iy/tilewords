# Unit of Work Dependencies — Squabble

## Dependency Matrix

| Unit | Depends On | Depended On By |
|---|---|---|
| U1: `dictionary` | *(none — leaf)* | U2, U3, U4, U5 |
| U2: `engine` | U1 (`dictionary` types passed as params) | U3, U4, U5 |
| U3: `ai` | U1 (`GADDAG` traversal), U2 (`Board`, `Rack`, `Move`, `Scorer`) | U4, U5 |
| U4: `ui` | U1 (dict loading), U2 (state, commands), U3 (AIPlayer, AIWorker) | U5 |
| U5: `cmd` | U1, U2, U3, U4 (wires everything) | *(none — root)* |

## Dependency Graph

```
U1: dictionary
      |
      +-------> U2: engine
      |               |
      |               +-------> U3: ai
      |               |               |
      +---------------+               +-------> U4: ui
      |                               |               |
      +-------------------------------+               +-------> U5: cmd
      |                                               |
      +-----------------------------------------------+
```

## Build Order (Strict Sequential)

1. **U1: `dictionary`** — no upstream dependencies; must complete first
2. **U2: `engine`** — depends on `dictionary` (for validation in rules); begins after U1 tests pass
3. **U3: `ai`** — depends on `dictionary` (GADDAG traversal) and `engine` (board, rack, move types); begins after U2 tests pass
4. **U4: `ui`** — depends on all of U1–U3; begins after U3 tests pass
5. **U5: `cmd`** — wires all packages; begins after U4 tests pass

## Integration Points

| Integration | Between Units | Mechanism | Risk |
|---|---|---|---|
| Word validation | U1 → U2 | `dictionary.Dictionary` passed to `engine.Rules.ValidatePlacement` | Low — pure function call |
| GADDAG traversal | U1 → U3 | `dictionary.GADDAG` methods used in move generation loop | Medium — performance-critical hot path |
| Board/rack reads | U2 → U3 | `engine.Board`, `engine.Rack` passed to `ai.Generator` | Low — read-only access |
| Move scoring | U2 → U3 | `engine.Scorer` called per candidate in generator | Medium — must score cross-words accurately |
| AI async result | U3 → U4 | `ai.AIWorker` channel polled in `ui.GameScreen.Update()` | Medium — concurrency; data race risk |
| Command execution | U2 → U4 | `engine.Command.Execute/Undo` called from `ui.GameScreen` | Low — clean command pattern boundary |
| Game state encode | U2 → U4 | `engine.GameState` gob-encoded by `ui.SaveManager` | Low — gob handles exported fields automatically |
| Entry-point wiring | U1–U4 → U5 | Direct construction in `cmd/squabble/main.go` | Low — compile-time verification |

## Testing Strategy Per Unit

| Unit | Test Type | Notes |
|---|---|---|
| U1 | Unit + PBT | Round-trip: serialise/deserialise GADDAG; invariant: word count preserved after dedup |
| U2 | Unit + PBT | Oracle: scorer vs manual calculation; stateful PBT on game state command sequences |
| U3 | Unit + PBT + Oracle | Generator output vs brute-force on small boards; race detection on AIWorker |
| U4 | Unit + integration | End-to-end game session on headless Ebitengine; save/load round-trip |
| U5 | Build + integration | Cross-platform `go build` matrix; `go vet`; `-race` flag on full `go test ./...` |
