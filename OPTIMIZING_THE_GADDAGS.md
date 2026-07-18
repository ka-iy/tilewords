# Optimizing GADDAG Dictionaries for Memory-Constrained Devices

*A measurement-driven study of five strategies for shrinking a Scrabble move-generation
automaton, and why automaton minimization subsumes cross-dictionary deduplication.*

**Project:** TileWords (Go + Fyne Scrabble)
**Status of measurements:** taken on Go 1.26.4, linux/amd64; figures are indicative of
this hardware and toolchain. The Android failure is from a `fyi.tilewords.game` low-memory
kill on an emulator.
**Implementation status:** Strategies I (CSR), II (load cache), and **V (minimization)**
are implemented in the codebase; III (deduplication) and IV (runtime merge) were
investigated and rejected. The figures for V in §8–§9 are the measured results *after*
implementation, not projections.

---

## Abstract

TileWords embeds four large word-list dictionaries as GADDAG automata for its AI move
generator. On Android, loading a second dictionary (the "load saved game" flow) crashed
the app: the process was terminated by the low-memory killer at ~2.4 GB resident. We
trace the failure to two independent causes — a memory-profligate in-memory representation
(nested hash maps) and the absence of a load cache — and then investigate three further
strategies to reduce dictionary footprint: cross-dictionary word-set deduplication,
runtime composition of a shared "common" automaton with per-dictionary "unique"
automata, and classical acyclic-automaton minimization.

Our central findings are:

1. Switching the in-memory representation from nested hash maps to a compressed
   sparse-row (CSR) layout reduces resident memory ~12× (843 MB → 68 MB for the largest
   dictionary) with no algorithmic change.
2. The dictionaries share 61.4% of their words, and deduplicating them into a common
   automaton plus per-dictionary remainders shrinks the embedded assets 56% — but is
   *runtime-memory-neutral*, because only one dictionary is resident at a time.
3. Composing the common and unique automata into a single graph at load time is
   **provably correct and reproduces the monolithic automaton exactly**, but costs a
   multi-second merge and a ~0.96 GB transient allocation — reintroducing the very memory
   pressure we set out to remove.
4. The reason the composition reproduces the monolithic graph *exactly* is that our
   automata are **un-minimized prefix tries**. Classical minimization (suffix sharing)
   shrinks each dictionary ~10× in both disk and RAM, requires no change to the traversal
   API or the AI, and adds no load-time cost.

The practical conclusion is that **automaton minimization strictly dominates
cross-dictionary deduplication** for this workload: set-level word overlap is a special
case of the sub-automaton redundancy that minimization eliminates globally. Combined with
CSR, minimization takes the largest dictionary from 843 MB to ~8 MB resident — a ~105×
reduction end-to-end.

---

## 1. Introduction and motivation

The GADDAG is the standard automaton for Scrabble move generation: it encodes, for every
dictionary word, every way of reading that word outward from an anchor square, enabling a
single left-to-right graph walk to enumerate all legal plays through a square
[Gordon 1994]. TileWords ships four dictionaries:

| Dictionary | Words (2–15, A–Z) | Character |
| --- | ---: | --- |
| `enable` | 168,551 | public-domain baseline |
| `pigpods` | 267,752 | largest; the outlier in overlap |
| `twirl06` | 178,691 | North-American tournament |
| `wordnik` | 194,152 | crowd-sourced |

Each is compiled offline into a `.gob`-serialized GADDAG and embedded in the binary via
`//go:embed`. The AI move generator (`ai/generate.go`, `ai/traverse.go`) consumes a
dictionary purely through three methods:

```
Root() NodeID
Successor(node NodeID, letter byte) (NodeID, bool)
IsTerminal(node NodeID) bool
```

Cross-word validation (`ai/crosscheck.go`) and candidate validation
(`engine.ValidatePlacement`) go through `Dictionary.Validate`, which is itself a GADDAG
walk. This narrow interface is important: it means the *representation* of the automaton
can change freely as long as those three methods are preserved.

### 1.1 The precipitating failure

On Android, the sequence *new game → play → save → main menu → load saved game* crashed.
The `logcat` evidence is unambiguous — this is not a Go panic but an OS kill:

```
lowmemorykiller: Kill 'fyi.tilewords.game' (…) to free 2459784kB anon rss …
                 reason: min watermark is breached even after kill
```

