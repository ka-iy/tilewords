# NFR Design Patterns — Unit 3: `ai`

## Pattern 1: Stateless Generator Functions (NFR-AI-T1)

**Problem**: `GenerateMoves`, `SelectMove`, and `ChooseMove` must be safe for concurrent
use and must not accumulate state between calls.

**Solution**: All three are package-level functions with no receiver and no side effects.
Every value they need is passed as a parameter; they return new values without mutating
inputs. The only mutation is to a local working copy of rack availability (see Pattern 4).

```go
// GenerateMoves is a pure function: same (board, rack, dict) always produces
// the same candidate set (modulo GADDAG traversal order, which is deterministic).
func GenerateMoves(
    board *engine.Board,
    rack  *engine.Rack,
    dict  *dictionary.Dictionary,
) []MoveCandidate { ... }
```

No package-level variables hold mutable state. The `crossCheckSet` and candidate slice
are local to each call.

---

## Pattern 2: Capacity-1 Channel Protocol for Non-Blocking UI (NFR-AI-T1, NFR-AI-R4)

**Problem**: Ebitengine's `Update()` loop runs on the main goroutine at ≥30 fps. If the
UI blocks waiting for the AI result, the frame rate drops.

**Solution**: `AIWorker` uses two buffered channels of capacity 1. The UI goroutine never
blocks: `Request` sends into the buffered `reqCh` (always has room because only one
request is in flight at a time), and `Poll` uses a non-blocking `select`.

```
UI goroutine                          AI goroutine
    |                                      |
    |-- reqCh <- aiRequest (buffered,1) -->|
    |                                      |-- ChooseMove (blocking, up to 500ms)
    |   (UI keeps drawing frames)          |
    |<-- resCh <- move (buffered,1) -------|
    |                                      |
Poll():                                    |
  select {                                 |
  case move := <-resCh: return move, true  |
  default: return nil, false               |
  }
```

**Why capacity 1**: The AI goroutine sends the result into `resCh` and then immediately
blocks on the next `reqCh` receive. If `resCh` were unbuffered, the AI goroutine would
block until the UI calls `Poll`. With capacity 1, the AI goroutine finishes and returns
to waiting for the next request without depending on the UI's polling cadence.

---

## Pattern 3: Cross-Check Precomputation (NFR-AI-P4)

**Problem**: During GADDAG traversal, `extendRight` must check whether placing a letter
at each cell creates a valid perpendicular word. Calling `dict.Validate` inside the
tight traversal loop would dominate runtime.

**Solution**: Before traversal begins, precompute the valid-letter set for every empty
cell. This is computed once per direction per `GenerateMoves` call and stored in a
`[15][15][26]bool` local array.

```go
// computeCrossChecks builds the valid-letter bitset for each empty cell.
// For a horizontal play, cross-checks constrain which letters may appear
// at each cell without forming an invalid vertical cross-word.
func computeCrossChecks(board *engine.Board, dict *dictionary.Dictionary, dir direction) [15][15][26]bool {
    var cc [15][15][26]bool
    for r := 0; r < 15; r++ {
        for c := 0; c < 15; c++ {
            if !board.IsEmpty(r, c) {
                setAllTrue(&cc[r][c]) // occupied cells: no constraint applies
                continue
            }
            prefix := collectPerp(board, r, c, dir, backward)
            suffix := collectPerp(board, r, c, dir, forward)
            if len(prefix) == 0 && len(suffix) == 0 {
                setAllTrue(&cc[r][c]) // no perpendicular neighbours: any letter valid
                continue
            }
            for l := byte('A'); l <= 'Z'; l++ {
                word := string(prefix) + string(l) + string(suffix)
                cc[r][c][l-'A'] = dict.Validate(word)
            }
        }
    }
    return cc
}
```

**Hot-path usage**:
```go
// Inside extendRight — O(1) array lookup, no dict call:
if !cc[pos.row][pos.col][letter-'A'] {
    continue // cross-check violation
}
```

---

## Pattern 4: Rack Working Copy via Letter-Count Array (Correctness + Performance)

**Problem**: `extendLeft` and `extendRight` must remove and restore rack tiles during
backtracking. Copying `[]Tile` slices on every recursive call is expensive and error-prone.

**Solution**: Represent available rack tiles as a `[28]int` count array (indices 0–25
for A–Z, index 26 for blank) rather than a slice. Remove is a decrement; restore is an
increment. No allocation on each recursive call.

