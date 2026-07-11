# Unit of Work Story Map — Squabble

## Mapping Summary

All 19 user stories are assigned. Each story has one **primary unit** (the unit whose code makes the story's acceptance criteria testable) and zero or more **contributing units** (units whose APIs the primary unit depends on for this story).

| Story | Title | Primary Unit | Contributing Units |
|---|---|---|---|
| US-01 | Launch the application | U4: ui | U5: cmd |
| US-02 | Select a dictionary | U4: ui | U1: dictionary |
| US-03 | Select AI difficulty | U4: ui | — |
| US-04 | Start a new game | U4: ui | U2: engine |
| US-05 | View the initial game board | U4: ui | U2: engine |
| US-06 | Draw initial tiles | U2: engine | U4: ui |
| US-07 | Place tiles on the board | U4: ui | U2: engine |
| US-08 | Word validation — no bluffing | U2: engine | U1: dictionary, U4: ui |
| US-09 | Score a valid move | U2: engine | U4: ui |
| US-10 | Exchange tiles | U2: engine | U4: ui |
| US-11 | Pass a turn | U2: engine | U4: ui |
| US-12 | Watch the AI take its turn | U4: ui | U3: ai, U2: engine |
| US-13 | AI difficulty affects move quality | U3: ai | — |
| US-14 | Undo last move | U2: engine | U4: ui |
| US-15 | Save the game | U4: ui | U2: engine |
| US-16 | Resume a saved game | U4: ui | U2: engine |
| US-17 | Game ends — rack exhausted | U2: engine | U4: ui |
| US-18 | Game ends — six consecutive passes | U2: engine | U4: ui |
| US-19 | View final scores and winner | U4: ui | U2: engine |

---

## Stories by Unit

### Unit 1: `dictionary`
**Primary**: *(none — dictionary is a supporting unit for US-02 and US-08)*
**Contributing to**: US-02, US-08

### Unit 2: `engine`
**Primary owner of**: US-06, US-08, US-09, US-10, US-11, US-14, US-17, US-18

| Story | Acceptance Criteria Owned |
|---|---|
| US-06 | Random draw of 7 tiles each; bag count display |
| US-08 | ValidatePlacement rejects invalid words pre-commit; no penalty |
| US-09 | Scoring: DL/TL/DW/TW multipliers; bingo +50; cross-word accumulation |
| US-10 | Exchange requires ≥7 tiles in bag; returns selected, draws same count |
| US-11 | Pass increments consecutive-pass counter |
| US-14 | Command.Undo restores board, racks, bag, scores, pass counter exactly |
| US-17 | IsGameOver detects rack-empty + bag-empty; ApplyEndgameScoring redistributes |
| US-18 | IsGameOver detects 6 consecutive passes; no score redistribution |

### Unit 3: `ai`
**Primary owner of**: US-13

| Story | Acceptance Criteria Owned |
|---|---|
| US-13 | Level 1 selects randomly; level 10 always picks max-score move (tie-break); levels 2–9 monotonically increase; no invalid moves at any level; ≤500 ms |

### Unit 4: `ui`
**Primary owner of**: US-01, US-02, US-03, US-04, US-05, US-07, US-12, US-15, US-16, US-19

| Story | Acceptance Criteria Owned |
|---|---|
| US-01 | App launches; main menu shows New Game / Resume / Quit; Resume disabled without save |
| US-02 | Setup screen lists 6 dict options; selected dict shown during play |
| US-03 | Difficulty selector 1–10; shown during play; locked after game start |
| US-04 | Board initialised; both racks (human + AI) always fully visible; centre square highlighted |
| US-05 | Board rendered with open-licensed images; all premium squares visually distinct |
| US-07 | Drag-and-drop and click-to-place; staged tile distinction; Play/Cancel buttons |
| US-12 | AI move displayed within 500 ms; tiles highlighted ≥1 s; "AI thinking..." indicator |
| US-15 | Save option in game menu; success confirmation; overwrite prompt; user-only permissions |
| US-16 | Resume enabled when save exists; restores full state; graceful corrupt-file handling |
| US-19 | End-game screen: both scores, winner/draw, end condition, New Game / Quit |

### Unit 5: `cmd`
**Primary**: *(none — cmd is the integration and build unit)*
**Contributing to**: US-01 (cross-platform launch), all stories (platform build correctness)

---

## Coverage Validation

- **Total stories**: 19
- **Stories with a primary unit assigned**: 19 ✓
- **Stories with no primary unit**: 0 ✓
- **Stories spanning >2 units**: US-08 (U1+U2+U4), US-12 (U3+U4+U2) — both handled correctly by primary unit owning end-to-end acceptance criteria
- **Gaps**: None — all FR-01 through FR-11 requirements are covered by at least one story
