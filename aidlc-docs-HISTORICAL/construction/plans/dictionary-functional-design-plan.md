# Functional Design Plan — Unit 1: `dictionary`

## Execution Checklist

- [x] Step A: Confirmed answers — Q1:A (map[byte]NodeID) Q2:B (skip+warn) Q3:A (normalise internally)
- [x] Step B: Generate domain-entities.md
- [x] Step C: Generate business-logic-model.md
- [x] Step D: Generate business-rules.md
- [x] Step E: PBT-01 testable properties documented in business-logic-model.md

---

## Functional Design Questions

Please answer each question by filling in the letter choice after the `[Answer]:` tag.
Let me know when done.

---

### Question 1
How should GADDAG nodes store their outgoing edges internally?

A) `map[byte]NodeID` — Go map keyed by letter byte; flexible, idiomatic, slightly more memory overhead
B) Sorted `[]struct{letter byte; node NodeID}` slice — compact, cache-friendly, binary-searchable; better for large graphs
C) Fixed-length `[27]NodeID` array indexed by letter (A=0…Z=25, arc-separator=26) — O(1) access, highest memory per node
D) Other (please describe after [Answer]: tag below)

[Answer]: A 

---

### Question 2
How should the GADDAG build tool handle words in source word lists that contain non-A-Z characters (e.g., hyphens, apostrophes, accented letters)?

A) Skip silently — ignore any word containing a character outside A-Z (uppercase after normalisation)
B) Skip with warning — log each skipped word to stderr; continue building
C) Fail loudly — return an error and abort if any non-A-Z word is encountered
D) Other (please describe after [Answer]: tag below)

[Answer]: B

---

### Question 3
Should `Dictionary.Validate` and `GADDAG.Contains` accept lowercase input, or require pre-normalised uppercase?

A) Accept either case — normalise to uppercase internally before lookup (caller convenience)
B) Require uppercase — callers must normalise; keeps the hot path allocation-free
C) Other (please describe after [Answer]: tag below)

[Answer]: A