Loading the saved game re-decodes the dictionary. With no cache and a
memory-heavy representation, a *second* live copy is briefly held alongside the first,
and the peak exceeds the device watermark. The ~2.4 GB anon RSS in the kill log matches
two live copies of the largest dictionary under the original representation (see §4).

This report documents the investigation that followed.

---

## 2. Background

### 2.1 GADDAG structure

A GADDAG node has labelled out-edges (letters `A`–`Z` plus the arc separator `+`) and a
terminal flag. For a word `w₁…wₙ` the builder inserts, for each split point `k`, the
string `wₖwₖ₋₁…w₁ + wₖ₊₁…wₙ`; the reversed prefix lets the move generator start at any
anchor and extend leftward, then cross `+` and extend rightward [Gordon 1994].

### 2.2 Our construction is a trie, not a minimized DAWG

TileWords's builder (`dictionary.Build`) inserts every GADDAG string into a graph, creating
a node whenever an edge is absent. This shares **prefixes** — two strings with a common
leading sequence reuse nodes — but it never shares **suffixes**. The result is a *trie*
(prefix tree), not the minimized directed acyclic word graph (DAWG) the literature assumes.

The signature of an un-minimized trie is visible in the raw counts: the edge/node ratio is
essentially 1.0 (every node has, on average, a single out-edge), i.e. the graph is
dominated by long non-branching chains — exactly the suffix chains that minimization would
collapse:

| Dictionary | Trie nodes | Trie edges | Edges/node |
| --- | ---: | ---: | ---: |
| `enable` | 4,003,703 | 4,003,701 | 1.00 |
| `pigpods` | 6,419,513 | 6,419,511 | 1.00 |
| `twirl06` | 4,202,930 | 4,202,928 | 1.00 |
| `wordnik` | 4,441,311 | 4,441,309 | 1.00 |

This redundancy is the target of minimization (§8).

### 2.3 Serialization and the mobile constraint

Assets are embedded in the APK, so their on-disk size is added to every install; and they
are decoded into resident memory at runtime on a device that may kill the process above a
watermark of a couple of GB. The two costs — **embedded size** and **resident memory** —
are distinct, and the strategies below trade against them differently.

---

## 3. Method

All figures are measured, not estimated, except where explicitly labelled *(structural
estimate)* or *(projected)*. Resident memory is Go `runtime.MemStats` `HeapAlloc` after a
forced GC; peak/transient figures are `HeapSys`. Structural size estimates use the CSR
byte layout of §4: `(nodes+1)·4 + edges·5 + nodes/8` bytes. Correctness is checked by
replaying every word in a source list and comparing acceptance against the monolithic
automaton, plus negative controls.

---

## 4. Optimization I — representation: nested maps → CSR *(implemented)*

### 4.1 The original representation

The GADDAG was stored as:

```go
edges     map[NodeID]map[byte]NodeID
terminals map[NodeID]bool
```

A map-of-maps carries enormous per-entry overhead: every node allocates an inner map with
its own bucket array, header, and load-factor slack. With ~6.4M nodes each holding a tiny
inner map, the overhead dwarfs the ~5 bytes of actual edge data per edge.

### 4.2 The CSR layout

We replace the maps with a compressed sparse-row adjacency, the standard representation
for sparse graphs:

```go
edgeOffsets []uint32  // node id's edges are the range [edgeOffsets[id], edgeOffsets[id+1])
edgeLetters []byte    // edge labels, grouped by source node, ascending within a node
edgeTargets []NodeID  // parallel to edgeLetters: destination node of each edge
terminal    []uint64  // bitset: node id terminal iff bit (id&63) of terminal[id>>6] is set
```

`Successor` becomes a binary search over a node's sorted edge range (≤ 27 entries);
`IsTerminal` becomes a bitset probe. The gob wire format stores these flat slices directly,
so decoding hands its slices straight into the live structure with no post-decode copy —
minimizing peak memory during load. The graph is validated on load (offsets monotonic and
in range, parallel arrays equal length) so a corrupt asset errors rather than indexing out
of bounds during a traversal.

### 4.3 Results

For `pigpods` (the largest):

| Metric | Nested maps | CSR |
| --- | ---: | ---: |
| Resident (1 copy, after GC) | 843 MB | **68 MB** |
| Peak during decode (`HeapSys`) | 1231 MB | 315 MB |
| Two copies resident (the crash) | 1875 MB heap / 2627 MB sys | ~136 MB |
| On-disk `.gob` | 76.2 MB | 58.6 MB |

