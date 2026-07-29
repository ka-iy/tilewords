# TileWords

A free, fully open-source, fully offline Scrabble®-like crossword tile game, written in Go with [Fyne](https://fyne.io/).

See the note at the very bottom for why I felt the need to make this. And yes, this was built using AI. The note explains that too.

## What's All This, Then?

TileWords is a Scrabble®-like offline crossword tile game in which you play against the computer, using individual letters to construct cromulent words on a game board to gain the endorphin hit from watching those sweet sweet points _racking_ (heh heh) up. Plus, since it shows the definitions for words whose definitions it knows, it's a learning tool too!

TileWords uses public-domain and libre/freely available word lists and definitions. No proprietary lexicons were harmed during the making of this game.

While it is currently a fully offline game, if there is sufficient interest in adding Player-vs-Player functionality, network access (and hence the requisite permissions on mobile platforms) will of course be required when it is implemented.

### Legal Guff (a.k.a. "Please don't sue me")

SCRABBLE® is a registered trademark. All intellectual property rights in and to SCRABBLE® are owned in the U.S.A. by Hasbro Inc., in Canada by Hasbro Canada Inc. and throughout the rest of the world by J.W. Spear & Sons Ltd. of England, a subsidiary of Mattel Inc.

The "Android" name, the Android logo, the "Google Play" brand, and other Google trademarks, are property of Google LLC (whom I non-lovingly call "Gurgle").

All other external-entity trademarks, names, logos, and service marks (collectively "trademarks") in this product are the registered and unregistered trademarks of their respective owners.

**TileWords is NOT affiliated with any of the above-mentioned products or entities.**

## What's It Got?

- **Free, offline and private:** the game is open-source, requires no payment, and will never track you or show you ads. It needs no network connection and requests no internet permission. Everything runs on your device.
- **Free and libre word lists:** at the start of each game, choose from three openly-licensed dictionaries - ENABLE2K, the Wordnik word list, and the atebits "Words" (Letterpress) list. Full attribution is in the [My Word!](#my-word) section.
   - The word list artefacts created during the build process are in an optimized form to minimize disk and memory usage on mobile devices.
- **Word definitions:** the meaning of each word played is shown where a definition is available, drawn from Wiktionary, Webster's 1913, WordNet, and other libre / public-domain glossaries. Words with no definition are noted rather than silently skipped.
   - Like the word lists, the definitions artefact is stored in a compact flat form and streamed in at load, so ~147,000 headwords cost about 19 MB of memory rather than the ~52 MB a conventional map of objects would. See the appendix to [OPTIMIZING_THE_GADDAGS.md](OPTIMIZING_THE_GADDAGS.md).
- **Two game modes:** Classic Mode uses the standard 15x15 premium-square layout and tile economy, while Interesting Mode uses an alternative pinwheel (4-fold rotational) layout with a different tile distribution and per-tile points. A preview shows each mode's board and tiles before you start.
- **Selectable AI difficulty:** choose how strongly the computer opponent plays, from 1 (easy) to 10 (hard). Even at level 10, the AI is not infallible - just like a human being. At level 11, it is - this is God Mode (for the AI, not you). To quote from `This is Spinal Tap`: _"Why don't you just make 10 louder and make 10 be the top number?"_ **/** _"...These go to ELEVEN!"_
- **Multiple ways to interact with the tiles and game board:** drag a tile onto the board, or click/press on the tile and then the board square to place it at that square. Double-tap a placed but not played board tile to return it to your rack. To recall all unplayed tiles to the rack, use the recall icon on top of your rack (looks like a download icon. Why? Because that's all that was available in the toolkit).
- **Move history:** a running log of every turn - who played, the words formed, and the score - with an option to show it in standard Scrabble notation.
- **Copy to clipboard:** the move-history and definitions panels are copyable. On desktop, select and copy; on a phone, long-press to copy the whole panel, while a finger-drag scrolls and a double or triple tap selects a word or line.
   - For convenience and visibility, a dedicated "Copy" button is also provided in the game UI which will copy the contents of the active tab (Move history or Definitions) to the system clipboard.
- **Undo:** take back the last full round - your move together with the computer's reply.
- **Save and restore:** keep a single saved game and resume it later. The save captures the board, racks, scores, move history, and game mode.
- **Remembered setup defaults:** optionally save your New Game choices - word list, game mode, difficulty, and notation - so that starting another game is a single tap.
- **Cross-platform:** runs on desktop and Android from a single codebase. Written in Go using the Fyne UI toolkit. MacOS may be supported at some time in the future, but iOS will probably never be - the Apple Store hates open source, the GPL, and developers who don't kowtow to them. Gurgle and Android are going the same way, sadly, in a horrifying game of simian mirror-neuron idiocy.
- **About and Lexicon:** an in-app dialog crediting the word lists and dictionaries, with the source links copyable to the clipboard.
- **Show/Hide AI rack:** for those times when you need to get a leg (or at least a toe) up on a frighteningly capable machine opponent.

## How's It Built (on Linux)?

The three word lists (`wordlists/*.txt`) and the prebuilt definitions asset
(`defs/assets/definitions/definitions.bin.gz`) are committed to the repository, so a
from-scratch build is simply: install the [prerequisites](#prerequisites), then run
`make`. The [word list](#fetch-the-word-lists) and [definitions](#fetch-and-build-the-definitions)
sections below are only needed to refresh or regenerate that data.

### Prerequisites

- **Go 1.25 or newer** (see `go.mod`). If your distribution ships an older Go, install a current release from <https://go.dev/dl/>.
- A **C toolchain** and **OpenGL / X11 development headers** (required by Fyne).
- The **fyne CLI**, which every build target in the Makefile drives:

  ```bash
  go install fyne.io/tools/cmd/fyne@latest
  ```

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
`wordlists/<name>.txt` is compiled into a GADDAG automatically at build time. One target
fetches all three from their upstream sources:

```bash
make download-wordlists
```

- **ENABLE2K** (public domain) - fetched and decompressed.
- **Wordnik word list** (MIT) - fetched, with upstream's surrounding quotes stripped.
- **atebits "Words"** (CC0) - fetched as-is.

Only a list that is **missing** is fetched, so this never silently replaces a committed
copy. To refresh one from upstream, delete it first:

```bash
rm wordlists/wordnik.txt && make download-wordlists
```

Adding a fourth list is just a matter of dropping `wordlists/<name>.txt` in place - it is
discovered and compiled automatically - and registering `<name>` in
`dictionary.AllDictNames` so the game offers it in the setup menu.

### Fetch and build the definitions (Optional)

The definitions asset is committed, so this step is optional too. It is only needed to
rebuild the in-game word definitions from scratch. One target does the whole job -
fetching every source it does not already have, building the base asset, and folding in
the supplements:

```bash
make defs
```

That assembles the asset from four sources, each with its own format and licence (all
recorded in [`defs/assets/definitions/SOURCES.md`](defs/assets/definitions/SOURCES.md)):

- **Wiktionary**, via the kaikki.org wiktextract JSONL extract (CC BY-SA 4.0) - the
  primary source, providing most headwords and every inflection edge.
- **Webster's Revised Unabridged Dictionary, 1913** (public domain) - archaic and
  technical headwords Wiktionary does not define.
- **Princeton WordNet 3.1** (WordNet licence) - headwords neither source above covers.
- **Curated public-domain glossaries**, committed as `defs/supplemental-glossary.tsv`
  (Jamieson's Scots dictionary and a Spenser glossary). Hand-checked, so it lives in the
  repository rather than being scraped at build time.

The first three are downloaded on demand and are git-ignored rather than committed. Be
aware that the Wiktionary extract alone is several GB, so a run that has to fetch it takes
a long time. Each source is fetched only when missing, so set `KAIKKI_EXTRACT`,
`WEBSTER_JSON` or `WORDNET_DICT` to reuse copies you already have:

```bash
make defs KAIKKI_EXTRACT=/path/to/kaikki-en.jsonl
```

A supplement only ever adds a word the primary source cannot resolve, and only when that
word is itself a headword in the supplement - no definition is inferred from a
near-spelling.

`make defs` regenerates the asset from whatever the sources on disk hold at that moment
rather than adding to the existing asset, so a rebuild tracks its sources: if any of them
has been revised since the last build, the new asset reflects that. Note that a source is
only downloaded when it is **missing**, so a re-run does not pick up an upstream revision
by itself - delete the local copy (or point the variable at a new one) to refresh it:

```bash
rm wordlists/kaikki-en.jsonl && make defs   # rebuild against a current Wiktionary extract
```

Report per-list coverage and the words that still have no definition:

```bash
make defs-audit
```

### Build and run

```bash
make              # (or: make linux) debug build of the Linux desktop binary
make linux-prod   # production build: stamped as production, stripped, -trimpath
```

Binaries are named for the platform they were built for, `tilewords-<goos>-<goarch>`, with
`-debug` appended for a debug build — so on 64-bit Linux:

```bash
./tilewords-linux-amd64-debug     # launch the debug build
./tilewords-linux-amd64           # launch the production build
```

A debug and a production build therefore sit side by side rather than overwriting each
other. Every binary reports which it is on startup and in its About dialog, so there is
no need to guess.

Other useful targets:

```bash
make test                    # run all tests with the race detector
make vet                     # run go vet
make clean                   # remove built binaries and packages
make clean-defs-sources      # remove the downloaded definition sources (frees GBs)
make clean-all-the-things    # the above plus every generated asset (needs re-downloads)
make debug-all               # debug build for desktop + Windows + Android
make release-all             # release build for desktop + Windows + Android
make help                    # list every target with a description
```

### Windows builds (optional)

A Windows `.exe` can be cross-compiled from Linux. Fyne's GUI is C-backed (GLFW/OpenGL), so
this needs cgo and a **mingw-w64** cross compiler — but nothing more than that: the Windows
libraries the GUI links against (`opengl32`, `gdi32`, `user32`) ship with mingw-w64, so
unlike a Linux build there are no separate development headers to install.

Debian / Ubuntu:

```bash
sudo apt-get install -y gcc-mingw-w64-x86-64
```

Fedora:

```bash
sudo dnf install -y mingw64-gcc
```

Arch:

```bash
sudo pacman -S --needed mingw-w64-gcc
```

Then build:

```bash
make windows-debug      # → ./tilewords-windows-amd64-debug.exe
make windows-release    # → ./tilewords-windows-amd64.exe  (stripped)
```

The result is a single self-contained executable: it imports only DLLs that are part of
Windows itself, so nothing has to be shipped alongside it. It is linked as a GUI binary, so
Windows does not open a console window behind the app, and `ui/Icon.png` is embedded as the
executable's own icon.

For a 32-bit build, install the 32-bit compiler and point the build at it — the architecture
is part of the artifact's name, so both can coexist:

```bash
sudo apt-get install -y gcc-mingw-w64-i686
make windows-debug WINDOWS_CC=i686-w64-mingw32-gcc WINDOWS_GOARCH=386
```

Note that a cross-compiled binary cannot be run or tested on the Linux host — `make test`
only exercises the native build, so the Windows `.exe` is verified as compiling, not as
running. Launch it on Windows to confirm behaviour.

### Android builds (optional)

Android builds need the Android SDK and NDK. Install the build CLIs and point the
environment at your SDK/NDK:

```bash
make install-mobile-tools     # install the fyne + gomobile CLIs
# then set ANDROID_HOME and ANDROID_NDK_HOME to your SDK/NDK locations
```

**Debug APK** (self-signed with a throwaway debug key, for local testing):

```bash
make android                  # debug APK for arm64-v8a
```

**Signed release build.** A release build must be signed, so it **requires a valid Java
keystore and signing certificate** that you generate yourself and keep private (never
commit it — `*.keystore` is git-ignored). Create one with the JDK's `keytool`:

```bash
keytool -genkey -v -keystore release.keystore \
        -alias tilewords -keyalg RSA -keysize 2048 -validity 10000
```

Then build, supplying the keystore path, its password, and the key alias (these default
to `release.keystore` / `changeme` / `tilewords`, so override at least `KEYSTORE_PASS`):

```bash
# signed .aab for all ABIs
make android-release \
  KEYSTORE=release.keystore KEYSTORE_PASS=<your-password> KEY_ALIAS=tilewords

# signed universal APK (needs bundletool on PATH: brew install bundletool)
make android-release-apk \
  KEYSTORE=release.keystore KEYSTORE_PASS=<your-password> KEY_ALIAS=tilewords
```

Run `make help` for the full list of per-ABI debug and release targets.

## My Word!

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

## A note from the (original) developer

This project was started as an experiment in using the [AWS AI-DLC framework](https://github.com/awslabs/aidlc-workflows) and Claude Code to build a word game with all the features I always wanted in my ideal word-game...uh...game _("What the hell does ZAX mean??")(It's a construction tool)_ but which I had not been able to find consolidated in one single game.

As an addendum to the experiment, I wanted to see whether I could do this without firing up my editor or manually changing stuff. I almost succeeded in that endeavor. Almost.

My takeaways thus far (as of August 2026):
- Agentic coding is definitely a development accelerator **provided that** the AI is constantly hand-held, stopped from going down senseless paths, steered in the correct direction, and generally treated like a precocious idiot-savant tween.
   - Thorough testing of _ALL THE THINGS_, followed by follow-up prompts to fix things, is pretty much _de rigeur_ for getting workable stuff out of an AI. And this wasn't even that complicated a project. OK, the UI was, since I'm not a UI guy - it actually did a decent job _(well, decent to a non-GUI guy, anyway)_ of coming with a rough initial layout which I could then use as a base to tweak.
   - It is, however,really good at typing much quicker than one would be able to even if one were able to code telepathically, plus (kinda) grok large wodges of code and come up with a halfway-sensible meaning of it all. It shaved literal months off the dev timeline as compared to my going at it alone. Also see below.
- Claude Sonnet _(whatever the version of it was that was available to Pro subscriptions before Opus 4.8 landed)_ did an **absolute shite** job of writing unsupervised code - buggy, non-functional, and generally quite useless.
  - Opus 4.8 (at effort level `high`) - the current avatar of Anthropic's offering available to folks without deep pockets -  was/is **much** better at everything, but _man_ does it eat up one's token/usage allotment like a starving Great White Shark let loose in a(n aquatic) pet store. It ate my 5-hour session allotment at roughly 1% per minute, getting about two hours' worth of work before it sat down on its hands and refused to do anything until the session usage reset. I shudder to think what Fable would have done to my usage rate - probably blown through it in about five minutes flat, for no appreciable improvement over Opus 4.8 (this last bit isn't conjecture - I tried Fable at work; suffice to say that I was less than impressed by it).
- AI-DLC is alright, but perhaps needs a bit more time to mature. Also, it is verbose as all hell, which is probably OK for Enterprise(tm) Development.
  - I think I'll follow the [BMAD](https://docs.bmad-method.org/) AI SLDC framework for future development especially since that lends itself well to collaborative efforts. I'll leave the `aidlc-docs` directory in the sources as a historical record of shenanigans perpetrated.

I also had lots of fun getting the AI to write a pseudoacademic "paper" - ( ͡° ͜ʖ ͡°) - on the optimization strategies used to reduce the size of the lexicon/glossary assets.
