# Business Logic Model — Unit 3: `ai`

## BL-AI-01: Anchor Square Identification

```
findAnchors(board *engine.Board) []anchorSquare:
  if !board.HasAnyTile():
    return [{row:7, col:7}]   // first move: only the centre is an anchor

  anchors := []anchorSquare{}
  dirs := [(-1,0),(1,0),(0,-1),(0,1)]
  for r in 0..14:
    for c in 0..14:
      if board.IsEmpty(r, c):
        for each (dr, dc) in dirs:
          if !board.IsEmpty(r+dr, c+dc):
            anchors = append(anchors, {r, c})
            break
  return anchors
```

Every valid play move places at least one tile on an anchor square.

---

## BL-AI-02: Cross-Check Precomputation (Appel-Jacobson §5)

Cross-checks are computed once per call to `GenerateMoves`, separately for each
play direction. For a horizontal play, cross-checks constrain which letters can appear
at each cell without forming an invalid vertical word; vice-versa for vertical.

```
computeCrossChecks(board *engine.Board, dict *dictionary.Dictionary, dir direction) [15][15][26]bool:
  var cc [15][15][26]bool
  for r in 0..14:
    for c in 0..14:
      if !board.IsEmpty(r, c):
        // Occupied cells: all letters are "valid" (cell won't be placed in)
        cc[r][c] = allTrue
        continue

      // Determine perpendicular direction
      prefix := tilesInDirection(board, r, c, perpendicular(dir), backward)
      suffix := tilesInDirection(board, r, c, perpendicular(dir), forward)

      if len(prefix) == 0 && len(suffix) == 0:
        // No perpendicular neighbours: any letter is valid here
        cc[r][c] = allTrue
        continue

      // For each letter A-Z, check if prefix + letter + suffix is a valid word
      for letter in 'A'..'Z':
        word := prefix + string(letter) + suffix
        cc[r][c][letter-'A'] = dict.Validate(word)
  return cc
```

---

## BL-AI-03: GADDAG Move Generation (Appel-Jacobson §5, GenerateMoves)

The main algorithm generates all valid plays for one direction. Called twice (once for
horizontal, once for vertical); results are merged and deduplicated.

```
generateForDirection(board, rack, dict, dir direction) []MoveCandidate:
  g := dict.GADDAG()
  anchors := findAnchors(board)
  cc := computeCrossChecks(board, dict, dir)
  var candidates []MoveCandidate

  for each anchor in anchors:
    existingLeft := tilesBeforeAnchor(board, anchor, dir)

    if len(existingLeft) > 0:
      // Case A: existing tiles constrain the left part.
      // Navigate the GADDAG along the reversed existing prefix.
      node, ok := traverseReversedPrefix(g, existingLeft)
      if !ok:
        continue  // no GADDAG path for this prefix — no word starts here
      // Move through the arc-separator to begin the forward (right) extension.
      node, ok = g.Successor(node, dictionary.ArcSep)
      if !ok:
        continue
      // Extend right from the anchor using rack tiles.
      extendRight(board, g, cc, rack, anchor, dir, node, existingLeft, &candidates)

    else:
      // Case B: no existing left tiles — enumerate left extensions from rack.
      maxLeft := distanceToLeftEdgeOrOccupied(board, anchor, dir)
      extendLeft(board, g, cc, rack, anchor, dir, g.Root(), maxLeft, nil, &candidates)

  return candidates
```

### extendLeft — recursive left extension (Appel-Jacobson §5)

```
extendLeft(board, g, cc, rack, anchor, dir, node NodeID, limit int, leftTiles []PlacedTile, candidates):
  // Attempt to transition through '+' and extend right from anchor.
  if rightNode, ok := g.Successor(node, ArcSep); ok:
    extendRight(board, g, cc, rack, anchor, dir, rightNode, leftTiles, candidates)

  // If we can still extend further left, try each rack letter.
  if limit > 0:
    for each (letter, tile) in availableRackLetters(rack):  // expands blanks to A-Z
      if nextNode, ok := g.Successor(node, letter); ok:
        removeTileFromRack(rack, tile)
        newLeft := prepend(PlacedTile{tile with letter, positionLeft(anchor, len(leftTiles)+1, dir)}, leftTiles)
        extendLeft(board, g, cc, rack, anchor, dir, nextNode, limit-1, newLeft, candidates)
        restoreTileToRack(rack, tile)
```