This ~12× resident reduction is a pure representation change; the automaton, the API, the
AI, and word acceptance are all identical. It **independently eliminates the OOM**: even an
uncached double-load is now ~136 MB, far below any device watermark.

---

## 5. Optimization II — load-time caching *(implemented)*

`Dictionary.Load` re-decoded from scratch on every call. Because the game plays one
dictionary at a time, a single-slot cache — keyed on dictionary name, dropping the previous
entry when a different dictionary is requested — makes re-loading the same dictionary
(exactly the save/load flow that crashed) a no-op that returns the existing instance. The
`Dictionary` is immutable after load and documented safe for concurrent use, so sharing the
pointer is sound. This bounds resident memory to a single dictionary and removes the second
decode entirely.

CSR (§4) and caching (§5) together fully resolve the reported crash. Everything that
follows targets the *remaining* cost: ~174 MB of embedded assets and ~68 MB resident.

---

## 6. Optimization III — cross-dictionary deduplication *(investigated)*

### 6.1 The overlap is large

The four lists overlap heavily. Of a 273,534-word union, **167,938 words (61.4%) are
common to all four**:

| Dictionary | Unique after extracting common-to-all |
| --- | ---: |
| `enable` | 613 |
| `pigpods` | 99,814 |
| `twirl06` | 10,753 |
| `wordnik` | 26,214 |

Pairwise containment (row ∩ col as a fraction of the row) shows `enable` is nearly a pure
subset of the others, `pigpods` the outlier:

| ∩ / row | enable | pigpods | twirl06 | wordnik |
| --- | ---: | ---: | ---: | ---: |
| **enable** | 100.0 | 99.8 | 99.6 | 100.0 |
| **pigpods** | 62.8 | 100.0 | 66.7 | 70.4 |
| **twirl06** | 94.0 | 100.0 | 100.0 | 100.0 |
| **wordnik** | 86.8 | 97.1 | 92.0 | 100.0 |

Storing the common words once plus per-dictionary remainders stores 305,332 word-entries
instead of 809,146 — a 62% reduction in stored words.

### 6.2 Disk shrinks, runtime does not

We built a `common.gob` (167,938 words) and four `<dict>_unique.gob` files and measured:

| | Embedded size |
| --- | ---: |
| Monolithic ×4 (CSR) | 173.9 MB |
| Split (common + 4 uniques) | **76.0 MB (−56%, −97.9 MB)** |

The 62% reduction in stored words more than offsets the prefix-sharing lost at the
common/unique boundary, so the embedded assets shrink. **Runtime memory, however, is
neutral** — loading `pigpods`
means holding `common` + `pigpods_unique` (64 MB) versus the monolithic 68 MB. With the
single-slot cache only one dictionary is ever resident, so the shared `common` is not
amortized across simultaneously-loaded dictionaries.

### 6.3 The structural catch

A split dictionary is **two automata, not one**. The AI's traversal walks a single graph by
node id, and node ids are not comparable across the two files. Supporting a split
dictionary therefore requires either (a) running move generation once per graph and
unioning candidates (correct, since each word lives wholly in one graph, and cross-checks
already route through `Validate` which can OR the two graphs — but it reworks the hot path),
or (b) composing the two graphs into one at load time (§7).

---

## 7. Optimization IV — runtime composition via product construction *(investigated)*

Merging `common` + `<dict>_unique` into a single GADDAG at load would leave the AI
untouched, since it would see one graph as before.

### 7.1 The merge

The union of two acyclic automata is a memoized product construction. `merge(c, u)`
produces one node per reachable pair `(c, u)` (either side may be absent, denoted ⊥):

```
merge(c, u):
    if memo[(c,u)] exists: return it
    n = new node
    memo[(c,u)] = n
    terminal[n] = isTerminal(c) or isTerminal(u)
    for each letter L in edges(c) ∪ edges(u):
        merge(child_c(L), child_u(L))   # ⊥ where a side lacks L
    return n
```

GADDAG strings have length ≤ 16, so recursion is shallow; memoization on `(c, u)` dedups
shared substructure. We verified correctness by replaying all 267,752 `pigpods` words and
negative controls: **0 mismatches** against the monolithic automaton.

