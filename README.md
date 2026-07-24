# TileWords

A Scrabble®-like offline crossword tile game, written in Go with [Fyne](https://fyne.io/).

## About

TileWords is a Scrabble®-like offline crossword tile game in which you play against the computer, using individual letters to construct cromulent words on a game board to gain the endorphin hit from watching those sweet sweet points _racking_ (heh heh) up. Plus, since it shows the definitions for words whose definitions it knows, it's a learning tool too!

TileWords uses public-domain and libre/freely available word lists and definitions. No proprietary lexicons were harmed during the making of this game.

### Disclaimer (a.k.a. "Please don't sue me")

SCRABBLE® is a registered trademark. All intellectual property rights in and to SCRABBLE® are owned in the U.S.A. by Hasbro Inc., in Canada by Hasbro Canada Inc. and throughout the rest of the world by J.W. Spear & Sons Ltd. of England, a subsidiary of Mattel Inc.

**TileWords is NOT affiliated with any of the above-mentioned products or entities.**

## Features

- **Offline and private:** the game needs no network connection and requests no internet permission. Everything runs on your device.
- **Free and libre word lists:** at the start of each game, choose from three openly-licensed dictionaries - ENABLE2K, the Wordnik word list, and the atebits "Words" (Letterpress) list. Full attribution is in the [Lexicon](#lexicon).
   - The word list artefacts created during the build process are in an optimized form to minimize disk and memory usage on mobile devices.
- **Word definitions:** the meaning of each word played is shown where a definition is available, drawn from Wiktionary, Webster's 1913, WordNet, and public-domain glossaries. Words with no definition are noted rather than silently skipped.
- **Two game modes:** Classic uses the standard 15x15 premium-square layout and tile economy, while Interesting uses an alternative pinwheel (4-fold rotational) layout with a different tile distribution and per-tile points. A preview shows each mode's board and tiles before you start.
- **Selectable AI difficulty:** choose how strongly the computer opponent plays, from 1 (easy) to 10 (hard).
- **Move history:** a running log of every turn - who played, the words formed, and the score - with an option to show it in standard Scrabble notation.
- **Copy to clipboard:** the move-history and definitions panels are copyable. On desktop, select and copy; on a phone, long-press to copy the whole panel, while a finger-drag scrolls and a double or triple tap selects a word or line.
   - For convenience and visibility, a dedicated "Copy" button is also provided in the game UI which will copy the contents of the active tab (Move history or Definitions) to the system clipboard.
- **Undo:** take back the last full round - your move together with the computer's reply.
- **Save and restore:** keep a single saved game and resume it later. The save captures the board, racks, scores, move history, and game mode.
- **Remembered setup defaults:** optionally save your New Game choices - word list, game mode, difficulty, and notation - so that starting another game is a single tap.
- **Cross-platform:** runs on desktop and Android from a single codebase.
- **About and Lexicon:** an in-app dialog crediting the word lists and dictionaries, with the source links copyable to the clipboard.

## Building on Linux

The three word lists (`wordlists/*.txt`) and the prebuilt definitions asset
(`defs/assets/definitions/definitions.gob.gz`) are committed to the repository, so a
from-scratch build is simply: install the [prerequisites](#prerequisites), then run
`make`. The [word list](#fetch-the-word-lists) and [definitions](#fetch-and-build-the-definitions)
sections below are only needed to refresh or regenerate that data.

### Prerequisites

- **Go 1.25 or newer** (see `go.mod`). If your distribution ships an older Go, install a current release from <https://go.dev/dl/>.
- A **C toolchain** and **OpenGL / X11 development headers** (required by Fyne).
- `make`, `git`, `curl`, `gzip`, and `tar`.

Debian / Ubuntu:

```bash
sudo apt-get update
sudo apt-get install -y golang-go gcc pkg-config libgl1-mesa-dev xorg-dev make git curl
```

Fedora:

```bash
sudo dnf install -y golang gcc pkgconf-pkg-config mesa-libGL-devel \
  libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel \
  libXxf86vm-devel make git curl
```

Arch:

```bash
sudo pacman -S --needed go gcc pkgconf mesa libxcursor libxrandr libxinerama libxi make git curl
```

### Get the source

```bash
git clone https://github.com/ka-iy/tilewords.git
cd tilewords
```

### Fetch the word lists (Optional)

The word lists are already committed, so this step is optional. Each
`wordlists/<name>.txt` is compiled into a GADDAG automatically at build time. To
refresh them from their upstream sources:

- **ENABLE2K** (public domain) - downloaded and decompressed automatically:

  ```bash
  make download-wordlists
  ```

- **Wordnik word list** (MIT) - the upstream file is quoted, so strip the quotes:

  ```bash
  curl -fsSL https://raw.githubusercontent.com/wordnik/wordlist/main/wordlist-20210729.txt \
    | tr -d '"' > wordlists/wordnik.txt
  ```

- **atebits "Words"** (CC0):

  ```bash
  curl -fsSL https://raw.githubusercontent.com/atebits/Words/master/Words/en.txt \
    > wordlists/atebits-letterpress.txt
  ```

### Fetch and build the definitions (Optional)

The definitions asset is committed, so this step is optional too. It is only needed
to rebuild the in-game word definitions from scratch. The asset is assembled in two
stages; run them in order (a fresh `make defs` overwrites the committed asset, and
`make defs-augment` then folds the extra sources back in).

**1. Base definitions from Wiktionary.** Download the kaikki wiktextract English
extract (large, ~3 GB) to the default path, then build. `builddefs` filters it down
to just the words the committed lists can form:

```bash
curl -fsSL -o wordlists/kaikki-en.jsonl \
  https://kaikki.org/dictionary/English/kaikki.org-dictionary-English.jsonl
make defs
```

**2. Augment with the supplemental sources.** Fetch Webster's 1913 and WordNet to
their default paths, then fold them (plus the committed Scots/Spenser glossary in
`defs/supplemental-glossary.tsv`) into the asset for words Wiktionary does not define:

```bash
# Webster's 1913 (public domain)
curl -fsSL -o wordlists/webster1913.json \
  https://raw.githubusercontent.com/matthewreagan/WebstersEnglishDictionary/master/dictionary_compact.json

# WordNet 3.1 (extract so that wordlists/wordnet/dict/ exists)
curl -fsSL -o /tmp/wn31.tar.gz https://wordnetcode.princeton.edu/wn3.1.dict.tar.gz
mkdir -p wordlists/wordnet && tar -xzf /tmp/wn31.tar.gz -C wordlists/wordnet

make defs-augment
```

Report per-list coverage and the words that still have no definition:

```bash
make defs-audit
```

### Build and run

```bash
make          # (or: make build) compiles the ./tilewords desktop binary
make run      # build and launch the app
```

Other useful targets:

```bash
make test     # run all tests with the race detector
make vet      # run go vet
make help     # list every target with a description
```

### Android builds (optional)

```bash
make install-mobile-tools     # install the fyne + gomobile CLIs
# then set ANDROID_HOME and ANDROID_NDK_HOME to your SDK/NDK locations
make android                  # debug APK for arm64-v8a
```

Signed release bundles/APKs require a keystore; see `make help` and the comments in
the `Makefile` for the release targets.

## Lexicon

TileWords is built on freely available word lists and dictionaries. Grateful acknowledgement is made to the authors and maintainers of the sources below, whose work makes possible this game and the lexicon it uses.

**Word lists**

- **ENABLE2K:** the Enhanced North American Benchmark LExicon (ENABLE), 2K edition - a Scrabble®-style word list compiled by Alan Beale and others, placed in the public domain by its creators. <https://github.com/BartMassey/wordlists>
- **Wordnik word list:** Copyright (c) 2020 Wordnik, used under the MIT License. <https://github.com/wordnik/wordlist>
- **atebits "Words" (Letterpress word list):** by Loren Brichter / atebits, released under CC0 1.0 (public domain dedication). <https://github.com/atebits/Words>

**Definitions**

- **Wiktionary (English):** © Wiktionary contributors, via the kaikki.org wiktextract project, used under CC BY-SA 4.0. Definitions derived from Wiktionary remain available under the same license. <https://kaikki.org/dictionary/English/>
- **Webster's Revised Unabridged Dictionary (1913):** public domain. <https://github.com/matthewreagan/WebstersEnglishDictionary>
- **Princeton WordNet 3.1:** Copyright 2011 by Princeton University, all rights reserved, used under the WordNet License (permissive, with attribution). <https://wordnet.princeton.edu/>
- **An Etymological Dictionary of the Scottish Language:** by John Jamieson, public domain (Project Gutenberg eBook #40521). <https://www.gutenberg.org/ebooks/40521>
- **Glossary to Edmund Spenser's "The Faerie Queene", Book I:** public domain, via Wikisource. <https://en.wikisource.org/wiki/The_Faerie_Queene_(unsourced)/Book_I/Glossary>

Definitions are shown for reference during play. Where more than one source defines a word, the Wiktionary sense is preferred; Webster's 1913, WordNet, and the glossaries fill gaps for archaic and dialectal words.

While (almost) every effort has been made to fill the gaps in the definitions, gaps do remain; where a played word has no definition that could be found, such will be indicated in the "Definitions" tab on the game screen.

----------

## A note from the original developer

This project was started as an experiment in using the [AWS AI-DLC framework](https://github.com/awslabs/aidlc-workflows) and Claude Code to build a word game with the features I always wanted _("What the hell does ZAX mean??")(It's a construction tool)_ but which I had not found in comparable projects. As an addendum to the experiment, I wanted to see whether I could do this without firing up my editor or manually changing stuff. I almost succeeded at that. Almost.

My takeaways thus far (as of July 2026):
- Agentic coding is definitely a development accelerator **provided that** the AI is constantly hand-held, stopped from going down senseless paths, steered in the correct direction, and generally treated like a precocious idiot-savant tween.
   - Thorough testing of _ALL THE THINGS_, followed by follow-up prompts to fix things, is pretty much _de rigeur_ for getting workable stuff out of an AI. And this wasn't even that complicated a project. OK, the UI was, since I'm not a UI guy - it actually did a decent job of the initial layout. Well, decent to a non-GUI guy, anyway.
   - It is, however,really good at typing much quicker than one would be able to even if one were able to code telepathically, plus grok (kinda) large wodges of code and come up with a halfway-sensible meaning of it all. Also see below.
- AI-DLC is alright, but perhaps needs a bit more time to mature. Also, it is verbose as all hell, which is probably OK for Enterprise(tm) Development.
  - I think I'll follow the [BMAD](https://docs.bmad-method.org/) AI SLDC framework for future development especially since that lends itself well to collaborative efforts. I'll leave the `aidlc-docs` directory in the sources as a historical record of shenanigans perpetrated.
- Claude Sonnet _(whatever the version of it was that was available to Pro subscriptions before Opus 4.8 landed)_ did a **horrible** job of writing code - buggy, non-functional, and generally quite useless.
  - Opus 4.8 (at effort level `high`) - the current avatar of Anthropic's offering available to folks without deep pockets -  was/is **much** better at everything, but _man_ does it eat up one's token/usage allotment like a starving Great White Shark let loose in a(n aquatic) pet store. It ate my 5-hour session allotment at roughly 1% per minute, getting about two hours' worth of work before it sat down on its hands and refused to do anything until the session usage reset. I shudder to think what Fable would have done to my usage rate - probably blown through it in about five minutes flat, for no appreciable improvement over Opus 4.8 (this last bit isn't conjecture - I tried Fable at work; suffice to say that I was less than impressed by it).

