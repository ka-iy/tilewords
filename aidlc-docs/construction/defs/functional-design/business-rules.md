# Business Rules — Unit 6: `defs`

## BR-D01: Definition Source and Licensing
- Definitions are sourced from Wiktionary via the kaikki.org `wiktextract` English dump.
- The data is licensed CC BY-SA; the game must attribute it (e.g. "Definitions from
  Wiktionary, CC BY-SA") in an about/credits surface.
- The raw ~3 GB extract is **not** committed; it is supplied by the developer via the
  `KAIKKI_EXTRACT` variable to `make defs` (opt-in build step).

## BR-D02: Headword Eligibility
- Only `lang_code == "en"` entries whose headword is a single lowercase A–Z word are kept.
- Multiword phrases, non-alphabetic entries, and non-English entries are discarded.

## BR-D03: Sense Cap and Gloss Length
- At most `MaxSensesPerEntry` (4) senses are retained per headword.
- Each gloss is whitespace-normalised and truncated to `maxGlossLen` (200 runes) with an
  ellipsis, keeping the shipped asset small.

## BR-D04: Sense Quality Ranking
- Proper-noun entries (`pos == "name"` — surnames, placenames, brands) are dropped: they
  are never the intended meaning of a lowercase game word.
- Initialism/abbreviation/acronym/symbol/clipping/contraction glosses are ranked **after**
  ordinary senses, so a common meaning is shown first (e.g. "za" → "Pizza", not the
  zinc-aluminium initialism; "children" → "a young human", not the surname).

## BR-D05: No Invented Definitions (Matcher Safety)
- The stem and fuzzy layers only accept a rewrite that is itself a real headword (or an
  edge to one). They can never map a word to an unrelated near-spelling.
- A blind edit-distance ("nearest headword") matcher was evaluated and **rejected**: on
  obscure word-game vocabulary it maps one rare word onto a different rare word one edit
  away (e.g. aerosat→aerostat), yielding wrong definitions. Only rule-based, headword-
  validated de-inflection and orthographic-variant matching are used.

## BR-D06: Deterministic Build
- Parsing runs concurrently, but the collector restores file order by line sequence before
  merging senses and resolving edges, so the choice of primary sense and every edge is
  deterministic. Two builds from the same inputs produce equivalent definitions.

## BR-D07: Runtime Immutability and Thread Safety
- A decoded `DB` is read-only; `Lookup` is safe to call concurrently from multiple
  goroutines without synchronisation.
- The game's definitions worker runs off the UI goroutine and marshals results back with
  the UI framework's thread-confinement call.

## BR-D08: Graceful Absence
- The definitions asset is optional. When it is not embedded (`make defs` not run), the
  package reports `Available() == false` and the UI shows an "unavailable" note; the game
  is otherwise fully playable.
- A played word with no definition returns `MatchNone`; the UI shows "(no definition
  found)" for that entry rather than failing.

## BR-D09: Homograph Handling (Lemma Merge)
- When a word is both a headword and a recorded inflected form of a different lemma, its
  own senses stay primary and the lemma reading is surfaced additively (BL-D06). This never
  demotes a common homograph to its inflected reading.

## BR-D10: Asset Format and Ignore Rules
- The shipped asset is a gzip-compressed gob at `defs/assets/definitions/definitions.gob.gz`,
  embedded via `//go:embed`, and decoded once (cached) at first use.
- Both `*.gob` and `*.gob.gz` are gitignored (only `.gitkeep` is tracked), matching the
  dictionary GADDAG assets; the asset is built locally, not committed.

## BR-D11: Trademark Compliance
- Consistent with the project-wide rule (NFR-09), no Hasbro trademark appears in the
  `defs` package identifiers, comments, or shipped strings.