### 7.2 The result: exact reproduction

| | Nodes | Edges |
| --- | ---: | ---: |
| `common` | 3,985,040 | 3,985,038 |
| `pigpods_unique` | 3,229,425 | 3,229,423 |
| **Merged** | **6,419,513** | **6,419,511** |
| Monolithic `pigpods` | 6,419,513 | 6,419,511 |

The merged automaton is **node-for-node identical to the monolithic one**.

### 7.3 Why: composition of tries yields the union trie

This exactness is not a coincidence; it is forced by §2.2.

> **Proposition.** Let `T_A`, `T_B` be tries — each node is reached from the root by a
> unique path, so the map *prefix → node* is a bijection onto reachable nodes. The memoized
> product construction visits exactly one node per pair `(a(p), b(p))`, where `p` ranges
> over the prefix-closure of `L(A) ∪ L(B)` and `a(p)`/`b(p)` are the (possibly ⊥) trie
> nodes for `p`. That prefix-closure is precisely the node set of the trie `T_{A∪B}`.
> Hence the product has exactly `|nodes(T_{A∪B})|` nodes and is isomorphic to `T_{A∪B}`.

*Proof sketch.* In a trie the reached node is a function of the consumed prefix `p`, so the
product state `(a(p), b(p))` is too; memoization thus allocates one node per distinct
reachable prefix. A prefix is reachable in the product iff it is a prefix of some string in
`A` or in `B`, i.e. iff it is in the prefix-closure of `L(A) ∪ L(B)` — which is the node set
of the union trie. ∎

Because the monolithic build is *also* a trie over `L(A) ∪ L(B)`, the merged graph equals
it exactly. Composing tries reconstructs the full trie and preserves all of its redundancy:
the merge produces no smaller a structure than the monolithic automaton, while adding the
cost of rebuilding it at load.

### 7.4 Cost

| Metric (merging `common` + `pigpods_unique`) | Value |
| --- | ---: |
| Merge time (desktop) | ~3.4 s |
| Transient resident around the merge (`HeapSys`) | ~0.96 GB |
| Correctness (words checked / mismatches) | 267,752 / 0 |

The ~0.96 GB transient (dominated by the memo and adjacency maps over 6.4M nodes)
reintroduces the mobile memory pressure we removed in §4, and multi-second load latency
would be worse on a phone. A map-free streaming merge could cut the transient to ~150 MB
(no memoization is even needed — §7.3 shows each product state is reached once), but the
CPU cost of materializing 6.4M nodes per load remains. The merge is correct, but it trades
embedded size for load latency and a load-time memory spike.

---

## 8. Optimization V — automaton minimization *(implemented)*

§2.2 and §7.3 point to the actual problem: the automata are un-minimized. Classical
minimization of an acyclic DFA [Revuz 1992] merges all equivalent sub-automata — nodes are
equivalent iff they have the same terminal flag and the same set of `(letter → equivalent
child)` edges. Because our builder assigns every child a higher id than its parent,
processing nodes in decreasing id order canonicalizes children before parents. A single
bottom-up *hash-consing* pass then minimizes the graph: each node is looked up in a hash
table keyed by its signature — its terminal flag and its list of `(letter, canonical-child
id)` edges — and if an equivalent node already exists it is reused instead of being
duplicated, so structurally identical sub-automata collapse to one.

### 8.1 Measurement

| Dictionary | Trie nodes | Trie ~MB* | Min nodes | Min edges | Min ~MB* | Node ratio |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `enable` | 4,003,703 | 34 | 401,729 | 836,835 | 5 | 10.0× |
| `pigpods` | 6,419,513 | 55 | 587,940 | 1,266,975 | 8 | 10.9× |
| `twirl06` | 4,202,930 | 36 | 413,548 | 873,040 | 5 | 10.2× |
| `wordnik` | 4,441,311 | 38 | 435,389 | 932,089 | 6 | 10.2× |

\* CSR structural estimate. The minimized edge/node ratio rises to ~2.1 (branching structure
survives; the non-branching suffix chains collapse).

Minimization yields a **~10× reduction in both nodes and edges** across all four
dictionaries.

### 8.2 Implementation and post-build results

