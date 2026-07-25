# Optimizing GADDAG Dictionaries for Memory-Constrained Devices

*A measurement-driven study of six strategies for shrinking a Scrabble move-generation
automaton, and why automaton minimization subsumes cross-dictionary deduplication.*

**Project:** TileWords (Go + Fyne Scrabble)
**Status of measurements:** taken on Go 1.26.4, linux/amd64; figures are indicative of
this hardware and toolchain. The Android failure is from a `fyi.tilewords.game` low-memory
kill on an emulator.
**Word lists:** the figures below are for the three shipped
openly-licensed lists — `enable` (ENABLE2K), `wordnik`, and `atebits-letterpress`. The
investigation was originally run with a fourth (tournament) list that has since been
removed; its rows have been dropped and the per-list numbers refreshed for the current
three. Figures from superseded representations (the nested-map layout of §4 and the
un-minimized CSR trie of §6–§7) are retained as historical baselines, since that code no
longer exists to re-measure; they are for the largest list and are within the "indicative"
tolerance.
**Implementation status:** Strategies I (CSR), II (load cache), **V (minimization)** and
**VI (streamed varint serialization)** are implemented in the codebase; III (deduplication)
and IV (runtime merge) were investigated and rejected, as were two encodings within VI (raw
fixed-width binary, and gzip). The figures for V and VI in §8–§10 are the measured results
*after* implementation, not projections. **Appendix A** carries the same representation change over
to the *definitions* asset — the heap's largest consumer once the dictionaries were
minimized — and is measured on the Android emulator rather than on the desktop.

---

## Abstract

TileWords embeds large word-list dictionaries as GADDAG automata for its AI move
generator. On Android, loading a second dictionary (the "load saved game" flow) crashed
the app: the process was terminated by the low-memory killer at ~2.4 GB resident. We
trace the failure to two independent causes — a memory-heavy in-memory representation
(nested hash maps) and the absence of a load cache — and then investigate three further
strategies to reduce dictionary footprint: cross-dictionary word-set deduplication,
runtime composition of a shared "common" automaton with per-dictionary "unique"
automata, and classical acyclic-automaton minimization. A sixth strategy addresses the
serialization of the minimized graph rather than the graph itself.

Our central findings are:

1. Switching the in-memory representation from nested hash maps to a compressed
   sparse-row (CSR) layout reduces resident memory ~12× (843 MB → 68 MB for the largest
   dictionary) with no algorithmic change.
2. The dictionaries share 61.4% of their words, and deduplicating them into a common
   automaton plus per-dictionary remainders shrinks the embedded assets ~47% — but is
   *runtime-memory-neutral*, because only one dictionary is resident at a time.
3. Composing the common and unique automata into a single graph at load time is
   **provably correct and reproduces the monolithic automaton exactly**, but costs a
   multi-second merge and a ~0.96 GB transient allocation — reintroducing the memory
   pressure §4 removed.
4. The reason the composition reproduces the monolithic graph *exactly* is that our
   automata are **un-minimized prefix tries**. Classical minimization (suffix sharing)
   shrinks each dictionary ~10× in nodes and edges — ~7× on disk and ~8× resident, the
   difference being that minimized graphs carry more edges per node — requires no change to
   the traversal API or the AI, and adds no load-time cost.
5. Once the graph is minimal, the remaining cost is its *encoding*. Storing per-node edge
   counts instead of absolute offsets shrinks the assets a further 33.6% (20.05 MB →
   13.31 MB) and streaming the read instead of buffering a `gob` message removes a ~2×
   allocation churn per load, cutting process peak 55.7 MB → 39.6 MB. Resident memory is
   unchanged, because the decoded slices were already the live structure.

**Automaton minimization strictly dominates
cross-dictionary deduplication** for this workload: set-level word overlap is a special
case of the sub-automaton redundancy that minimization eliminates globally. Combined with
CSR, minimization takes the largest dictionary from 843 MB to ~8.4 MB resident — a ~100×
reduction end-to-end — and re-encoding takes the embedded assets to 13.31 MB.

---

## 1. Introduction and motivation

The GADDAG is the standard automaton for Scrabble move generation: it encodes, for every
dictionary word, every way of reading that word outward from an anchor square, enabling a
single left-to-right graph walk to enumerate all legal plays through a square
[Gordon 1994]. TileWords ships three dictionaries, all openly licensed (see the project's
Lexicon):

| Dictionary | Words (2–15, A–Z) | Character |
| --- | ---: | --- |
| `enable` | 169,266 | public-domain baseline (ENABLE2K) |
| `wordnik` | 194,152 | crowd-sourced |
| `atebits-letterpress` | 270,652 | largest; the outlier in overlap |

Each is compiled offline into a serialized GADDAG (§9) and embedded in the binary via
`//go:embed`. The AI move generator (`ai/generate.go`, `ai/traverse.go`) consumes a
dictionary only through three methods:

```
Root() NodeID
Successor(node NodeID, letter byte) (NodeID, bool)
IsTerminal(node NodeID) bool
```

Cross-word validation (`ai/crosscheck.go`) and candidate validation
(`engine.ValidatePlacement`) go through `Dictionary.Validate`, which is itself a GADDAG
walk. The narrow interface means the *representation* of the automaton
can change freely as long as those three methods are preserved.

