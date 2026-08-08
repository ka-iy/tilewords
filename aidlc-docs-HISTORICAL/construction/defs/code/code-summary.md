# Code Summary — Unit 6: `defs`

## Files

| File | Description |
|---|---|
| `defs/entry.go` | Package doc; `Sense`, `Entry`, `MatchKind` (+`String`), `Result` (incl. `AlsoForm`/`AlsoFormWord`) |
| `defs/db.go` | `DB`, `NewDB`, `Len`, `FormCount`, `FormLemma`, layered `Lookup` (+ `resolveHeadword`), gzip `Encode`/`Decode` |
| `defs/inflect.go` | `candidateStems` (de-inflection incl. classical plurals), `undouble`, helpers; `minStemLen` |
| `defs/variant.go` | `variantCandidates` (orthographic variants: -ise/-ize, -our/-or, ae/e, …) |
| `defs/build.go` | Streaming extract parser: `BuildFilteredDB`, `WordList`, `ListCoverage`, `Report`, `extractLine`, sense classification (`classifySense`), `cleanGloss`, `resolveEdges`, deterministic seq-ordered collector; `MaxSensesPerEntry`, `maxGlossLen` |
| `defs/loader.go` | `//go:embed all:assets/definitions`, `Available`, cached `Load` (`sync.Once`) |
| `defs/assets/definitions/.gitkeep` | Tracks the asset dir; `definitions.gob.gz` is gitignored and built by `make defs` |
| `defs/defs_test.go` | Stem/variant/fuzzy tests, `Lookup` layers, encode/decode round-trip, homograph merge, sense classification |
| `defs/loader_test.go` | Embedded-asset load (skipped when the asset is absent) |

## Developer Tools

| File | Description |
|---|---|
| `tools/builddefs/main.go` | CLI: `-kaikki`, `-input`, `-output`; builds the filtered `.gob.gz`; prints the coverage report |
| `tools/defslookup/main.go` | CLI: resolve words; `-audit <list> -kind exact\|formof\|stem\|fuzzy`; `-mergeaudit` |
| `tools/memcheck/main.go` | CLI: reports the DB's and each dictionary GADDAG's retained heap (`-dict`, `-defs`) |

## Build Integration

- `make defs KAIKKI_EXTRACT=<path>` filters the Wiktionary extract to `wordlists/*.txt`
  and writes `defs/assets/definitions/definitions.gob.gz` (opt-in; `build` does not depend
  on it — the game runs without it).
- `.gitignore`: adds `*.gob.gz` alongside `*.gob`.

## UI Integration (see `construction/ui/functional-design/move-history-and-definitions.md`)

- `ui/definitions.go` — the definitions worker: `startDefinitions`, channel dispatch
  (`dispatchDefinitions`, `dispatchHistoryDefinitions`), `runDefinitionsWorker`,
  `appendDefinition`, `formatDefinitionEntry`.
- `ui/tabpanel.go` — the two-tab Move history / Definitions switcher + Copy button.
- `ui/dragscroll.go` — mobile drag-to-scroll overlay with long-press copy.
- `engine/state.go` — `MoveRecord.Words` (persists a play's words so definitions repopulate
  on load).

## Coverage (measured)

| List | Covered |
|---|---|
| ENABLE | 99.2% |
| Wordnik | 98.6% |
| atebits-letterpress | 96.1% |

Asset: ~145k headwords + ~124k inflection edges; ~7.8 MB gzipped. Retained heap ~52 MB.

## Tests

`go test ./defs/...` passes (headless; no external data). The embedded-load test runs only
when `make defs` has produced the asset.