Minimization is implemented in `dictionary.Build` (function `minimizeTrie`) as Revuz's
algorithm: after the trie is built, a single high-to-low id pass hash-conses each node on
the exact signature `(terminal, [(letter, canonicalChild)…])`, then a breadth-first walk
from the root renumbers the surviving nodes (root → `RootNodeID`) into the CSR arrays. The
child-id > parent-id property of the builder makes the bottom-up pass a plain descending
loop. Determinism is preserved: the BFS follows edges in ascending letter order, so map
iteration order never reaches the output. The load path, the traversal API, `Validate`, and
the AI are untouched — a minimized dictionary is simply a smaller CSR graph.

Rebuilding the four embedded `.gob` assets with the minimizing `Build` gives the measured
results below (versus the pre-minimization CSR trie):

| Dictionary | Words | On-disk (trie CSR) | On-disk (minimized) | Resident (minimized) |
| --- | ---: | ---: | ---: | ---: |
| `enable` | 168,551 | 36.5 MB | **5.52 MB** | — |
| `pigpods` | 267,752 | 58.6 MB | **8.34 MB** | **8 MB** |
| `twirl06` | 178,691 | 38.3 MB | **5.74 MB** | — |
| `wordnik` | 194,152 | 40.5 MB | **6.10 MB** | — |
| **Total** | | **173.9 MB** | **25.7 MB (−85%)** | |

The largest dictionary is now **8 MB resident** (measured `HeapAlloc` after GC), down from
68 MB (CSR trie) and 843 MB (original nested maps). Correctness was re-verified after
implementation: all 267,752 `pigpods` words are accepted (0 missing), negative controls are
rejected, the full test suite passes — including the AI move-generation tests, which confirm
the minimized DAWG is behaviourally identical for move generation — and a regression test
(`TestBuild_Minimized`) asserts the accept sinks are merged so minimization cannot be
silently disabled.

### 8.3 Why it dominates: minimization subsumes deduplication

The deduplication of §6 exploits *exact-word overlap* between dictionaries. But two words
sharing a suffix — whether they are the "same word in two dictionaries" or simply
"two words ending the same way" — induce a shared sub-automaton that minimization merges.
Cross-dictionary word overlap is therefore a **special case** of the redundancy
minimization already removes, and it removes it *within* a dictionary as well as across the
notional common/unique partition. Minimization is strictly more general.

This is borne out by the numbers: dedup removes 62% of stored words; minimization removes
~90% of nodes. And the two do not usefully stack — once `pigpods` is ~8 MB resident, a
further common/unique split saves a few MB of disk at the cost of the two-graph complexity
of §6.3–§7.

---

## 9. Results: the full comparison

| Strategy | Embedded (4 dicts) | Resident (active dict) | AI change | Load cost | Status |
| --- | ---: | ---: | --- | --- | --- |
| Original (nested maps, trie) | 226 MB† | 843 MB | — | baseline | replaced |
| **I. CSR** | 174 MB | **68 MB** | none | faster decode | **done** |
| **II. Load cache** | 174 MB | 68 MB (bounded) | none | 0 on re-load | **done** |
| III. Dedup split | 76 MB | ~same (64 MB) | rework or merge | — | rejected |
| IV. Dedup + runtime merge | 76 MB | ~same | none | +3.4 s, +~0.96 GB | rejected |
| **V. Minimization** | **25.7 MB** | **8 MB** | none | faster decode | **done** |

† original nested-map `.gob` sizes summed.

End-to-end, CSR + minimization takes the largest dictionary from **843 MB → 8 MB resident
(~105×)** and the embedded assets from **226 MB → 25.7 MB (~9×)**, with no change to the
traversal API or the AI. These are the final implemented figures.

---

## 10. Discussion

Three ideas organize the whole investigation:

- **Separate the two costs.** Embedded size and resident memory are different budgets.
  Deduplication (§6) helps the first and not the second; representation (§4) and
  minimization (§8) help both. Conflating them makes the dedup approach look better than it
  is for a device that holds one dictionary at a time.

- **Representation gates whether a structural win is realized.** The ~10× node reduction of
  minimization is only worth ~10× in RAM because CSR's per-node overhead is a few bytes;
  under the original nested maps, per-node overhead would have swamped the node-count
  saving. CSR and minimization are complementary: one removes per-node overhead, the other
  removes nodes.

- **Minimization subsumes set-level deduplication.** Storing shared words once (§6) is a
  coarse, manual approximation of what suffix-minimization does automatically and more
  completely. The product-construction result of §7 is the formal reason the manual scheme
  cannot beat it: composing the pieces reconstructs the full redundant trie.

