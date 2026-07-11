# Tech Stack Decisions — Unit 3: `ai`

## Language
| Decision | Choice | Rationale |
|---|---|---|
| Implementation language | Go 1.22+ | Project-wide requirement |

## Algorithm
| Decision | Choice | Rationale |
|---|---|---|
| Move generation algorithm | GADDAG left-extension (Appel-Jacobson 1998 §5) | Required by FR-02; enables O(rack-size) traversal with cross-check precomputation |
| Cross-check storage | `[15][15][26]bool` local variable | Cache-friendly; zero allocation per direction; 5,850 bytes fits in L1 cache |
| Candidate collection | `[]MoveCandidate` (growing slice) | Simple; bounded size; sorted once after collection |
| Deduplication | `map[moveKey]bool` (move key = start+end position + direction) | O(1) lookup; map is small (≤10K entries) |
| Candidate sorting | `sort.Slice` (unstable) | O(n log n); sufficient for ≤10K candidates well within 500ms budget |

## Randomness
| Decision | Choice | Rationale |
|---|---|---|
| RNG for SelectMove and AIWorker seed | `math/rand.Rand` (stdlib) | Injected via parameter; deterministic in tests; no global state |
| AIWorker RNG seeding | `rand.New(rand.NewSource(time.Now().UnixNano()))` at `Request()` call | Fresh seed per AI turn; prevents repetitive play patterns |

## Concurrency
| Decision | Choice | Rationale |
|---|---|---|
| AI computation model | Single dedicated goroutine + buffered channels (cap 1) | Keeps UI goroutine non-blocking (Ebitengine requirement); simple ownership model |
| State transfer to AI | Deep clone of `GameState` passed by value over channel | Eliminates data races without locks; consistent with engine thread-safety model |
| Result retrieval | Non-blocking `select` in `Poll()` | UI calls `Poll()` each `Update()` frame; no busy-waiting |

## Dependencies
| Package | Version | Purpose |
|---|---|---|
| `squabble/dictionary` | internal | GADDAG traversal API (`Successor`, `IsTerminal`, `Root`, `ArcSep`) |
| `squabble/engine` | internal | `Board`, `Rack`, `GameState`, `ValidatePlacement`, `Score`, move types |
| `math`, `math/rand`, `sort`, `fmt`, `time` | stdlib | Floating-point k-computation, RNG, sorting, error wrapping, AIWorker seeding |
| `pgregory.net/rapid` | v1.2.0 (test only) | Property-based testing |

## Testing
| Decision | Choice | Rationale |
|---|---|---|
| Unit test framework | `testing` (stdlib) | Standard Go |
| Property-based testing | `pgregory.net/rapid` | Project-wide selection |
| Dictionary in tests | `dictionary.NewFromWords` with curated word list | No licensed assets needed; reuses Unit 1 infrastructure |
| Race detection | `go test -race ./ai/...` | Required by SECURITY-AI-3 |
| AIWorker testing | Functional test: `Request` + `Poll` loop until result | Tests the channel protocol end-to-end |
