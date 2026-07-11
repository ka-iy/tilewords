# Functional Design Plan — Unit 3: `ai`

## Context

Unit 3 (`ai`) implements:
1. The Appel-Jacobson (1998) GADDAG left-extension move generator
2. The 10-level difficulty model (random → optimal interpolation)
3. The async AIWorker goroutine + non-blocking Poll() interface

One question requires an explicit answer. All other decisions are derivable from FR-05,
the Appel-Jacobson paper, and the application design.

---

## Q1 — OpponentAccess Definition

FR-05 states: "Ties [at level 10] are broken by preferring moves that open **fewer
premium squares** for the opponent (i.e., minimise the opponent's scoring opportunities
on their next turn)."

`MoveCandidate.OpponentAccess` is the tie-breaking metric. Two interpretations:

**Option A — Total exposure** (recommended):
After placing the move, count all **empty** premium squares (DL, TL, DW, TW) that are
orthogonally adjacent to **any** tile on the board. Lower = better. This measures the
total premium-square opportunity available to the opponent after our move.

**Option B — Incremental exposure**:
Count only the empty premium squares that **newly** become adjacent to a board tile as a
direct result of this move (squares that were not adjacent before). Lower = better.
This measures only what this move "opens up" in isolation.

**Recommended**: Option A, because it gives a more complete picture of how exposed the
board state is — a move that happens to extend toward many pre-existing premium squares
is still risky even if the move itself doesn't create the exposure.

---

## Pre-answered Questions (no input needed)

**Q2 — AI no-valid-plays fallback**: When GenerateMoves returns no candidates, ChooseMove
prefers ExchangeMove (if bag ≥ 7) over PassMove. This matches standard tournament
strategy and is consistent with FR-07.

**Q3 — Blank tile enumeration**: During GADDAG traversal, blank tiles are tried for all
26 letters (A–Z) at each position. This is a move-generation implementation detail,
not a user decision.

**Q4 — Cross-check precomputation**: Cross-check sets (the set of letters that can
legally appear at each empty cell perpendicular to the play direction) are computed
once per call to GenerateMoves before the main traversal. This is the standard
Appel-Jacobson optimisation (§5).

**Q5 — Move deduplication**: The same word at the same position can be generated
multiple times (e.g., once from each anchor). GenerateMoves deduplicates by (word,
startRow, startCol, direction) before returning.

**Q6 — AIWorker concurrency model**: The AIWorker owns exactly one goroutine that blocks
on a request channel. The UI goroutine sends a GameState clone + level + dict via the
request channel, then polls for the result each frame via Poll(). Only one request may
be in flight at a time; sending a second request before the first is done panics.
