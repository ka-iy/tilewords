# NFR Design Patterns — Unit 1: `dictionary`

## Pattern 1: Fast-Path Validation (Performance — NFR-DICT-P2)

**Problem**: `Dictionary.Validate` is called thousands of times per AI move generation cycle. Allocation and unnecessary GADDAG traversal must be avoided.

**Solution**: Three-tier fast rejection before GADDAG access.

```
Validate(word string) bool:
  Tier 1 — Length check (no allocation):
    if len(word) < MinWordLen || len(word) > MaxWordLen → return false

  Tier 2 — Case normalisation + byte validity (single pass, no alloc if already upper):
    upper := toUpperInPlace(word)   // returns word unchanged if already uppercase
    for each byte b in upper:
      if b < 'A' || b > 'Z' → return false

  Tier 3 — GADDAG traversal (full-reverse path):
    node := g.root
    for i := len(upper)-1; i >= 0; i--:
      node, ok = g.edges[node][upper[i]]
      if !ok → return false
    return g.terminals[node]
```

**Allocation-free guarantee**: `toUpperInPlace` returns the original string if it contains no lowercase bytes (detected via a pre-scan). Only allocates a new `[]byte` + conversion if lowercase bytes are present (rare in the AI hot path, which operates on uppercase rack letters).

---

## Pattern 2: Immutable Post-Construction (Thread Safety — NFR-DICT-T1)

**Problem**: The AI goroutine and the rules engine both read the dictionary concurrently.

**Solution**: Constructor-enforced immutability. The `GADDAG` struct exposes no mutation methods. After `Load` returns, the struct is treated as deeply immutable:

```
// GADDAG is read-only after Load returns.
// All exported methods are safe for concurrent use without additional locking.
type GADDAG struct {
    edges     map[NodeID]map[byte]NodeID  // written only during Load
    terminals map[NodeID]bool             // written only during Load
    root      NodeID
}
```

No `sync.RWMutex` is needed; Go's memory model guarantees that a value fully written before a goroutine is started is safely readable by that goroutine without synchronisation.

**Enforcement in code review**: any PR adding a write to `edges` or `terminals` outside of `Load` is a blocking defect.

---

## Pattern 3: Contextual Error Wrapping (Reliability — SECURITY-15)

**Problem**: Errors from gob decoding are opaque without context.

**Solution**: All error returns are wrapped with the originating function and package:

```go
// Load wraps all errors so callers can identify the source.
func Load(data []byte) (*GADDAG, error) {
    ...
    if err := dec.Decode(&g); err != nil {
        return nil, fmt.Errorf("dictionary.Load: gob decode failed: %w", err)
    }
    if g.root != rootNodeID {
        return nil, fmt.Errorf("dictionary.Load: invalid root node %d (want %d)", g.root, rootNodeID)
    }
    ...
}
```

**Rule**: No exported function returns a bare `errors.New` or an unwrapped sentinel. Every error includes the package-qualified function name as prefix and wraps the upstream error with `%w` for `errors.Is`/`errors.As` compatibility.

---

## Pattern 4: Build-Time Asset Pipeline (Binary Size — NFR-DICT-B1)

**Problem**: Raw word lists cannot be embedded directly (text format is large; runtime GADDAG construction is slow and violates NFR-DICT-P1).

**Solution**: Offline build pipeline triggered by `go generate`:

```
Source word list .txt files (not committed)
         |
         v
tools/buildgaddag/main.go
  - reads word list(s)
  - normalises, filters, deduplicates
  - constructs GADDAG (Appel-Jacobson §3-4)
  - gob-encodes to assets/dictionaries/{name}.gob
         |
         v
assets/dictionaries/*.gob  (committed to repo)
         |
    //go:embed
         |
         v
dictionary.Load(embeddedBytes) → *GADDAG  (runtime, ≤500ms)
```

**Determinism enforcement**: the build tool sorts the input word list before GADDAG construction and uses a fixed node ID allocation sequence. Same input → same output bytes → reproducible builds (SECURITY-13).

---

## Pattern 5: PBT Generator Design (PBT-07 — Generator Quality)

**Problem**: Naive random string generators produce mostly invalid inputs that trivially return `false`, providing poor coverage of GADDAG traversal paths.

**Solution**: Three purpose-built generators for `rapid`:

```
WordFromDictGenerator(dict *Dictionary):
  // Generates known-valid words by sampling from a pre-loaded reference word list.
  // Ensures Contains(word) == true tests exercise actual GADDAG paths.

RandomAlphaStringGenerator(minLen, maxLen int):
  // Generates strings of A-Z only, lengths in [minLen, maxLen].
  // Used for oracle comparison: Contains(s) must match brute-force list membership.

InvalidStringGenerator():
  // Generates strings containing at least one non-A-Z character.
  // Used to verify fast-path rejection: Contains(s) must always return false.
```

**Centralization**: all generators live in `dictionary/testhelpers_test.go` (test-only file); shared across all `_test.go` files in the package.

---

## Pattern 6: GoDoc + Algorithm Commentary (NFR-10)

**Mandatory structure for algorithm-critical files**:

```go
// Package dictionary implements the GADDAG word graph data structure
// and provides word validation for the Squabble crossword board game.
//
// The GADDAG is described in:
//   Appel, A. W. & Jacobson, G. J. (1988). "The World's Fastest Scrabble Program."
//   Communications of the ACM, 31(5), 572–578.
package dictionary

// Load deserialises a GADDAG from gob-encoded bytes produced by tools/buildgaddag.
// The bytes are typically supplied via //go:embed from assets/dictionaries/.
// Returns an error if the data is malformed or the root node is invalid.
func Load(data []byte) (*GADDAG, error) { ... }

// Successor returns the NodeID reached by following the edge labelled letter
// from node, and whether such an edge exists.
// letter must be an uppercase A-Z byte or the arc-separator ArcSep ('+').
// This method is used by the AI move generator to traverse the GADDAG during
// left-extension move enumeration (Appel-Jacobson §5, GenerateMoves algorithm).
func (g *GADDAG) Successor(node NodeID, letter byte) (NodeID, bool) { ... }
```

All named constants must appear before their first use with a comment explaining their value and source:

```go
// ArcSep is the arc-separator byte used in GADDAG strings to delimit
// the reversed prefix from the forward suffix (Appel-Jacobson §3).
const ArcSep byte = '+'

// RootNodeID is the fixed NodeID of the GADDAG root node.
// The build tool always assigns 1 to the root.
const RootNodeID NodeID = 1

// MinWordLen and MaxWordLen define valid word lengths for a 15x15 board.
const MinWordLen = 2
const MaxWordLen = 15
```