### 1.1 The precipitating failure

On Android, the sequence *new game → play → save → main menu → load saved game* crashed.
The `logcat` line records an OS kill, not a Go panic:

```
lowmemorykiller: Kill 'fyi.tilewords.game' (…) to free 2459784kB anon rss …
                 reason: min watermark is breached even after kill
```

Loading the saved game re-decodes the dictionary. With no cache and a
memory-heavy representation, a *second* live copy is briefly held alongside the first,
and the peak exceeds the device watermark. The ~2.4 GB anon RSS in the kill log matches
two live copies of the largest dictionary under the original representation (see §4).

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
1.00 (every node has, on average, a single out-edge), so the graph is dominated by long
non-branching chains — the suffix chains minimization collapses:

| Dictionary | Trie nodes | Trie edges | Edges/node |
| --- | ---: | ---: | ---: |
| `enable` | 4,017,937 | 4,017,936 | 1.00 |
| `wordnik` | 4,441,310 | 4,441,309 | 1.00 |
| `atebits-letterpress` | 6,474,718 | 6,474,717 | 1.00 |

This redundancy is the target of minimization (§8).

### 2.3 Serialization and the mobile constraint

Assets are embedded in the APK, so their on-disk size is added to every install; and they
are decoded into resident memory at runtime on a device that may kill the process above a
watermark of a couple of GB. The two costs — **embedded size** and **resident memory** —
are distinct, and the strategies below trade against them differently.

---

## 3. Method

All figures are measured, not estimated, except where labelled *(structural
estimate)* or *(projected)*. Resident memory is Go `runtime.MemStats` `HeapAlloc` after a
forced GC; peak/transient figures are `HeapSys`. Structural size estimates use the CSR
byte layout of §4: `(nodes+1)·4 + edges·5 + nodes/8` bytes. Units follow the tool that
produced them: heap figures are MiB (2²⁰ B), as `runtime.MemStats` reports them, while
on-disk and structural-estimate figures are decimal MB (10⁶ B). The two coincide for the
largest minimized dictionary — 8.83 MB structural, 8.41 MB on disk, 8.4 MiB resident are
the same object measured three ways. Correctness is checked by
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

A map-of-maps carries per-entry overhead: every node allocates an inner map with its own
bucket array, header, and load-factor slack. The 843 MB of §4.3 spans 6.47M edges — over
100 bytes per edge, against the ~5 bytes of edge data an edge holds.

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
`IsTerminal` becomes a bitset probe. The wire format stores these flat slices directly, so
decoding hands them straight into the live structure with no post-decode copy. This layout
was first serialized with `encoding/gob`; peak during load was then several times the
resident structure (§4.3), because gob buffers a whole message before decoding it. §9
replaces the encoding and removes that cost. The graph is validated on load (offsets monotonic and
in range, parallel arrays equal length) so a corrupt asset errors rather than indexing out
of bounds during a traversal.

### 4.3 Results

For `atebits-letterpress` (the largest). The nested-map figures are historical (that
representation was replaced); the CSR on-disk size is the structural estimate for the
current list, as serialized by gob, the encoding of the time (§9):

| Metric | Nested maps | CSR |
| --- | ---: | ---: |
| Resident (1 copy, after GC) | 843 MB | **68 MB** |
| Peak during decode (`HeapSys`) | 1231 MB | 315 MB |
| Two copies resident (the crash) | 1875 MB heap / 2627 MB sys | ~136 MB |
| On-disk asset (gob) | 76.2 MB | 59.1 MB |

This ~12× resident reduction is a pure representation change; the automaton, the API, the
AI, and word acceptance are all identical. It **independently eliminates the OOM**: even an
uncached double-load is ~136 MB, well below the watermark of §1.1.

---

## 5. Optimization II — load-time caching *(implemented)*

`Dictionary.Load` re-decoded from scratch on every call. Because the game plays one
dictionary at a time, a single-slot cache — keyed on dictionary name, dropping the previous
entry when a different dictionary is requested — makes re-loading the same dictionary
(exactly the save/load flow that crashed) a no-op that returns the existing instance. The
`Dictionary` is immutable after load and documented safe for concurrent use, so sharing the
pointer is sound. This bounds resident memory to a single dictionary and removes the second
decode.

CSR (§4) and caching (§5) together resolve the reported crash. Everything that
follows targets the *remaining* cost: ~136 MB of embedded assets (CSR trie) and ~68 MB
resident.

---

## 6. Optimization III — cross-dictionary deduplication *(investigated)*

### 6.1 The overlap is large

Of a 275,412-word union, **169,126 words (61.4%) are
common to all three**:

| Dictionary | Unique after extracting common-to-all |
| --- | ---: |
| `enable` | 140 |
| `wordnik` | 25,026 |
| `atebits-letterpress` | 101,526 |

Pairwise containment (row ∩ col as a fraction of the row) shows `enable` is nearly a pure
subset of the others, `atebits-letterpress` the outlier:

| ∩ / row | enable | wordnik | atebits |
| --- | ---: | ---: | ---: |
| **enable** | 100.0 | 100.0 | 99.9 |
| **wordnik** | 87.2 | 100.0 | 97.5 |
| **atebits-letterpress** | 62.5 | 70.0 | 100.0 |

