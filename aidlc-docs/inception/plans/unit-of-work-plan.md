# Unit of Work Plan — Squabble

## Execution Checklist

- [x] Step A: Confirmed answers — Q1:A Q2:A Q3:A (5 units, sequential, separate cmd unit)
- [x] Step B: Generate unit-of-work.md
- [x] Step C: Generate unit-of-work-dependency.md
- [x] Step D: Generate unit-of-work-story-map.md
- [x] Step E: Validate all stories assigned; no gaps or overlaps

---

## Decomposition Questions

Please answer each question by filling in the letter choice after the `[Answer]:` tag.
Let me know when done.

---

### Question 1
The GADDAG build tool (`go generate` script) converts raw word list text files into pre-built `.gob` files embedded in the binary. How should this tool be treated?

A) Part of the `dictionary` unit — the build tool lives in `tools/buildgaddag/` and is documented, built, and tested within Unit 1
B) A separate Unit 0 (pre-requisite) that must be completed and its output committed before Unit 1 begins
C) Other (please describe after [Answer]: tag below)

[Answer]: A 

---

### Question 2
What is the preferred development sequence for the 4 units?

A) Strictly sequential: Unit 1 fully complete (code + tests passing) before Unit 2 begins, and so on
B) Overlapping: start stubbing the next unit's interface as soon as the current unit's API is stable, before all tests pass
C) Other (please describe after [Answer]: tag below)

[Answer]: A

---

### Question 3
Should the `cmd/squabble` entry-point wiring be a separate named unit, or treated as the final step of the `ui` unit?

A) Separate unit — Unit 5: `cmd` (wires all packages, entry point, platform build configuration)
B) Part of Unit 4 (`ui`) — `cmd/squabble/main.go` is generated as the last step of the ui unit
C) Other (please describe after [Answer]: tag below)

[Answer]: A
