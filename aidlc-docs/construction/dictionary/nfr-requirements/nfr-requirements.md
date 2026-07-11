# NFR Requirements — Unit 1: `dictionary`

## Performance

### NFR-DICT-P1: GADDAG Load Time
- All dictionaries selected by the user must be fully deserialised and ready within the global **3-second startup budget** (NFR-01).
- For a single dictionary: target ≤500 ms deserialisation.
- For the combined "all" dictionary: target ≤1 s (largest gob file).
- Measurement: wall-clock time from `Loader.Load()` call to `*Dictionary` ready.

### NFR-DICT-P2: Word Validation Throughput
- `Dictionary.Validate(word)` must complete in **O(n)** time where n = word length (≤15).
- No heap allocations on the hot path for uppercase input (fast path in BR-01).
- Target: ≤1 µs per validation call on target hardware (Intel/AMD 2018+, ARM64).
- Rationale: the AI move generator calls validation thousands of times per move.

### NFR-DICT-P3: GADDAG Traversal Latency
- `GADDAG.Successor` and `GADDAG.IsTerminal` must each complete in **O(1)** amortised time (single map lookup).
- No locks or synchronisation overhead (read-only after Load; see thread safety below).

---

## Binary Size

### NFR-DICT-B1: Embedded Asset Budget
- All 6 `.gob` files combined must fit within **20 MB** of the 50 MB binary budget, leaving headroom for other packages and assets.
- Expected GADDAG sizes (estimated): CSW ~3 MB, SOWPODS ~3 MB, OSPD ~2 MB, NASPA ~2 MB, OTCWL ~2 MB, All ~5 MB ≈ 17 MB total.
- If any individual file exceeds its estimate by >50%, the build tool must be optimised (node sharing, more compact edge representation) before committing.

---

## Reliability

### NFR-DICT-R1: Corrupt Asset Detection
- `GADDAG.Load` must return a descriptive `error` (not panic) if gob decoding fails for any reason.
- `Loader.Load` must propagate this error to the caller; it must not return a partially initialised `*Dictionary`.
- Rationale: embedded assets can theoretically be corrupted by binary patching; the game must fail safely with a clear error message (SECURITY-15).

### NFR-DICT-R2: Build Determinism
- Two runs of the build tool on the same input word lists must produce byte-identical `.gob` files.
- Ensures reproducible builds (SECURITY-13 artifact integrity).

---

## Thread Safety

### NFR-DICT-T1: Concurrent Read Safety
- All methods on `*GADDAG` and `*Dictionary` are safe to call concurrently from multiple goroutines after `Load` returns.
- No internal synchronisation is required (achieved by making the struct read-only post-construction).
- The AI goroutine and the rules engine goroutine both read the dictionary concurrently; no locking overhead permitted on the hot path.

---

## Testability

### NFR-DICT-TS1: Independent Testability
- The `dictionary` package must be fully testable with `go test ./dictionary/...` without Ebitengine, engine, ai, or ui dependencies.
- All PBT properties identified in the functional design must have corresponding tests using the `rapid` framework.

### NFR-DICT-TS2: Build Tool Testability
- `tools/buildgaddag` must have its own unit tests covering: single-word insertion, multi-word deduplication, non-A-Z skip-with-warning, length-boundary rejection.

---

## Security

### SECURITY-10 (Supply Chain)
- `go.sum` must be committed and pin all transitive dependencies.
- The `rapid` test dependency is the only non-stdlib dependency for this unit; it must be sourced from its official Go module proxy path.

### SECURITY-15 (Exception Handling)
- `GADDAG.Load` and `Loader.Load` must wrap all errors with context using `fmt.Errorf("dictionary: %w", err)`.
- No `panic` in any exported function under any input condition.
- The build tool must exit with a non-zero status code on any error.

### SECURITY-03 (Logging) — N/A for runtime package
- The `dictionary` package does not perform logging (it is a pure library).
- The build tool (`tools/buildgaddag`) writes warnings to `stderr` per BR-04; this is acceptable for a CLI tool.

### SECURITY-01, 02, 04, 05, 06, 07, 08, 11, 12, 13, 14 — N/A
- This unit has no network I/O, no authentication, no user input (runtime), no web endpoints, no database, and no infrastructure resources.

---

## Maintainability / NFR-10

### NFR-DICT-M1: Code Commentary
- Every exported type, function, constant, and method must have a GoDoc comment.
- `gaddag.go` and `loader.go` must include inline comments referencing the Appel-Jacobson paper by section number where the algorithm is implemented.
- The `buildgaddag` tool must document the GADDAG string-insertion algorithm step-by-step in comments.
- No magic numbers; all constants (arc-separator byte, min/max word length, root NodeID) must be named.