Storing the common words once plus per-dictionary remainders stores 295,818 word-entries
instead of 634,070 — a 53% reduction in stored words.

### 6.2 Disk shrinks, runtime does not

We partition each list into `common` (169,126 words) plus a `<dict>_unique` remainder and
size the CSR-trie of each *(structural estimate)*:

| | Embedded size (CSR trie) |
| --- | ---: |
| Monolithic ×3 | 136.3 MB |
| Split (common + 3 uniques) | **72.6 MB (−47%, −63.7 MB)** |

The 53% reduction in stored words more than offsets the prefix-sharing lost at the
common/unique boundary, so the embedded assets shrink. **Runtime memory, however, is
neutral** — loading `atebits-letterpress` means holding `common` + `atebits-letterpress_unique`
(~66 MB) versus the monolithic ~59 MB, so the split is neutral-to-slightly-worse at
runtime. With the single-slot cache only one dictionary is ever resident, so the shared
`common` is not amortized across simultaneously-loaded dictionaries.

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
shared substructure. We verified correctness by replaying all 270,652 `atebits-letterpress`
words and negative controls: **0 mismatches** against the monolithic automaton.

### 7.2 The result: exact reproduction

| | Nodes | Edges |
| --- | ---: | ---: |
| `common` | 4,014,173 | 4,014,172 |
| `atebits-letterpress_unique` | 3,261,251 | 3,261,250 |
| **Merged** | **6,474,718** | **6,474,717** |
| Monolithic `atebits-letterpress` | 6,474,718 | 6,474,717 |

The merged automaton is **node-for-node identical to the monolithic one** (the 7,275,424
nodes of the two inputs dedup exactly to the monolithic 6,474,718).

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

| Metric (merging `common` + `atebits-letterpress_unique`) | Value |
| --- | ---: |
| Merge time (desktop) | ~3.4 s |
| Transient resident around the merge (`HeapSys`) | ~0.96 GB |
| Correctness (words checked / mismatches) | 270,652 / 0 |

The ~0.96 GB transient (dominated by the memo and adjacency maps over 6.5M nodes)
reintroduces the mobile memory pressure we removed in §4, and multi-second load latency
would be worse on a phone. A map-free streaming merge could cut the transient to ~150 MB
(no memoization is even needed — §7.3 shows each product state is reached once), but the
CPU cost of materializing 6.5M nodes per load remains. The merge is correct, but it trades
embedded size for load latency and a load-time memory spike.

---

## 8. Optimization V — automaton minimization *(implemented)*

§2.2 and §7.3 identify the problem: the automata are un-minimized. Classical
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
| `enable` | 4,017,937 | 37 | 402,979 | 839,880 | 6 | 10.0× |
| `wordnik` | 4,441,310 | 40 | 435,389 | 932,089 | 6 | 10.2× |
| `atebits-letterpress` | 6,474,718 | 59 | 591,712 | 1,277,150 | 9 | 10.9× |

\* CSR structural estimate. The minimized edge/node ratio rises to ~2.1 (branching structure
survives; the non-branching suffix chains collapse).

Minimization yields a **~10× reduction in both nodes and edges** across all three
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

Rebuilding the three embedded assets with the minimizing `Build` gives the measured
results below (the trie-CSR column is a *structural estimate*, since the build now always
minimizes; the minimized on-disk sizes are the actual embedded bytes, as serialized by gob,
the encoding of the time — §9 shrinks them by a further third):

| Dictionary | Words | On-disk (trie CSR) | On-disk (minimized) | Resident (minimized) |
| --- | ---: | ---: | ---: | ---: |
| `enable` | 169,266 | 36.7 MB | **5.54 MB** | — |
| `wordnik` | 194,152 | 40.5 MB | **6.10 MB** | — |
| `atebits-letterpress` | 270,652 | 59.1 MB | **8.41 MB** | **8.4 MB** |
| **Total** | | **136.3 MB** | **20.05 MB (−85%)** | |

The largest dictionary is now **8.4 MB resident** (measured `HeapAlloc` after GC), down from
68 MB (CSR trie) and 843 MB (original nested maps). Correctness was re-verified after
implementation: all 270,652 `atebits-letterpress` words are accepted (0 missing), negative
controls are rejected, the full test suite passes — including the AI move-generation tests,
which confirm the minimized DAWG is behaviourally identical for move generation — and a
regression test (`TestBuild_Minimized`) asserts the accept sinks are merged so minimization
cannot be silently disabled.

### 8.3 Why it dominates: minimization subsumes deduplication

The deduplication of §6 exploits *exact-word overlap* between dictionaries. But two words
sharing a suffix — whether they are the "same word in two dictionaries" or simply
"two words ending the same way" — induce a shared sub-automaton that minimization merges.
Cross-dictionary word overlap is therefore a **special case** of the redundancy
minimization already removes, and it removes it *within* a dictionary as well as across the
notional common/unique partition. Minimization is strictly more general.

This is borne out by the numbers: dedup removes 53% of stored words; minimization removes
~90% of nodes. And the two do not usefully stack — once `atebits-letterpress` is ~8.4 MB
resident, a further common/unique split saves a few MB of disk at the cost of the two-graph
complexity of §6.3–§7.

