# Functional Design Plan — Unit 2: `engine`

## Context

Unit 2 (`engine`) implements the board, tile bag, player racks, scoring, placement validation,
game-end detection, end-game scoring, and the Command/Undo system. It has no external
dependencies at runtime and is consumed by both `ai` and `ui`.

Two questions require explicit answers before the design documents are generated.
All other engine design decisions are derivable from existing requirements and the
application design.

---

## Q1 — First Turn

**FR-08 states**: "Human plays first by default (or randomised — TBD in design)."

Who takes the first turn?

**Option A**: Human always plays first.
**Option B**: Randomly determined at game start (coin flip via `rng`).

**Design implication**: If Option B, `GameState.New()` must set `CurrentTurn` based on a
random draw and the UI must reflect who was selected to go first.

---

## Q2 — Tile Exchange and the 6-Pass End Condition

**FR-07 states**: "Six consecutive passes have been made across both players."

Does a tile exchange (`ExchangeMove`) count as a pass for the purpose of this rule?

**Option A**: Only an explicit `PassMove` increments `ConsecutivePasses`. An `ExchangeMove`
resets `ConsecutivePasses` to 0 (it is an active move, not a pass). This matches standard
NASPA/NWL tournament rules.

**Option B**: Both `PassMove` and `ExchangeMove` increment `ConsecutivePasses`.

**Design implication**: Affects `ExchangeCommand.Execute`. Option A is the standard
tournament interpretation of FR-07.

---

## Pre-answered Questions (no input needed)

The following were resolved from existing requirements and are recorded here for traceability:

**Q3 — Undo scope**: FR-09 is explicit: "The AI's previous move is also undone." Undo at the
start of the human's turn reverts the AI's immediately-preceding move first, then the human's
move before that. `GameState` will store `LastHumanCommand` and `LastAICommand` separately.

**Q4 — Blank tile representation**: Decided in application design: blank in rack = `Tile{Letter: 0, IsBlank: true}`; when played, `Tile{IsBlank: true, AssignedLetter: 'X', Points: 0}`.

**Q5 — Coordinate system**: `(row, col)`, row 0 = top, col 0 = left. Already decided.

**Q6 — Tile exchange bag requirement**: FR-07 explicitly requires ≥7 tiles in the bag to exchange.
