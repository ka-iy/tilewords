# Domain Entities — Unit 6: `defs`

> Retroactively documented. The `defs` unit was added after the initial five units to
> show the meaning of every word formed during gameplay. It sources definitions from
> Wiktionary (via the kaikki.org wiktextract dump), filters them to the shipped word
> lists at build time, and resolves a played word to a definition at runtime through a
> layered matcher.

## Entity: `Sense`

**Description**: One definition of a headword.
**Attributes**:
```
Sense {
    POS   string  // part of speech, e.g. "noun", "verb"; "" when Wiktionary recorded none
    Gloss string  // one human-readable definition line
}
```

---

## Entity: `Entry`

**Description**: Every definition Wiktionary records for one headword.
**Attributes**:
```
Entry {
    Word   string   // lowercase headword these senses define
    Senses []Sense  // in Wiktionary order, capped at MaxSensesPerEntry (4)
}
```

---

## Entity: `MatchKind`

**Type**: `uint8` (typed constant)
**Valid values** (ordered most- to least-reliable):
- `MatchNone` — no definition found
- `MatchExact` — the queried word is itself a headword
- `MatchFormOf` — Wiktionary records the word as an inflected form of the headword
- `MatchStem` — rule-based de-inflection reduced the word to the headword
- `MatchFuzzy` — a known orthographic-variant rewrite reached the headword

`String()` returns a short label (`"exact"`, `"formof"`, `"stem"`, `"fuzzy"`, `"none"`).

---

## Entity: `Result`

**Description**: The outcome of `DB.Lookup`.
**Attributes**:
```
Result {
    Entry        *Entry     // definitions found, or nil when Kind == MatchNone
    Headword     string     // DB key that supplied Entry; may differ from the queried word
    Kind         MatchKind  // which resolution layer matched
    AlsoForm     *Entry     // set only on an exact match whose word is ALSO an inflected form
    AlsoFormWord string     // the lemma AlsoForm belongs to, "" when AlsoForm is nil
}
```
`AlsoForm` carries the lemma reading for a homograph (e.g. "mice" is a rare verb yet
chiefly the plural of "mouse"), so a caller can show both readings.

---

## Entity: `DB`

**Description**: A set of headword definitions plus the inflection edges needed to
resolve a played word to one of them. Immutable after construction; safe for concurrent
`Lookup`.
**Internal structure**:
```
DB {
    entries map[string]*Entry   // lowercase headword → definitions
    formOf  map[string]string   // lowercase inflected form → lemma headword present in entries
}
```
On-disk shape (`gobDB`) persists only the two maps; the DB is gzip-compressed gob.

---

## Build-time entities

### Entity: `WordList`
Names a word list file to measure coverage against and to filter definitions down to.
```
WordList { Name string; Path string }
```

### Entity: `ListCoverage`
Per-list breakdown of how words resolved: `Total`, `Exact`, `FormOf`, `Stem`, `Fuzzy`,
`Miss`, `SampleMisses []string`; `Covered()` = Exact+FormOf+Stem+Fuzzy.

### Entity: `Report`
Build summary: per-list `Lists []ListCoverage`, plus `FullHeadwords`, `FullForms`,
`ShippedHeadwords`, `ShippedForms`.

### Entity: `kaikkiEntry` (wire subset)
The subset of one Wiktionary-extract JSONL line the parser reads: `Word`, `Pos`,
`LangCode`, `Senses []{Glosses, Tags, FormOf}`, `Forms []{Form, Tags}`. Only
`lang_code == "en"` single-word alphabetic headwords are kept.

---

## Entity relationships

```
WordList(s) ---filter---> DB
                          |
              +-----------+-----------+
              |                       |
          entries                  formOf
      (headword → Entry)     (form → lemma headword)
              |
           Senses (1:N Sense)

Runtime:  played word --DB.Lookup--> Result{Entry, Kind, AlsoForm}
          layers: exact → formOf → stem(candidateStems) → fuzzy(variantCandidates)
```
