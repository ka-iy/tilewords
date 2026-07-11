# Tech Stack Decisions — Unit 2: `engine`

## Language
| Decision | Choice | Rationale |
|---|---|---|
| Implementation language | Go 1.22+ | Project-wide requirement; cross-platform compilation |

## Randomness
| Decision | Choice | Rationale |
|---|---|---|
| RNG for bag shuffle and opening draw | `math/rand.Rand` (stdlib) | Caller-supplied `*rand.Rand` makes all randomness injectable and deterministic in tests; no global state |
| Shuffle algorithm | Fisher-Yates (in-place) | O(n) time, O(1) extra space; standard and correct for uniform shuffle |

## Data Structures
| Decision | Choice | Rationale |
|---|---|---|
| Board grid | `[15][15]Cell` (fixed array) | Zero allocation after construction; direct index access O(1); no bounds ambiguity |
| Rack tiles | `[]Tile` (slice, cap 7) | Simple; small bounded size; append/remove via slice operations |
| Bag tiles | `[]Tile` (slice, cap 100) | Draw from end (O(1) pop); shuffle in-place |
| Premium square lookup | Hard-coded `[15][15]SquareType` initialised in `NewBoard` | Compile-time constant layout; no map lookup overhead on hot scoring path |

## Serialisation
| Decision | Choice | Rationale |
|---|---|---|
| Save/load format | `encoding/gob` (in `ui` package) | Stdlib; binary; compact; no engine-level gob coupling |
| Board serialisation | `Board.GobEncode` / `Board.GobDecode` | Needed because `cells` is unexported; custom encoder exposes the cell grid |
| Command serialisation | Excluded from save | `Command` interface values require `gob.Register` per concrete type; FR-09 permits undo loss after load |

## Dependencies
| Package | Version | Purpose |
|---|---|---|
| `math/rand` | stdlib | Bag shuffle, opening draw for first turn |
| `fmt`, `errors` | stdlib | Error wrapping (SECURITY-ENG-1) |
| All other dependencies | stdlib only | No external runtime dependencies for this unit |

## Testing
| Decision | Choice | Rationale |
|---|---|---|
| Unit test framework | `testing` (stdlib) | Standard Go |
| Property-based testing | `pgregory.net/rapid` | Already selected project-wide (NFR-07) |
| Race detection | `go test -race ./engine/...` | Required by SECURITY-ENG-4 |
| Dictionary mock | Test-built GADDAG via `dictionary.Build()` | Avoids licensed word lists in tests; reuses Unit 1's `Build` function |
| Deterministic bag | `NewTestBag(tiles []Tile)` test helper | Provides fixed tile sequence without shuffle for scoring and validation tests |
