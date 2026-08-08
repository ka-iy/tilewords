# Logical Components — Unit 3: `ai`

## Component Map

```
GenerateMoves
├── Anchor Finder          (findAnchors)
├── Cross-Check Engine     (computeCrossChecks × 2 directions)
└── GADDAG Traversal Engine
    ├── extendLeft
    ├── extendRight
    └── traverseExistingRight
        └── Candidate Recorder
            ├── recordCandidate
            └── computeOpponentAccess

SelectMove                 (Difficulty Model)

ChooseMove                 (Orchestrator)
└── calls GenerateMoves → SelectMove → fallback

AIWorker                   (Goroutine + Channels)
└── run() calls ChooseMove
```

---

## Component 1: Anchor Finder

**Function**: `findAnchors(board *engine.Board) []anchorSquare`

**Responsibility**: Identify all anchor squares — empty cells that are either adjacent to an
existing board tile or, on an empty board, the centre cell (7,7).

**Input/Output**:
- Input: `*engine.Board`
- Output: `[]anchorSquare` where `anchorSquare` holds `{row, col int}` and a `limit int`
  (maximum left-extension tiles for the `extendLeft` recursion limit)

**Limit computation** (Appel-Jacobson §5): For anchor at column `c`, `limit` is the number
of consecutive empty cells immediately to the left of `c` (bounded by column 0 and any
occupied cell). This prevents `extendLeft` from generating the same word from two different
anchors.

**NFR trace**: NFR-AI-P1 (no wasted traversal); stateless per-call (NFR-AI-T1).

---

## Component 2: Cross-Check Engine

**Function**: `computeCrossChecks(board *engine.Board, dict *dictionary.Dictionary, dir direction) [15][15][26]bool`

**Responsibility**: For every empty cell, compute the set of letters that may be placed there
without forming an invalid perpendicular word.

**Algorithm**:
1. For each empty cell `(r, c)`, collect the perpendicular prefix (`collectPerp` going
   backward) and suffix (`collectPerp` going forward).
2. If both are empty → all-true (no perpendicular constraint).
3. Otherwise, for each letter A–Z, check `dict.Validate(prefix + letter + suffix)` and
   store the result in `cc[r][c][letter-'A']`.

**Called**: Once for horizontal direction, once for vertical direction per `GenerateMoves`
call. Results are passed by pointer to the traversal engine.

**NFR trace**: NFR-AI-P4 (≤10 ms); NFR-AI-M1 (`[15][15][26]bool` = 5,850 bytes on stack).

---

## Component 3: GADDAG Traversal Engine

**Functions**:
- `extendLeft(board, g, cc, counts, anchor, dir, node, limit, leftTiles, candidates)`
- `extendRight(board, g, cc, counts, anchor, dir, node, leftTiles, rightTiles, candidates)`
- `traverseExistingRight(board, g, cc, counts, anchor, dir, node, leftTiles, rightTiles, candidates)`

**Responsibility**: Generate all legal word placements reachable from each anchor using the
GADDAG left-extension algorithm (Appel-Jacobson 1998 §5).

### extendLeft
- **Case 1 — cross the separator**: if the current GADDAG node has an ArcSep (`'+'`)
  successor, call `extendRight` with the tiles placed left of the anchor.
- **Case 2 — extend further left**: if `limit > 0`, try each rack letter, decrement the
  count, recurse with `limit-1`, then restore the count.
- On empty board (first move), limit is always 6 (MaxRackSize-1) and the single anchor is
  (7, 7).

### extendRight
- If the current cell is occupied: follow the existing tile's arc in the GADDAG; if the arc
  exists, call `traverseExistingRight`.
- If the cell is empty:
  - If the GADDAG node is terminal: call `recordCandidate`.
  - For each rack letter, check `cc[r][c][letter-'A']`; if valid, place the tile, recurse,
    restore.

### traverseExistingRight
- Reads the letter on the board, follows its GADDAG arc, continues `extendRight`.

**Shared state** (all stack-local):
- `counts *rackCounts` — mutated and restored on each recursive call (Pattern 4).
- `cc *[15][15][26]bool` — read-only after construction (Pattern 3).
- `candidates *[]MoveCandidate` — append-only.

**NFR trace**: NFR-AI-P1 (GADDAG bounding); NFR-AI-T1 (no global state); NFR-AI-M3
(traversal depth ≤ 22).

---

## Component 4: Candidate Recorder

**Functions**:
- `recordCandidate(board, dict, placed, candidates)`
- `computeOpponentAccess(board, placed) int`

### recordCandidate
1. Construct `engine.PlayMove{Placed: placed}`.
2. Call `engine.ValidatePlacement` — defensive gate (Pattern 5); discard on error.
3. Call `engine.Score` (requires `WordsFormed` populated by `ValidatePlacement`).
4. Call `computeOpponentAccess`.
5. Append `MoveCandidate{Move, Score, OpponentAccess}` to the candidates slice.

