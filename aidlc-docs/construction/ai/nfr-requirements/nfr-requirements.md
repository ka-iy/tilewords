# NFR Requirements — Unit 3: `ai`

## Performance (NFR-01)

| ID | Requirement | Threshold | Rationale |
|---|---|---|---|
| NFR-AI-P1 | `GenerateMoves` wall-clock time | ≤ 500 ms on modern desktop (Intel/AMD 2018+, ARM64) | FR-01 / NFR-01 hard limit; UI shows "AI thinking…" during computation |
| NFR-AI-P2 | `GenerateMoves` wall-clock time on mid-range Android (2020+) | ≤ 1500 ms | Mobile tolerance; still produces a result within one Ebitengine Draw cycle budget |
| NFR-AI-P3 | `SelectMove` execution time | ≤ 1 ms | Pure in-memory sort slice + index; negligible |
| NFR-AI-P4 | Cross-check precomputation | ≤ 10 ms | O(225 × 26 × avg_word_len) dict lookups; bounded and fast |
| NFR-AI-P5 | `computeOpponentAccess` per candidate | ≤ 50 µs | O(225 × 4) loop; called once per candidate |

**Performance strategy**: The GADDAG traversal is bounded by rack size (≤7 tiles) and the GADDAG structure. On large dictionaries (e.g., CSW ~280,000 words) the total candidate count is typically 1,000–10,000 moves. No additional performance optimisations (e.g., anchor-pruning, candidate pre-filtering) are required for v1, but the design does not preclude them.

---

## Memory (NFR-03)

| ID | Requirement | Threshold | Rationale |
|---|---|---|---|
| NFR-AI-M1 | `crossCheckSet` per call | ≤ 6 KB | [15][15][26]bool = 5,850 bytes; allocated on stack or as a local var |
| NFR-AI-M2 | `candidates` slice per call | ≤ 4 MB | ≤10,000 candidates × ~400 bytes per MoveCandidate (PlayMove + words + score) |
| NFR-AI-M3 | AIWorker goroutine stack | OS default (≥ 8 KB, grows as needed) | Single goroutine; stack usage bounded by GADDAG traversal depth (≤ 7 + board length ≈ 22) |

---

## Thread Safety (NFR-AI-T1)

`GenerateMoves`, `SelectMove`, and `ChooseMove` are stateless package-level functions.
They are safe for concurrent calls with independent arguments (no shared mutable state).

`AIWorker` encapsulates all goroutine communication:
- `reqCh` and `resCh` are buffered channels (capacity 1), preventing blocking on send.
- The `busy` flag is written and read only on the UI goroutine; the AI goroutine never reads it.
- No mutexes are needed: the channel protocol enforces happens-before.

---

## Reliability (NFR-04)

| ID | Requirement |
|---|---|
| NFR-AI-R1 | `GenerateMoves` must not panic on any valid (board, rack, dict) input, including an empty rack or a board in any legal game state |
| NFR-AI-R2 | `SelectMove` panics only when `candidates` is empty — this is a programming error (callers must check len > 0 before calling) |
| NFR-AI-R3 | `AIWorker.Request` panics if called while busy — the UI is responsible for checking `busy` before calling |
| NFR-AI-R4 | The AIWorker goroutine must never deadlock; it blocks only on `reqCh` receive (unbounded wait) and `resCh <- result` (buffered, capacity 1, never blocks if Poll is eventually called) |
| NFR-AI-R5 | `ChooseMove` must always return a non-nil `engine.Move`; it must never return nil |

---

## Correctness (NFR-AI-C1)

Every `MoveCandidate` returned by `GenerateMoves` must be a legal move: it must pass
`engine.ValidatePlacement` with the same board and dictionary. This is enforced by
`recordCandidate` calling `ValidatePlacement` as a defensive gate after GADDAG traversal.
Violations are silently discarded (not returned as errors).

---

## Security (NFR-08 / SECURITY-15)

| ID | Requirement |
|---|---|
| SECURITY-AI-1 | No exported function in `ai` returns a raw error; all errors are wrapped with the function name prefix: `fmt.Errorf("ai.GenerateMoves: …")` |
| SECURITY-AI-2 | The AI goroutine operates on a deep clone of `GameState`; it never holds a reference to the original state, preventing concurrent mutation bugs |
| SECURITY-AI-3 | `go test -race ./ai/...` must pass with no reported races |

---

## Testability (NFR-07)

| ID | Requirement |
|---|---|
| NFR-AI-TEST-1 | `GenerateMoves` is fully testable without a running UI or real embedded assets; tests use `dictionary.NewFromWords` and `engine.NewTestBag` |
| NFR-AI-TEST-2 | `SelectMove` is testable with a hand-constructed `[]MoveCandidate` slice — no board or GADDAG needed |
| NFR-AI-TEST-3 | `AIWorker` is testable by injecting a mock `ChooseMove`-equivalent via a function parameter or interface |
| NFR-AI-TEST-4 | PBT framework `pgregory.net/rapid` is used for property-based tests (PBT-AI-01 through PBT-AI-07) |
| NFR-AI-TEST-5 | `go test -race ./ai/...` must pass; race detector catches any accidental shared-state access in AIWorker |

---

## Code Commentary (NFR-10)

| ID | Requirement |
|---|---|
| NFR-AI-C1 | Every exported function and type in `ai` has a GoDoc comment |
| NFR-AI-C2 | `extendLeft` and `extendRight` have inline comments at each step referencing Appel-Jacobson §5 and naming the algorithm phase (e.g., "// left-extension phase: navigate the GADDAG reversed prefix — Appel-Jacobson §5") |
| NFR-AI-C3 | The cross-check precomputation loop is commented to explain why cross-checks are per-direction and why an empty-neighbour cell gets the all-true set |
| NFR-AI-C4 | The `SelectMove` k-computation formula is commented with the FR-05 reference and an example showing level→k mapping |
| NFR-AI-C5 | `computeOpponentAccess` is commented to explain Option A (total post-move exposure) and the Q1 design decision |
