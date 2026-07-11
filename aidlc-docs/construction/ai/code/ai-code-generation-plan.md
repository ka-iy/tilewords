# Code Generation Plan — Unit 3: `ai`

## Files to Generate

| Step | File | Description |
|---|---|---|
| 1 | `ai/doc.go` | Package GoDoc with algorithm citation and usage example |
| 2 | `ai/moves.go` | `MoveCandidate`, `direction`, `anchorSquare`, `rackCounts`, `moveKey` types; `rackToCountArray`, `findAnchors` |
| 3 | `ai/crosscheck.go` | `computeCrossChecks`, `collectPerp`, `setAllTrue` |
| 4 | `ai/traverse.go` | `extendLeft`, `extendRight`, `traverseExistingRight` (GADDAG traversal core) |
| 5 | `ai/record.go` | `recordCandidate`, `computeOpponentAccess`, deduplication map logic |
| 6 | `ai/generate.go` | `GenerateMoves` (top-level: anchors → cross-checks × 2 → traversal → sort → dedup) |
| 7 | `ai/select.go` | `SelectMove` (k-formula, level-10 determinism) |
| 8 | `ai/choose.go` | `ChooseMove` (orchestrator: generate → select → exchange/pass fallback) |
| 9 | `ai/worker.go` | `AIWorker`, `aiRequest`, `NewAIWorker`, `Start`, `Request`, `Poll`, `run` |
| 10 | `ai/ai_test.go` | Example tests (unit + integration) |
| 11 | `ai/ai_pbt_test.go` | PBT tests (PBT-AI-01 through PBT-AI-07) |
| 12 | `aidlc-docs/construction/ai/code/code-summary.md` | Code summary |

---

## Step Details

### Step 1 — `ai/doc.go`
Package-level GoDoc: describes the GADDAG left-extension algorithm (Appel-Jacobson 1998 §5),
the AIWorker concurrency model, and a usage example showing `NewAIWorker` → `Start` →
`Request` → `Poll` loop.

### Step 2 — `ai/moves.go`
Types:
- `MoveCandidate{Move engine.PlayMove; Score int; OpponentAccess int}`
- `direction` (`dirHorizontal`, `dirVertical`)
- `anchorSquare{row, col, limit int}`
- `rackCounts [27]int` (indices 0–25 = A–Z, index 26 = blank)
- `moveKey{startRow, startCol, endRow, endCol int; dir direction}`

Functions:
- `rackToCountArray(rack *engine.Rack) rackCounts`
- `findAnchors(board *engine.Board) []anchorSquare` — adjacent-to-tile cells + centre on
  empty board; limit = consecutive empty cells to the left (horizontal) or above (vertical)

### Step 3 — `ai/crosscheck.go`
- `computeCrossChecks(board *engine.Board, dict *dictionary.Dictionary, dir direction) [15][15][26]bool`
- `collectPerp(board *engine.Board, r, c int, dir direction, backward bool) []byte`
- `setAllTrue(cc *[26]bool)` — fills all 26 entries with true

Cross-check is computed per direction: for horizontal play, perpendicular = vertical words;
for vertical play, perpendicular = horizontal words.

### Step 4 — `ai/traverse.go`
All three functions are mutually recursive:
```
extendLeft → extendRight → traverseExistingRight → extendRight
```
Signatures:
```go
func extendLeft(board *engine.Board, g *dictionary.GADDAG, cc *[15][15][26]bool,
    counts *rackCounts, anchor anchorSquare, dir direction,
    node dictionary.NodeID, limit int, leftTiles []engine.PlacedTile,
    candidates *[]MoveCandidate, seen map[moveKey]bool)

func extendRight(board *engine.Board, g *dictionary.GADDAG, cc *[15][15][26]bool,
    counts *rackCounts, anchor anchorSquare, dir direction,
    node dictionary.NodeID, leftTiles, rightTiles []engine.PlacedTile,
    candidates *[]MoveCandidate, seen map[moveKey]bool)

func traverseExistingRight(board *engine.Board, g *dictionary.GADDAG, cc *[15][15][26]bool,
    counts *rackCounts, anchor anchorSquare, dir direction,
    node dictionary.NodeID, leftTiles, rightTiles []engine.PlacedTile,
    candidates *[]MoveCandidate, seen map[moveKey]bool)
```
`seen` map passed through for O(1) deduplication without post-processing.

### Step 5 — `ai/record.go`
```go
func recordCandidate(board *engine.Board, dict *dictionary.Dictionary,
    placed []engine.PlacedTile, candidates *[]MoveCandidate,
    seen map[moveKey]bool)

func computeOpponentAccess(board *engine.Board, placed []engine.PlacedTile) int
```
`recordCandidate`: build `moveKey`, check `seen`, call `ValidatePlacement`, call `Score`,
call `computeOpponentAccess`, append `MoveCandidate`, mark `seen`.

`computeOpponentAccess`: iterate all 15×15 cells; for each empty premium square, check 4
orthogonal neighbours in the union of board tiles + placed tiles; count if any neighbour
is occupied.