```go
// rackCounts[letter-'A'] = number of that letter available in the rack.
// rackCounts[26] = number of blank tiles available.
type rackCounts [27]int

func rackToCountArray(rack *engine.Rack) rackCounts {
    var counts rackCounts
    for _, t := range rack.Tiles() {
        if t.IsBlank {
            counts[26]++
        } else {
            counts[t.Letter-'A']++
        }
    }
    return counts
}
```

Blank expansion during traversal:
```go
// For each rack letter (including blank→A-Z expansion):
for l := byte('A'); l <= 'Z'; l++ {
    isBlank := false
    if counts[l-'A'] > 0 {
        counts[l-'A']--
    } else if counts[26] > 0 {
        counts[26]--
        isBlank = true
    } else {
        continue
    }
    // ... recurse ...
    if isBlank { counts[26]++ } else { counts[l-'A']++ } // restore
}
```

This pattern eliminates all allocations inside the traversal hot path.

---

## Pattern 5: Defensive ValidatePlacement Gate (BR-AI-01)

**Problem**: The GADDAG traversal is complex; edge cases (e.g., single-tile plays
adjacent to existing tiles forming very short cross-words) could slip through.

**Solution**: `recordCandidate` calls `engine.ValidatePlacement` on every complete
candidate before adding it to the result slice. Invalid candidates are silently discarded.
This is O(word_length × cross_word_count) — fast relative to traversal time — and ensures
the AI never produces an illegal move regardless of traversal bugs.

```go
func recordCandidate(board *engine.Board, dict *dictionary.Dictionary,
    placed []engine.PlacedTile, candidates *[]MoveCandidate) {

    move := engine.PlayMove{Placed: placed}
    // Defensive gate — Appel-Jacobson §5 traversal is sufficient but not formally proven.
    if _, err := engine.ValidatePlacement(board, &move, dict); err != nil {
        return // discard; traversal produced an edge-case invalid move
    }
    score, _ := engine.Score(board, &move)
    access := computeOpponentAccess(board, placed)
    *candidates = append(*candidates, MoveCandidate{
        Move: move, Score: score, OpponentAccess: access,
    })
}
```

---

## Pattern 6: Deterministic Level-10, Seeded Level-1–9 (FR-05)

**Problem**: Level 10 must always return the same move for the same board state (no
randomness). Levels 1–9 must be random but reproducible for debugging.

**Solution**: `SelectMove` accepts an explicit `*rand.Rand`. Level 10 never calls `rng`;
levels 1–9 use `rng.Intn(k)`. The AIWorker seeds a new `*rand.Rand` from
`time.Now().UnixNano()` at each `Request()` call, so production play is varied, but
tests can inject a fixed-seed RNG for reproducibility.

```go
func SelectMove(candidates []MoveCandidate, level int, rng *rand.Rand) MoveCandidate {
    if len(candidates) == 0 {
        panic("ai.SelectMove: empty candidates slice")
    }
    if level == 10 {
        return candidates[0] // deterministic; sorted by score desc, access asc
    }
    // k = max(1, round(total × (1 - (level-1)/9))) — FR-05
    fraction := 1.0 - float64(level-1)/9.0
    k := int(math.Round(float64(len(candidates)) * fraction))
    if k < 1 { k = 1 }
    if k > len(candidates) { k = len(candidates) }
    return candidates[rng.Intn(k)]
}
```

---

## Pattern 7: GoDoc + Algorithm Commentary (NFR-10)

Algorithm-critical functions carry paper section references at the function level and
inline comments at each non-obvious step:

```go
// extendLeft generates all left extensions from anchor by recursively placing
// rack tiles to the left, navigating the GADDAG reversed-prefix arcs.
// This implements the "BeforeAnchor" phase of the GenerateMoves algorithm
// described in Appel-Jacobson (1998) §5.
//
// node is the current GADDAG node reached by the letters placed so far.
// limit is the maximum number of additional tiles that may be placed left of anchor.
func extendLeft(board *engine.Board, g *dictionary.GADDAG, cc *[15][15][26]bool,
    counts *rackCounts, anchor anchorSquare, dir direction,
    node dictionary.NodeID, limit int, leftTiles []engine.PlacedTile,
    candidates *[]MoveCandidate) {

    // Attempt to cross the arc-separator and begin the right extension.
    // In the GADDAG string for word w₁…wₙ at anchor position k,
    // the separator '+' separates the reversed prefix wₖ…w₁ from the suffix wₖ₊₁…wₙ.
    // Appel-Jacobson §5: "if the current arc set contains '+', call AfterAnchor."
    if rightNode, ok := g.Successor(node, dictionary.ArcSep); ok {
        extendRight(board, g, cc, counts, anchor, dir, rightNode, leftTiles, nil, candidates)
    }
    ...
}
```
