# Business Rules — Unit 3: `ai`

## BR-AI-01: GenerateMoves Must Never Return an Invalid Move

Every `MoveCandidate` returned by `GenerateMoves` must pass `engine.ValidatePlacement`
when the same board and dictionary are used. The GADDAG traversal is the primary filter,
but `recordCandidate` calls `ValidatePlacement` as a defensive safety net. Any candidate
that fails validation is silently discarded; a failing candidate at this stage indicates
a bug in the traversal logic.

---

## BR-AI-02: Candidate Moves Must Be Fully Scored Before Return

`MoveCandidate.Score` must be set to the result of `engine.Score` before the candidate
is added to the result slice. `engine.Score` requires `move.WordsFormed` to be populated,
which is done by `engine.ValidatePlacement` inside `recordCandidate`. The AI never
returns a candidate with `Score == 0` due to missing scoring — 0 is a valid score only
for a word consisting entirely of blank tiles.

---

## BR-AI-03: GenerateMoves Result Is Sorted by Score Descending

Candidates are always returned sorted: primary key is `Score` descending; secondary key
is `OpponentAccess` ascending (stable tiebreaker for level 10). `SelectMove` relies on
this ordering and must not re-sort internally.

---

## BR-AI-04: Level-10 Move Selection Is Deterministic

At level 10, `SelectMove` returns `candidates[0]` — the candidate with the highest score
and, among equal-score candidates, the lowest `OpponentAccess`. No randomness is used.
The same board+rack always produces the same level-10 move.

---

## BR-AI-05: k Computation for Levels 1–9

```
k = max(1, round(float64(total) × (1 - float64(level-1)/9)))
```

`round` uses `math.Round` (round half-to-even). `k` is clamped to `[1, total]`.
`SelectMove` samples uniformly from `candidates[:k]` using `rng.Intn(k)`.

---

## BR-AI-06: OpponentAccess Definition (Q1: Option A — Total Exposure)

After placing a move, `OpponentAccess` is the count of empty premium squares
(DoubleLetter, TripleLetter, DoubleWord, TripleWord, Centre) that have at least one
orthogonally adjacent tile (existing board tiles + newly placed tiles). This is the
"total exposure" interpretation: the full count of premium squares the opponent can
reach on their next turn.

---

## BR-AI-07: Blank Tile Enumeration

During GADDAG traversal, each blank tile in the rack is expanded to all 26 letters A–Z.
A blank played as letter `l` is represented in `PlacedTile.Tile` as
`Tile{IsBlank:true, AssignedLetter:l, Points:0}`. The blank always scores 0 regardless
of `l` and regardless of any premium square it is placed on (carried over from BR-E13).

---

## BR-AI-08: AI Uses Its Own Rack

`GenerateMoves` and `ChooseMove` always use `state.AIRack`. The human rack is never
consulted during AI move generation. `state.AIRack` is always accessible (its tiles
are always visible per US-04/FR-08), but only the AI reads it for move generation.

---

## BR-AI-09: No Request Overlap in AIWorker

`AIWorker.Request` panics if called while `w.busy == true` (a prior request has not yet
been consumed by `Poll`). The UI is responsible for never calling `Request` a second time
before receiving the result of the first request via `Poll`.

---

## BR-AI-10: Exchange Strategy When No Valid Plays

When `GenerateMoves` returns an empty slice, the AI exchanges all tiles if
`state.Bag.Count() >= engine.MaxRackSize`, otherwise passes. The AI always exchanges
all tiles (not a subset) to maximise rack refresh. This is a fixed strategy, not
level-dependent.

---

## BR-AI-11: Cross-Check Must Pass for Every Placed Tile

During right extension, a rack tile may only be placed at an empty cell `(r, c)` if
`crossCheckSet[r][c][letter-'A'] == true`. If the cell has no perpendicular neighbours,
the cross-check set is all-true (any letter is valid). This rule enforces BR-E05 without
calling `ValidatePlacement` on every partial word.

---

## BR-AI-12: The AI Goroutine Has No Access to Global State

`AIWorker.run()` receives a deep clone of `GameState` via channel. It must not read
from any shared pointer, global variable, or the original `GameState`. The clone is
its sole input; the result channel is its sole output.

---

## BR-AI-13: GenerateMoves Is Stateless and Reentrant

`GenerateMoves` is a pure function of its inputs (board, rack, dict). It creates no
goroutines, no global state, and no persistent data. Calling it concurrently from
multiple goroutines with independent inputs is safe (it is called from the AI goroutine
only, but the design permits concurrent use).

---

## BR-AI-14: Move Deduplication By Position

Two candidates with identical `(startRow, startCol, endRow, endCol, direction)` and
identical word (same letters at same positions) are considered duplicates even if
generated from different anchors. Only the first occurrence is kept (after sorting,
this preserves the one with the correct scoring, which will be identical for both since
they represent the same physical play).

---

## BR-AI-15: No "Scrabble" in Any Identifier or Comment

Per NFR-09 and BR-E18, the word "Scrabble" must not appear in any Go identifier,
comment, string literal, log message, or UI text in the `ai` package. Use "crossword",
"board game", or "Squabble" as replacements.
