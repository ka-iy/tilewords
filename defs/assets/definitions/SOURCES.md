# Definitions asset — sources and licenses

The embedded `definitions.gob.gz` is a build artifact (gitignored) assembled from
several dictionaries. This file records each source, its license, and how to
regenerate the asset. Attribution below is a redistribution requirement of the
respective sources.

## Sources

- **Primary — Wiktionary (English), via the kaikki.org wiktextract project**
  - License: **CC BY-SA 4.0**. © Wiktionary contributors.
  - Provides the great majority of headwords and every inflection edge.
  - Built by `tools/builddefs` from the ~3 GB kaikki JSONL extract
    (`KAIKKI_EXTRACT`); see `make defs`.
  - URL: https://kaikki.org/dictionary/English/
  - Extract: https://kaikki.org/dictionary/English/kaikki.org-dictionary-English.jsonl

- **Supplement — Webster's Revised Unabridged Dictionary (1913)**
  - License: **public domain** (published pre-1929 in the United States).
  - Fills in archaic/technical headwords Wiktionary does not define.
  - Source JSON: `matthewreagan/WebstersEnglishDictionary` (`dictionary_compact.json`).
  - URL: https://github.com/matthewreagan/WebstersEnglishDictionary
  - File: https://raw.githubusercontent.com/matthewreagan/WebstersEnglishDictionary/master/dictionary_compact.json

- **Supplement — Princeton WordNet 3.1**
  - License: **WordNet License** (permissive; the notice below must be retained).
  - Fills in headwords not covered by either source above.
  - "WordNet 3.1 Copyright 2011 by Princeton University. All rights reserved.
    THIS SOFTWARE AND DATABASE IS PROVIDED 'AS IS' AND PRINCETON UNIVERSITY MAKES
    NO REPRESENTATIONS OR WARRANTIES, EXPRESS OR IMPLIED." (full text ships with
    the WordNet distribution).
  - URL: https://wordnet.princeton.edu/
  - Data: https://wordnetcode.princeton.edu/wn3.1.dict.tar.gz

- **Supplement — curated public-domain glossaries** (`defs/supplemental-glossary.tsv`)
  - A committed, reviewable TSV of glosses extracted for words the sources above do
    not define. Each entry's source is marked in a section comment. Currently:
    - **John Jamieson, *An Etymological Dictionary of the Scottish Language*** —
      public domain. Covers Scots/dialect words (`cannie`, `cranreuch`, `sheuch`,
      `leuch`).
      Project Gutenberg eBook #40521: https://www.gutenberg.org/ebooks/40521
      (text: https://www.gutenberg.org/cache/epub/40521/pg40521.txt)
    - **Glossary to Edmund Spenser's *The Faerie Queene*, Book I** — public domain,
      via Wikisource. Covers Spenserian archaisms (`teene`, `talaunts`,
      `counterfesaunce`).
      https://en.wikisource.org/wiki/The_Faerie_Queene_(unsourced)/Book_I/Glossary

## Precedence and integrity

- Supplemental sources only add a definition for a word the primary source cannot
  already resolve; existing Wiktionary headwords and inflection edges always win
  (`defs.DB.WithSupplement`).
- Only a word that is **itself** a headword in a supplemental source is added — no
  definition is inferred from a near-spelling — so supplemental glosses cannot be
  wrong for the word shown. The game's runtime de-inflection then reaches the
  regular inflected forms of those added headwords.
- Webster's 1913 takes precedence over WordNet where both define a word (its
  public-domain gloss is preferred).

## Regenerating the asset

```bash
make defs           # base asset from Wiktionary (needs KAIKKI_EXTRACT)
make defs-augment   # fold in Webster's 1913 + WordNet (needs WEBSTER_JSON, WORDNET_DICT)
                    # and the committed defs/supplemental-glossary.tsv
make defs-audit     # report per-list coverage and the remaining undefined words
```

The supplemental sources are external downloads (not committed); the download URLs
are documented in the `Makefile` next to the `defs-augment` target.
