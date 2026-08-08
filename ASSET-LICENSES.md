# Licensing of TileWords

TileWords is two different kinds of thing under one roof, and they are licensed
separately:

| Part | What it is | Licence |
|---|---|---|
| **Code** | everything under `cpu/`, `buildinfo/`, `cmd/`, `defs/`, `dictionary/`, `engine/`, `ui/`, `tools/`, plus the `Makefile` and build scripts | **GPL-3.0-or-later** — see [`LICENSE`](LICENSE) |
| **Word-list assets** | `dictionary/assets/dictionaries/*.bin` (generated GADDAGs) | terms of the source word lists — see below |
| **Definitions asset** | `defs/assets/definitions/definitions.bin.gz`, and its reviewable source `defs/supplemental-glossary.tsv` | **CC BY-SA 4.0** — see below |

Copyright © 2026 Kartikeya IYER, for the code.

The data assets are **not** covered by the GPL. They are third-party works, shipped
alongside the program rather than derived from it, and each carries the terms its
source imposes. Redistributing TileWords means honouring both columns.

`defs/assets/definitions/SOURCES.md` is the authoritative, per-source record with
full attribution and regeneration instructions. `LEXICON.txt` carries the same
attribution in the form shown to players in the About dialog. This file is the
summary of how the two licences divide.

## Why the split

The definitions asset contains Wiktionary text under CC BY-SA 4.0. ShareAlike means
adaptations must stay under CC BY-SA 4.0 or a licence Creative Commons has declared
compatible — a list that contains only the Free Art License 1.3 and the GNU GPLv3, and
in one direction only. Keeping the asset under CC BY-SA 4.0 rather than sweeping it
into the GPL avoids relying on that one-way route, and keeps the obligations owed to
each upstream source unambiguous.

## Word lists

Each shipped `.bin` is a GADDAG built from one word list. The generated file is a
transformation of that list, so the list's terms travel with it:

- **ENABLE2K** (`enable.bin`) — public domain, compiled by Alan Beale and others.
- **Wordnik word list** (`wordnik.bin`) — Copyright © 2020 Wordnik, MIT License.
- **atebits "Words"** (`atebits-letterpress.bin`) — Loren Brichter / atebits, CC0 1.0.

Lexicons and glossaries with non-libre copyright are deliberately **not** bundled.
F Big Words, amirite guise?

## Definitions

`definitions.bin.gz` is a merge of several dictionaries. The most restrictive terms
in the mix are CC BY-SA 4.0, so the asset as a whole is offered under **CC BY-SA
4.0**: attribute the sources, keep adaptations under the same licence, add no further
restrictions, and apply no DRM that would stop a recipient exercising those rights.

- **Wiktionary (English)**, via kaikki.org wiktextract — CC BY-SA 4.0. Supplies most
  headwords and every inflection edge.
- **Extended Free Zyzzyva Dictionary (OWL 2.1)** — definitions by **Joseph Petree**,
  via the WOW24 Zyzzyva import compiled by Mitch Bayersdorfer and released under
  CC BY-SA 4.0. His permission asks that his name appear in the definitions file and
  that the text not be severely altered; his wording is kept verbatim and only
  Zyzzyva's formatting notation is stripped. See the caveat in `SOURCES.md`
  concerning the bundle's non-commercial note, which derives from WGPO's terms for the
  WOW24 *word list* — a list this project does not use.
- **Webster's Revised Unabridged Dictionary (1913)** — public domain.
- **Webster's New Modern English Dictionary (1922)** — public domain.
- **Princeton WordNet 3.1** — WordNet License; the copyright notice must be retained.
- **Jamieson's Etymological Dictionary of the Scottish Language** — public domain.
- **Glossary to Spenser's "The Faerie Queene", Book I** — public domain.
- **The Online Plain Text English Dictionary (OPTED) v0.03** — public domain.

## Notes for anyone redistributing

- **Aggregation, not linkage.** The assets are embedded with `//go:embed`, but they are
  data, not code linked into the program. GPLv3 section 5 treats a compilation of a
  covered work with separate independent works as an aggregate, and the assets stay
  under their own terms.
- **The generators are code.** `tools/builddefs`, `tools/mergedefs`,
  `tools/buildgaddag` and the `defs`/`dictionary` packages are GPL-3.0-or-later, so a
  distributed modification must publish them. That is what makes the assets
  reproducible rather than opaque; `SOURCES.md` documents the commands.
- **App stores that apply DRM.** CC BY-SA 4.0 forbids applying technological measures
  that prevent recipients exercising the licensed rights, with no exception for also
  offering an unrestricted copy. Distributing the definitions asset inside a
  DRM-wrapped store binary is therefore doubtful — independently of the code licence,
  and not something the copyright holder of this project can waive, since the
  definitions are not his to relicense.
