# Business Rules — Unit 2: `engine`

## BR-E01: Board Dimensions and Coordinate System

The board is always 15 rows × 15 columns. Coordinates are `(row, col)` with row 0 at the
top and col 0 at the left (standard matrix order). Any method receiving `(row, col)` outside
`[0,14]` must panic with a clear out-of-bounds message (not silently corrupt state).

---

## BR-E02: First Move Must Cover Centre

The first `PlayMove` of the game must place at least one tile on the centre square `(7,7)`.
This is enforced in `ValidatePlacement`. `MoveNumber == 0` is the trigger condition.

---

## BR-E03: Tile Placement — Single Row or Column, No Gaps

All tiles in a `PlayMove` must be placed in exactly one row (horizontal play) or one column
(vertical play). A single-tile play may be either.

The span from the first to last placed tile (in the varying dimension) must be fully covered:
every intermediate position must contain either an existing board tile or a newly placed tile.
A gap (an empty cell between two placed tiles) is always invalid.

---

## BR-E04: Adjacency Rule (Moves 2 and Beyond)

From move 2 onwards, at least one newly placed tile must be orthogonally adjacent to an
existing board tile. Diagonal adjacency does not count.

---

## BR-E05: All Formed Words Must Be Dictionary-Valid

Every word formed by a play — the main word and all cross-words — must pass
`dict.Validate`. A cross-word of length 1 (a single tile with no orthogonal neighbours)
does not constitute a word and is excluded from validation. Words of length ≥ 2 must
be valid.

---

## BR-E06: Premium Squares Are Single-Use

Premium square multipliers apply only when a **new** tile is placed on that square during
the current turn. Tiles placed on premium squares in previous turns contribute only their
face value to future scoring. The `Score` function checks `isNewTile(r,c, move)` before
applying any multiplier.

---

## BR-E07: Bingo Bonus

Playing all 7 tiles from a rack in a single turn earns a 50-point bonus, added after all
multipliers are applied. `len(PlayMove.Placed) == 7` is the trigger condition.

---

## BR-E08: Tile Exchange — Minimum Bag Size

A player may only exchange tiles when the bag contains at least 7 tiles. `ExchangeCommand.Execute`
returns an error if `state.Bag.Count() < 7`.

---

## BR-E09: ConsecutivePasses Increment Rule (Q2 Decision)

Both `PassMove` and `ExchangeMove` increment `state.ConsecutivePasses` by 1.
`PlayMove` resets `state.ConsecutivePasses` to 0.
This was explicitly selected (Option B) by the project owner.

---

## BR-E10: Six-Pass Game-End Condition

The game ends when `state.ConsecutivePasses >= 6`. This is checked in `IsGameOver` after
every `Command.Execute`. Since both players share the counter, 6 consecutive non-play moves
across both players triggers the end condition.

---

## BR-E11: Rack Exhaustion Game-End Condition

The game ends when the bag is empty (`state.Bag.Count() == 0`) AND at least one rack is
empty (`state.HumanRack.Count() == 0 || state.AIRack.Count() == 0`). This is checked after
rack replenishment inside `PlayCommand.Execute`.

**Precedence over BR-E10.** Both conditions can become true on the same turn: a zero-scoring
play (e.g. an all-blank word on plain squares) that empties the last rack is itself a
scoreless turn, so it can take `ConsecutivePasses` to 6 while also playing the player out.
`IsGameOver` reports `RackExhausted` in that case, not `SixConsecutivePasses`:

- Going out is the stronger claim — the player used their last letter, which is what earns
  the going-out bonus under BR-E12.
- The scoreless-turn rule exists for a game in which nobody can play at all.
- Reporting the six-pass ending instead would deduct both racks and deny the bonus to a
  player who had in fact gone out.

---

## BR-E12: End-Game Score Adjustment

When end condition is `RackExhausted`:
- The player who emptied their rack gains the sum of points of all tiles remaining in the
  other player's rack.
- The other player's score is reduced by the same amount.

When end condition is `SixConsecutivePasses`:
- Each player's score is reduced by the sum of points of their own remaining tiles.
- No redistribution between players.

Score reductions may produce negative total scores; no floor is applied.

---

## BR-E13: Blank Tile Point Value

A blank tile always scores 0 points regardless of its `AssignedLetter` or any premium
square it is placed on. This applies to letter multipliers and cross-word scoring equally.

---

## BR-E14: Command Pattern Is the Only Mutation Mechanism

`GameState` fields must not be mutated directly anywhere in the codebase. All mutations
must go through `Command.Execute`. All reversals must go through `Command.Undo`. This is
the enforcement mechanism for undo correctness.

---

## BR-E15: Undo Scope — One Full Round

`UndoLastRound` reverts exactly one full human+AI round: the AI's most recent command first,
then the human's command before that. It is only callable when:
1. `state.CurrentTurn == HumanTurn` (it is the human's turn to move)
2. Both `state.LastHumanCommand` and `state.LastAICommand` are non-nil

The UI is responsible for checking these preconditions before offering the undo option.

---

## BR-E16: Rack Capacity

A rack may hold at most 7 tiles. `Rack.Add` must return an error if adding the supplied
tiles would exceed this limit. `Rack.Replenish` draws `min(7 - current, bag.Count())` tiles.

---

## BR-E17: Tile Conservation Invariant

At all times, the total number of tiles across the bag, both racks, and all board cells
must equal exactly 100. This invariant is verified by PBT-E07 and is a blocking defect
if violated.

---

## BR-E18: No "Scrabble" in Any Identifier or Comment

Per NFR-09 and BR-09 (from dictionary unit), the word "Scrabble" must not appear in any
Go identifier, comment, string literal, log message, or UI text within the `engine` package.
Use "crossword game", "board game", or "Squabble" as replacements.

---

## BR-E19: First Turn Is Decided By An Opening Draw

`engine.New` decides the first player with the standard opening draw: each player draws
one tile, and the player whose letter is nearest the start of the alphabet plays first.
A blank counts as earlier than 'A' and so wins the draw; a tie (same letter, or two
blanks) is re-drawn. The drawn tiles are returned to the bag and reshuffled before the
racks are dealt. The result is stored in `GameState.CurrentTurn`, and the letter each
player drew is recorded in `GameState.OpeningDraw`. The UI logs the drawn letters and
displays which player was selected to go first.

---

## BR-E20: gob Serialisation Compatibility

All fields of `GameState`, `Board`, `Bag`, `Rack`, `Tile`, and command types that must
survive a save/load cycle must be exported (uppercase first letter). Unexported fields are
silently skipped by `encoding/gob`. Any field needed after load must be exported or
re-derived from exported fields during load.
