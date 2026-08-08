# Business Logic Model — Unit 6: `defs`

## BL-D01: Definitions Asset Build (Build Tool — `tools/builddefs`)

**Input**: A Wiktionary extract JSONL file (kaikki.org wiktextract dump, English) and one
or more word list `.txt` files.
**Output**: One gzip-compressed gob (`definitions.gob.gz`) filtered to the words the lists
can form, plus a per-list coverage report.

### Algorithm (`BuildFilteredDB`)
```
1. Read every word list → per-list ordered slices + the lowercase union set `needed`.
2. Stream the extract concurrently (one JSON worker per CPU); each line:
   - keep only lang_code == "en" and a single lowercase A–Z headword;
   - for each sense: form-of senses become inflection edges (form → lemma); other
     senses become real definitions, classified primary vs low-value (BR-D04);
   - for each row of a lemma's inflection table (Forms) whose form is in `needed`,
     record an edge form → this lemma.
3. Restore file order by sequence number (deterministic despite concurrency), then
   merge senses per headword: primary senses first, then low-value, capped at 4.
4. Resolve inflection edges to a form→lemma map, preferring a lemma that has real senses.
5. Build the full DB, then measure coverage: DB.Lookup every word of every list; tally
   exact / formof / stem / fuzzy / miss; collect the headwords reached.
6. Filter: ship only the entries some list word reaches, plus the edges needed to reach
   them. Encode the shipped DB (BL-D05) and print the Report.
```
**Source**: Wiktionary via kaikki.org (`wiktextract`); definitions are CC BY-SA.

---

## BL-D02: Layered Word Resolution (`DB.Lookup`)

**Input**: A played word (any case).
**Output**: `(Result, bool)` — the definition and how it matched, or not found.

### Algorithm
```
lw := lower(trim(word))
1. Exact:  entries[lw] present → Result{Exact}. If lw is ALSO a recorded inflected form
           of a different lemma with an entry, attach that lemma via AlsoForm (BL-D06).
2. FormOf: formOf[lw] → lemma with an entry → Result{FormOf}.
3. Stem:   for each candidate in candidateStems(lw): entries[candidate] or its formOf
           lemma present → Result{Stem}.
4. Fuzzy:  for each candidate in variantCandidates(lw): resolve as in step 3 → Result{Fuzzy}.
5. Otherwise → not found.
```
Layers are ordered most- to least-reliable. Both the stem and fuzzy layers only accept a
rewrite that is itself a real headword, so neither can invent a definition (BR-D05).

---

## BL-D03: Rule-Based De-Inflection (`candidateStems`)

**Purpose**: Reduce an inflected word to plausible lemma candidates, most likely first.
Each candidate is validated against the DB, so over-generation is harmless.

Covers: noun plurals / third-person `-s -es -ies -ves`; past/participle `-ed -ied`;
gerund `-ing`; comparative/superlative `-er -est -ier -iest`; adverb `-ly -ily`; noun
`-ness`; and classical plurals `-ae→-a/-e`, `-es→-is`, `-ata→-a`, `-i→-us`, `-a→-um/-on`.
Consonant doubling (`running→run`) and silent-e restoration (`baking→bake`) are attempted.

---

## BL-D04: Orthographic-Variant Rewrites (`variantCandidates`)

**Purpose**: Map a spelling to a variant of the same word via known orthographic
correspondences: `-ise/-ize`, `-isation/-ization`, `-yse/-yze`, `-ogue/-og` (both
directions) and the contractions `ae→e`, `oe→e`, `our→or`. Validated against headwords.

---

## BL-D05: Asset Serialisation (`DB.Encode` / `Decode`)

`Encode` gzip-compresses a gob of the two DB maps (~a third the size of the raw gob).
`Decode` reverses it. The runtime asset (`definitions.gob.gz`) is ~7.8 MB.

---

## BL-D06: Runtime Lemma Merge (AlsoForm)

On an exact match whose word is also a recorded inflected form of a different lemma, the
lemma's definitions are attached via `Result.AlsoForm` so the caller can show both
readings (e.g. "mice" shows its rare verb sense plus "form of mouse"). The word's own
senses stay primary, so a common homograph (e.g. "rose", "found") is never demoted.

---

## BL-D07: Coverage Reporting

For each list, `Lookup` is run over every word and tallied by match kind, with sample
misses retained. Measured coverage: ENABLE 99.2%, Wordnik 98.6%, atebits-letterpress
96.1%. The `tools/defslookup` tool audits individual layers; `tools/memcheck` reports the
DB's retained heap.

---

## Testable Properties

| Property | Category | Description |
|---|---|---|
| **Encode/Decode round-trip** | Round-trip | A DB encoded then decoded resolves the same words to the same headwords/kinds. |
| **Stem correctness** | Oracle | `candidateStems` contains the known lemma for representative inflections (cats→cat, baking→bake, wolves→wolf, cacti→cactus). |
| **Variant correctness** | Oracle | `variantCandidates` contains the known variant (activise→activize, flavour→flavor). |
| **No invented matches** | Invariant | Stem/fuzzy only resolve to headwords actually present in the DB. |
| **Deterministic build** | Determinism | Two builds from the same inputs produce equivalent definitions (verified by 0 diffs over a sampled lookup). |
| **Homograph safety** | Invariant | An exact match keeps the word's own senses primary; the inflection reading is additive via AlsoForm. |
