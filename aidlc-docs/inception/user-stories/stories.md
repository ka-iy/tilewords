# User Stories — Squabble
**Breakdown**: User Journey-Based | **Acceptance Criteria**: Concise bullets | **Sizing**: None

---

## Journey Stage 1: App Launch & Setup

### US-01: Launch the Application
**As** a player,
**I want** to launch Squabble and be presented with a main menu,
**so that** I can configure and start a game.

**Acceptance Criteria:**
- The application launches without error on Windows, macOS, Linux, and Android.
- The main menu displays options: New Game, Resume Game, and Quit.
- "Resume Game" is greyed out / disabled when no save file exists.
- The app title does not contain any Hasbro-trademarked names.

**References**: FR-01, NFR-09

---

### US-02: Select a Dictionary
**As** a player,
**I want** to choose which word list the game uses,
**so that** I can play with the dictionary I prefer or am most familiar with.

**Acceptance Criteria:**
- The new game setup screen lists all six options: CSW, SOWPODS, OSPD, NWL/NASPA, OTCWL, All Dictionaries.
- Selecting "All Dictionaries" silently deduplicates words before the game begins.
- The selected dictionary is shown on the game screen throughout play.
- Only one dictionary (or the combined set) is active per game session.

**References**: FR-03

---

### US-03: Select AI Difficulty Level
**As** a player,
**I want** to choose a difficulty level from 1 to 10 before the game starts,
**so that** I can play a game matched to my skill level.

**Acceptance Criteria:**
- The new game setup screen presents a selector for levels 1–10.
- The selected level is shown on the game screen throughout play.
- Level cannot be changed after the game has started.
- Tooltip or label describes level 1 as "Easiest" and level 10 as "Expert".

**References**: FR-05

---

### US-04: Start a New Game
**As** a player,
**I want** to start a new game after selecting my dictionary and difficulty,
**so that** the board, tile bag, and racks are initialised and play begins.

**Acceptance Criteria:**
- Pressing "Start Game" initialises a 15×15 board, shuffles a standard 100-tile bag, and deals 7 tiles to each player.
- The human player's rack is always visible. The AI's rack is hidden by default; a "Show AI Rack" / "Hide AI Rack" toggle button reveals or conceals it at any time, so the player can optionally verify the AI only plays tiles it legitimately holds.
- The centre star square is highlighted as the required starting position.
- The human player takes the first turn.

**References**: FR-04, FR-06, FR-08

---

## Journey Stage 2: Game Start

### US-05: View the Initial Game Board
**As** a player,
**I want** to see a clear 15×15 board with premium squares marked,
**so that** I can plan my opening move.

**Acceptance Criteria:**
- All 15×15 squares are rendered using open-licensed board graphics.
- Double Letter (DL), Triple Letter (TL), Double Word (DW), Triple Word (TW), and centre star squares are visually distinct and labelled.
- The board layout matches the standard Scrabble board premium square positions.
- The board does not reproduce Hasbro's copyrighted board artwork.

**References**: FR-01, NFR-06, NFR-09

---

### US-06: Draw Initial Tiles
**As** a player,
**I want** my 7 starting tiles to be drawn randomly from the shuffled bag,
**so that** each game starts with a fair and unpredictable rack.

**Acceptance Criteria:**
- Exactly 7 tiles are dealt to the human player and 7 to the AI from the same shuffled bag.
- Tiles display the letter and point value using open-licensed tile images.
- Blank tiles are shown with no letter until played.
- The remaining bag count is shown on the game screen.

**References**: FR-06, NFR-06

---

## Journey Stage 3: Human Player Turn

### US-07: Place Tiles on the Board to Form a Word
**As** a player,
**I want** to drag tiles from my rack onto the board to form a word,
**so that** I can make my move.

**Acceptance Criteria:**
- Tiles can be placed by drag-and-drop or by click-select then click-to-place.
- Placed-but-uncommitted tiles are visually distinguished from already-committed tiles.
- The player may reposition uncommitted tiles freely before committing.
- Pressing "Play" commits the move; pressing "Cancel" returns tiles to the rack.
- The first word played must cover the centre square.
- Subsequent words must connect to at least one tile already on the board.

**References**: FR-01, NFR-05

---

### US-08: Word Validation — No Bluffing
**As** a player,
**I want** my word to be validated against the selected dictionary before it is accepted,
**so that** invalid words are never committed to the board.

**Acceptance Criteria:**
- When the player presses "Play", every new word formed (main word and any cross-words) is checked against the active dictionary.
- If any word is invalid, the move is rejected and the player is shown which word(s) failed.
- The rejected tiles remain on the board in their staged positions so the player can adjust.
- No penalty is applied for an invalid attempt.
- Valid moves are committed immediately without a second confirmation step.

**References**: FR-04

---

### US-09: Score a Valid Move (Including Bingo Bonus)
**As** a player,
**I want** my score to be calculated correctly when I play a valid word,
**so that** premium squares and bingo bonuses are properly rewarded.

**Acceptance Criteria:**
- Letter values are multiplied by DL/TL modifiers before word multipliers are applied.
- DW and TW multipliers are applied to the full word score after letter multipliers.
- If the player uses all 7 tiles in one move, a 50-point bingo bonus is added.
- The player's running score is updated on screen immediately after the move.
- Multiple word formations in one move (cross-words) are each scored and summed.

**References**: FR-02

---

### US-10: Exchange Tiles
**As** a player,
**I want** to exchange any subset of my tiles when the bag has 7 or more tiles,
**so that** I can improve a poor rack without passing.

**Acceptance Criteria:**
- An "Exchange" button is available on the player's turn.
- The player selects 1–7 tiles to exchange; the selection is visually highlighted.
- Pressing "Confirm Exchange" returns selected tiles to the bag, redraws the same count, and ends the player's turn.
- Exchange is disabled (greyed out) when fewer than 7 tiles remain in the bag.
- Exchanging counts as the player's turn; the consecutive-pass counter is reset to zero.

