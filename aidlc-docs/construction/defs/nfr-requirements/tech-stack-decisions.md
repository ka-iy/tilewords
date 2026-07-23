# Tech Stack Decisions — Unit 6: `defs`

## Language
| Decision | Choice | Rationale |
|---|---|---|
| Implementation language | Go | Project-wide; shares the module with the game |

## Definition Source
| Decision | Choice | Rationale |
|---|---|---|
| Corpus | Wiktionary via kaikki.org `wiktextract` (English JSONL) | Best coverage of obscure word-game vocabulary (2-letter words, inflections, archaic terms); machine-readable; CC BY-SA |
| Filtering | To the union of the shipped word lists | Ships only definitions the player can actually form; bounds asset size |

## Serialisation
| Decision | Choice | Rationale |
|---|---|---|
| Wire format | gzip-compressed `encoding/gob` | stdlib; `Encode`/`Decode` gzip transparently; ~1/3 the raw size (~7.8 MB) |
| Asset embedding | `//go:embed all:assets/definitions` | Single-binary deployment; tolerates a missing `.gob.gz` at compile time (surfaces at load) |

## Matching Strategy
| Decision | Choice | Rationale |
|---|---|---|
| Resolution | Layered: exact → form-of → stem → orthographic variant | Most-reliable-first; each fallback validated against real headwords |
| Rejected | Blind edit-distance nearest-headword | Produced wrong definitions on obscure vocabulary (BR-D05) |
| Inflection data | Wiktionary `form_of` senses + lemma inflection tables | Highest-quality inflection→lemma edges, better than heuristic stemming alone |

## Build Pipeline (`tools/builddefs`)
| Decision | Choice | Rationale |
|---|---|---|
| Extract parsing | streaming JSONL, `encoding/json`, N worker goroutines | The extract is ~3 GB; streaming + concurrency keeps build time reasonable |
| Determinism | line-sequence-ordered merge | Concurrent parse, deterministic output (BR-D06) |
| Build trigger | `make defs KAIKKI_EXTRACT=<path>` | Opt-in; the extract is large and not committed |

## Runtime Load
| Decision | Choice | Rationale |
|---|---|---|
| Caching | `sync.Once` singleton | The DB is large and immutable; decode once, share the instance |
| Availability gate | `defs.Available()` | Lets the UI hide the feature when the asset was not built |

## Data Structures
| Decision | Choice | Rationale |
|---|---|---|
| Headword store | `map[string]*Entry` | O(1) exact lookup |
| Inflection edges | `map[string]string` (form → lemma) | O(1) form-of resolution; smaller than duplicating glosses per inflection |

## Testing
| Package | Purpose |
|---|---|
| `testing` (stdlib) | Unit tests over in-memory DBs (`NewDB`) — no external data needed |

## Developer Tools
| Tool | Purpose |
|---|---|
| `tools/builddefs` | Build the filtered asset + print coverage |
| `tools/defslookup` | Inspect the asset; audit a match layer by eye |
| `tools/memcheck` | Report the DB's (and GADDAGs') retained heap |

## Dependencies
| Package | Purpose |
|---|---|
| stdlib only (`gob`, `gzip`, `json`, `embed`, `sync`) | No external runtime dependency for this unit |
