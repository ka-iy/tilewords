# Logical Components — Unit 6: `defs`

## Overview
The `defs` unit is a pure in-memory library plus an offline build pipeline and developer
tools. No network, database, queue, or cache infrastructure.

---

## Component 1: Build-Time Definitions Compiler (`tools/builddefs`)

**Type**: CLI tool (offline, developer-run via `make defs`)
**Inputs**: Wiktionary extract JSONL (`KAIKKI_EXTRACT`, not committed) + word list `.txt` files
**Outputs**: `defs/assets/definitions/definitions.gob.gz` (built locally) + coverage report

```
[kaikki English extract .jsonl]     [word lists .txt]
            |                              |
            v                              v
   [Streaming JSON workers] ----needed---> [union set]
            |
            v
   [Sense/edge extractor]      -- real senses (ranked) + inflection edges
            |
            v
   [Deterministic collector]   -- restore file order, merge senses, cap at 4
            |
            v
   [Edge resolver]             -- form → lemma (prefer lemma with senses)
            |
            v
   [Coverage + filter]         -- Lookup every list word; keep reached entries
            |
            v
   [gzip gob Encoder] ---> definitions.gob.gz  (+ Report to stdout)
```

---

## Component 2: Runtime Definitions Loader (`defs.Load` / `Available`)

**Type**: In-process, cached (`sync.Once`)
**Trigger**: First lookup after a game screen is shown
```
[//go:embed all:assets/definitions] --Available()--> bool (gate)
                 |
            ReadFile(.gob.gz)
                 |
            [gzip gob Decoder] --> *DB (immutable, cached)
```

---

## Component 3: Layered Matcher (runtime, in `defs`)

**Type**: In-process method calls on `*DB`
**Consumer**: the UI definitions worker
```
[played word] → Lookup()
     exact → form-of → stem(candidateStems) → fuzzy(variantCandidates) → none
                                                       |
                                      (+ AlsoForm merge on exact homographs)
```
No caching layer; the maps are the optimal structure.

---

## Component 4: Developer Inspection Tools

| Tool | Role |
|---|---|
| `tools/defslookup` | Resolve individual words; audit one match layer (exact/formof/stem/fuzzy) over a list |
| `tools/memcheck` | Load the DB and each GADDAG, report retained heap deltas |

---

## Infrastructure Components: None

| Type | Present | Rationale |
|---|---|---|
| Network / HTTP | No | Pure local library; the extract is fetched manually by the developer |
| Database / store | No | Asset embedded in binary |
| Message queue | No | Synchronous calls; the UI worker uses an in-process channel |
| Cache | No (beyond the `sync.Once` singleton) | Maps are O(1); no external cache warranted |