Stated generally: *for a family of dictionaries compiled to acyclic word automata,
cross-dictionary word-set deduplication is dominated by per-dictionary automaton
minimization, because inter-dictionary word overlap is a strict subset of the intra- and
inter-word suffix redundancy that minimization eliminates.* On the TileWords corpus, despite
61% word overlap, minimization yields a larger reduction than exploiting that overlap
directly.

---

## 11. Recommendation and outcome

The recommendation was to implement **minimization inside `dictionary.Build`** (Revuz
bottom-up hash-consing, using the existing child-id > parent-id ordering) and regenerate the
four `.gob` assets, and **not** to pursue the cross-dictionary split (runtime-neutral,
complicates the AI) or the runtime merge (reintroduces a load-time memory spike), since
minimization dominates both.

**This has been done.** Minimization is implemented in `minimizeTrie` and the assets are
regenerated (§8.2). The change was contained and low-risk as predicted:

- The load path, the `Root`/`Successor`/`IsTerminal` API, `Validate`, and the entire AI are
  unchanged — a minimized automaton is simply a smaller CSR graph.
- No runtime merge, no two-graph bookkeeping, no load-time spike; decode is faster.
- Correctness is confirmed by full-corpus acceptance parity, the property-based tests, the
  AI move-generation tests, and a minimization regression guard (§8.2).

Result: embedded assets 174 MB → 25.7 MB, and the largest dictionary 68 MB → 8 MB resident.

---

## 12. Reproducibility

All measurements come from throwaway in-package Go tests (the `dictionary` package, so they
can read the unexported CSR fields) plus a Python overlap script over `wordlists/*.txt`:

- **Overlap (§6.1):** intersect the four word sets filtered to valid 2–15 A–Z words.
- **Split sizes (§6.2):** emit `common.txt` and `<dict>_unique.txt`, compile each with
  `tools/buildgaddag`, and sum file sizes.
- **Resident memory (§4, §6.2):** `dictionary.Load`, force GC, read `HeapAlloc`.
- **Merge (§7):** memoized product construction over two loaded graphs; replay all words
  for correctness; time the merge and read `HeapSys`.
- **Minimization (§8):** now production code in `dictionary.Build` (`minimizeTrie`); the
  node/edge counts came from a throwaway hash-cons over a loaded graph, and the on-disk and
  resident figures from rebuilding the assets (`make gaddag`) and loading them.

Except for `minimizeTrie` (committed), these harnesses are not committed (they touch
unexported internals and load multi-hundred-MB assets); they are described here so the
numbers can be regenerated on demand.

---

## 13. Related work: the Kurnia Word Graph (KWG)

The Kurnia Word Graph (KWG), used by Andy Kurnia's *wolges* engine (and by MAGPIE and the
Woogles stack), is a minimized acyclic GADDAG stored in a flat array — the same structure as
Strategy V, with a denser encoding. This section compares the two.

### 13.1 What KWG is

