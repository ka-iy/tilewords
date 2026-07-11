# Code Summary — Unit 1: `dictionary`

## Files Created

| File | Description |
|---|---|
| `go.mod` | Module `squabble`, Go 1.22, requires `pgregory.net/rapid` v1.2.0 |
| `.gitignore` | Appended: ignores `dictionary/assets/dictionaries/*.txt` (licensed word lists) and standard Go artifacts |
| `dictionary/assets/dictionaries/.gitkeep` | Placeholder so the assets directory is tracked in git while .gob files (built by `go generate`) are not |
| `dictionary/doc.go` | Package-level GoDoc with full Appel-Jacobson citation, GADDAG string-insertion summary, usage example, and `//go:generate` directive |
| `dictionary/names.go` | `DictName` type; constants `DictCSW`, `DictSOWPODS`, `DictOSPD`, `DictNASPA`, `DictOTCWL`, `DictAll`; `AllDictNames` slice; `DisplayName()` |
| `dictionary/gaddag.go` | `NodeID`, `GADDAG` struct, `loadGADDAG()`, `Root()`, `Successor()`, `IsTerminal()`, `contains()` (3-tier fast-path), `Build()` (Appel-Jacobson §3–4 GADDAG construction), `dedup()`, `toUpper()` |
| `dictionary/dictionary.go` | `Dictionary` struct, `Name()`, `WordCount()`, `Validate()` (case-insensitive), `GADDAG()` |
| `dictionary/loader.go` | `//go:embed all:assets/dictionaries`, `embeddedAssets embed.FS`, `assetPath()`, `Load(names ...DictName)` |
| `tools/buildgaddag/main.go` | CLI: flags `-input`, `-output`, `-name`; reads/normalises/deduplicates word lists; calls `dictionary.Build()`; atomic write (temp → rename) |
| `dictionary/testhelpers_test.go` | `TestMain` builds test GADDAG from `testWords` in memory; `wordFromDictGen`, `randomAlphaGen`, `invalidStringGen` rapid generators |
| `dictionary/dictionary_test.go` | 8 example-based tests + 6 PBT tests (round-trip, oracle, case-invariance, invalid-rejection, dedup-invariant, idempotent-load) |

## Build Tool Usage

The GADDAG assets must be built before compiling for production. For each dictionary:

```sh
go run ./tools/buildgaddag \
    -input /path/to/wordlist.txt \
    -output dictionary/assets/dictionaries/naspa.gob \
    -name naspa
```

For the combined "all" dictionary, pass comma-separated input paths:

```sh
go run ./tools/buildgaddag \
    -input csw.txt,sowpods.txt,ospd.txt,naspa.txt,otcwl.txt \
    -output dictionary/assets/dictionaries/all.gob \
    -name all
```

Or trigger all via:

```sh
go generate ./dictionary/...
```

(after updating the `//go:generate` line in `dictionary/doc.go` with actual word list paths)

## Test Fixture Note

Unit tests do NOT require pre-built `.gob` assets. `TestMain` constructs an in-memory GADDAG
from the embedded `testWords` slice using `Build()` directly. This means:

- `go test ./dictionary/...` works out of the box with no word list files.
- Integration tests that exercise `Load(DictName)` (loading embedded assets) require `go generate` to be run first.

## Test Results

All tests pass:

```
ok  squabble/dictionary  1.940s
```

Race detector enabled (`-race`). 8 example tests + 6 PBT property checks executed.
