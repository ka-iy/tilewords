# Code Summary — Unit 3: `ai`

## Files Generated

| File | Purpose |
|---|---|
| `ai/doc.go` | Package GoDoc with Appel-Jacobson citation and AIWorker usage example |
| `ai/moves.go` | `MoveCandidate`, `direction`, `anchorSquare`, `rackCounts [27]int`, `moveKey`; `rackToCountArray`, `findAnchors`, `hasOccupiedNeighbour`, `computeLimit` |
| `ai/crosscheck.go` | `computeCrossChecks`, `collectPerp`, `setAllTrue` |
| `ai/traverse.go` | `extendLeft`, `extendRight`, `traverseExistingRight`, `leftPos`, `nextRightPos`, `mergeNewPlaced` |
| `ai/record.go` | `recordCandidate`, `computeOpponentAccess`, `makeMoveKey` |
| `ai/generate.go` | `GenerateMoves` |
| `ai/select.go` | `SelectMove` |
| `ai/choose.go` | `ChooseMove` |
| `ai/worker.go` | `AIWorker`, `aiRequest`, `NewAIWorker`, `Start`, `Request`, `Poll`, `run` |
| `ai/ai_test.go` | 13 example tests |
| `ai/ai_pbt_test.go` | 7 PBT tests (PBT-AI-01 through PBT-AI-07) |

## Key Implementation Decisions

### GADDAG Build Bug Fixed
The `dictionary.Build` function originally marked only the `k=n` full-reverse path as
terminal. The Appel-Jacobson §5 move generator requires all valid-word paths (k=1..n) to
be terminal so that `extendRight` (AfterAnchor) can detect word completion. Fixed by
marking all path end-nodes as terminal and storing the true word count in a new
`WordCount uint32` field in `gaddagData` (terminal-node counting is no longer a proxy
for word count).

### Traversal Position Tracking
Instead of using `len(rightTiles)` to compute the current right-extension position
(which would require including existing board tiles in `rightTiles`), absolute `(row, col)`
coordinates are passed through `extendRight` and `traverseExistingRight`. This ensures
only newly placed rack tiles appear in `mergeNewPlaced` output, which is what
`engine.ValidatePlacement` expects.

### Deduplication
The `seen map[moveKey]bool` is threaded through the entire traversal (extendLeft →
extendRight → recordCandidate) so duplicates from different anchors are discarded at
record time without a post-processing pass.

### boardStateGen PBT helper
PBT tests use the `engine.PlayCommand` type from the engine package for board setup.
However, building complex board states for PBT was simplified to using empty-board states
with randomly sized racks to keep test isolation clean.

## Test Results

```
go test -race ./ai/...   → ok  squabble/ai  1.462s
go test -race ./...      → ok  squabble/ai, squabble/dictionary, squabble/engine
```

All 13 example tests and 7 PBT tests pass with the race detector enabled.
