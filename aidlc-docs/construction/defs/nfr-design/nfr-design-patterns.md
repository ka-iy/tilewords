# NFR Design Patterns — Unit 6: `defs`

## Pattern 1: Layered, Headword-Validated Resolution (Correctness)
**Problem**: A played word may be an exact headword, an inflection, a spelling variant, or
unknown; a wrong definition is worse than none.
**Solution**: Try layers most- to least-reliable (exact → form-of → stem → variant). The
stem and variant layers generate *candidate* strings but only accept ones that are real
headwords (or edges to them), so they can never invent a definition. Blind edit-distance was
rejected precisely because it can (BR-D05).

## Pattern 2: Wiktionary Inflection Data over Heuristics (Coverage/Correctness)
**Problem**: Rule-based stemming alone misses irregular inflections and over-generates.
**Solution**: Prefer Wiktionary's own `form_of` senses and lemma inflection tables as exact
form→lemma edges; fall back to rule-based de-inflection (incl. classical plurals) only when
no explicit edge exists. This yields ~99% coverage on the curated lists.

## Pattern 3: Deterministic Concurrent Build (Reproducibility — NFR-DEFS-R2)
**Problem**: Parsing ~3 GB single-threaded is slow, but concurrent workers emit out of order,
which would make the primary-sense choice and edge resolution non-deterministic.
**Solution**: Workers tag each extract with its file line number; the collector sorts by that
sequence before merging senses and resolving edges. Concurrency for speed, file order for
determinism.

## Pattern 4: Transparent Compression at the Serialisation Boundary (Asset Size)
**Problem**: The raw gob is ~20 MB; shipping that in every binary is wasteful.
**Solution**: `Encode`/`Decode` wrap the gob stream in gzip, so the on-disk/embedded asset is
~7.8 MB and every call site (build tool, loader, inspector) gets compression for free.

## Pattern 5: Cached Immutable Singleton (Memory/Performance — NFR-DEFS-M1/P2)
**Problem**: The DB is ~52 MB and slow to decode; decoding per lookup or per game is wasteful.
**Solution**: `Load` decodes once under `sync.Once` and returns the shared, immutable `*DB`;
concurrent `Lookup` needs no locking.

## Pattern 6: Optional Feature Gate (Reliability — NFR-DEFS-R1)
**Problem**: The asset is large and built from a non-committed extract; a build may lack it.
**Solution**: `//go:embed all:assets/definitions` embeds the directory even when it holds only
`.gitkeep`; `Available()` reports whether the `.gob.gz` is present so the UI can hide the
feature and degrade gracefully instead of failing.

## Pattern 7: Additive Homograph Merge (UX Correctness — BL-D06)
**Problem**: A word can be both a rare headword and a common inflection (mice, oxen); showing
only one reading misleads.
**Solution**: On an exact match, attach the inflected lemma's senses via `Result.AlsoForm`
without demoting the word's own senses, so common homographs (rose, found) are never harmed.

## Pattern 8: GoDoc + Rationale Commentary (NFR-10)
**Mandatory**: Every exported symbol has GoDoc; the resolution order, the rejection of blind
edit-distance, the determinism seam, and the compression choice are documented in comments so
the "why" survives.