---

## 9. Optimization VI — serialization: gob → streamed varint *(implemented)*

Minimization (§8) leaves the graph as small as the automaton can be. What remained was the
*encoding* of that graph on disk: 20.05 MB of embedded assets for a set of graphs whose
in-memory form is ~21 MB. This strategy changes only the encoding, not the graph.

### 9.1 What gob was costing

The CSR arrays were serialized with `encoding/gob`. Two costs, measured on the real assets:

- **Absolute offsets are expensive to store.** `EdgeOffsets` holds one running total per
  node, reaching the edge count; gob spends four to five bytes on each. For
  `atebits-letterpress` that is **2,366,856 B** of the 8.41 MB asset.
- **Decoding allocates the graph twice.** gob buffers a whole message before decoding it, so
  a load allocates the encoded bytes as well as the structure. Measured `TotalAlloc` around
  a single `loadGADDAG`, against a structure of known size:

| Dictionary | Structure | Allocated during load | Ratio |
| --- | ---: | ---: | ---: |
| `enable` | 5.86 MB | 11.44 MB | 1.95× |
| `wordnik` | 6.46 MB | 12.58 MB | 1.95× |
| `atebits-letterpress` | 8.83 MB | 17.26 MB | 1.96× |

gob was not, however, wasteful about integers: it varint-encodes them, so the gob asset
(8.41 MB) is *smaller* than the same arrays written as fixed-width little-endian binary
(8.83 MB). Replacing gob with a raw fixed-width array — the obvious "drop the reflection"
move — is a 5% regression on disk, which is why it was rejected.

### 9.2 Candidates

Five encodings of the same three minimized graphs, measured by serializing the real assets:

| Encoding | Total, 3 dicts | vs. gob |
| --- | ---: | ---: |
| gob, uncompressed *(superseded baseline)* | 20.05 MB | — |
| Raw fixed-width binary | 21.14 MB | **+5.5%** |
| gob + gzip | 12.63 MB | −37.0% |
| **Varint edge counts + varint targets** | **13.31 MB** | **−33.6%** |
| Varint counts + varint targets + gzip | 8.87 MB | −55.7% |
| Varint counts + zigzag delta targets + gzip | 8.63 MB | −57.0% |

The decisive change is storing each node's **edge count** instead of its absolute offset.
Counts are at most 27 — the alphabet plus the arc separator — so each costs one varint byte,
against four or five for an offset. For `atebits-letterpress`, 2,366,856 B of offsets becomes
591,713 B of counts. Offsets are rebuilt by prefix sum while reading, so they cannot drift
from the arrays they index.

Delta-encoding targets against their source node is *worse* uncompressed (+114 KB on the
largest graph, since minimization makes many edges point backwards to shared suffixes, and
zigzag doubles small negative deltas) and better only once gzipped.

### 9.3 Why not gzip

gzip is the larger reduction on disk, and it was rejected. It buys ~22 further points at the
cost of decompressing on every load — CPU plus a transient buffer, which pushes in exactly
the wrong direction for the constraint of §1.1 — and a compressed asset cannot be `mmap`-ed,
foreclosing the option §14.5 identifies as KWG's structural advantage. The uncompressed
varint form keeps the asset a flat array of the graph, so that option stays open.

### 9.4 Results

Assets rebuilt with the new writer, and the heap measured on the Android emulator:

| Metric | gob | Streamed varint | Change |
| --- | ---: | ---: | ---: |
| Embedded assets, 3 dicts | 20.05 MB | **13.31 MB** | **−33.6%** |
| `enable` | 5.54 MB | 3.68 MB | −33.6% |
| `wordnik` | 6.10 MB | 4.05 MB | −33.6% |
| `atebits-letterpress` | 8.41 MB | 5.58 MB | −33.6% |
| Resident, active dictionary | 8.4 MB | **8.4 MB** | none |
| Process peak (`HeapSys`, defs + 1 dict) | 55.7 MB | **39.6 MB** | **−29%** |
| Total from OS (`Sys`) | 59.4 MB | **43.1 MB** | −27% |

**Resident memory is unchanged, and that is expected.** gob already handed its decoded
slices straight into the `GADDAG` with no post-decode copy (§4.2), so the steady-state
footprint was already just the CSR arrays; there was no intermediate representation to
remove. The gain is on disk and in the load spike, where the ~2× churn of §9.1 disappears.

The reader validates as it goes — edge counts must sum to the declared edge total, and every
target must address a node that exists — so a corrupt asset is rejected at load rather than
indexing out of range mid-traversal. That is strictly more checking than the gob path
performed: it validated offsets and array lengths, but never the edge targets.

---

## 10. Results: the full comparison