Per the wolges `details.txt` [wolges], a KWG is "a flat array of nodes `(tile, accepts,
is_end, arc_index)`. Each entry is 32-bit. tile is 8 bits (subject to change). accepts and
is_end are 1 bit each" — leaving 22 bits for `arc_index`. Each 32-bit entry is an edge, not
a node: a state is a contiguous run of sibling entries with strictly increasing `tile`, and
"it is guaranteed that the last node always has `is_end=true`." Following a letter means
scanning that sorted run; `arc_index` then points to the first entry of the child state;
`accepts` marks that the string read so far is a word. This is the packed-arc DAWG cell (one
letter, flags, and child pointer per machine word) applied to a GADDAG.

Two further properties are relevant here:

- **Minimization with tail-sharing.** "It is guaranteed that this graph is acyclic and
  there are no redundant nodes," and, beyond ordinary node minimization, "different from
  other implementations of similar structures, it is allowed to share the tail end of a
  node." Because a state is a sorted sibling run terminated by `is_end`, two states whose
  edge-lists share a common suffix can share the same physical tail entries.
- **Unified DAWG + GADDAG ("Gaddawg").** "A GADDAG necessarily contains all but the root
  node of the DAWG … This makes including the DAWG a negligible cost: it's just one
  additional root node." KWG stores both in one graph and is "about 33% smaller than typical
  GADDAG files."

### 13.2 Mapping onto Strategy V

The two share the same structure — a minimized, acyclic, sorted-edge GADDAG in flat arrays —
and differ in encoding:

| Aspect | Strategy V (CSR) | KWG |
| --- | --- | --- |
| Unit of the array | node (via `edgeOffsets`) + parallel edge arrays | edge (one 32-bit cell each) |
| Letter | `edgeLetters[i]` (1 byte) | `tile` field (8 bits) in the cell |
| Child pointer | `edgeTargets[i]` (4-byte node id) → `edgeOffsets` | `arc_index` (22 bits) → first child cell |
| Node's edge range | explicit `edgeOffsets[n]…[n+1]` | implicit: sibling run to `is_end` |
| Terminal flag | separate `terminal` bitset | `accepts` bit in the cell |
| Minimization | whole-node (Revuz hash-consing) | whole-node plus edge-list tail-sharing |
| Also stores DAWG | no (GADDAG-only; validation walks the full-reverse path) | yes, at ~1 extra node |
| Serialization | gob of flat slices; heap-decoded | raw little-endian array; memory-mappable |
| Pointer width | 32-bit node id (≤ ~4.29 B nodes) | 22-bit arc_index (≤ ~4.19 M cells) |

KWG fuses the node and edge into a single 32-bit word and folds the terminal flag into it,
where the CSR keeps three parallel arrays, an offsets table, and a bitset. The CSR spends
1 byte (letter) + 4 bytes (target) = 5 bytes per edge, plus 4 bytes per node for offsets,
plus the bitset — about 6.9 bytes per transition. KWG spends 4 bytes per transition and
nothing else.

### 13.3 Quantitative comparison

Applying KWG's 4-bytes-per-transition encoding to the measured minimized graphs (an upper
bound for KWG, since its tail-sharing reduces the cell count below the edge count):

| Dict | Nodes | Edges | Strategy V (CSR, in-mem) | KWG (≤ 4 B/edge) | Ratio |
| --- | ---: | ---: | ---: | ---: | ---: |
| `enable` | 401,729 | 836,835 | 5.84 MB | 3.35 MB | 1.75× |
| `pigpods` | 587,940 | 1,266,975 | 8.76 MB | 5.07 MB | 1.73× |
| `twirl06` | 413,548 | 873,040 | 6.07 MB | 3.49 MB | 1.74× |
| `wordnik` | 435,389 | 932,089 | 6.46 MB | 3.73 MB | 1.73× |
| **Total** | | | **27.1 MB** | **≤ 15.6 MB** | **1.74×** |

A KWG of the same dictionaries is ~1.7× smaller than the CSR, both on disk and resident, and
smaller still once tail-sharing and DAWG-node reuse are counted. The 22-bit-vs-32-bit
pointer difference is not the driver — the largest graph, 1.27M edges, fits in 22 bits — the
driver is field fusion and tail-sharing.

### 13.4 Assessment

- Strategy V accounts for the reduction from an un-minimized trie in hash maps (843 MB) to a
  minimized automaton in flat arrays (8 MB), ~105×. KWG's denser layout yields a further
  ~1.7× (§13.3). That factor does not change the outcome for TileWords: the OOM is resolved
  under either encoding.
- A KWG is a raw little-endian array and can be `mmap`-ed directly from an uncompressed
  asset: near-zero heap, no decode step, pages faulted in on demand. The gob format decodes
  into ~8 MB of heap-allocated slices at load. On a memory-constrained device an mmap-ed KWG
  uses less resident memory and loads without a decode pass — a larger difference than the
  1.7× byte reduction.
- KWG's tail-sharing is a stronger minimization: Strategy V merges whole equivalent nodes
  (Revuz), while KWG additionally shares suffixes of sibling edge-lists. Expressing it
  requires indexing children by edge position rather than by node id.

Closing the gap would require switching to a packed 32-bit-per-edge cell array (letter +
accepts + is_end + arc_index), indexing children by cell position, and `mmap`-ing it, which
would take TileWords from ~8 MB heap to ~5 MB or less mapped with near-zero heap. The Gaddawg
DAWG-union does not apply to TileWords, which never anagrams; only the KWG encoding, not its
dual-graph role, would be relevant.

### 13.5 When CSR is preferable

For move generation in isolation, KWG is smaller, memory-mappable, and no worse on the
traversal hot path, so the CSR layout's advantages are conditional on requirements the KWG
encoding trades away. Most of them stem from one difference: the CSR keeps a separate
`edgeOffsets` table — the source of its ~1.7× size disadvantage — and that table gives every
node a dense, stable integer id in `[0, nodeCount)`. KWG has no node table: a state is a
start position in the shared edge array, and tail-sharing makes states physically overlap,
so there is no dense node-to-index mapping.

- **Graphs larger than ~4.19M edges.** The documented 32-bit KWG cell is
  `tile:8 + accepts:1 + is_end:1 + arc_index:22`, so the whole graph must fit in 2²² =
  4,194,304 cells. pigpods is 1.27M edges, but a larger or merged lexicon can exceed that, at
  which point the standard KWG needs a wider cell variant. The CSR's 32-bit node ids and edge
  offsets address ~4.29B nodes/edges without a format change.

- **Dense per-node auxiliary data.** Decorating nodes — precomputed cross-set masks, subtree
  word counts, weights, probabilities, or a memoization table — maps directly onto the dense
  ids as a parallel `aux[nodeID]` array. KWG's overlapping states have no dense node
  numbering, so such data must live in a separate structure; this is why Kurnia Leave Values
  (KLV) store leaves as a separate KWG plus a float array rather than as per-node fields.

- **Extending the encoding.** The CSR is a *struct-of-arrays* (SoA) layout — separate arrays
  for letters, targets, and terminal bits. A field can be widened (e.g. 64-bit targets) or a
  new per-node/per-edge array added without disturbing the others or renegotiating a bit
  budget. KWG's fused 32-bit cell is a fixed budget; any addition is a new cell format.

- **Build and verification simplicity.** Whole-node hash-consing (§8) is a short pass that is
  simple to test. KWG's tail-sharing is a stronger but more intricate construction, so the
  simpler build is easier to verify.

- **Format evolution.** gob is self-describing (field names and types), so adding a field to
  `gaddagData` is backward- and forward-tolerant. A raw little-endian array has no such
  tolerance — any layout change is a hard version break, and the format is endianness- and
  alignment-dependent. (The reverse also holds: gob is Go-only, while a raw KWG array is
  readable from any language.)

- **Bulk field sweeps.** The SoA layout keeps all targets (or all letters, or the terminal
  bitset) contiguous, which suits a vectorized scan over one field — analytics or a bulk
  validation sweep. This does not help traversal: there KWG's *array-of-structs* (AoS)
  layout, with each cell holding a letter and its child pointer together, has better cache
  locality for the follow-edge step.

- **This project.** Strategy V is implemented, tested, and brings pigpods to 8 MB, below any
  relevant watermark. KWG's byte and mmap gains do not change that outcome, so for TileWords
  the operative advantage is an existing, verified implementation with no migration.

Where KWG wins and the CSR cannot follow without becoming KWG: raw byte size (field fusion
plus tail-sharing) and memory-mappability. Those matter most for move generation, which is
why KWG is the better format for that task in isolation.

---

## References

- S. A. Gordon. *A Faster Scrabble Move Generation Algorithm.* Software: Practice and
  Experience, 24(2):219–232, 1994. (The GADDAG.)
- A. W. Appel and G. J. Jacobson. *The World's Fastest Scrabble Program.* Communications of
  the ACM, 31(5):572–578, 1988. (The DAWG-based predecessor.)
- D. Revuz. *Minimisation of Acyclic Deterministic Automata in Linear Time.* Theoretical
  Computer Science, 92(1):181–189, 1992.
- J. Daciuk, S. Mihov, B. W. Watson, R. E. Watson. *Incremental Construction of Minimal
  Acyclic Finite-State Automata.* Computational Linguistics, 26(1):3–16, 2000.
- Compressed sparse row (CSR): standard sparse-matrix/graph adjacency representation.
- [wolges] A. Kurnia. *wolges* — Scrabble engine and the Kurnia Word Graph (KWG) format.
  Format specification: <https://github.com/andy-k/wolges/blob/main/details.txt>. Repository:
  <https://github.com/andy-k/wolges>.

---

*This document records the state of the investigation. Strategies I, II, and V are
implemented in the codebase; III and IV were investigated and rejected. The §8–§9 figures
for V are measured after implementation.*
