# How to build all this stuff (on Linux)

This document is written for building on Linux. You can probably adapt the instructions here to
build on Windows or on a Mac, although that's not been tested and of course you or may not have
the libraries required to cross-compile for the other desktop operating systems.

A `Makefile` is provided as your one-stop-shop build harness.
Run `make help` to see the list of available build targets.
You should _make_ sure (heh heh) to install the pre-requisites before using the `Makefile`.

The three word lists (`wordlists/*.txt`) are committed to the repository, so a from-scratch
build is: install the [prerequisites](#prerequisites), `make install-fyne-cli` for the
patched fyne CLI every target drives, then run `make`.

The definitions asset (`defs/assets/definitions/definitions.bin.gz`) is **not** committed —
F-Droid does not accept a prebuilt binary asset in a source tree — so every build target
builds it when it is missing, fetching the sources it is made from. **Be aware that this
means a first build downloads several GB**, the Wiktionary extract being nearly all of it,
and takes a correspondingly long time. Once the asset exists it is left alone; `make test`
and `make vet` never ask for it, and `make defs` is what deliberately regenerates one. See
[Fetch and build the definitions](#fetch-and-build-the-definitions) for how to point the
build at copies of the sources you already have.

### Prerequisites

- **Go 1.25 or newer** (see `go.mod`). If your distribution ships an older Go, install a current release from <https://go.dev/dl/>.
- A **C toolchain** and **OpenGL / X11 development headers** (required by Fyne).
- The **fyne CLI**, which every build target in the Makefile drives. It must be the
  **forked and patched** build from the `honor-user-ldflags` branch of
  <https://github.com/ka-iy/fyne-tools>, **not** the upstream `fyne.io/tools` one — every
  target here needs it, desktop and Android alike. One target installs it:

  ```bash
  make install-fyne-cli
  ```

  That clones the fork into a temporary directory, switches to the branch, runs
  `go install ./cmd/fyne`, and throws the clone away — so there is no checkout to keep
  up to date. It prints where the binary landed (`go env GOBIN`, or `$(go env GOPATH)/bin`
  when `GOBIN` is unset), and warns if that directory is not on your `PATH`, which it must
  be for the build targets to find it.

  The fork fixes a few things this build needs that upstream does not do:

  - it honours the linker flags the Makefile passes through `GOFLAGS` on **every** target —
    Linux, Windows and Android — which is what stamps the version, build type and build
    timestamp into the artifact; built with the upstream CLI, every artifact reports the
    `buildinfo` defaults instead;
  - for Android it signs debug APKs with the v1/v2/v3 schemes that `targetSdk` 30 and newer
    require, and emits the `<apk>.idsig` (v4) sidecar that lets `adb install` take the
    incremental path.

  **If you ever install upstream's CLI** — `go install fyne.io/tools/cmd/fyne@latest`, or
  anything that runs it for you — it overwrites the patched one, because both write the same
  `fyne` binary and the last one installed wins. Run `make install-fyne-cli` again to put the
  patched CLI back.

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
### The quick-n-dirty build instructions

```bash
make clean
make debug-all # The first build will take a while
```

### The in-excruciating-detail build instructions

**Note:** To bump up the app version numbering for a release, see the [Bumping the version](#Bumping%20the%20version) section below.

#### Fetch the word lists (Optional)

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

#### Fetch and build the definitions

The definitions asset is not committed, so a build makes one when it is missing. Running
this by hand is only needed to regenerate an asset that already exists, or to fetch the
sources ahead of a build. One target does the whole job - fetching every source it does not
already have, building the base asset, and folding in the supplements:

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
- **The committed glossary**, `defs/supplemental-glossary.tsv` - 8,889 entries that
  close the rest of the gap. It is committed as plain reviewable text, rather than being
  scraped at build time, so every gloss the game ships can be read and corrected. Most of
  it is the obscure tournament vocabulary the three sources above simply do not carry; the
  remainder comes from curated public-domain glossaries (Jamieson's Scots dictionary, a
  Spenser glossary, and others). Each section names its own source.

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

As of the current asset that reports 100% on all three lists. The only words it still
counts as missing are the ones listed in `defs/possibly_invalid_words.txt`, which
`buildgaddag` drops from the compiled dictionaries, so they are never playable.

#### Build and run

```bash
make               # (or: make linux) debug build of the Linux desktop binary
make linux-release # production build: stamped as production, stripped, -trimpath
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
make debug-all               # debug build for Linux + Windows + Android
make release-all             # release build for Linux + Windows + Android
make help                    # list every target with a description
```

#### Windows builds (optional)

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

#### Android builds (optional)

Android builds need the Android SDK and NDK. Install the build CLIs and point the
environment at your SDK/NDK:

```bash
make install-mobile-tools     # installs the patched fyne CLI + gomobile
# then set ANDROID_HOME and ANDROID_NDK_HOME to your SDK/NDK locations
```

That target includes `install-fyne-cli`, so it installs the patched fyne CLI (see
[Prerequisites](#prerequisites)) rather than replacing it with upstream's; gomobile is
upstream's own.

**Debug APK** (self-signed with a throwaway debug key, for local testing):

```bash
make android-debug            # 4 debug APKs (see below)
```

That builds four APKs: one for each of the three ABIs, plus a universal one holding every
ABI in a single file (roughly four times the size). To iterate on a device, build just the
one you need — `make android-debug-arm64-v8a` for a modern phone, or
`make android-debug-x86_64` for an emulator.

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
# 4 .aab files: one per ABI, plus one universal holding them all
make android-release \
  KEYSTORE=release.keystore KEYSTORE_PASS=<your-password> KEY_ALIAS=tilewords

# 4 APKs, split the same way (needs bundletool: brew install bundletool)
make android-release-apk \
  KEYSTORE=release.keystore KEYSTORE_PASS=<your-password> KEY_ALIAS=tilewords
```

Each of those targets builds four artifacts: one per ABI, plus a universal one holding
every ABI in a single file. To produce only one, name it — `make android-release-arm64-v8a`
for a single-ABI bundle, or `make android-release-universal` for the universal one alone.

#### Bumping the version

The version and build number live in three files, which have to agree:

| File | Version | Build |
| --- | --- | --- |
| `Makefile` | `APP_VERSION` | `APP_BUILD` |
| `FyneApp.toml` | `Version` | `Build` |
| `cmd/tilewords/AndroidManifest.xml` | `android:versionName` | `android:versionCode` |

Nothing in the build checks that they do, so editing one by hand goes unnoticed until a
packaged app reports a version it was not built as, or the Play Store turns an upload
away. `version-bump.sh` is the one thing that moves them together:

```bash
./version-bump.sh -a           # raise the patch level and the build number by one
./version-bump.sh -u           # take the version from the most recent tag, patch + 1
./version-bump.sh -v 0.3.0     # set the version; the build number still rises
./version-bump.sh -b 12        # set the build number; the version holds
./version-bump.sh -i           # be asked for the values
./version-bump.sh -d -v 1.0.0  # show what would change, write nothing
./version-bump.sh -h           # the full usage
```

It has to be told what to do: run with no options, it prints that usage and changes
nothing, so a bump is never something that happens by accident.

It reads the values in force from the Makefile, and refuses anything that would not move
forwards:

- The version may repeat while the build number rises, but it may never fall. Comparison
  follows semver, so `0.10.0` beats `0.9.0` and a prerelease ranks below its release.
- The build number must rise every time, because the Play Store refuses an upload whose
  `versionCode` is not above the last one.
- If the three files already disagree, it reports the disagreement and writes nothing
  rather than guessing which of them is right.

`-u` bases the version on `git describe --tags --abbrev=0` — the most recent tag
reachable from `HEAD` — rather than on the Makefile, which is what to use when the tag
is the release of record. The tag's patch level is raised by one and any prerelease
suffix is dropped, so `v0.2.0-beta` gives `0.2.1`; the build number still rises by one.
No tag, no `git`, or a tag that is not semver is an error, and so is a tag old enough
that following it would move the version backwards.

Nothing is written until all three rewrites have been produced and read back. A pattern
that stops matching, or a file that is read-only, therefore aborts the run with the tree
as it was, rather than leaving some files bumped and others behind.