### extendRight — recursive right extension (Appel-Jacobson §5)

```
extendRight(board, g, cc, rack, anchor, dir, node NodeID, leftTiles []PlacedTile, candidates):
  pos := positionRight(anchor, len(rightTilesPlacedSoFar), dir)
  if pos is off-board:
    return

  if !board.IsEmpty(pos.row, pos.col):
    // Cell is occupied: must follow the existing tile's letter.
    letter := existingLetter(board, pos)
    if nextNode, ok := g.Successor(node, letter); ok:
      if nextNode.IsTerminal && nextPositionIsEmpty(pos, dir, board):
        recordCandidate(leftTiles, existingTilesUsed, &candidates)
      extendRight(board, g, cc, rack, anchor, dir, nextNode, leftTiles, candidates)

  else:
    // Cell is empty: try each rack letter that passes the cross-check.
    for each (letter, tile) in availableRackLetters(rack):
      if !cc[pos.row][pos.col][letter-'A']:
        continue   // cross-check violation
      if nextNode, ok := g.Successor(node, letter); ok:
        newRight := append(rightTiles, PlacedTile{tile with letter, pos})
        allPlaced := leftTiles + newRight
        if nextNode.IsTerminal && nextPositionIsEmpty(pos, dir, board):
          recordCandidate(allPlaced, &candidates)
        removeTileFromRack(rack, tile)
        extendRight(board, g, cc, rack, anchor, dir, nextNode, leftTiles, candidates)
        restoreTileToRack(rack, tile)
```

### recordCandidate — validate, score, and add to candidates

```
recordCandidate(placed []PlacedTile, board *engine.Board, dict *dictionary.Dictionary, candidates *[]MoveCandidate):
  move := engine.PlayMove{Placed: placed}
  // ValidatePlacement is the authoritative check; catches edge cases missed by traversal.
  if _, err := engine.ValidatePlacement(board, &move, dict); err != nil:
    return  // skip invalid moves (defensive; should not happen if traversal is correct)
  score, _ := engine.Score(board, &move)
  access := computeOpponentAccess(board, placed)
  *candidates = append(*candidates, MoveCandidate{Move: move, Score: score, OpponentAccess: access})
```

---

## BL-AI-04: Blank Tile Expansion (`availableRackLetters`)

```
availableRackLetters(rack *engine.Rack) []struct{ letter byte; tile engine.Tile }:
  var result []...
  for each tile in rack.Tiles():
    if tile.IsBlank:
      for l in 'A'..'Z':
        result = append(result, {letter: l, tile: Tile{IsBlank:true, AssignedLetter:l, Points:0}})
    else:
      result = append(result, {letter: tile.Letter, tile: tile})
  return result
```

Blanks expand to all 26 letters. `AssignedLetter` is set so `PlayMove.Placed` records
which letter the blank was used as, enabling correct board display and scoring (blank
always scores 0 regardless of assigned letter).

---

## BL-AI-05: OpponentAccess Calculation (Option A — Q1 decision)

After a hypothetical move is placed, count all empty premium squares that have at
least one board-tile neighbour. Premium squares = DoubleLetter, TripleLetter,
DoubleWord, TripleWord, Centre.

```
computeOpponentAccess(board *engine.Board, placed []PlacedTile) int:
  // Build a virtual board with placed tiles overlaid.
  count := 0
  dirs := [(-1,0),(1,0),(0,-1),(0,1)]
  for r in 0..14:
    for c in 0..14:
      sq := board.Cell(r,c).Square
      if sq == Normal:
        continue   // not a premium square; skip
      if !isEmptyOnVirtualBoard(board, placed, r, c):
        continue   // premium square is occupied; skip
      for each (dr, dc) in dirs:
        if !isEmptyOnVirtualBoard(board, placed, r+dr, c+dc):
          count++
          break    // one adjacent tile is enough to count this square
  return count
```

---

## BL-AI-06: Move Deduplication

The same word at the same board position can be generated from multiple anchors.
After generating candidates for both directions, deduplicate by key
`(minRow, minCol, maxRow, maxCol, direction)` where minRow/Col is the first tile
position and maxRow/Col is the last.

```
dedup(candidates []MoveCandidate) []MoveCandidate:
  seen := map[moveKey]bool{}
  var result []MoveCandidate
  for each c in candidates:
    k := makeMoveKey(c.Move)
    if !seen[k]:
      seen[k] = true
      result = append(result, c)
  return result
```