**Deduplication**: A `map[moveKey]bool` (where `moveKey` is start+end position+direction)
is checked before appending; duplicates from different anchors are discarded (BR-AI-14).

### computeOpponentAccess
- Counts empty premium squares (DoubleLetter/TripleLetter/DoubleWord/TripleWord/Centre) that
  have at least one orthogonally adjacent tile, considering existing board tiles plus the
  newly placed tiles (Option A — BR-AI-06).

**NFR trace**: NFR-AI-C1 (correctness gate); NFR-AI-P5 (≤50 µs per candidate).

---

## Component 5: Difficulty Model

**Function**: `SelectMove(candidates []MoveCandidate, level int, rng *rand.Rand) MoveCandidate`

**Responsibility**: Choose one candidate from the sorted slice according to the difficulty
level (FR-05 / BR-AI-04 / BR-AI-05).

**Algorithm**:
- `candidates` is sorted: score descending, `OpponentAccess` ascending (BR-AI-03).
- Level 10: count the leading candidates scoring within `topPlayMargin` of the best, then
  return one of those at random (BR-AI-04). Uses the RNG, so level 10 is not deterministic.
- Levels 1–9: compute `k = max(1, round(total × (1 - (level-1)/9)))`; return
  `candidates[rng.Intn(k)]` (BR-AI-05).

**Panics** if `candidates` is empty (NFR-AI-R2 — programming error; callers must guard).

**NFR trace**: NFR-AI-P3 (≤1 ms); NFR-AI-T1 (stateless, injected RNG).

---

## Component 6: ChooseMove Orchestrator

**Function**: `ChooseMove(state *engine.GameState, dict *dictionary.Dictionary, level int, rng *rand.Rand) engine.Move`

**Responsibility**: Top-level move decision for one AI turn.

**Algorithm**:
1. Call `GenerateMoves(state.Board, state.AIRack, dict)`.
2. If candidates is non-empty: call `SelectMove(candidates, level, rng)`; return the
   selected `engine.PlayMove`.
3. If candidates is empty (BR-AI-10):
   - If `state.Bag.Count() >= engine.MaxRackSize`: return `engine.ExchangeMove{Tiles: state.AIRack.Tiles()}`.
   - Otherwise: return `engine.PassMove{}`.

**NFR trace**: NFR-AI-R5 (always returns non-nil Move); BR-AI-08 (uses AIRack only).

---

## Component 7: AIWorker

**Type**: `AIWorker` (exported struct)

**Fields** (all unexported):
```go
type AIWorker struct {
    reqCh chan aiRequest   // buffered, cap 1
    resCh chan engine.Move // buffered, cap 1
    busy  bool            // UI-goroutine only; never read by AI goroutine
}
```

**API**:
```go
func NewAIWorker(choose func(*engine.GameState, *dictionary.Dictionary, int, *rand.Rand) engine.Move) *AIWorker
func (w *AIWorker) Start()
func (w *AIWorker) Request(state *engine.GameState, dict *dictionary.Dictionary, level int)
func (w *AIWorker) Poll() (engine.Move, bool)
```

**Internal type**:
```go
type aiRequest struct {
    state *engine.GameState   // deep clone — BR-AI-12
    dict  *dictionary.Dictionary
    level int
    rng   *rand.Rand          // freshly seeded at Request() time
}
```

**run() loop**:
```go
func (w *AIWorker) run(choose func(...) engine.Move) {
    for req := range w.reqCh {
        move := choose(req.state, req.dict, req.level, req.rng)
        w.resCh <- move
    }
}
```

**NFR trace**: NFR-AI-T1 (channels only; no mutex); NFR-AI-R4 (no deadlock — cap-1 channels);
SECURITY-AI-2 (deep clone passed over channel); NFR-AI-R3 (panic on busy double-Request).

---

## Infrastructure Dependencies

| Dependency | Usage |
|---|---|
| `squabble/engine` | `Board`, `Rack`, `GameState`, `GameState.Clone()`, `ValidatePlacement`, `Score`, move types (`PlayMove`, `ExchangeMove`, `PassMove`) |
| `squabble/dictionary` | `Dictionary.Validate()`, `GADDAG.Successor()`, `GADDAG.IsTerminal()`, `GADDAG.Root()`, `ArcSep` constant |
| `math/rand` | `*rand.Rand` (injected) for `SelectMove` and `AIWorker.Request` seeding |
| `math` | `math.Round` for k-computation |
| `sort` | `sort.Slice` for candidate sorting |
| `fmt` | `fmt.Errorf` for error wrapping (SECURITY-AI-1) |
| `time` | `time.Now().UnixNano()` for AIWorker RNG seeding |

No new infrastructure (database, file I/O, network, CGO) is introduced by Unit 3.