**References**: FR-07

---

### US-11: Pass a Turn
**As** a player,
**I want** to pass my turn without placing tiles or exchanging,
**so that** I can concede a turn when I have no useful move.

**Acceptance Criteria:**
- A "Pass" button is always available on the player's turn.
- Passing ends the player's turn with no score change and no tile change.
- The consecutive-pass counter increments by 1.
- After the player passes, the AI takes its turn.

**References**: FR-07

---

## Journey Stage 4: AI Player Turn

### US-12: Watch the AI Take Its Turn
**As** a player,
**I want** to see the AI's move play out on the board after my turn,
**so that** I can follow the game and understand what the AI played.

**Acceptance Criteria:**
- After the human's turn is committed, the AI's move is computed and displayed within 500 ms.
- The AI's played tiles are animated or highlighted on the board for at least 1 second so the player can see what was played.
- The word(s) played and the AI's score for that turn are shown briefly on screen.
- The AI's total score is updated in the score display.
- If the AI passes or exchanges, a brief message indicates this.

**References**: FR-05, NFR-01

---

### US-13: AI Difficulty Affects Move Quality
**As** a player,
**I want** the AI's playing strength to match the difficulty level I chose,
**so that** lower levels feel beatable and level 10 feels like a tournament expert.

**Acceptance Criteria:**
- At level 1, the AI selects randomly from all valid moves (observably non-optimal play).
- At level 10, the AI always selects the highest-scoring valid move; ties are broken by minimising premium-square access for the human.
- At intermediate levels, move quality increases monotonically with level (no level N is consistently stronger than level N+1).
- The AI never plays an invalid word at any difficulty level.
- The AI never takes longer than 500 ms to select a move at any difficulty level.

**References**: FR-05, NFR-01

---

## Journey Stage 5: Mid-Game Actions

### US-14: Undo Last Move
**As** a player,
**I want** to undo my most recent completed move immediately after making it,
**so that** I can correct a misclick or reconsider my play.

**Acceptance Criteria:**
- An "Undo" button becomes active immediately after the human player commits a move (and the AI has responded).
- Pressing "Undo" restores the board, both tile racks, the bag, and both scores to the state before the human's move.
- The AI's responding move is also undone.
- "Undo" is only available once per move (one level of undo); after undoing, the button is disabled until the next move is committed.
- Undo is not available for exchange or pass actions that preceded an AI turn where no tiles were placed by the human.

**References**: FR-09

---

### US-15: Save the Game
**As** a player,
**I want** to save my current game at any point during play,
**so that** I can resume it later without losing my progress.

**Acceptance Criteria:**
- A "Save Game" option is available in the game menu at any point during the human player's turn.
- The game state (board, both racks, bag, scores, consecutive-pass count, difficulty, dictionary selection) is saved to the platform's standard app data directory.
- A success confirmation is shown after saving.
- If a save file already exists, it is overwritten after a brief confirmation prompt.
- Saved files do not require special access permissions; they are readable only by the current user.

**References**: FR-10, NFR-08

---

### US-16: Resume a Saved Game
**As** a player,
**I want** to load my saved game from the main menu,
**so that** I can continue a game I started earlier.

**Acceptance Criteria:**
- "Resume Game" on the main menu is enabled only when a save file exists.
- Loading restores the board, racks, scores, bag, difficulty, and dictionary exactly as saved.
- If the save file is corrupted or unreadable, a clear error message is shown and the option to start a new game is offered.
- After loading, play resumes on the human player's turn.

**References**: FR-10, NFR-04

---

## Journey Stage 6: End Game

### US-17: Game Ends When a Player Exhausts Tiles
**As** a player,
**I want** the game to end automatically when a player plays their last tile and the bag is empty,
**so that** the standard Scrabble end condition is enforced.

**Acceptance Criteria:**
- When a player plays their last tile and the bag contains zero tiles, the game ends immediately after that move is scored.
- The player who emptied their rack earns bonus points equal to the sum of the opponent's remaining tile values.
- The opponent's score is reduced by the same amount.
- An end-game screen appears showing both players' final scores and the winner.

**References**: FR-07

---

### US-18: Game Ends After Six Consecutive Passes
**As** a player,
**I want** the game to end if neither player makes a scoring move for six consecutive turns,
**so that** deadlocked games do not continue indefinitely.

**Acceptance Criteria:**
- The game tracks a consecutive-pass counter (incremented by pass or exchange, reset to zero by any tile placement).
- When the counter reaches 6, the game ends immediately.
- No bonus points are redistributed for remaining tiles in this end condition.
- The end-game screen shows both final scores and the winner (or "Draw" if scores are equal).

**References**: FR-07

---

### US-19: View Final Scores and Winner
**As** a player,
**I want** to see a clear end-game summary with final scores and the winner declared,
**so that** I know the outcome of the game.

**Acceptance Criteria:**
- The end-game screen shows: human player's final score, AI's final score, winner declaration (or "Draw").
- The end condition that triggered the game end is stated (e.g., "Tiles exhausted" or "Six consecutive passes").
- The screen offers "New Game" and "Quit" options.
- Scores and outcome are presented without reference to any Hasbro trademarks.

**References**: FR-07, NFR-09

---

## INVEST Compliance Summary

| Story | Independent | Negotiable | Valuable | Estimable | Small | Testable |
|---|---|---|---|---|---|---|
| US-01 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-02 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-03 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-04 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-05 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-06 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-07 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-08 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-09 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-10 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-11 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-12 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-13 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-14 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-15 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-16 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-17 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-18 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
| US-19 | ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |
