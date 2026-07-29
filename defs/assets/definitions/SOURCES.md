# Definitions asset — sources and licenses

The embedded `definitions.bin.gz` is a build artifact assembled from several
dictionaries. It is committed, so building and running the game needs none of the
sources below — they are only required to regenerate it. This file records each source,
its license, and how to regenerate the asset. Attribution below is a redistribution
requirement of the respective sources.

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
      `leuch`, `taigle`, `drouk`, `kae`).
      Project Gutenberg eBook #40521: https://www.gutenberg.org/ebooks/40521
      Where Jamieson carries two entries for one headword, both senses are given
      (`blume`, `tirr`). Regional markers ("S.", "S. B.", "Galloway") are dropped, and
      the archaic verb spelling "reach" is given as "retch" (`cowk`).
    - **Webster's New Modern English Dictionary** (Consolidated Book Publishers, 1922) —
      public domain; every copyright date in the volume is 1908-1922, so US copyright has
      expired. OCR text from the Internet Archive, item `webstersnewmoder00web`. Because OCR
      output cannot be trusted unchecked, each gloss was read back against the source page
      (`befana`, `eboulement`, `siffleur`, `stegomyia`).
      https://archive.org/details/webstersnewmoder00web
    - **Joseph Petree's OWL 2.1 (2016) from WOW24 ** — definitions by **Joseph
      Petree**, obtained via the WOW24 import compiled by Mitch Bayersdorfer and
      released under **CC BY-SA 4.0** (© 2026 Mitch Bayersdorfer). This is the only source
      found that defines the Scrabble-tournament vocabulary absent from every general
      dictionary (`acta`, `aerosat`, `pistolero`, `wab`), and supplies the large majority of
      this glossary.
      https://pdxscrabble.neocities.org/word-lists/WOW24-Dictionary-Import
      Joseph Petree's permission requires that his name appear in the definitions file and
      that the text not be severely altered, so his definitions are kept verbatim and only
      the database's formatting notation is stripped (inflection tables, cross-reference prefixes,
      the `~` non-tournament flag, provenance markers).
      Note: the bundle's README attaches a non-commercial note to its contents, deriving from
      WGPO's terms for the **WOW24 word list**. That word list is not used here — only Joseph
      Petree's definition text, keyed to this project's own word lists — but anyone redistributing
      this asset commercially should confirm that reading for themselves.
    - **The Online Plain Text English Dictionary (OPTED) v0.03**, Ralph S. Sutherland —
      public domain, derived from the Project Gutenberg etext of Webster's Unabridged
      Dictionary (1913), the same 1913 edition already credited above. Used only for the
      irregular Latin plurals it records as cross-references and that no de-inflection rule
      derives (`animi`, `semina`, `vertigines`); each entry names the singular and gives that
      singular's OPTED definition. OPTED asks that OPTED, Project Gutenberg and Webster's
      1913 all be acknowledged, which this section and the Webster entry above do.
      https://www.mso.anu.edu.au/~ralph/OPTED/
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
make defs        # build the complete asset from every source above
make defs-audit  # report per-list coverage and the remaining undefined words
```

`make defs` does the whole job in one pass: it fetches any source it does not already
have, builds the base asset from the Wiktionary extract, then folds in Webster's 1913,
WordNet and the committed glossary. The downloaded sources are third-party data and are
NOT committed (they are git-ignored); each is fetched only when absent, so point
`KAIKKI_EXTRACT`, `WEBSTER_JSON` or `WORDNET_DICT` at copies you already have to reuse
them. Expect the first run to take a long time — the Wiktionary extract alone is
several GB.

The download URLs are the ones listed under each source above, and are defined next to
the `defs` target in the `Makefile`.

`make defs` regenerates the asset from the sources as they stand on disk; it does not
add to the existing asset. A rebuild therefore tracks its sources — if any has been
revised since the last build, the coverage changes with it. Only when the sources are
unchanged does a rebuild reproduce the same coverage.

Because sources are fetched only when absent, an upstream revision is not picked up on
its own. Delete the local copy, or point `KAIKKI_EXTRACT` / `WEBSTER_JSON` /
`WORDNET_DICT` at a newer one, to rebuild against current data.

Two consequences of that worth knowing, since this asset is committed:

- Rebuilding is reproducible: the asset is written from flat arrays in a fixed order, so
  the same inputs produce byte-identical output (guarded by
  `defs.TestEncodeIsDeterministic`). A rebuild that leaves a diff in git therefore means
  the sources genuinely changed — worth reading, rather than the noise a rebuild used to
  produce. `make defs-audit` reports what the change did to coverage.
- Re-running the merge step alone over an already-augmented asset does add nothing, as a
  supplement can only fill a gap the primary source leaves. That is a property of the
  merge, not of `make defs` as a whole.
