# Application Design Plan — Squabble

## Execution Checklist

- [x] Step A: Confirmed answers — Q1:B Q2:C Q3:A Q4:B Q5:A Q6:A Q7:B
- [x] Step B: Generate components.md
- [x] Step C: Generate component-methods.md
- [x] Step D: Generate services.md
- [x] Step E: Generate component-dependency.md
- [x] Step F: Generate application-design.md (consolidated)
- [x] Step G: Validate design completeness and consistency

---

## Design Questions

Please answer each question by filling in the letter choice after the `[Answer]:` tag.
Let me know when done.

---

### Question 1
What Go module/workspace structure should the project use?

A) Single Go module (`go.mod` at repo root) with internal packages: `internal/dictionary`, `internal/engine`, `internal/ai`, `internal/ui`
B) Single Go module with top-level packages: `dictionary/`, `engine/`, `ai/`, `ui/`
C) Go workspace (`go.work`) with one sub-module per unit (separate `go.mod` per package)
D) Other (please describe after [Answer]: tag below)

[Answer]: B 

---

### Question 2
How should the game state be managed to support undo?

A) Mutable game state with a single snapshot saved before each human move (copy-on-write of the state struct)
B) Immutable game state — each move produces a new state value; undo restores the previous value from a stack
C) Mutable game state with a command/inverse-command pattern (store the inverse operation, not a full snapshot)
D) Other (please describe after [Answer]: tag below)

[Answer]: C

---

### Question 3
How should the AI move computation be executed to keep the UI responsive?

A) Run AI computation in a separate goroutine; UI shows an "AI thinking..." indicator and receives the move via a channel
B) Run AI computation synchronously on the main/game-loop goroutine (acceptable if ≤500 ms target is met)
C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

### Question 4
What file format should be used for saved games?

A) JSON (human-readable, easy to debug, slightly larger)
B) Go `encoding/gob` binary (compact, fast, Go-native)
C) Other (please describe after [Answer]: tag below)

[Answer]: B

---

### Question 5
How should blank tiles be represented in the GADDAG and on the board?

A) Blank in rack = a wildcard byte (e.g., `0`); when played, the chosen letter is stored on the board with a flag marking it as a blank-origin tile (zero score)
B) Blank tiles are pre-expanded: one GADDAG entry per possible letter assignment (A–Z) at generation time
C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

### Question 6
What coordinate system should the board use internally?

A) `(row, col)` with row 0 at the top, col 0 at the left (standard matrix notation)
B) `(x, y)` with x increasing right, y increasing down (screen-space convention)
C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

### Question 7
How should dictionary word lists be sourced and embedded?

A) Store raw word list text files (one word per line) in a `assets/dictionaries/` directory, compress with gzip at build time using `go generate`, embed compressed files with `//go:embed`
B) Store pre-built serialised GADDAG structures (one per dictionary) in `assets/`, embed with `//go:embed`, deserialise at startup
C) Store raw word list text files and build the GADDAG in memory at startup (no pre-compilation step)
D) Other (please describe after [Answer]: tag below)

[Answer]: B