### Step 6 — `ai/generate.go`
```go
func GenerateMoves(board *engine.Board, rack *engine.Rack,
    dict *dictionary.Dictionary) []MoveCandidate
```
Algorithm:
1. `counts := rackToCountArray(rack)`
2. `anchors := findAnchors(board)`
3. `ccH := computeCrossChecks(board, dict, dirHorizontal)`
4. `ccV := computeCrossChecks(board, dict, dirVertical)`
5. `seen := make(map[moveKey]bool)`
6. For each anchor, for each direction, call `extendLeft` with limit from anchor.
7. `sort.Slice(candidates, ...)` — score desc, OpponentAccess asc.
8. Return candidates.

### Step 7 — `ai/select.go`
```go
func SelectMove(candidates []MoveCandidate, level int, rng *rand.Rand) MoveCandidate
```
- Panic if empty.
- Level 10: return `candidates[0]`.
- Levels 1–9: k-formula with `math.Round`; clamp to [1, len]; return `candidates[rng.Intn(k)]`.

### Step 8 — `ai/choose.go`
```go
func ChooseMove(state *engine.GameState, dict *dictionary.Dictionary,
    level int, rng *rand.Rand) engine.Move
```
- Generate → select if non-empty → return PlayMove.
- Empty + bag ≥ MaxRackSize → ExchangeMove with all AI rack tiles.
- Empty + bag < MaxRackSize → PassMove.

### Step 9 — `ai/worker.go`
```go
type AIWorker struct { reqCh chan aiRequest; resCh chan engine.Move; busy bool }
type aiRequest struct { state *engine.GameState; dict *dictionary.Dictionary; level int; rng *rand.Rand }
func NewAIWorker(choose func(*engine.GameState, *dictionary.Dictionary, int, *rand.Rand) engine.Move) *AIWorker
func (w *AIWorker) Start()
func (w *AIWorker) Request(state *engine.GameState, dict *dictionary.Dictionary, level int)
func (w *AIWorker) Poll() (engine.Move, bool)
func (w *AIWorker) run(choose func(*engine.GameState, *dictionary.Dictionary, int, *rand.Rand) engine.Move)
```
`Request`: panics if `w.busy`; calls `state.Clone()`; seeds fresh `*rand.Rand`; sends to `reqCh`.
`Poll`: non-blocking select on `resCh`; clears `busy` on success.

### Step 10 — `ai/ai_test.go`
Example tests:
- `TestFindAnchors_EmptyBoard` — centre cell (7,7) only, limit=7
- `TestFindAnchors_OccupiedBoard` — anchors adjacent to placed tiles, correct limits
- `TestComputeCrossChecks_NoNeighbours` — all-true for isolated cells
- `TestComputeCrossChecks_WithNeighbour` — only valid cross-words pass
- `TestGenerateMoves_EmptyRack` — returns empty slice, no panic
- `TestGenerateMoves_FirstMove` — single word placed at centre returns correct candidates
- `TestGenerateMoves_AllValid` — every returned candidate passes `engine.ValidatePlacement`
- `TestGenerateMoves_Sorted` — candidates in score-desc then OpponentAccess-asc order
- `TestSelectMove_Level10` — always returns candidates[0]
- `TestSelectMove_Level1` — returns from full slice
- `TestSelectMove_EmptyPanics` — panics on empty slice
- `TestChooseMove_HasCandidates` — returns PlayMove
- `TestChooseMove_NoCandidates_LargeBag` — returns ExchangeMove
- `TestChooseMove_NoCandidates_SmallBag` — returns PassMove
- `TestAIWorker_RequestPoll` — sends request, polls until result
- `TestAIWorker_DoublePanics` — second Request while busy panics

### Step 11 — `ai/ai_pbt_test.go`
PBT tests (pgregory.net/rapid):
- `TestPBT_AI_AllCandidatesValid` (PBT-AI-01) — for random (board, rack, dict), every candidate passes ValidatePlacement
- `TestPBT_AI_CandidatesSorted` (PBT-AI-02) — result slice is score-desc, OpponentAccess-asc
- `TestPBT_AI_NoDuplicates` (PBT-AI-03) — no two candidates have same moveKey
- `TestPBT_AI_Level10Deterministic` (PBT-AI-04) — two calls with same (board, rack, dict) return identical MoveCandidate
- `TestPBT_AI_SelectMove_RangeCorrect` (PBT-AI-05) — for level L, selected candidate is always within candidates[:k]
- `TestPBT_AI_ChooseMove_NonNil` (PBT-AI-06) — ChooseMove never returns nil
- `TestPBT_AI_Worker_NoRace` (PBT-AI-07) — concurrent Request/Poll cycle produces valid moves (run with -race)

---

## Dependency Order

Steps 1–5 have no internal dependencies on each other within the `ai` package (all depend
only on `engine` and `dictionary`). Steps 6–9 depend on 2–5. Steps 10–11 depend on all
source steps.

Planned generation order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12.

## Verification

After all files are generated:
```
go build ./ai/...
go test -race ./ai/...
```
Both must pass with no errors.
