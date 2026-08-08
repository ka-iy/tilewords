# Requirements Clarification Questions — Squabble (Cross-Platform Scrabble in Go)

Please answer each question by filling in the letter choice after the `[Answer]:` tag.
If none of the options match, choose the last option (Other/X) and describe your preference after the tag.
Let me know when you are done.

---

## Question 1
Which graphical framework should be used for cross-platform UI (desktop + mobile)?

A) Ebitengine (2D game engine, pure Go, supports desktop + mobile + web via WASM)
B) Fyne (Go GUI toolkit, desktop + mobile, widget-based)
C) Gio (Immediate-mode Go UI, desktop + mobile)
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 2
Which mobile platforms must be supported?

A) Android only
B) iOS only
C) Both Android and iOS
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 3
Which desktop platforms must be supported?

A) Linux only
B) Windows and Linux
C) Windows, macOS, and Linux
D) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question 4
Should the game also run in a web browser (e.g., via WebAssembly)?

A) Yes — web browser support required
B) No — desktop and mobile only
C) Nice to have but not required
D) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question 5
For the Appel-Jacobson engine (GADDAG / DAWG data structure), which word list format should be used internally?

A) GADDAG (as described in the 1998 paper — faster move generation)
B) DAWG/CDAWG (more memory efficient)
C) Let the implementation decide (GADDAG recommended for correctness to the paper)
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 6
How should the dictionaries be bundled with the game?

A) Embedded in the binary at compile time (no external files required at runtime)
B) Downloaded at first launch from an online source
C) Shipped as separate data files alongside the binary
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 7
Which dictionaries should be available by default (the user picks at game start)?

A) SOWPODS and TWL06/OTCWL only (most popular internationally)
B) All five listed: Collins Scrabble Words (CSW), SOWPODS, OSPD, NASPA Word List (NWL/TWL), OTCWL
C) All five plus a combined "all dictionaries" option with deduplication
D) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question 8
For computer AI difficulty (levels 1–10), which strategy model should govern the lower levels?

A) Random valid move selection (level 1) up to optimal move (level 10) with linear interpolation
B) Score threshold: level N plays any move scoring ≥ N×(max_score/10), else plays a lower-scoring move
C) Introduce deliberate mistakes: level 1 plays the worst valid move, level 10 plays the best
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 9
What should happen when the computer AI at level 10 has multiple moves with the same maximum score?

A) Pick any one arbitrarily (first found)
B) Break ties by preferring moves that open fewer premium squares for the opponent
C) Break ties randomly
D) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 10
Should the game support saving and resuming games?

A) Yes — save/load game state to a local file
B) No — single-session play only
C) Yes — with cloud sync (requires account/login)
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 11
How many human players can play per game?

A) Exactly one human vs. the computer
B) One or two humans vs. the computer
C) Up to four players (mix of human and computer)
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 12
What tile bag configuration should be used?

A) Standard North American English Scrabble distribution (100 tiles)
B) Configurable per selected dictionary/locale
C) Always standard 100-tile bag regardless of dictionary
D) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question 13
How should the game board and tiles be rendered?

A) Vector/2D sprites drawn by the game engine (custom art assets)
B) Use open-licensed Scrabble-style board and tile images
C) Procedurally drawn tiles and board using primitive shapes (no external assets)
D) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Question 14
Should the game enforce a strict no-bluffing rule (as you specified)?

A) Yes — the played word is always validated against the selected dictionary in real time; invalid words are rejected before placement
B) Yes — invalid words are rejected AND a penalty is applied to the player who attempted to play one
C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 15
What should happen if a player cannot play any valid word?

A) Player must exchange tiles (or pass if bag is empty)
B) Player may pass; three consecutive passes end the game
C) Follow official Scrabble tournament rules (exchange or pass; six consecutive passes end game)
D) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## Question 16
Should the application include an undo move feature for the human player?

A) Yes — undo last move (human only)
B) No — moves are final once committed
C) Yes — unlimited undo history
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 17
What is the primary target language/locale for the game at launch?

A) English only
B) English with a roadmap for other languages
C) Multiple languages from launch (specify which after [Answer] if Other)
D) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 18 — Extension: Property-Based Testing
Should property-based testing (PBT) rules be enforced for this project?

A) Yes — enforce all PBT rules as blocking constraints (recommended for projects with business logic, data transformations, serialization, or stateful components)
B) Partial — enforce PBT rules only for pure functions and serialization round-trips (suitable for projects with limited algorithmic complexity)
C) No — skip all PBT rules (suitable for simple CRUD applications, UI-only projects, or thin integration layers with no significant business logic)
X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Question 19 — Extension: Security Baseline
Should security extension rules be enforced for this project?

A) Yes — enforce all SECURITY rules as blocking constraints (recommended for production-grade applications)
B) No — skip all SECURITY rules (suitable for PoCs, prototypes, and experimental projects)
X) Other (please describe after [Answer]: tag below)

[Answer]: A
