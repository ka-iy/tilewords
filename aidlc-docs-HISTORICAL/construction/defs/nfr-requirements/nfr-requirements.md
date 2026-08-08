# NFR Requirements — Unit 6: `defs`

## Performance

### NFR-DEFS-P1: Lookup Latency
- `DB.Lookup` must be O(1) for an exact match (single map lookup) and bounded for the
  fallback layers (a small, fixed number of candidate strings per word).
- Target: sub-microsecond for exact/form-of; a played word triggers at most one lookup.

### NFR-DEFS-P2: Asset Decode
- The DB is decoded once, lazily, off the UI goroutine, and cached (`sync.Once`). Later
  `Load` calls return the same instance without re-decoding.
- Decode of the ~7.8 MB gzipped asset must complete within a second or so on target
  hardware; it must not block the UI goroutine.

---

## Memory

### NFR-DEFS-M1: Retained Heap
- The decoded DB retains roughly **52 MB** of heap (≈145k headwords + ≈124k edges + glosses).
- This is the dominant data-structure cost at runtime (the largest GADDAG is ~8 MB); it is
  a tracked figure, measurable via `tools/memcheck`. If it becomes a pressure point on
  low-end devices, the follow-up options are capping senses and interning part-of-speech
  strings, or lazy-loading on first lookup.

---

## Binary / Asset Size

### NFR-DEFS-B1: Embedded Asset Budget
- The shipped asset is a gzip-compressed gob, ~**7.8 MB** (down from ~20 MB raw) — a ~61%
  reduction — filtered to only the words the bundled lists can form.

---

## Reliability

### NFR-DEFS-R1: Optional Asset / Graceful Absence
- The package must build and the game must run whether or not the asset is present
  (`Available()` gates the feature). A missing or malformed asset must never panic; `Decode`
  returns a wrapped error, and `Load` surfaces it once (cached).

### NFR-DEFS-R2: Build Determinism
- Two runs of `builddefs` on the same extract + word lists must produce equivalent
  definitions (BR-D06), for reproducible assets.

---

## Thread Safety

### NFR-DEFS-T1: Concurrent Lookup
- A decoded `DB` is immutable; `Lookup` is safe for concurrent use with no locking.
- The UI definitions worker reads the DB on a background goroutine and marshals appends back
  to the UI goroutine; sends to it never block the UI (bounded, non-blocking channel).

---

## Testability

### NFR-DEFS-TS1: Headless Unit Tests
- `go test ./defs/...` must pass with no external data: the matcher, variant, stem, encode/
  decode, and lemma-merge logic are tested on small in-memory DBs built with `NewDB`.
- The embedded-asset load path is exercised only when the asset is present and is skipped
  otherwise.

---

## Security

### SECURITY-15 (Exception Handling)
- `Encode`/`Decode`/`Load` wrap all errors with `defs.` context; no exported function panics.

### SECURITY-10 (Supply Chain)
- Runtime dependencies are stdlib only (`encoding/gob`, `compress/gzip`, `encoding/json`,
  `embed`). The `builddefs`/`defslookup`/`memcheck` tools are developer-run and add no
  runtime dependency.

### SECURITY-03 (Logging)
- The runtime package does not log. The build/inspection tools write to stdout/stderr, which
  is acceptable for CLI tools.

### SECURITY-01, 02, 04–08, 11–14 — N/A
- No network I/O, authentication, user-supplied input at runtime, web endpoints, database,
  or infrastructure.

---

## Maintainability / NFR-10

### NFR-DEFS-C1: Code Commentary
- Every exported type, function, and method has a GoDoc comment.
- The layered resolution order, the rejection of blind edit-distance matching, and the
  build's determinism seam are documented in comments.