| Strategy | Embedded (3 dicts) | Resident (active dict) | AI change | Load cost | Status |
| --- | ---: | ---: | --- | --- | --- |
| Original (nested maps, trie) | —† | 843 MB | — | baseline | replaced |
| **I. CSR** | 136 MB | **68 MB** | none | faster decode | **done** |
| **II. Load cache** | 136 MB | 68 MB (bounded) | none | 0 on re-load | **done** |
| III. Dedup split | 72.6 MB | ~66 MB | rework or merge | — | rejected |
| IV. Dedup + runtime merge | 72.6 MB | ~same | none | +3.4 s, +~0.96 GB | rejected |
| **V. Minimization** | **20.05 MB** | **8.4 MB** | none | faster decode | **done** |
| VI. Raw fixed-width binary | 21.14 MB | 8.4 MB | none | no buffering | rejected |
| VI. gzip | 8.87 MB‡ | 8.4 MB | none | +decompress, no mmap | rejected |
| **VI. Streamed varint** | **13.31 MB** | **8.4 MB** | none | no buffering | **done** |

† The superseded nested-map asset sizes were not re-measured for the current three-list
set; CSR (Strategy I) is the first re-derivable embedded baseline.

‡ gzip over the chosen varint layout; gzip over the gob layout was 12.63 MB.

End-to-end, CSR + minimization takes the largest dictionary from **843 MB → 8.4 MB resident
(~100×)**, and the embedded assets from **136 MB (CSR trie) → 20.05 MB minimized → 13.31 MB
streamed (~10× overall)**, with no change to the traversal API or the AI. Process peak on the
emulator, with the definitions asset also resident, is **43.1 MB** (§9.4, Appendix A.5).
These are the final implemented figures.

---

## 11. Discussion

Three ideas organize the investigation:

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

- **Structure first, then encoding.** §4–§8 change what is stored; §9 changes only how it is
  written down, and only after the structure was minimal. Taken in the other order the
  encoding work would have been spent compressing redundancy that minimization then deleted:
  a third off 136 MB of trie is worth less than a third off 20 MB of minimized graph, and the
  offsets that §9 shrinks are proportional to node count, which §8 had already cut ~10×.

Stated generally: *for a family of dictionaries compiled to acyclic word automata,
cross-dictionary word-set deduplication is dominated by per-dictionary automaton
minimization, because inter-dictionary word overlap is a strict subset of the intra- and
inter-word suffix redundancy that minimization eliminates.* On the TileWords corpus, despite
61% word overlap, minimization yields a larger reduction than exploiting that overlap
directly.

---

## 12. Recommendation and outcome

The recommendation was to implement **minimization inside `dictionary.Build`** (Revuz
bottom-up hash-consing, using the existing child-id > parent-id ordering) and regenerate the
embedded assets, and **not** to pursue the cross-dictionary split (runtime-neutral,
complicates the AI) or the runtime merge (reintroduces a load-time memory spike), since
minimization dominates both.

**This has been done.** Minimization is implemented in `minimizeTrie` and the assets are
regenerated (§8.2). The change was contained:

- The load path, the `Root`/`Successor`/`IsTerminal` API, `Validate`, and the entire AI are
  unchanged — a minimized automaton is simply a smaller CSR graph.
- No runtime merge, no two-graph bookkeeping, no load-time spike; decode is faster.
- Correctness is confirmed by full-corpus acceptance parity, the property-based tests, the
  AI move-generation tests, and a minimization regression guard (§8.2).

Serialization (§9) was then changed for the same reasons in a narrower place: the encoding,
not the graph. Storing per-node edge counts rather than absolute offsets, and streaming the
read instead of buffering a gob message, took the assets down a further third and removed the
~2× allocation churn per load. Two candidate encodings were rejected on measurement — raw
fixed-width binary, 5.5% *larger* than the gob it would have replaced, and gzip, which is
smaller still but adds decompression to every load and forecloses `mmap`.

Result: embedded assets 136 MB (CSR trie) → 20.05 MB minimized → 13.31 MB streamed, the
largest dictionary 68 MB → 8.4 MB resident, and process peak 55.7 MB → 39.6 MB.

---

## 13. Reproducibility

All measurements come from throwaway in-package Go tests (the `dictionary` package, so they
can read the unexported CSR fields) plus a Python overlap script over `wordlists/*.txt`:

- **Overlap (§6.1):** intersect the three word sets filtered to valid 2–15 A–Z words.
- **Split sizes (§6.2):** emit `common.txt` and `<dict>_unique.txt`, compile each with
  `tools/buildgaddag`, and size each CSR trie via the §3 structural formula.
- **Resident memory (§4, §8.2):** `dictionary.Load`, force GC, read `HeapAlloc` (also via
  `tools/memcheck`).
- **Merge (§7):** memoized product construction over two loaded graphs; replay all words
  for correctness; time the merge and read `HeapSys`.
- **Minimization (§8):** now production code in `dictionary.Build` (`minimizeTrie`); the
  node/edge counts came from a throwaway trie build plus `minimizeTrie` over a loaded list,
  and the on-disk and resident figures from rebuilding the assets (`make gaddag`) and
  loading them.
- **Serialization candidates (§9.2):** a throwaway in-package test re-serialized each loaded
  graph in every candidate encoding and gzipped it, so the comparison is over the real assets.
  The load churn of §9.1 is a `TotalAlloc` delta across one `loadGADDAG` call, against the
  structure size from the §3 formula. The final figures come from rebuilding with
  `make gaddag` and re-running `tools/memcheck` on the emulator.

