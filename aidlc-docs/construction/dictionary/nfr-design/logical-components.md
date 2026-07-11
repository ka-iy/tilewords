# Logical Components — Unit 1: `dictionary`

## Overview

The `dictionary` unit is a pure in-memory library with no network, database, queue, or cache infrastructure. Its logical components are limited to the build-time pipeline and the runtime asset loader.

---

## Component 1: Build-Time GADDAG Compiler (`tools/buildgaddag`)

**Type**: CLI tool (offline, developer-run)
**Trigger**: `go generate ./dictionary/...`
**Inputs**: Raw word list `.txt` files (developer-supplied, not committed)
**Outputs**: `assets/dictionaries/*.gob` files (committed)

```
[Word list .txt files]
        |
        v
  [Input Parser]         -- reads lines, normalises to uppercase, filters
        |
        v
  [Deduplicator]         -- sort + adjacent-dedup (for combined "all" dict)
        |
        v
  [GADDAG Constructor]   -- Appel-Jacobson §3-4 string insertion
        |
        v
  [gob Encoder]          -- encoding/gob serialisation to []byte
        |
        v
  [File Writer]          -- atomic write to assets/dictionaries/{name}.gob
```

**Error handling**: any stage failure writes to stderr and exits non-zero. No partial `.gob` files are written (write to temp path, rename on success).

---

## Component 2: Runtime Asset Loader (`dictionary.Loader`)

**Type**: In-process function call
**Trigger**: Called once at game startup by `ui.SetupScreen` after the user selects a dictionary
**Inputs**: `DictName` constant → maps to an embedded `//go:embed` byte slice
**Outputs**: `*Dictionary` (immutable after return)

```
[DictName constant]
        |
        v
  [Asset Selector]       -- maps DictName to embedded []byte (compile-time constant)
        |
        v
  [gob Decoder]          -- GADDAG.Load(bytes) → *GADDAG
        |
        v
  [Dictionary Wrapper]   -- Dictionary{name, gaddag, wordCount}
        |
        v
  [*Dictionary]          -- read-only; passed to engine.Rules and ai.Generator
```

**No caching**: only one dictionary is active per game session; the `*Dictionary` pointer is held for the session lifetime. No LRU or lazy loading needed.

---

## Component 3: GADDAG Query Engine (runtime, in `dictionary` package)

**Type**: In-process method calls on `*GADDAG`
**Consumers**: `engine.Rules.ValidatePlacement` (word validation) and `ai.Generator.GenerateMoves` (left-extension traversal)

```
Validation path:
  [word string] → Validate() → [3-tier fast-path] → GADDAG traversal → bool

AI traversal path:
  [node NodeID] → Successor(letter) → [map lookup] → (NodeID, bool)
  [node NodeID] → IsTerminal()      → [map lookup] → bool
```

**No caching layer**: the GADDAG itself is the optimal data structure; adding an LRU cache would increase memory and introduce synchronisation overhead without benefit.

---

## Component 4: PBT Test Harness (test-only)

**Type**: Test infrastructure (`*_test.go` files only)
**Framework**: `pgregory.net/rapid`

```
[rapid.T] → WordFromDictGenerator    → valid words for oracle tests
          → RandomAlphaGenerator     → arbitrary A-Z strings for invariant tests
          → InvalidStringGenerator   → non-A-Z strings for rejection tests
                    |
                    v
            [Property assertions]
              Contains(valid word) == true         (oracle)
              Contains(after round-trip) == before (round-trip)
              Contains(invalid string) == false    (invariant)
              wordCount(combined) ≤ Σ wordCount(i) (dedup invariant)
```

---

## Infrastructure Components: None

| Infrastructure Type | Present | Rationale |
|---|---|---|
| Network / HTTP | No | Pure local library |
| Database / persistent store | No | Assets embedded in binary |
| Message queue | No | Synchronous library calls |
| Cache (LRU, Redis, etc.) | No | GADDAG is O(1)/O(n); caching adds overhead |
| Circuit breaker | No | No external dependencies at runtime |
| Rate limiter | No | Internal library; no external callers |
| Load balancer | No | Single-process application |
| CDN / object storage | No | Assets embedded; no runtime download |
