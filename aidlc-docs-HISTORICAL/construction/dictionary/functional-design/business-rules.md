# Business Rules — Unit 1: `dictionary`

> **Correction addendum** — The shipped dictionaries are **enable**, **wordnik**, and
> **atebits-letterpress** (public-domain / open lists), not the placeholder
> `csw/sowpods/ospd/naspa/otcwl/all` names used below, and there is no combined "all" GADDAG.
> The project name is **TileWords**. See `aidlc-docs/corrections.md`.

## BR-01: Letter Normalisation
- All word inputs (from source word lists and from validation callers) are normalised to uppercase before processing.
- Normalisation is applied byte-by-byte: bytes in range `a`–`z` (0x61–0x7A) are converted to `A`–`Z` (0x41–0x5A). All other bytes are left unchanged.
- Normalisation must not allocate a new string unless the input actually contains lowercase bytes (fast path: scan first, convert only if needed).

## BR-02: Valid Letter Set
- Only bytes `A`–`Z` (0x41–0x5A) are valid in words stored in or queried against the GADDAG.
- The arc-separator byte `+` (0x2B) is reserved for internal GADDAG traversal; it must never appear in source words or validation inputs.
- Any word or query string containing a byte outside `A`–`Z` (after normalisation per BR-01) is treated as invalid.

## BR-03: Word Length Constraints
- Minimum valid word length: **2 letters**.
- Maximum valid word length: **15 letters** (board size constraint).
- Words outside this range in source word lists are silently discarded during build (they cannot be placed on a 15×15 board).
- Validation queries for strings outside this range return `false` immediately without GADDAG traversal.

## BR-04: Invalid-Word Handling in Build Tool
- If a word in a source word list contains a non-A-Z character (after normalisation), it is **skipped** and a warning is written to `stderr` in the format:
  `[buildgaddag] skipping word %q: contains non-A-Z character`
- Building continues after the skip; the tool does not abort.
- Words skipped due to BR-03 (length) are also logged in the same format.

## BR-05: Deduplication for Combined Dictionary
- When building the `all.gob` combined dictionary, words from all 5 source lists are merged into a single slice.
- The merged slice is sorted lexicographically (byte order, ascending).
- Adjacent duplicates are removed in a single pass (standard deduplicate-sorted-slice pattern).
- The resulting deduplicated, sorted slice is then used to construct the GADDAG.
- This ensures the combined GADDAG is exactly equivalent to the set-union of the 5 word lists.

## BR-06: GADDAG Asset Ownership
- Pre-built `.gob` files in `assets/dictionaries/` are the authoritative runtime representation.
- The build tool (`tools/buildgaddag`) is the only process that writes `.gob` files.
- The `dictionary` package only reads `.gob` files; it never constructs a GADDAG at runtime.
- Source word list `.txt` files are **not** committed to the repository (licensing constraints); only the derived `.gob` files are committed.

## BR-07: Thread Safety
- A `*GADDAG` and `*Dictionary` are read-only after `Load` returns.
- All public methods (`Contains`, `Validate`, `Successor`, `IsTerminal`, `Root`, `GADDAG`) are safe to call concurrently from multiple goroutines without synchronisation.
- The `Loader.Load` function itself is not required to be concurrency-safe (it is called once at game start, before any concurrent access).

## BR-08: Validation Semantics
- `Dictionary.Validate(word string) bool` returns `true` if and only if:
  1. `word`, after normalisation, consists entirely of A-Z bytes, AND
  2. `len(word)` is in [2, 15], AND
  3. The GADDAG's full-reverse path for the word terminates at a terminal node.
- Any failure of conditions 1 or 2 returns `false` without GADDAG traversal (fast rejection).

## BR-09: Trademark Compliance
- The string "Scrabble" must not appear in any exported or unexported identifier, constant, comment, or log message within the `dictionary` package.
- Dictionary names and UI labels use only the official abbreviation (CSW, SOWPODS, OSPD, NWL/NASPA, OTCWL, All Dictionaries).