`minimizeTrie` is production code; the harnesses are not retained (they touch unexported
internals and load multi-hundred-MB assets), and are described here so the numbers can be
regenerated on demand.

---

## 14. Related work: the Kurnia Word Graph (KWG)

The Kurnia Word Graph (KWG), used by Andy Kurnia's *wolges* engine (and by MAGPIE and the
Woogles stack), is a minimized acyclic GADDAG stored in a flat array — the same structure as
Strategy V, with a denser encoding.

### 14.1 What KWG is

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

### 14.2 Mapping onto Strategy V

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
| Serialization | streamed varints of flat slices; heap-decoded | raw little-endian array; memory-mappable |
| Pointer width | 32-bit node id (≤ ~4.29 B nodes) | 22-bit arc_index (≤ ~4.19 M cells) |

KWG fuses the node and edge into a single 32-bit word and folds the terminal flag into it,
where the CSR keeps three parallel arrays, an offsets table, and a bitset. The CSR spends
1 byte (letter) + 4 bytes (target) = 5 bytes per edge, plus 4 bytes per node for offsets,
plus the bitset — about 6.9 bytes per transition. KWG spends 4 bytes per transition and
nothing else.

### 14.3 Quantitative comparison

Applying KWG's 4-bytes-per-transition encoding to the measured minimized graphs (an upper
bound for KWG, since its tail-sharing reduces the cell count below the edge count):

| Dict | Nodes | Edges | Strategy V (CSR, in-mem) | KWG (≤ 4 B/edge) | Ratio |
| --- | ---: | ---: | ---: | ---: | ---: |
| `enable` | 402,979 | 839,880 | 5.86 MB | 3.36 MB | 1.74× |
| `wordnik` | 435,389 | 932,089 | 6.46 MB | 3.73 MB | 1.73× |
| `atebits-letterpress` | 591,712 | 1,277,150 | 8.83 MB | 5.11 MB | 1.73× |
| **Total** | | | **21.1 MB** | **≤ 12.2 MB** | **1.73×** |

A KWG of the same dictionaries is ~1.7× smaller than the CSR, both on disk and resident, and
smaller still once tail-sharing and DAWG-node reuse are counted. The 22-bit-vs-32-bit
pointer difference is not the driver — the largest graph, 1.28M edges, fits in 22 bits — the
driver is field fusion and tail-sharing.

### 14.4 Assessment

- Strategy V accounts for the reduction from an un-minimized trie in hash maps (843 MB) to a
  minimized automaton in flat arrays (8.4 MB), ~100×. KWG's denser layout yields a further
  ~1.7× (§14.3). That factor does not change the outcome for TileWords: the OOM is resolved
  under either encoding.
- A KWG is a raw little-endian array and can be `mmap`-ed directly from an uncompressed
  asset: near-zero heap, no decode step, pages faulted in on demand. The varint format of §9
  decodes into ~8.4 MB of heap-allocated slices at load. On a memory-constrained device an mmap-ed
  KWG uses less resident memory and loads without a decode pass — a larger difference than
  the 1.7× byte reduction.
- KWG's tail-sharing is a stronger minimization: Strategy V merges whole equivalent nodes
  (Revuz), while KWG additionally shares suffixes of sibling edge-lists. Expressing it
  requires indexing children by edge position rather than by node id.

Closing the gap would require switching to a packed 32-bit-per-edge cell array (letter +
accepts + is_end + arc_index), indexing children by cell position, and `mmap`-ing it, which
would take TileWords from ~8.4 MB heap to ~5 MB or less mapped with near-zero heap. The
Gaddawg DAWG-union does not apply to TileWords, which never anagrams; only the KWG encoding,
not its dual-graph role, would be relevant.

### 14.5 When CSR is preferable

For move generation in isolation, KWG is smaller, memory-mappable, and no worse on the
traversal hot path, so the CSR layout's advantages are conditional on requirements the KWG
encoding trades away. Most of them stem from one difference: the CSR keeps a separate
`edgeOffsets` table — the source of its ~1.7× size disadvantage — and that table gives every
node a dense, stable integer id in `[0, nodeCount)`. KWG has no node table: a state is a
start position in the shared edge array, and tail-sharing makes states physically overlap,
so there is no dense node-to-index mapping.

- **Graphs larger than ~4.19M edges.** The documented 32-bit KWG cell is
  `tile:8 + accepts:1 + is_end:1 + arc_index:22`, so the whole graph must fit in 2²² =
  4,194,304 cells. atebits-letterpress is 1.28M edges, but a larger or merged lexicon can
  exceed that, at which point the standard KWG needs a wider cell variant. The CSR's 32-bit
  node ids and edge offsets address ~4.29B nodes/edges without a format change.

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

- **Format evolution.** This advantage was gob's, and §9 gave it up: gob was self-describing
  (field names and types), so adding a field was backward- and forward-tolerant. The varint
  stream instead carries a magic-and-version header, so a stale asset is refused with a clear
  message and rebuilt — cheap here, since assets are build products, but it is a version break
  rather than a tolerated one. KWG's raw array has the same property, and unlike either Go
  encoding it is readable from any language.