---

## BL-AI-07: GenerateMoves (top-level)

```
GenerateMoves(board *engine.Board, rack *engine.Rack, dict *dictionary.Dictionary) []MoveCandidate:
  hCandidates := generateForDirection(board, rack, dict, horizontal)
  vCandidates := generateForDirection(board, rack, dict, vertical)
  all := append(hCandidates, vCandidates...)
  all = dedup(all)
  // Sort by Score descending (primary); OpponentAccess ascending (secondary for level-10 stability).
  sort.Slice(all, func(i, j int) bool {
    if all[i].Score != all[j].Score:
      return all[i].Score > all[j].Score
    return all[i].OpponentAccess < all[j].OpponentAccess
  })
  return all
```

---

## BL-AI-08: SelectMove — Difficulty Model (FR-05)

```
SelectMove(candidates []MoveCandidate, level int, rng *rand.Rand) MoveCandidate:
  if len(candidates) == 0:
    panic("SelectMove called with empty candidates")

  if level == 10:
    // Candidates are already sorted: highest score first, then lowest OpponentAccess.
    return candidates[0]

  // Level 1..9: select uniformly from the top k candidates.
  // k = max(1, round(total × (1 - (level-1)/9)))
  fraction := 1.0 - float64(level-1)/9.0
  k := int(math.Round(float64(len(candidates)) * fraction))
  if k < 1:
    k = 1
  if k > len(candidates):
    k = len(candidates)
  return candidates[rng.Intn(k)]
```

**Interpolation check**:
| Level | Fraction | k (if 100 candidates) |
|---|---|---|
| 1 | 1.000 | 100 (all) |
| 2 | 0.889 | 89 |
| 5 | 0.556 | 56 |
| 9 | 0.111 | 11 |
| 10 | special | 1 (top) |

---

## BL-AI-09: ChooseMove

```
ChooseMove(state *engine.GameState, dict *dictionary.Dictionary, level int, rng *rand.Rand) engine.Move:
  candidates := GenerateMoves(state.Board, state.AIRack, dict)

  if len(candidates) > 0:
    c := SelectMove(candidates, level, rng)
    return c.Move   // engine.PlayMove

  // No valid plays: prefer exchange over pass (BR-AI-11).
  if state.Bag.Count() >= engine.MaxRackSize:
    return engine.ExchangeMove{Tiles: state.AIRack.Tiles()}

  return engine.PassMove{}
```

---

## BL-AI-10: AIWorker Goroutine

```
NewAIWorker() *AIWorker:
  w := &AIWorker{
    reqCh: make(chan aiRequest, 1),
    resCh: make(chan engine.Move, 1),
  }
  go w.run()
  return w

AIWorker.run():  // background goroutine — runs for process lifetime
  for req := range w.reqCh:
    move := ChooseMove(req.state, req.dict, req.level, req.rng)
    w.resCh <- move

AIWorker.Request(state *engine.GameState, dict *dictionary.Dictionary, level int):
  if w.busy:
    panic("ai.AIWorker.Request: request already in flight")
  w.busy = true
  clone := state.Clone()
  rng := rand.New(rand.NewSource(time.Now().UnixNano()))
  w.reqCh <- aiRequest{state: clone, dict: dict, level: level, rng: rng}

AIWorker.Poll() (engine.Move, bool):
  select {
  case move := <-w.resCh:
    w.busy = false
    return move, true
  default:
    return nil, false
  }
```

---

## PBT Property Summary

| ID | Property | Oracle |
|---|---|---|
| PBT-AI-01 | Every MoveCandidate passes engine.ValidatePlacement | call ValidatePlacement on each returned candidate |
| PBT-AI-02 | Level-10 SelectMove returns the highest-scoring candidate | brute-force max |
| PBT-AI-03 | Level-1 SelectMove returns a candidate from the full set | index in [0, len) |
| PBT-AI-04 | SelectMove at level N returns a candidate with rank ≤ k | rank check |
| PBT-AI-05 | OpponentAccess ≥ 0 for any move | non-negative check |
| PBT-AI-06 | GenerateMoves on an empty board with a known rack includes all valid words | dictionary oracle |
| PBT-AI-07 | Score of every MoveCandidate equals engine.Score result | re-score and compare |