- **Bulk field sweeps.** The SoA layout keeps all targets (or all letters, or the terminal
  bitset) contiguous, which suits a vectorized scan over one field — analytics or a bulk
  validation sweep. This does not help traversal: there KWG's *array-of-structs* (AoS)
  layout, with each cell holding a letter and its child pointer together, has better cache
  locality for the follow-edge step.

- **This project.** Strategy V is implemented, tested, and brings atebits-letterpress to
  8.4 MB, below the watermark of §1.1. KWG's byte and mmap gains do not change that outcome,
  so for TileWords the operative advantage is an existing, verified implementation with no
  migration.

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

## Appendix A. The same treatment for the definitions asset *(implemented)*

Everything above concerns the GADDAG dictionaries. Once minimization (§8) had taken the
largest of those to 8.4 MB resident, the dominant consumer of the process heap was no longer
a dictionary at all — it was the *definitions* database, at 51.9 MB. The same exercise
follows: the representation change of §4 in a setting that is not a graph, plus a load-path
change with no counterpart in the main body.

The figures in §4–§10 were taken on `linux/amd64`. The figures here were taken **on-device**,
on the Android x86_64 emulator, by cross-compiling `tools/memcheck` for `android/amd64` and
running it under `adb`, mobile being the binding constraint.

### A.1 The starting point

The definitions DB held two Go maps:

```go
entries map[string]*Entry   // lowercase headword -> definitions
formOf  map[string]string   // inflected form -> lemma headword
```

with `Entry{Word string; Senses []Sense}` and `Sense{POS, Gloss string}`. For the shipped
asset that is 146,859 headwords, 251,845 senses and 124,332 inflection edges.

The payload — the text being stored — totals **14.94 MB** (headwords 1.25 MB, glosses
12.59 MB, forms 1.10 MB). The structure cost **51.9 MB** resident, so 37.0 MB, or 2.5× the
payload, went on representation: a map bucket slot and a string header per key, an `*Entry`
allocation and a slice header per headword, and two more string headers per sense.

The pattern is the one §4 identifies — a map of pointers to small objects — but the
achievable gain is much smaller. §4 cut resident memory 12× because a trie node carries no
payload, so almost all of the 843 MB was representation. Here 14.94 MB of the 51.9 MB is
text that any representation must store, which bounds the reduction at 3.5× before any
design choices are made.

### A.2 The flat layout

The dictionaries' CSR layout (§4.2) generalizes even though there is no automaton here. The
rows are headwords and the entries within a row are senses, so the same offset-plus-payload
shape applies:

```go
headBlob  []byte     // every headword, sorted, concatenated
headOff   []uint32   // headword i is headBlob[headOff[i]:headOff[i+1]]
senseOff  []uint32   // headword i's senses are [senseOff[i], senseOff[i+1])
glossBlob []byte     // every gloss, concatenated
glossOff  []uint32   // sense j is glossBlob[glossOff[j]:glossOff[j+1]]
sensePOS  []uint32   // sense j's part of speech, interned into posTable
posTable  []string   // 20 distinct values for 251,845 senses
formBlob  []byte     // every inflected form, sorted, concatenated
formOff   []uint32
formLemma []uint32   // form k resolves to headword formLemma[k]
```

Two differences from §4. There is no graph, so lookups cannot follow edges: the blobs are
sorted and searched by binary search over `headOff`, which replaces hashing. And parts of
speech are interned — 20 distinct strings serve 251,845 senses, where the map form stored a
string header for each.

The structural cost is the blobs plus the index arrays,
`(headwords+1)·8 + (senses+1)·4 + senses·4 + (forms+1)·4 + forms·4` bytes:

| Component | Size |
| --- | ---: |
| Text blobs (headwords, glosses, forms) | 14.94 MB |
| Index arrays (offsets, POS, lemma) | 3.99 MB |
| **Structural total** | **18.93 MB** |
| Measured live heap after the change | **19.0 MB** |

The measured live heap is 0.07 MB above the structural total: payload and index arrays
account for all of it.

### A.3 Choosing the on-disk encoding before implementing

Memory was the target, but embedded size and resident memory are distinct costs (§2.3), and a
layout can improve one while worsening the other. Four candidate encodings were therefore
measured by serializing the real asset in each and gzipping the result. The measurement used
a throwaway in-package test and preceded the implementation:

| Candidate encoding | Gzipped | vs. baseline |
| --- | ---: | ---: |
| Map of pointers, `gob` (baseline) | 8.22 MB | — |
| Flat, absolute `uint32` offsets | 7.62 MB | −7.3% |
| Flat, per-item lengths, raw binary | 6.20 MB | −24.6% |
| **Flat, per-item lengths, varint** | **6.15 MB** | **−25.2%** |

The format follows from the third and fourth rows: **store lengths, not offsets.** Absolute
offsets are large and monotonically increasing, which compresses poorly; the lengths they are
derived from are small and compress well. Offsets are rebuilt by prefix sum while reading,
so they cannot drift from the blobs they index. Fixed-width `uint32` fields cost more than
varints, which spend 1–2 bytes on values a fixed field spends 4 on.

The flat layout therefore reduces both resident memory and on-disk size; it does not trade
one for the other.

### A.4 Streaming the load: peak is a separate problem from steady state

The first implementation kept `gob` as the container. Steady-state memory fell by more than
half; peak fell by 8%:

| Metric | Maps + `gob` | Flat + `gob` |
| --- | ---: | ---: |
| Live heap, defs DB | 51.9 MB | 22.0 MB |
| Peak during load (`HeapSys`) | 95.6 MB | 87.6 MB |

Two causes, both inherent to a reflective encoder. `gob` buffers an entire message before
handing it over, so decoding a single ~19 MB top-level value holds a full copy of the
encoded stream *and* the decoded result at once. And the per-item length arrays were
materialized as slices before being converted into the offset arrays, so both existed
simultaneously.

The fix was to drop reflective encoding. The asset is now a gzip-compressed byte
stream read field by field: counts first, then each blob into a slice sized exactly from its
declared length, with offsets accumulated by prefix sum *as the lengths are read*. Nothing
larger than the DB's own structures is ever allocated, and the only buffers are a 64 KB
`bufio` window and gzip's own. The lengths arrays now never exist at all, which is where the
last 3 MB of steady state went.

On mobile the allocation spike at startup, not the resting footprint, is what the low-memory
killer reacts to — the failure mode of §1.1.

### A.5 Results

| Metric | Maps + `gob` | Flat + `gob` | Flat + streamed | Change |
| --- | ---: | ---: | ---: | ---: |
| Live heap, defs DB | 51.9 MB | 22.0 MB | **19.0 MB** | **−63%** |
| Live heap, defs + one dictionary | 58.2 MB | 28.3 MB | **25.3 MB** | −57% |
| Heap in use (`HeapInuse`) | 66.1 MB | 28.7 MB | **25.6 MB** | −61% |
| Peak during load (`HeapSys`) | 95.6 MB | 87.6 MB | **55.7 MB** | **−42%** |
| Total from OS (`Sys`) | 100.9 MB | 91.2 MB | **59.4 MB** | −41% |
| On-disk asset | 8.22 MB | 6.15 MB | **6.13 MB** | **−25.4%** |

Lookup results, the `Lookup` API and every consumer are unchanged; only the internals and the
asset format moved. The asset was renamed `definitions.gob.gz` → `definitions.bin.gz`, since
it is no longer a gob.

The format also became **byte-deterministic**. The flat arrays have a fixed order, so
rebuilding from unchanged sources reproduces the asset exactly; the map form's output varied
between builds, because `gob` serializes map keys in nondeterministic order. And since every
count, length and index is validated as it is read, a truncated or corrupt asset is reported
as an error at load rather than panicking on an out-of-range slice at first lookup.

### A.6 What carried over from the main study

- **Representation dominates.** As in §4, the reduction came from the layout, not from
  storing less: the payload is byte-identical, and 37.0 MB of per-object overhead became
  3.99 MB of index arrays.
- **Deduplication applies without an automaton.** There is nothing here to minimize in the
  §8 sense, but the same principle holds at the string level: 20 interned part-of-speech
  strings replace 251,845 string headers.
- **Encoding choices are measurable in advance.** §6–§7 were rejected on measurements taken
  after implementation; A.3 chose between four encodings before writing any.
- **Peak and steady state are separate problems.** §4.3 reports both `HeapAlloc` and
  `HeapSys`; A.4 is the case where the two diverged, and only the second change moved the
  one that matters on-device.

### A.7 Remaining headroom

Peak is still 55.7 MB against 25.3 MB live. The remainder is Go heap arena growth and the
GC not returning pages to the OS promptly, not an identifiable buffer, so it is not
addressable by another encoding change. Reducing it further would mean mapping the blobs
from a file instead of embedding and decompressing them (`mmap`), which is the same
memory-mappability argument §14.5 makes for KWG — and it would give up the single
self-contained binary. Not pursued.

### A.8 Reproducibility

- **Resident and peak memory:** `tools/memcheck`, cross-compiled with
  `CGO_ENABLED=1 GOOS=android GOARCH=amd64 CC=<ndk>/x86_64-linux-android33-clang`, pushed to
  the emulator with `adb push` and run there. It loads each structure, forces a GC, and
  reports the `HeapAlloc` delta, then the process `HeapInuse`/`HeapSys`/`Sys`.
- **Candidate encoding sizes (A.3):** a throwaway in-package test in `defs` (so it could read
  the unexported fields) that re-serialized the loaded asset in each candidate layout and
  gzipped it. Not retained.
- **On-disk size and coverage:** `make defs` to rebuild, then `make defs-audit` to confirm
  the rebuild reports identical coverage (146,859 headwords, 124,332 edges).
- **Correctness:** the `defs` package tests, including a determinism test, a round-trip
  test, and ten malformed-asset cases — the magic header, all four length-sum checks
  (headword, gloss, sense, form), both index-range checks (part-of-speech, form lemma), and
  a mid-stream truncation; plus `tools/defslookup` spot-checks of exact matches, the
  homograph/inflection path (`mice`→`mouse`, `rose`→`rise`) and supplemental-glossary
  entries.

---

*This document records the state of the investigation. Strategies I, II, and V are
implemented in the codebase; III and IV were investigated and rejected. The §8–§10 figures
for V are measured after implementation, refreshed for the current three shipped word lists.
Appendix A applies the same representation change to the definitions asset, measured on the
Android emulator.*
